package remoteportforward

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	"github.com/wim-web/tnnl/internal/input"
)

var localPortName = "local-port"
var remotePortName = "remote-port"
var hostName = "host"
var inputFileName = "input-file"

type remotePortforwardRunner func(context.Context, input.RemotePortForwardInput) error

func newRemotePortforwardCommand(run remotePortforwardRunner) *cobra.Command {
	c := &cobra.Command{
		Use:   "remoteportforward",
		Short: "Forward a local port through an ECS container to a remote host",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString(inputFileName)
			if err != nil {
				return err
			}

			overrides := input.RemotePortForwardOverrides{}
			if cmd.Flags().Changed(remotePortName) {
				value, err := cmd.Flags().GetString(remotePortName)
				if err != nil {
					return err
				}
				overrides.RemotePort = &value
			}
			if cmd.Flags().Changed(localPortName) {
				value, err := cmd.Flags().GetString(localPortName)
				if err != nil {
					return err
				}
				overrides.LocalPort = &value
			}
			if cmd.Flags().Changed(hostName) {
				value, err := cmd.Flags().GetString(hostName)
				if err != nil {
					return err
				}
				overrides.Host = &value
			}

			resolved, err := input.ResolveRemotePortForward(path, overrides)
			if err != nil {
				return err
			}
			return run(cmd.Context(), resolved)
		},
	}
	c.Flags().StringP(localPortName, "l", "", "local port (leave empty for automatic assignment)")
	c.Flags().StringP(remotePortName, "r", "", "remote port")
	c.Flags().String(hostName, "", "remote host")
	c.Flags().String(inputFileName, "", "input JSON; generate with `tnnl remoteportforward make-input-file`")
	return c
}

var RemoteportforwardCmd = newRemotePortforwardCommand(handler.RemotePortforwardHandler)

func init() {
	cmd.RootCmd.AddCommand(RemoteportforwardCmd)
}
