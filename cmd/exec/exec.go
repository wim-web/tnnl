package exec

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	"github.com/wim-web/tnnl/internal/input"
)

var cmdName = "command"
var waitName = "wait"
var inputFileName = "input-file"

type execRunner func(context.Context, input.ExecInput) error

func newExecCommand(run execRunner) *cobra.Command {
	c := &cobra.Command{
		Use:   "exec",
		Short: "Run an interactive command in an ECS container",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString(inputFileName)
			if err != nil {
				return err
			}

			overrides := input.ExecOverrides{}
			if cmd.Flags().Changed(cmdName) {
				value, err := cmd.Flags().GetString(cmdName)
				if err != nil {
					return err
				}
				overrides.Command = &value
			}
			if cmd.Flags().Changed(waitName) {
				value, err := cmd.Flags().GetInt(waitName)
				if err != nil {
					return err
				}
				overrides.Wait = &value
			}

			resolved, err := input.ResolveExec(path, overrides)
			if err != nil {
				return err
			}
			return run(cmd.Context(), resolved)
		},
	}
	c.Flags().String(cmdName, "sh", "command to run (default: sh)")
	c.Flags().Int(waitName, 0, "seconds to wait for an eligible task")
	c.Flags().String(inputFileName, "", "input JSON; generate with `tnnl exec make-input-file`")
	return c
}

var ExecCmd = newExecCommand(handler.ExecHandler)

func init() {
	cmd.RootCmd.AddCommand(ExecCmd)
}
