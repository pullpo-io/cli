package login

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

const tokenUser = "x-access-token"

type config interface {
	Token(string) (string, string)
	User(string) (string, error)
}

type CredentialOptions struct {
	IO     *iostreams.IOStreams
	Config func() (config, error)

	Operation string
}

func NewCmdCredential(f *cmdutil.Factory, runF func(*CredentialOptions) error) *cobra.Command {
	opts := &CredentialOptions{
		IO: f.IOStreams,
		Config: func() (config, error) {
			cfg, err := f.Config()
			if err != nil {
				return nil, err
			}
			return cfg.Authentication(), nil
		},
	}

	cmd := &cobra.Command{
		Use:    "git-credential",
		Args:   cobra.ExactArgs(1),
		Short:  "Implements git credential helper protocol",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Operation = args[0]

			if runF != nil {
				return runF(opts)
			}
			return helperRun(opts)
		},
	}

	return cmd
}

func helperRun(opts *CredentialOptions) error {
	if opts.Operation == "store" {
		// We pretend to implement the "store" operation, but do nothing since we already have a cached token.
		return nil
	}

	if opts.Operation == "erase" {
		// We pretend to implement the "erase" operation, but do nothing since we don't want git to cause user to be logged out.
		return nil
	}

	if opts.Operation != "get" {
		return fmt.Errorf("pullpo auth git-credential: %q operation not supported", opts.Operation)
	}

	wants := map[string]string{}

	s := bufio.NewScanner(opts.IO.In)
	for s.Scan() {
		line := s.Text()
		if line == "" {
			break
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if key == "url" {
			u, err := url.Parse(value)
			if err != nil {
				return err
			}
			wants["protocol"] = u.Scheme
			wants["host"] = u.Host
			wants["path"] = u.Path
			wants["username"] = u.User.Username()
			wants["password"], _ = u.User.Password()
		} else {
			wants[key] = value
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	if wants["protocol"] != "https" {
		return cmdutil.SilentError
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	lookupHost := wants["host"]
	var gotUser string
	gotToken, source := cfg.Token(lookupHost)
	if gotToken == "" && strings.HasPrefix(lookupHost, "gist.") {
		lookupHost = strings.TrimPrefix(lookupHost, "gist.")
		gotToken, source = cfg.Token(lookupHost)
	}

	if strings.HasSuffix(source, "_TOKEN") {
		gotUser = tokenUser
	} else {
		gotUser, _ = cfg.User(lookupHost)
		if gotUser == "" {
			gotUser = tokenUser
		}
	}

	if gotUser == "" || gotToken == "" {
		return cmdutil.SilentError
	}

	if wants["username"] != "" && gotUser != tokenUser && !strings.EqualFold(wants["username"], gotUser) {
		return cmdutil.SilentError
	}

	fmt.Fprint(opts.IO.Out, "protocol=https\n")
	fmt.Fprintf(opts.IO.Out, "host=%s\n", wants["host"])
	fmt.Fprintf(opts.IO.Out, "username=%s\n", gotUser)
	fmt.Fprintf(opts.IO.Out, "password=%s\n", gotToken)

	return nil
}
