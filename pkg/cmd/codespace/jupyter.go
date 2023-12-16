package codespace

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/cli/cli/v2/internal/codespaces"
	"github.com/cli/cli/v2/internal/codespaces/portforwarder"
	"github.com/cli/cli/v2/internal/codespaces/rpc"
	"github.com/spf13/cobra"
)

func newJupyterCmd(app *App) *cobra.Command {
	var selector *CodespaceSelector

	jupyterCmd := &cobra.Command{
		Use:   "jupyter",
		Short: "Open a codespace in JupyterLab",
		Args:  noArgsConstraint,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Jupyter(cmd.Context(), selector)
		},
	}

	selector = AddCodespaceSelector(jupyterCmd, app.apiClient)

	return jupyterCmd
}

func (a *App) Jupyter(ctx context.Context, selector *CodespaceSelector) (err error) {
	// Ensure all child tasks (e.g. port forwarding) terminate before return.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	codespace, err := selector.Select(ctx)
	if err != nil {
		return err
	}

	codespaceConnection, err := codespaces.GetCodespaceConnection(ctx, a, a.apiClient, codespace)
	if err != nil {
		return fmt.Errorf("error connecting to codespace: %w", err)
	}

	fwd, err := portforwarder.NewPortForwarder(ctx, codespaceConnection)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}
	defer safeClose(fwd, &err)

	var (
		invoker    rpc.Invoker
		serverPort int
		serverUrl  string
	)
	err = a.RunWithProgress("Starting JupyterLab on codespace", func() (err error) {
		invoker, err = rpc.CreateInvoker(ctx, fwd)
		if err != nil {
			return
		}

		serverPort, serverUrl, err = invoker.StartJupyterServer(ctx)
		return
	})
	if invoker != nil {
		defer safeClose(invoker, &err)
	}
	if err != nil {
		return err
	}

	// Pass 0 to pick a random port
	listen, _, err := codespaces.ListenTCP(0, false)
	if err != nil {
		return err
	}
	defer listen.Close()
	destPort := listen.Addr().(*net.TCPAddr).Port

	tunnelClosed := make(chan error, 1)
	go func() {
		opts := portforwarder.ForwardPortOpts{
			Port: serverPort,
		}
		tunnelClosed <- fwd.ForwardPortToListener(ctx, opts, listen)
	}()

	// Server URL contains an authentication token that must be preserved
	targetUrl := strings.Replace(serverUrl, fmt.Sprintf("%d", serverPort), fmt.Sprintf("%d", destPort), 1)
	err = a.browser.Browse(targetUrl)
	if err != nil {
		return fmt.Errorf("failed to open JupyterLab in browser: %w", err)
	}

	fmt.Fprintln(a.io.Out, targetUrl)

	select {
	case err := <-tunnelClosed:
		return fmt.Errorf("tunnel closed: %w", err)
	case <-ctx.Done():
		return nil // success
	}
}
