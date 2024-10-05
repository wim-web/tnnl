package exec

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	"github.com/wim-web/tnnl/internal/input"
)

var cmdName = "command"
var waitName = "wait"
var inputFileName = "input-file"

var ExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "like ecs execute-command",
	Run: func(cmd *cobra.Command, args []string) {
		var execInput input.ExecInput

		inputFile, err := cmd.Flags().GetString(inputFileName)
		if err != nil {
			log.Fatalln(err)
		}

		if inputFile != "" {
			input.ReadInputFile(&execInput, inputFile)
		}

		if execInput.Cmd == "" {
			command, err := cmd.Flags().GetString(cmdName)
			if err != nil {
				log.Fatalln(err)
			}

			execInput.Cmd = command
		}

		if execInput.Wait == 0 {
			wait, err := cmd.Flags().GetInt("wait")
			if err != nil {
				log.Fatalln(err)
			}

			execInput.Wait = wait
		}

		err = handler.ExecHandler(execInput)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	cmd.RootCmd.AddCommand(ExecCmd)

	commandDefault := "sh"
	ExecCmd.Flags().String(cmdName, commandDefault, fmt.Sprintf("exec command(default: %s)", commandDefault))

	waitDefault := 0
	ExecCmd.Flags().Int(waitName, waitDefault, fmt.Sprintf("the number of seconds to wait for task to launch(default: %v)", waitDefault))

	inputFileDefault := ""
	ExecCmd.Flags().String(inputFileName, inputFileDefault, "input file path\nyou can make file, using exec make-input-file")
}
