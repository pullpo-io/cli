package list

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/workflow/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ListOptions
		wantsErr bool
	}{
		{
			name: "no arguments",
			wants: ListOptions{
				Limit: defaultLimit,
			},
		},
		{
			name: "all flag",
			cli:  "--all",
			wants: ListOptions{
				Limit: defaultLimit,
				All:   true,
			},
		},
		{
			name: "limit flag",
			cli:  "--limit 100",
			wants: ListOptions{
				Limit: 100,
			},
		},
		{
			name:     "invalid limit flag",
			cli:      "--limit 0",
			wantsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()

			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
			assert.Equal(t, tt.wants.All, gotOpts.All)
		})
	}
}

func TestListRun(t *testing.T) {
	workflows := []shared.Workflow{
		{
			Name:  "Go",
			State: shared.Active,
			ID:    707,
		},
		{
			Name:  "Linter",
			State: shared.Active,
			ID:    666,
		},
		{
			Name:  "Release",
			State: shared.DisabledManually,
			ID:    451,
		},
	}
	payload := shared.WorkflowsPayload{Workflows: workflows}

	tests := []struct {
		name       string
		opts       *ListOptions
		wantErr    bool
		wantOut    string
		wantErrOut string
		stubs      func(*httpmock.Registry)
		tty        bool
	}{
		{
			name: "lists worrkflows nontty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			wantOut: "Go\tactive\t707\nLinter\tactive\t666\n",
		},
		{
			name: "lists workflows tty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			tty:     true,
			wantOut: "NAME    STATE   ID\nGo      active  707\nLinter  active  666\n",
		},
		{
			name: "lists workflows with limit tty",
			opts: &ListOptions{
				Limit: 1,
			},
			tty:     true,
			wantOut: "NAME  STATE   ID\nGo    active  707\n",
		},
		{
			name: "show all workflows tty",
			opts: &ListOptions{
				Limit: defaultLimit,
				All:   true,
			},
			tty:     true,
			wantOut: "NAME     STATE              ID\nGo       active             707\nLinter   active             666\nRelease  disabled_manually  451\n",
		},
		{
			name: "no results nontty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{}),
				)
			},
			wantErr: true,
		},
		{
			name: "paginates workflows nontty",
			opts: &ListOptions{
				Limit: 101,
			},
			stubs: func(reg *httpmock.Registry) {
				workflows := []shared.Workflow{}
				var flowID int64
				for flowID = 0; flowID < 103; flowID++ {
					workflows = append(workflows, shared.Workflow{
						ID:    flowID,
						Name:  fmt.Sprintf("flow %d", flowID),
						State: shared.Active,
					})
				}
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{
						Workflows: workflows[0:100],
					}))
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(shared.WorkflowsPayload{
						Workflows: workflows[100:],
					}))
			},
			wantOut: longOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			defer reg.Verify(t)
			if tt.stubs == nil {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/workflows"),
					httpmock.JSONResponse(payload),
				)
			} else {
				tt.stubs(reg)
			}

			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}

			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.tty)
			tt.opts.IO = ios

			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.FromFullName("OWNER/REPO")
			}

			err := listRun(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantOut, stdout.String())
			assert.Equal(t, tt.wantErrOut, stderr.String())
		})
	}
}

const longOutput = "flow 0\tactive\t0\nflow 1\tactive\t1\nflow 2\tactive\t2\nflow 3\tactive\t3\nflow 4\tactive\t4\nflow 5\tactive\t5\nflow 6\tactive\t6\nflow 7\tactive\t7\nflow 8\tactive\t8\nflow 9\tactive\t9\nflow 10\tactive\t10\nflow 11\tactive\t11\nflow 12\tactive\t12\nflow 13\tactive\t13\nflow 14\tactive\t14\nflow 15\tactive\t15\nflow 16\tactive\t16\nflow 17\tactive\t17\nflow 18\tactive\t18\nflow 19\tactive\t19\nflow 20\tactive\t20\nflow 21\tactive\t21\nflow 22\tactive\t22\nflow 23\tactive\t23\nflow 24\tactive\t24\nflow 25\tactive\t25\nflow 26\tactive\t26\nflow 27\tactive\t27\nflow 28\tactive\t28\nflow 29\tactive\t29\nflow 30\tactive\t30\nflow 31\tactive\t31\nflow 32\tactive\t32\nflow 33\tactive\t33\nflow 34\tactive\t34\nflow 35\tactive\t35\nflow 36\tactive\t36\nflow 37\tactive\t37\nflow 38\tactive\t38\nflow 39\tactive\t39\nflow 40\tactive\t40\nflow 41\tactive\t41\nflow 42\tactive\t42\nflow 43\tactive\t43\nflow 44\tactive\t44\nflow 45\tactive\t45\nflow 46\tactive\t46\nflow 47\tactive\t47\nflow 48\tactive\t48\nflow 49\tactive\t49\nflow 50\tactive\t50\nflow 51\tactive\t51\nflow 52\tactive\t52\nflow 53\tactive\t53\nflow 54\tactive\t54\nflow 55\tactive\t55\nflow 56\tactive\t56\nflow 57\tactive\t57\nflow 58\tactive\t58\nflow 59\tactive\t59\nflow 60\tactive\t60\nflow 61\tactive\t61\nflow 62\tactive\t62\nflow 63\tactive\t63\nflow 64\tactive\t64\nflow 65\tactive\t65\nflow 66\tactive\t66\nflow 67\tactive\t67\nflow 68\tactive\t68\nflow 69\tactive\t69\nflow 70\tactive\t70\nflow 71\tactive\t71\nflow 72\tactive\t72\nflow 73\tactive\t73\nflow 74\tactive\t74\nflow 75\tactive\t75\nflow 76\tactive\t76\nflow 77\tactive\t77\nflow 78\tactive\t78\nflow 79\tactive\t79\nflow 80\tactive\t80\nflow 81\tactive\t81\nflow 82\tactive\t82\nflow 83\tactive\t83\nflow 84\tactive\t84\nflow 85\tactive\t85\nflow 86\tactive\t86\nflow 87\tactive\t87\nflow 88\tactive\t88\nflow 89\tactive\t89\nflow 90\tactive\t90\nflow 91\tactive\t91\nflow 92\tactive\t92\nflow 93\tactive\t93\nflow 94\tactive\t94\nflow 95\tactive\t95\nflow 96\tactive\t96\nflow 97\tactive\t97\nflow 98\tactive\t98\nflow 99\tactive\t99\nflow 100\tactive\t100\n"
