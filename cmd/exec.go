package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/handler"
)

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "like ecs execute-command",
	Run: func(cmd *cobra.Command, args []string) {
		command, err := cmd.Flags().GetString("command")
		if err != nil {
			log.Fatalln(err)
		}
		err = handler.ExecHandler(command)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd)

	commandDefault := "sh"
	execCmd.Flags().String("command", commandDefault, fmt.Sprintf("exec command(default: %s)", commandDefault))
}
