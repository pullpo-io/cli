package extension

import (
	"errors"
	"fmt"
	gio "io"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/git"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/extension/browse"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/extensions"
	"github.com/cli/cli/v2/pkg/search"
	"github.com/spf13/cobra"
)

var alreadyInstalledError = errors.New("alreadyInstalledError")

func NewCmdExtension(f *cmdutil.Factory) *cobra.Command {
	m := f.ExtensionManager
	io := f.IOStreams
	gc := f.GitClient
	prompter := f.Prompter
	config := f.Config
	browser := f.Browser
	httpClient := f.HttpClient

	extCmd := cobra.Command{
		Use:   "extension",
		Short: "Manage pullpo extensions",
		Long: heredoc.Docf(`
			GitHub CLI extensions are repositories that provide additional pullpo commands.

			The name of the extension repository must start with %[1]sgh-%[1]s and it must contain an
			executable of the same name. All arguments passed to the %[1]spullpo <extname>%[1]s invocation
			will be forwarded to the %[1]sgh-<extname>%[1]s executable of the extension.

			An extension cannot override any of the core pullpo commands. If an extension name conflicts
			with a core pullpo command, you can use %[1]spullpo extension exec <extname>%[1]s.

			For the list of available extensions, see <https://github.com/topics/gh-extension>.
		`, "`"),
		Aliases: []string{"extensions", "ext"},
	}

	upgradeFunc := func(name string, flagForce, flagDryRun bool) error {
		cs := io.ColorScheme()
		err := m.Upgrade(name, flagForce)
		if err != nil {
			if name != "" {
				fmt.Fprintf(io.ErrOut, "%s Failed upgrading extension %s: %s\n", cs.FailureIcon(), name, err)
			} else if errors.Is(err, noExtensionsInstalledError) {
				return cmdutil.NewNoResultsError("no installed extensions found")
			} else {
				fmt.Fprintf(io.ErrOut, "%s Failed upgrading extensions\n", cs.FailureIcon())
			}
			return cmdutil.SilentError
		}
		if io.IsStdoutTTY() {
			successStr := "Successfully"
			if flagDryRun {
				successStr = "Would have"
			}
			extensionStr := "extension"
			if name == "" {
				extensionStr = "extensions"
			}
			fmt.Fprintf(io.Out, "%s %s upgraded %s\n", cs.SuccessIcon(), successStr, extensionStr)
		}
		return nil
	}

	extCmd.AddCommand(
		func() *cobra.Command {
			query := search.Query{
				Kind: search.KindRepositories,
			}
			qualifiers := search.Qualifiers{
				Topic: []string{"gh-extension"},
			}
			var order string
			var sort string
			var webMode bool
			var exporter cmdutil.Exporter

			cmd := &cobra.Command{
				Use:   "search [<query>]",
				Short: "Search extensions to the GitHub CLI",
				Long: heredoc.Docf(`
					Search for pullpo extensions.

					With no arguments, this command prints out the first 30 extensions
					available to install sorted by number of stars. More extensions can
					be fetched by specifying a higher limit with the %[1]s--limit%[1]s flag.

					When connected to a terminal, this command prints out three columns.
					The first has a ✓ if the extension is already installed locally. The
					second is the full name of the extension repository in %[1]sOWNER/REPO%[1]s
					format. The third is the extension's description.

					When not connected to a terminal, the ✓ character is rendered as the
					word "installed" but otherwise the order and content of the columns
					are the same.

					This command behaves similarly to %[1]spullpo search repos%[1]s but does not
					support as many search qualifiers. For a finer grained search of
					extensions, try using:

						pullpo search repos --topic "gh-extension"

					and adding qualifiers as needed. See %[1]spullpo help search repos%[1]s to learn
					more about repository search.

					For listing just the extensions that are already installed locally,
					see:

						pullpo ext list
				`, "`"),
				Example: heredoc.Doc(`
					# List the first 30 extensions sorted by star count, descending
					$ pullpo ext search

					# List more extensions
					$ pullpo ext search --limit 300

					# List extensions matching the term "branch"
					$ pullpo ext search branch

					# List extensions owned by organization "github"
					$ pullpo ext search --owner github

					# List extensions, sorting by recently updated, ascending
					$ pullpo ext search --sort updated --order asc

					# List extensions, filtering by license
					$ pullpo ext search --license MIT

					# Open search results in the browser
					$ pullpo ext search -w
				`),
				RunE: func(cmd *cobra.Command, args []string) error {
					cfg, err := config()
					if err != nil {
						return err
					}
					client, err := httpClient()
					if err != nil {
						return err
					}

					if cmd.Flags().Changed("order") {
						query.Order = order
					}
					if cmd.Flags().Changed("sort") {
						query.Sort = sort
					}

					query.Keywords = args
					query.Qualifiers = qualifiers

					host, _ := cfg.Authentication().DefaultHost()
					searcher := search.NewSearcher(client, host)

					if webMode {
						url := searcher.URL(query)
						if io.IsStdoutTTY() {
							fmt.Fprintf(io.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(url))
						}
						return browser.Browse(url)
					}

					io.StartProgressIndicator()
					result, err := searcher.Repositories(query)
					io.StopProgressIndicator()
					if err != nil {
						return err
					}

					if exporter != nil {
						return exporter.Write(io, result.Items)
					}

					if io.IsStdoutTTY() {
						if len(result.Items) == 0 {
							return errors.New("no extensions found")
						}
						fmt.Fprintf(io.Out, "Showing %d of %d extensions\n", len(result.Items), result.Total)
						fmt.Fprintln(io.Out)
					}

					cs := io.ColorScheme()
					installedExts := m.List()

					isInstalled := func(repo search.Repository) bool {
						searchRepo, err := ghrepo.FromFullName(repo.FullName)
						if err != nil {
							return false
						}
						for _, e := range installedExts {
							// TODO consider a Repo() on Extension interface
							if u, err := git.ParseURL(e.URL()); err == nil {
								if r, err := ghrepo.FromURL(u); err == nil {
									if ghrepo.IsSame(searchRepo, r) {
										return true
									}
								}
							}
						}
						return false
					}

					tp := tableprinter.New(io, tableprinter.WithHeader("", "REPO", "DESCRIPTION"))
					for _, repo := range result.Items {
						if !strings.HasPrefix(repo.Name, "gh-") {
							continue
						}

						installed := ""
						if isInstalled(repo) {
							if io.IsStdoutTTY() {
								installed = "✓"
							} else {
								installed = "installed"
							}
						}

						tp.AddField(installed, tableprinter.WithColor(cs.Green))
						tp.AddField(repo.FullName, tableprinter.WithColor(cs.Bold))
						tp.AddField(repo.Description)
						tp.EndRow()
					}

					return tp.Render()
				},
			}

			// Output flags
			cmd.Flags().BoolVarP(&webMode, "web", "w", false, "Open the search query in the web browser")
			cmdutil.AddJSONFlags(cmd, &exporter, search.RepositoryFields)

			// Query parameter flags
			cmd.Flags().IntVarP(&query.Limit, "limit", "L", 30, "Maximum number of extensions to fetch")
			cmdutil.StringEnumFlag(cmd, &order, "order", "", "desc", []string{"asc", "desc"}, "Order of repositories returned, ignored unless '--sort' flag is specified")
			cmdutil.StringEnumFlag(cmd, &sort, "sort", "", "best-match", []string{"forks", "help-wanted-issues", "stars", "updated"}, "Sort fetched repositories")

			// Qualifier flags
			cmd.Flags().StringSliceVar(&qualifiers.License, "license", nil, "Filter based on license type")
			cmd.Flags().StringSliceVar(&qualifiers.User, "owner", nil, "Filter on owner")

			return cmd
		}(),
		&cobra.Command{
			Use:     "list",
			Short:   "List installed extension commands",
			Aliases: []string{"ls"},
			Args:    cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				cmds := m.List()
				if len(cmds) == 0 {
					return cmdutil.NewNoResultsError("no installed extensions found")
				}
				cs := io.ColorScheme()
				t := tableprinter.New(io, tableprinter.WithHeader("NAME", "REPO", "VERSION"))
				for _, c := range cmds {
					// TODO consider a Repo() on Extension interface
					var repo string
					if u, err := git.ParseURL(c.URL()); err == nil {
						if r, err := ghrepo.FromURL(u); err == nil {
							repo = ghrepo.FullName(r)
						}
					}

					t.AddField(fmt.Sprintf("pullpo %s", c.Name()))
					t.AddField(repo)
					version := displayExtensionVersion(c, c.CurrentVersion())
					if c.IsPinned() {
						t.AddField(version, tableprinter.WithColor(cs.Cyan))
					} else {
						t.AddField(version)
					}

					t.EndRow()
				}
				return t.Render()
			},
		},
		func() *cobra.Command {
			var forceFlag bool
			var pinFlag string
			cmd := &cobra.Command{
				Use:   "install <repository>",
				Short: "Install a pullpo extension from a repository",
				Long: heredoc.Docf(`
					Install a GitHub repository locally as a GitHub CLI extension.

					The repository argument can be specified in %[1]sOWNER/REPO%[1]s format as well as a full URL.
					The URL format is useful when the repository is not hosted on github.com.

					To install an extension in development from the current directory, use %[1]s.%[1]s as the
					value of the repository argument.

					For the list of available extensions, see <https://github.com/topics/gh-extension>.
				`, "`"),
				Example: heredoc.Doc(`
					$ pullpo extension install owner/gh-extension
					$ pullpo extension install https://git.example.com/owner/gh-extension
					$ pullpo extension install .
				`),
				Args: cmdutil.MinimumArgs(1, "must specify a repository to install from"),
				RunE: func(cmd *cobra.Command, args []string) error {
					if args[0] == "." {
						if pinFlag != "" {
							return fmt.Errorf("local extensions cannot be pinned")
						}
						wd, err := os.Getwd()
						if err != nil {
							return err
						}
						return m.InstallLocal(wd)
					}

					repo, err := ghrepo.FromFullName(args[0])
					if err != nil {
						return err
					}

					cs := io.ColorScheme()

					if ext, err := checkValidExtension(cmd.Root(), m, repo.RepoName(), repo.RepoOwner()); err != nil {
						// If an existing extension was found and --force was specified, attempt to upgrade.
						if forceFlag && ext != nil {
							return upgradeFunc(ext.Name(), forceFlag, false)
						}

						if errors.Is(err, alreadyInstalledError) {
							fmt.Fprintf(io.ErrOut, "%s Extension %s is already installed\n", cs.WarningIcon(), ghrepo.FullName(repo))
							return nil
						}

						return err
					}

					io.StartProgressIndicator()
					err = m.Install(repo, pinFlag)
					io.StopProgressIndicator()

					if err != nil {
						if errors.Is(err, releaseNotFoundErr) {
							return fmt.Errorf("%s Could not find a release of %s for %s",
								cs.FailureIcon(), args[0], cs.Cyan(pinFlag))
						} else if errors.Is(err, commitNotFoundErr) {
							return fmt.Errorf("%s %s does not exist in %s",
								cs.FailureIcon(), cs.Cyan(pinFlag), args[0])
						} else if errors.Is(err, repositoryNotFoundErr) {
							return fmt.Errorf("%s Could not find extension '%s' on host %s",
								cs.FailureIcon(), args[0], repo.RepoHost())
						}
						return err
					}

					if io.IsStdoutTTY() {
						fmt.Fprintf(io.Out, "%s Installed extension %s\n", cs.SuccessIcon(), args[0])
						if pinFlag != "" {
							fmt.Fprintf(io.Out, "%s Pinned extension at %s\n", cs.SuccessIcon(), cs.Cyan(pinFlag))
						}
					}
					return nil
				},
			}
			cmd.Flags().BoolVar(&forceFlag, "force", false, "force upgrade extension, or ignore if latest already installed")
			cmd.Flags().StringVar(&pinFlag, "pin", "", "pin extension to a release tag or commit ref")
			return cmd
		}(),
		func() *cobra.Command {
			var flagAll bool
			var flagForce bool
			var flagDryRun bool
			cmd := &cobra.Command{
				Use:   "upgrade {<name> | --all}",
				Short: "Upgrade installed extensions",
				Args: func(cmd *cobra.Command, args []string) error {
					if len(args) == 0 && !flagAll {
						return cmdutil.FlagErrorf("specify an extension to upgrade or `--all`")
					}
					if len(args) > 0 && flagAll {
						return cmdutil.FlagErrorf("cannot use `--all` with extension name")
					}
					if len(args) > 1 {
						return cmdutil.FlagErrorf("too many arguments")
					}
					return nil
				},
				RunE: func(cmd *cobra.Command, args []string) error {
					var name string
					if len(args) > 0 {
						name = normalizeExtensionSelector(args[0])
					}
					if flagDryRun {
						m.EnableDryRunMode()
					}
					return upgradeFunc(name, flagForce, flagDryRun)
				},
			}
			cmd.Flags().BoolVar(&flagAll, "all", false, "Upgrade all extensions")
			cmd.Flags().BoolVar(&flagForce, "force", false, "Force upgrade extension")
			cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Only display upgrades")
			return cmd
		}(),
		&cobra.Command{
			Use:   "remove <name>",
			Short: "Remove an installed extension",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				extName := normalizeExtensionSelector(args[0])
				if err := m.Remove(extName); err != nil {
					return err
				}
				if io.IsStdoutTTY() {
					cs := io.ColorScheme()
					fmt.Fprintf(io.Out, "%s Removed extension %s\n", cs.SuccessIcon(), extName)
				}
				return nil
			},
		},
		func() *cobra.Command {
			var debug bool
			var singleColumn bool
			cmd := &cobra.Command{
				Use:   "browse",
				Short: "Enter a UI for browsing, adding, and removing extensions",
				Long: heredoc.Docf(`
					This command will take over your terminal and run a fully interactive
					interface for browsing, adding, and removing pullpo extensions. A terminal
					width greater than 100 columns is recommended.

					To learn how to control this interface, press %[1]s?%[1]s after running to see
					the help text.

					Press %[1]sq%[1]s to quit.

					Running this command with %[1]s--single-column%[1]s should make this command
					more intelligible for users who rely on assistive technology like screen
					readers or hipullpo zoom.

					For a more traditional way to discover extensions, see:

						pullpo ext search

					along with %[1]spullpo ext install%[1]s, %[1]spullpo ext remove%[1]s, and %[1]spullpo repo view%[1]s.
				`, "`"),
				Args: cobra.NoArgs,
				RunE: func(cmd *cobra.Command, args []string) error {
					if !io.CanPrompt() {
						return errors.New("this command runs an interactive UI and needs to be run in a terminal")
					}
					cfg, err := config()
					if err != nil {
						return err
					}
					host, _ := cfg.Authentication().DefaultHost()
					client, err := f.HttpClient()
					if err != nil {
						return err
					}

					searcher := search.NewSearcher(api.NewCachedHTTPClient(client, time.Hour*24), host)

					gc.Stderr = gio.Discard

					opts := browse.ExtBrowseOpts{
						Cmd:          cmd,
						IO:           io,
						Browser:      browser,
						Searcher:     searcher,
						Em:           m,
						Client:       client,
						Cfg:          cfg,
						Debug:        debug,
						SingleColumn: singleColumn,
					}

					return browse.ExtBrowse(opts)
				},
			}
			cmd.Flags().BoolVar(&debug, "debug", false, "log to /tmp/extBrowse-*")
			cmd.Flags().BoolVarP(&singleColumn, "single-column", "s", false, "Render TUI with only one column of text")
			return cmd
		}(),
		&cobra.Command{
			Use:   "exec <name> [args]",
			Short: "Execute an installed extension",
			Long: heredoc.Docf(`
				Execute an extension using the short name. For example, if the extension repository is
				%[1]sowner/gh-extension%[1]s, you should pass %[1]sextension%[1]s. You can use this command when
				the short name conflicts with a core pullpo command.

				All arguments after the extension name will be forwarded to the executable
				of the extension.
			`, "`"),
			Example: heredoc.Doc(`
				# execute a label extension instead of the core pullpo label command
				$ pullpo extension exec label
			`),
			Args:               cobra.MinimumNArgs(1),
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				if found, err := m.Dispatch(args, io.In, io.Out, io.ErrOut); !found {
					return fmt.Errorf("extension %q not found", args[0])
				} else {
					return err
				}
			},
		},
		func() *cobra.Command {
			promptCreate := func() (string, extensions.ExtTemplateType, error) {
				extName, err := prompter.Input("Extension name:", "")
				if err != nil {
					return extName, -1, err
				}
				options := []string{"Script (Bash, Ruby, Python, etc)", "Go", "Other Precompiled (C++, Rust, etc)"}
				extTmplType, err := prompter.Select("What kind of extension?",
					options[0],
					options)
				return extName, extensions.ExtTemplateType(extTmplType), err
			}
			var flagType string
			cmd := &cobra.Command{
				Use:   "create [<name>]",
				Short: "Create a new extension",
				Example: heredoc.Doc(`
					# Use interactively
					pullpo extension create

					# Create a script-based extension
					pullpo extension create foobar

					# Create a Go extension
					pullpo extension create --precompiled=go foobar

					# Create a non-Go precompiled extension
					pullpo extension create --precompiled=other foobar
				`),
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					if cmd.Flags().Changed("precompiled") {
						if flagType != "go" && flagType != "other" {
							return cmdutil.FlagErrorf("value for --precompiled must be 'go' or 'other'. Got '%s'", flagType)
						}
					}
					var extName string
					var err error
					tmplType := extensions.GitTemplateType
					if len(args) == 0 {
						if io.IsStdoutTTY() {
							extName, tmplType, err = promptCreate()
							if err != nil {
								return fmt.Errorf("could not prompt: %w", err)
							}
						}
					} else {
						extName = args[0]
						if flagType == "go" {
							tmplType = extensions.GoBinTemplateType
						} else if flagType == "other" {
							tmplType = extensions.OtherBinTemplateType
						}
					}

					var fullName string

					if strings.HasPrefix(extName, "gh-") {
						fullName = extName
						extName = extName[3:]
					} else {
						fullName = "gh-" + extName
					}

					cs := io.ColorScheme()

					commitIcon := cs.SuccessIcon()
					if err := m.Create(fullName, tmplType); err != nil {
						if errors.Is(err, ErrInitialCommitFailed) {
							commitIcon = cs.FailureIcon()
						} else {
							return err
						}
					}

					if !io.IsStdoutTTY() {
						return nil
					}

					var goBinChecks string

					steps := fmt.Sprintf(
						"- run 'cd %[1]s; pullpo extension install .; pullpo %[2]s' to see your new extension in action",
						fullName, extName)

					if tmplType == extensions.GoBinTemplateType {
						goBinChecks = heredoc.Docf(`
						%[1]s Downloaded Go dependencies
						%[1]s Built %[2]s binary
						`, cs.SuccessIcon(), fullName)
						steps = heredoc.Docf(`
						- run 'cd %[1]s; pullpo extension install .; pullpo %[2]s' to see your new extension in action
						- run 'go build && pullpo %[2]s' to see changes in your code as you develop`, fullName, extName)
					} else if tmplType == extensions.OtherBinTemplateType {
						steps = heredoc.Docf(`
						- run 'cd %[1]s; pullpo extension install .' to install your extension locally
						- fill in script/build.sh with your compilation script for automated builds
						- compile a %[1]s binary locally and run 'pullpo %[2]s' to see changes`, fullName, extName)
					}
					link := "https://docs.github.com/github-cli/github-cli/creating-github-cli-extensions"
					out := heredoc.Docf(`
						%[1]s Created directory %[2]s
						%[1]s Initialized git repository
						%[7]s Made initial commit
						%[1]s Set up extension scaffolding
						%[6]s
						%[2]s is ready for development!

						%[4]s
						%[5]s
						- run 'pullpo repo create' to share your extension with others

						For more information on writing extensions:
						%[3]s
					`, cs.SuccessIcon(), fullName, link, cs.Bold("Next Steps"), steps, goBinChecks, commitIcon)
					fmt.Fprint(io.Out, out)
					return nil
				},
			}
			cmd.Flags().StringVar(&flagType, "precompiled", "", "Create a precompiled extension. Possible values: go, other")
			return cmd
		}(),
	)

	return &extCmd
}

func checkValidExtension(rootCmd *cobra.Command, m extensions.ExtensionManager, extName, extOwner string) (extensions.Extension, error) {
	if !strings.HasPrefix(extName, "gh-") {
		return nil, errors.New("extension repository name must start with `gh-`")
	}

	commandName := strings.TrimPrefix(extName, "gh-")
	if c, _, _ := rootCmd.Find([]string{commandName}); c != rootCmd && c.GroupID != "extension" {
		return nil, fmt.Errorf("%q matches the name of a built-in command or alias", commandName)
	}

	for _, ext := range m.List() {
		if ext.Name() == commandName {
			if ext.Owner() == extOwner {
				return ext, alreadyInstalledError
			}
			return ext, fmt.Errorf("there is already an installed extension that provides the %q command", commandName)
		}
	}

	return nil, nil
}

func normalizeExtensionSelector(n string) string {
	if idx := strings.IndexRune(n, '/'); idx >= 0 {
		n = n[idx+1:]
	}
	return strings.TrimPrefix(n, "gh-")
}

func displayExtensionVersion(ext extensions.Extension, version string) string {
	if !ext.IsBinary() && len(version) > 8 {
		return version[:8]
	}
	return version
}
