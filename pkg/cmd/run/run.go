package run

import (
	cmdCancel "github.com/cli/cli/v2/pkg/cmd/run/cancel"
	cmdDelete "github.com/cli/cli/v2/pkg/cmd/run/delete"
	cmdDownload "github.com/cli/cli/v2/pkg/cmd/run/download"
	cmdList "github.com/cli/cli/v2/pkg/cmd/run/list"
	cmdRerun "github.com/cli/cli/v2/pkg/cmd/run/rerun"
	cmdView "github.com/cli/cli/v2/pkg/cmd/run/view"
	cmdWatch "github.com/cli/cli/v2/pkg/cmd/run/watch"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRun(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run <command>",
		Short:   "View details about workflow runs",
		Long:    "List, view, and watch recent workflow runs from GitHub Actions.",
		GroupID: "actions",
	}
	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(cmdList.NewCmdList(f, nil))
	cmd.AddCommand(cmdView.NewCmdView(f, nil))
	cmd.AddCommand(cmdRerun.NewCmdRerun(f, nil))
	cmd.AddCommand(cmdDownload.NewCmdDownload(f, nil))
	cmd.AddCommand(cmdWatch.NewCmdWatch(f, nil))
	cmd.AddCommand(cmdCancel.NewCmdCancel(f, nil))
	cmd.AddCommand(cmdDelete.NewCmdDelete(f, nil))

	return cmd
}
