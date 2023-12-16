package list

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/cache/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wants    ListOptions
		wantsErr string
	}{
		{
			name:  "no arguments",
			input: "",
			wants: ListOptions{
				Limit: 30,
				Order: "desc",
				Sort:  "last_accessed_at",
			},
		},
		{
			name:  "with limit",
			input: "--limit 100",
			wants: ListOptions{
				Limit: 100,
				Order: "desc",
				Sort:  "last_accessed_at",
			},
		},
		{
			name:     "invalid limit",
			input:    "-L 0",
			wantsErr: "invalid limit: 0",
		},
		{
			name:  "with sort",
			input: "--sort created_at",
			wants: ListOptions{
				Limit: 30,
				Order: "desc",
				Sort:  "created_at",
			},
		},
		{
			name:  "with order",
			input: "--order asc",
			wants: ListOptions{
				Limit: 30,
				Order: "asc",
				Sort:  "last_accessed_at",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}
			argv, err := shlex.Split(tt.input)
			assert.NoError(t, err)
			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr != "" {
				assert.EqualError(t, err, tt.wantsErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
			assert.Equal(t, tt.wants.Sort, gotOpts.Sort)
			assert.Equal(t, tt.wants.Order, gotOpts.Order)
		})
	}
}

func TestListRun(t *testing.T) {
	var now = time.Date(2023, 1, 1, 1, 1, 1, 1, time.UTC)
	tests := []struct {
		name       string
		opts       ListOptions
		stubs      func(*httpmock.Registry)
		tty        bool
		wantErr    bool
		wantErrMsg string
		wantStderr string
		wantStdout string
	}{
		{
			name: "displays results tty",
			tty:  true,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/caches"),
					httpmock.JSONResponse(shared.CachePayload{
						ActionsCaches: []shared.Cache{
							{
								Id:             1,
								Key:            "foo",
								CreatedAt:      time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC),
								LastAccessedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
								SizeInBytes:    100,
							},
							{
								Id:             2,
								Key:            "bar",
								CreatedAt:      time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC),
								LastAccessedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
								SizeInBytes:    1024,
							},
						},
						TotalCount: 2,
					}),
				)
			},
			wantStdout: heredoc.Doc(`

Showing 2 of 2 caches in OWNER/REPO

ID  KEY  SIZE      CREATED            ACCESSED
1   foo  100 B     about 2 years ago  about 1 year ago
2   bar  1.00 KiB  about 2 years ago  about 1 year ago
`),
		},
		{
			name: "displays results non-tty",
			tty:  false,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/caches"),
					httpmock.JSONResponse(shared.CachePayload{
						ActionsCaches: []shared.Cache{
							{
								Id:             1,
								Key:            "foo",
								CreatedAt:      time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC),
								LastAccessedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
								SizeInBytes:    100,
							},
							{
								Id:             2,
								Key:            "bar",
								CreatedAt:      time.Date(2021, 1, 1, 1, 1, 1, 1, time.UTC),
								LastAccessedAt: time.Date(2022, 1, 1, 1, 1, 1, 1, time.UTC),
								SizeInBytes:    1024,
							},
						},
						TotalCount: 2,
					}),
				)
			},
			wantStdout: "1\tfoo\t100 B\t2021-01-01T01:01:01Z\t2022-01-01T01:01:01Z\n2\tbar\t1.00 KiB\t2021-01-01T01:01:01Z\t2022-01-01T01:01:01Z\n",
		},
		{
			name: "displays no results",
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/caches"),
					httpmock.JSONResponse(shared.CachePayload{
						ActionsCaches: []shared.Cache{},
						TotalCount:    0,
					}),
				)
			},
			wantErr:    true,
			wantErrMsg: "No caches found in OWNER/REPO",
		},
		{
			name: "displays list error",
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/caches"),
					httpmock.StatusStringResponse(404, "Not Found"),
				)
			},
			wantErr:    true,
			wantErrMsg: "X Failed to get caches: HTTP 404 (https://api.github.com/repos/OWNER/REPO/actions/caches?per_page=100)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			if tt.stubs != nil {
				tt.stubs(reg)
			}
			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)
			tt.opts.IO = ios
			tt.opts.Now = now
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.New("OWNER", "REPO"), nil
			}
			defer reg.Verify(t)

			err := listRun(&tt.opts)
			if tt.wantErr {
				if tt.wantErrMsg != "" {
					assert.EqualError(t, err, tt.wantErrMsg)
				} else {
					assert.Error(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}

func Test_humanFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{
			name: "min bytes",
			size: 1,
			want: "1 B",
		},
		{
			name: "max bytes",
			size: 1023,
			want: "1023 B",
		},
		{
			name: "min kibibytes",
			size: 1024,
			want: "1.00 KiB",
		},
		{
			name: "max kibibytes",
			size: 1024*1024 - 1,
			want: "1023.99 KiB",
		},
		{
			name: "min mibibytes",
			size: 1024 * 1024,
			want: "1.00 MiB",
		},
		{
			name: "fractional mibibytes",
			size: 1024*1024*12 + 1024*350,
			want: "12.34 MiB",
		},
		{
			name: "max mibibytes",
			size: 1024*1024*1024 - 1,
			want: "1023.99 MiB",
		},
		{
			name: "min gibibytes",
			size: 1024 * 1024 * 1024,
			want: "1.00 GiB",
		},
		{
			name: "fractional gibibytes",
			size: 1024 * 1024 * 1024 * 1.5,
			want: "1.50 GiB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanFileSize(tt.size); got != tt.want {
				t.Errorf("humanFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
