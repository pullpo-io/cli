package refresh

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/git"
	"github.com/cli/cli/v2/internal/authflow"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/pkg/cmd/auth/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/set"
	"github.com/spf13/cobra"
)

type RefreshOptions struct {
	IO         *iostreams.IOStreams
	Config     func() (config.Config, error)
	HttpClient *http.Client
	GitClient  *git.Client
	Prompter   shared.Prompt

	MainExecutable string

	Hostname     string
	Scopes       []string
	RemoveScopes []string
	ResetScopes  bool
	AuthFlow     func(*config.AuthConfig, *iostreams.IOStreams, string, []string, bool, bool) error

	Interactive     bool
	InsecureStorage bool
}

func NewCmdRefresh(f *cmdutil.Factory, runF func(*RefreshOptions) error) *cobra.Command {
	opts := &RefreshOptions{
		IO:     f.IOStreams,
		Config: f.Config,
		AuthFlow: func(authCfg *config.AuthConfig, io *iostreams.IOStreams, hostname string, scopes []string, interactive, secureStorage bool) error {
			if secureStorage {
				cs := io.ColorScheme()
				fmt.Fprintf(io.ErrOut, "%s Using secure storage could break installed extensions", cs.WarningIcon())
			}
			token, username, err := authflow.AuthFlow(hostname, io, "", scopes, interactive, f.Browser)
			if err != nil {
				return err
			}
			_, loginErr := authCfg.Login(hostname, username, token, "", secureStorage)
			return loginErr
		},
		HttpClient: &http.Client{},
		GitClient:  f.GitClient,
		Prompter:   f.Prompter,
	}

	cmd := &cobra.Command{
		Use:   "refresh",
		Args:  cobra.ExactArgs(0),
		Short: "Refresh stored authentication credentials",
		Long: heredoc.Docf(`Expand or fix the permission scopes for stored credentials.

			The %[1]s--scopes%[1]s flag accepts a comma separated list of scopes you want
			your pullpo credentials to have. If no scopes are provided, the command
			maintains previously added scopes.

			The %[1]s--remove-scopes%[1]s flag accepts a comma separated list of scopes you
			want to remove from your pullpo credentials. Scope removal is idempotent.
			The minimum set of scopes (%[1]srepo%[1]s, %[1]sread:org%[1]s, and %[1]sgist%[1]s) cannot be removed.

			The %[1]s--reset-scopes%[1]s flag resets the scopes for your pullpo credentials to
			the default set of scopes for your auth flow.
		`, "`"),
		Example: heredoc.Doc(`
			$ pullpo auth refresh --scopes write:org,read:public_key
			# => open a browser to add write:org and read:public_key scopes

			$ pullpo auth refresh
			# => open a browser to ensure your authentication credentials have the correct minimum scopes

			$ pullpo auth refresh --remove-scopes delete_repo
			# => open a browser to idempotently remove the delete_repo scope

			$ pullpo auth refresh --reset-scopes
			# => open a browser to re-authenticate with the default minimum scopes
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Interactive = opts.IO.CanPrompt()

			if !opts.Interactive && opts.Hostname == "" {
				return cmdutil.FlagErrorf("--hostname required when not running interactively")
			}

			opts.MainExecutable = f.Executable()
			if runF != nil {
				return runF(opts)
			}
			return refreshRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The GitHub host to use for authentication")
	cmd.Flags().StringSliceVarP(&opts.Scopes, "scopes", "s", nil, "Additional authentication scopes for pullpo to have")
	cmd.Flags().StringSliceVarP(&opts.RemoveScopes, "remove-scopes", "r", nil, "Authentication scopes to remove from gh")
	cmd.Flags().BoolVar(&opts.ResetScopes, "reset-scopes", false, "Reset authentication scopes to the default minimum set of scopes")
	// secure storage became the default on 2023/4/04; this flag is left as a no-op for backwards compatibility
	var secureStorage bool
	cmd.Flags().BoolVar(&secureStorage, "secure-storage", false, "Save authentication credentials in secure credential store")
	_ = cmd.Flags().MarkHidden("secure-storage")

	cmd.Flags().BoolVarP(&opts.InsecureStorage, "insecure-storage", "", false, "Save authentication credentials in plain text instead of credential store")

	return cmd
}

func refreshRun(opts *RefreshOptions) error {
	cfg, err := opts.Config()
	if err != nil {
		return err
	}
	authCfg := cfg.Authentication()

	candidates := authCfg.Hosts()
	if len(candidates) == 0 {
		return fmt.Errorf("not logged in to any hosts. Use 'pullpo auth login' to authenticate with a host")
	}

	hostname := opts.Hostname
	if hostname == "" {
		if len(candidates) == 1 {
			hostname = candidates[0]
		} else {
			selected, err := opts.Prompter.Select("What account do you want to refresh auth for?", "", candidates)
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}
			hostname = candidates[selected]
		}
	} else {
		var found bool
		for _, c := range candidates {
			if c == hostname {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("not logged in to %s. use 'pullpo auth login' to authenticate with this host", hostname)
		}
	}

	if src, writeable := shared.AuthTokenWriteable(authCfg, hostname); !writeable {
		fmt.Fprintf(opts.IO.ErrOut, "The value of the %s environment variable is being used for authentication.\n", src)
		fmt.Fprint(opts.IO.ErrOut, "To refresh credentials stored in GitHub CLI, first clear the value from the environment.\n")
		return cmdutil.SilentError
	}

	additionalScopes := set.NewStringSet()

	if !opts.ResetScopes {
		if oldToken, _ := authCfg.Token(hostname); oldToken != "" {
			if oldScopes, err := shared.GetScopes(opts.HttpClient, hostname, oldToken); err == nil {
				for _, s := range strings.Split(oldScopes, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						additionalScopes.Add(s)
					}
				}
			}
		}
	}

	credentialFlow := &shared.GitCredentialFlow{
		Executable: opts.MainExecutable,
		Prompter:   opts.Prompter,
		GitClient:  opts.GitClient,
	}
	gitProtocol := cfg.GitProtocol(hostname)
	if opts.Interactive && gitProtocol == "https" {
		if err := credentialFlow.Prompt(hostname); err != nil {
			return err
		}
		additionalScopes.AddValues(credentialFlow.Scopes())
	}

	additionalScopes.AddValues(opts.Scopes)

	additionalScopes.RemoveValues(opts.RemoveScopes)

	if err := opts.AuthFlow(authCfg, opts.IO, hostname, additionalScopes.ToSlice(), opts.Interactive, !opts.InsecureStorage); err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	fmt.Fprintf(opts.IO.ErrOut, "%s Authentication complete.\n", cs.SuccessIcon())

	if credentialFlow.ShouldSetup() {
		username, _ := authCfg.User(hostname)
		password, _ := authCfg.Token(hostname)
		if err := credentialFlow.Setup(hostname, username, password); err != nil {
			return err
		}
	}

	return nil
}
