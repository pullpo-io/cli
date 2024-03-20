package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/update"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/safeexec"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var updaterEnabled = ""

type exitCode int

const (
	exitOK      exitCode = 0
	exitError   exitCode = 1
	exitCancel  exitCode = 2
	exitAuth    exitCode = 4
	exitPending exitCode = 8
)

func main() {
	code := mainRun()
	os.Exit(int(code))
}

// isExtensionCommand returns true if args resolve to an extension command.

func printError(out io.Writer, err error, cmd *cobra.Command, debug bool) {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		fmt.Fprintf(out, "error connecting to %s\n", dnsError.Name)
		if debug {
			fmt.Fprintln(out, dnsError)
		}
		fmt.Fprintln(out, "check your internet connection or https://githubstatus.com")
		return
	}

	fmt.Fprintln(out, err)

	var flagError *cmdutil.FlagError
	if errors.As(err, &flagError) || strings.HasPrefix(err.Error(), "unknown command ") {
		if !strings.HasSuffix(err.Error(), "\n") {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, cmd.UsageString())
	}
}

func shouldCheckForUpdate() bool {
	if os.Getenv("GH_NO_UPDATE_NOTIFIER") != "" {
		return false
	}
	if os.Getenv("CODESPACES") != "" {
		return false
	}
	return updaterEnabled != "" && !isCI() && isTerminal(os.Stdout) && isTerminal(os.Stderr)
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// based on https://github.com/watson/ci-info/blob/HEAD/index.js
func isCI() bool {
	return os.Getenv("CI") != "" || // GitHub Actions, Travis CI, CircleCI, Cirrus CI, GitLab CI, AppVeyor, CodeShip, dsari
		os.Getenv("BUILD_NUMBER") != "" || // Jenkins, TeamCity
		os.Getenv("RUN_ID") != "" // TaskCluster, dsari
}

func checkForUpdate(ctx context.Context, f *cmdutil.Factory, currentVersion string) (*update.ReleaseInfo, error) {
	if !shouldCheckForUpdate() {
		return nil, nil
	}
	httpClient, err := f.HttpClient()
	if err != nil {
		return nil, err
	}
	repo := updaterEnabled
	stateFilePath := filepath.Join(config.StateDir(), "state.yml")
	return update.CheckForUpdate(ctx, httpClient, stateFilePath, repo, currentVersion)
}

func isRecentRelease(publishedAt time.Time) bool {
	return !publishedAt.IsZero() && time.Since(publishedAt) < time.Hour*24
}

// Check whether the pullpo binary was found under the Homebrew prefix
func isUnderHomebrew(ghBinary string) bool {
	brewExe, err := safeexec.LookPath("brew")
	if err != nil {
		return false
	}

	brewPrefixBytes, err := exec.Command(brewExe, "--prefix").Output()
	if err != nil {
		return false
	}

	brewBinPrefix := filepath.Join(strings.TrimSpace(string(brewPrefixBytes)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(ghBinary, brewBinPrefix)
}
