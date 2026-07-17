package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:           "tnnl",
	Short:         "Use ECS Exec and port forwarding through AWS Systems Manager Session Manager",
	Long:          "tnnl is a CLI for running ECS Exec sessions and port forwarding through AWS Systems Manager Session Manager.",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if shortVersion {
			return writeVersion(cmd)
		}

		return cmd.Help()
	},
}

func ExecuteContext(ctx context.Context) error {
	return RootCmd.ExecuteContext(ctx)
}

func init() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.Flags().BoolVarP(&shortVersion, "version", "v", false, "Print the version")
}

var Version string
var shortVersion bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return writeVersion(cmd)
	},
}

func writeVersion(cmd *cobra.Command) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), Version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	return nil
}
