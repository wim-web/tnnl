package portforward

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	"github.com/wim-web/tnnl/internal/input"
)

var localPortName = "local-port"
var targetPortName = "target-port"
var inputFileName = "input-file"

type portforwardRunner func(context.Context, input.PortForwardInput) error

func newPortforwardCommand(run portforwardRunner) *cobra.Command {
	c := &cobra.Command{
		Use:   "portforward",
		Short: "Forward a local port to an ECS container",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString(inputFileName)
			if err != nil {
				return err
			}

			overrides := input.PortForwardOverrides{}
			if cmd.Flags().Changed(targetPortName) {
				value, err := cmd.Flags().GetString(targetPortName)
				if err != nil {
					return err
				}
				overrides.TargetPort = &value
			}
			if cmd.Flags().Changed(localPortName) {
				value, err := cmd.Flags().GetString(localPortName)
				if err != nil {
					return err
				}
				overrides.LocalPort = &value
			}

			resolved, err := input.ResolvePortForward(path, overrides)
			if err != nil {
				return err
			}
			return run(cmd.Context(), resolved)
		},
	}
	c.Flags().StringP(localPortName, "l", "", "local port (leave empty for automatic assignment)")
	c.Flags().StringP(targetPortName, "t", "", "target port")
	c.Flags().String(inputFileName, "", "input JSON; generate with `tnnl portforward make-input-file`")
	return c
}

var PortforwardCmd = newPortforwardCommand(handler.PortforwardHandler)

func init() {
	cmd.RootCmd.AddCommand(PortforwardCmd)
}
