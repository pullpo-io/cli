package browse

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cli/cli/v2/git"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdBrowse(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		factory  func(*cmdutil.Factory) *cmdutil.Factory
		wants    BrowseOptions
		wantsErr bool
	}{
		{
			name:     "no arguments",
			cli:      "",
			wantsErr: false,
		},
		{
			name: "settings flag",
			cli:  "--settings",
			wants: BrowseOptions{
				SettingsFlag: true,
			},
			wantsErr: false,
		},
		{
			name: "projects flag",
			cli:  "--projects",
			wants: BrowseOptions{
				ProjectsFlag: true,
			},
			wantsErr: false,
		},
		{
			name: "releases flag",
			cli:  "--releases",
			wants: BrowseOptions{
				ReleasesFlag: true,
			},
			wantsErr: false,
		},
		{
			name: "wiki flag",
			cli:  "--wiki",
			wants: BrowseOptions{
				WikiFlag: true,
			},
			wantsErr: false,
		},
		{
			name: "no browser flag",
			cli:  "--no-browser",
			wants: BrowseOptions{
				NoBrowserFlag: true,
			},
			wantsErr: false,
		},
		{
			name: "branch flag",
			cli:  "--branch main",
			wants: BrowseOptions{
				Branch: "main",
			},
			wantsErr: false,
		},
		{
			name:     "branch flag without a branch name",
			cli:      "--branch",
			wantsErr: true,
		},
		{
			name: "combination: settings projects",
			cli:  "--settings --projects",
			wants: BrowseOptions{
				SettingsFlag: true,
				ProjectsFlag: true,
			},
			wantsErr: true,
		},
		{
			name: "combination: projects wiki",
			cli:  "--projects --wiki",
			wants: BrowseOptions{
				ProjectsFlag: true,
				WikiFlag:     true,
			},
			wantsErr: true,
		},
		{
			name: "passed argument",
			cli:  "main.go",
			wants: BrowseOptions{
				SelectorArg: "main.go",
			},
			wantsErr: false,
		},
		{
			name:     "passed two arguments",
			cli:      "main.go main.go",
			wantsErr: true,
		},
		{
			name:     "passed argument and projects flag",
			cli:      "main.go --projects",
			wantsErr: true,
		},
		{
			name:     "passed argument and releases flag",
			cli:      "main.go --releases",
			wantsErr: true,
		},
		{
			name:     "passed argument and settings flag",
			cli:      "main.go --settings",
			wantsErr: true,
		},
		{
			name:     "passed argument and wiki flag",
			cli:      "main.go --wiki",
			wantsErr: true,
		},
		{
			name: "empty commit flag",
			cli:  "--commit",
			wants: BrowseOptions{
				Commit: emptyCommitFlag,
			},
			wantsErr: false,
		},
		{
			name: "commit flag with a hash",
			cli:  "--commit=12a4",
			wants: BrowseOptions{
				Commit: "12a4",
			},
			wantsErr: false,
		},
		{
			name: "commit flag with a hash and a file selector",
			cli:  "main.go --commit=12a4",
			wants: BrowseOptions{
				Commit:      "12a4",
				SelectorArg: "main.go",
			},
			wantsErr: false,
		},
		{
			name:     "passed both branch and commit flags",
			cli:      "main.go --branch main --commit=12a4",
			wantsErr: true,
		},
		{
			name:     "passed both number arg and branch flag",
			cli:      "1 --branch trunk",
			wantsErr: true,
		},
		{
			name:     "passed both number arg and commit flag",
			cli:      "1 --commit=12a4",
			wantsErr: true,
		},
		{
			name:     "passed both commit SHA arg and branch flag",
			cli:      "de07febc26e19000f8c9e821207f3bc34a3c8038 --branch trunk",
			wantsErr: true,
		},
		{
			name:     "passed both commit SHA arg and commit flag",
			cli:      "de07febc26e19000f8c9e821207f3bc34a3c8038 --commit=12a4",
			wantsErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmdutil.Factory{}
			var opts *BrowseOptions
			cmd := NewCmdBrowse(&f, func(o *BrowseOptions) error {
				opts = o
				return nil
			})
			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)
			cmd.SetArgs(argv)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			_, err = cmd.ExecuteC()

			if tt.wantsErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wants.Branch, opts.Branch)
			assert.Equal(t, tt.wants.SelectorArg, opts.SelectorArg)
			assert.Equal(t, tt.wants.ProjectsFlag, opts.ProjectsFlag)
			assert.Equal(t, tt.wants.ReleasesFlag, opts.ReleasesFlag)
			assert.Equal(t, tt.wants.WikiFlag, opts.WikiFlag)
			assert.Equal(t, tt.wants.NoBrowserFlag, opts.NoBrowserFlag)
			assert.Equal(t, tt.wants.SettingsFlag, opts.SettingsFlag)
			assert.Equal(t, tt.wants.Commit, opts.Commit)
		})
	}
}

type testGitClient struct{}

func (gc *testGitClient) LastCommit() (*git.Commit, error) {
	return &git.Commit{Sha: "6f1a2405cace1633d89a79c74c65f22fe78f9659"}, nil
}

func Test_runBrowse(t *testing.T) {
	s := string(os.PathSeparator)
	t.Setenv("GIT_DIR", "../../../git/fixtures/simple.git")
	tests := []struct {
		name          string
		opts          BrowseOptions
		httpStub      func(*httpmock.Registry)
		baseRepo      ghrepo.Interface
		defaultBranch string
		expectedURL   string
		wantsErr      bool
	}{
		{
			name: "no arguments",
			opts: BrowseOptions{
				SelectorArg: "",
			},
			baseRepo:    ghrepo.New("jlsestak", "cli"),
			expectedURL: "https://github.com/jlsestak/cli",
		},
		{
			name: "settings flag",
			opts: BrowseOptions{
				SettingsFlag: true,
			},
			baseRepo:    ghrepo.New("bchadwic", "ObscuredByClouds"),
			expectedURL: "https://github.com/bchadwic/ObscuredByClouds/settings",
		},
		{
			name: "projects flag",
			opts: BrowseOptions{
				ProjectsFlag: true,
			},
			baseRepo:    ghrepo.New("ttran112", "7ate9"),
			expectedURL: "https://github.com/ttran112/7ate9/projects",
		},
		{
			name: "releases flag",
			opts: BrowseOptions{
				ReleasesFlag: true,
			},
			baseRepo:    ghrepo.New("ttran112", "7ate9"),
			expectedURL: "https://github.com/ttran112/7ate9/releases",
		},
		{
			name: "wiki flag",
			opts: BrowseOptions{
				WikiFlag: true,
			},
			baseRepo:    ghrepo.New("ravocean", "ThreatLevelMidnight"),
			expectedURL: "https://github.com/ravocean/ThreatLevelMidnight/wiki",
		},
		{
			name:          "file argument",
			opts:          BrowseOptions{SelectorArg: "path/to/file.txt"},
			baseRepo:      ghrepo.New("ken", "mrprofessor"),
			defaultBranch: "main",
			expectedURL:   "https://github.com/ken/mrprofessor/tree/main/path/to/file.txt",
		},
		{
			name: "issue argument",
			opts: BrowseOptions{
				SelectorArg: "217",
			},
			baseRepo:    ghrepo.New("kevin", "MinTy"),
			expectedURL: "https://github.com/kevin/MinTy/issues/217",
		},
		{
			name: "issue with hashtag argument",
			opts: BrowseOptions{
				SelectorArg: "#217",
			},
			baseRepo:    ghrepo.New("kevin", "MinTy"),
			expectedURL: "https://github.com/kevin/MinTy/issues/217",
		},
		{
			name: "branch flag",
			opts: BrowseOptions{
				Branch: "trunk",
			},
			baseRepo:    ghrepo.New("jlsestak", "CouldNotThinkOfARepoName"),
			expectedURL: "https://github.com/jlsestak/CouldNotThinkOfARepoName/tree/trunk",
		},
		{
			name: "branch flag with file",
			opts: BrowseOptions{
				Branch:      "trunk",
				SelectorArg: "main.go",
			},
			baseRepo:    ghrepo.New("bchadwic", "LedZeppelinIV"),
			expectedURL: "https://github.com/bchadwic/LedZeppelinIV/tree/trunk/main.go",
		},
		{
			name: "branch flag within dir",
			opts: BrowseOptions{
				Branch:           "feature-123",
				PathFromRepoRoot: func() string { return "pkg/dir" },
			},
			baseRepo:    ghrepo.New("bstnc", "yeepers"),
			expectedURL: "https://github.com/bstnc/yeepers/tree/feature-123",
		},
		{
			name: "branch flag within dir with .",
			opts: BrowseOptions{
				Branch:           "feature-123",
				SelectorArg:      ".",
				PathFromRepoRoot: func() string { return "pkg/dir" },
			},
			baseRepo:    ghrepo.New("bstnc", "yeepers"),
			expectedURL: "https://github.com/bstnc/yeepers/tree/feature-123/pkg/dir",
		},
		{
			name: "branch flag within dir with dir",
			opts: BrowseOptions{
				Branch:           "feature-123",
				SelectorArg:      "inner/more",
				PathFromRepoRoot: func() string { return "pkg/dir" },
			},
			baseRepo:    ghrepo.New("bstnc", "yeepers"),
			expectedURL: "https://github.com/bstnc/yeepers/tree/feature-123/pkg/dir/inner/more",
		},
		{
			name: "file with line number",
			opts: BrowseOptions{
				SelectorArg: "path/to/file.txt:32",
			},
			baseRepo:      ghrepo.New("ravocean", "angur"),
			defaultBranch: "trunk",
			expectedURL:   "https://github.com/ravocean/angur/blob/trunk/path/to/file.txt?plain=1#L32",
		},
		{
			name: "file with line range",
			opts: BrowseOptions{
				SelectorArg: "path/to/file.txt:32-40",
			},
			baseRepo:      ghrepo.New("ravocean", "angur"),
			defaultBranch: "trunk",
			expectedURL:   "https://github.com/ravocean/angur/blob/trunk/path/to/file.txt?plain=1#L32-L40",
		},
		{
			name: "invalid default branch",
			opts: BrowseOptions{
				SelectorArg: "chocolate-pecan-pie.txt",
			},
			baseRepo:      ghrepo.New("andrewhsu", "recipes"),
			defaultBranch: "",
			wantsErr:      true,
		},
		{
			name: "file with invalid line number after colon",
			opts: BrowseOptions{
				SelectorArg: "laptime-notes.txt:w-9",
			},
			baseRepo: ghrepo.New("andrewhsu", "sonoma-raceway"),
			wantsErr: true,
		},
		{
			name: "file with invalid file format",
			opts: BrowseOptions{
				SelectorArg: "path/to/file.txt:32:32",
			},
			baseRepo: ghrepo.New("ttran112", "ttrain211"),
			wantsErr: true,
		},
		{
			name: "file with invalid line number",
			opts: BrowseOptions{
				SelectorArg: "path/to/file.txt:32a",
			},
			baseRepo: ghrepo.New("ttran112", "ttrain211"),
			wantsErr: true,
		},
		{
			name: "file with invalid line range",
			opts: BrowseOptions{
				SelectorArg: "path/to/file.txt:32-abc",
			},
			baseRepo: ghrepo.New("ttran112", "ttrain211"),
			wantsErr: true,
		},
		{
			name: "branch with issue number",
			opts: BrowseOptions{
				SelectorArg: "217",
				Branch:      "trunk",
			},
			baseRepo:    ghrepo.New("ken", "grc"),
			wantsErr:    false,
			expectedURL: "https://github.com/ken/grc/tree/trunk/217",
		},
		{
			name: "opening branch file with line number",
			opts: BrowseOptions{
				Branch:      "first-browse-pull",
				SelectorArg: "browse.go:32",
			},
			baseRepo:    ghrepo.New("github", "ThankYouGitHub"),
			wantsErr:    false,
			expectedURL: "https://github.com/github/ThankYouGitHub/blob/first-browse-pull/browse.go?plain=1#L32",
		},
		{
			name: "no browser with branch file and line number",
			opts: BrowseOptions{
				Branch:        "3-0-stable",
				SelectorArg:   "init.rb:6",
				NoBrowserFlag: true,
			},
			httpStub: func(r *httpmock.Registry) {
				r.Register(
					httpmock.REST("HEAD", "repos/mislav/will_paginate"),
					httpmock.StringResponse("{}"),
				)
			},
			baseRepo:    ghrepo.New("mislav", "will_paginate"),
			wantsErr:    false,
			expectedURL: "https://github.com/mislav/will_paginate/blob/3-0-stable/init.rb?plain=1#L6",
		},
		{
			name: "open last commit",
			opts: BrowseOptions{
				Commit:    emptyCommitFlag,
				GitClient: &testGitClient{},
			},
			baseRepo:    ghrepo.New("vilmibm", "gh-user-status"),
			wantsErr:    false,
			expectedURL: "https://github.com/vilmibm/gh-user-status/tree/6f1a2405cace1633d89a79c74c65f22fe78f9659",
		},
		{
			name: "open last commit with a file",
			opts: BrowseOptions{
				Commit:      emptyCommitFlag,
				SelectorArg: "main.go",
				GitClient:   &testGitClient{},
			},
			baseRepo:    ghrepo.New("vilmibm", "gh-user-status"),
			wantsErr:    false,
			expectedURL: "https://github.com/vilmibm/gh-user-status/tree/6f1a2405cace1633d89a79c74c65f22fe78f9659/main.go",
		},
		{
			name: "open number only commit hash",
			opts: BrowseOptions{
				Commit:    "1234567890",
				GitClient: &testGitClient{},
			},
			baseRepo:    ghrepo.New("yanskun", "ILoveGitHub"),
			wantsErr:    false,
			expectedURL: "https://github.com/yanskun/ILoveGitHub/tree/1234567890",
		},
		{
			name: "open file with the repository state at a commit hash",
			opts: BrowseOptions{
				Commit:      "12a4",
				SelectorArg: "main.go",
				GitClient:   &testGitClient{},
			},
			baseRepo:    ghrepo.New("yanskun", "ILoveGitHub"),
			wantsErr:    false,
			expectedURL: "https://github.com/yanskun/ILoveGitHub/tree/12a4/main.go",
		},
		{
			name: "relative path from browse_test.go",
			opts: BrowseOptions{
				SelectorArg: filepath.Join(".", "browse_test.go"),
				PathFromRepoRoot: func() string {
					return "pkg/cmd/browse/"
				},
			},
			baseRepo:      ghrepo.New("bchadwic", "gh-graph"),
			defaultBranch: "trunk",
			expectedURL:   "https://github.com/bchadwic/gh-graph/tree/trunk/pkg/cmd/browse/browse_test.go",
			wantsErr:      false,
		},
		{
			name: "relative path to file in parent folder from browse_test.go",
			opts: BrowseOptions{
				SelectorArg: ".." + s + "pr",
				PathFromRepoRoot: func() string {
					return "pkg/cmd/browse/"
				},
			},
			baseRepo:      ghrepo.New("bchadwic", "gh-graph"),
			defaultBranch: "trunk",
			expectedURL:   "https://github.com/bchadwic/gh-graph/tree/trunk/pkg/cmd/pr",
			wantsErr:      false,
		},
		{
			name: "does not use relative path when has repo override",
			opts: BrowseOptions{
				SelectorArg:     "README.md",
				HasRepoOverride: true,
				PathFromRepoRoot: func() string {
					return "pkg/cmd/browse/"
				},
			},
			baseRepo:      ghrepo.New("bchadwic", "gh-graph"),
			defaultBranch: "trunk",
			expectedURL:   "https://github.com/bchadwic/gh-graph/tree/trunk/README.md",
			wantsErr:      false,
		},
		{
			name: "use special characters in selector arg",
			opts: BrowseOptions{
				SelectorArg: "?=hello world/ *:23-44",
				Branch:      "branch/with spaces?",
			},
			baseRepo:    ghrepo.New("bchadwic", "test"),
			expectedURL: "https://github.com/bchadwic/test/blob/branch/with%20spaces%3F/%3F=hello%20world/%20%2A?plain=1#L23-L44",
			wantsErr:    false,
		},
		{
			name: "commit hash in selector arg",
			opts: BrowseOptions{
				SelectorArg: "77507cd94ccafcf568f8560cfecde965fcfa63e7",
			},
			baseRepo:    ghrepo.New("bchadwic", "test"),
			expectedURL: "https://github.com/bchadwic/test/commit/77507cd94ccafcf568f8560cfecde965fcfa63e7",
			wantsErr:    false,
		},
		{
			name: "short commit hash in selector arg",
			opts: BrowseOptions{
				SelectorArg: "6e3689d5",
			},
			baseRepo:    ghrepo.New("bchadwic", "test"),
			expectedURL: "https://github.com/bchadwic/test/commit/6e3689d5",
			wantsErr:    false,
		},

		{
			name: "commit hash with extension",
			opts: BrowseOptions{
				SelectorArg: "77507cd94ccafcf568f8560cfecde965fcfa63e7.txt",
				Branch:      "trunk",
			},
			baseRepo:    ghrepo.New("bchadwic", "test"),
			expectedURL: "https://github.com/bchadwic/test/tree/trunk/77507cd94ccafcf568f8560cfecde965fcfa63e7.txt",
			wantsErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			browser := browser.Stub{}

			reg := httpmock.Registry{}
			defer reg.Verify(t)
			if tt.defaultBranch != "" {
				reg.StubRepoInfoResponse(tt.baseRepo.RepoOwner(), tt.baseRepo.RepoName(), tt.defaultBranch)
			}

			if tt.httpStub != nil {
				tt.httpStub(&reg)
			}

			opts := tt.opts
			opts.IO = ios
			opts.BaseRepo = func() (ghrepo.Interface, error) {
				return tt.baseRepo, nil
			}
			opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: &reg}, nil
			}
			opts.Browser = &browser
			if opts.PathFromRepoRoot == nil {
				opts.PathFromRepoRoot = func() string { return "" }
			}

			err := runBrowse(&opts)
			if tt.wantsErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if opts.NoBrowserFlag {
				assert.Equal(t, fmt.Sprintf("%s\n", tt.expectedURL), stdout.String())
				assert.Equal(t, "", stderr.String())
				browser.Verify(t, "")
			} else {
				assert.Equal(t, "", stdout.String())
				assert.Equal(t, "", stderr.String())
				browser.Verify(t, tt.expectedURL)
			}
		})
	}
}

func Test_parsePathFromFileArg(t *testing.T) {
	tests := []struct {
		name         string
		currentDir   string
		fileArg      string
		expectedPath string
	}{
		{
			name:         "empty paths",
			currentDir:   "",
			fileArg:      "",
			expectedPath: "",
		},
		{
			name:         "root directory",
			currentDir:   "",
			fileArg:      ".",
			expectedPath: "",
		},
		{
			name:         "relative path",
			currentDir:   "",
			fileArg:      filepath.FromSlash("foo/bar.py"),
			expectedPath: "foo/bar.py",
		},
		{
			name:         "go to parent folder",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.FromSlash("../"),
			expectedPath: "pkg/cmd",
		},
		{
			name:         "current folder",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      ".",
			expectedPath: "pkg/cmd/browse",
		},
		{
			name:         "current folder (alternative)",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.FromSlash("./"),
			expectedPath: "pkg/cmd/browse",
		},
		{
			name:         "file that starts with '.'",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      ".gitignore",
			expectedPath: "pkg/cmd/browse/.gitignore",
		},
		{
			name:         "file in current folder",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.Join(".", "browse.go"),
			expectedPath: "pkg/cmd/browse/browse.go",
		},
		{
			name:         "file within parent folder",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.Join("..", "browse.go"),
			expectedPath: "pkg/cmd/browse.go",
		},
		{
			name:         "file within parent folder uncleaned",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.FromSlash(".././//browse.go"),
			expectedPath: "pkg/cmd/browse.go",
		},
		{
			name:         "different path from root directory",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.Join("..", "..", "..", "internal/build/build.go"),
			expectedPath: "internal/build/build.go",
		},
		{
			name:         "go out of repository",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.FromSlash("../../../../../../"),
			expectedPath: "",
		},
		{
			name:         "go to root of repository",
			currentDir:   "pkg/cmd/browse/",
			fileArg:      filepath.Join("../../../"),
			expectedPath: "",
		},
		{
			name:         "empty fileArg",
			fileArg:      "",
			expectedPath: "",
		},
	}
	for _, tt := range tests {
		path, _, _, _ := parseFile(BrowseOptions{
			PathFromRepoRoot: func() string {
				return tt.currentDir
			}}, tt.fileArg)
		assert.Equal(t, tt.expectedPath, path, tt.name)
	}
}
