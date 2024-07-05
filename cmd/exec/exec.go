package exec

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	inputfile "github.com/wim-web/tnnl/internal/input-file"
)

var ExecCmd = &cobra.Command{
	Use:   "exec",
	Short: "like ecs execute-command",
	Run: func(cmd *cobra.Command, args []string) {
		command, err := cmd.Flags().GetString("command")
		if err != nil {
			log.Fatalln(err)
		}

		wait, err := cmd.Flags().GetInt("wait")
		if err != nil {
			log.Fatalln(err)
		}

		inputFile, err := cmd.Flags().GetString("input-file")
		if err != nil {
			log.Fatalln(err)
		}

		var input inputfile.ExecInputFile

		if inputFile != "" {
			file, err := os.Open(inputFile)
			if err != nil {
				log.Fatalln("Error opening file:", err)
			}
			defer file.Close()

			jsonData, err := io.ReadAll(file)
			if err != nil {
				log.Fatalln("Error reading file:", err)
			}

			err = json.Unmarshal(jsonData, &input)
			if err != nil {
				log.Fatalln("Error decoding JSON:", err)
			}
		}

		err = handler.ExecHandler(command, wait, input.Cluster, input.Service)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	cmd.RootCmd.AddCommand(ExecCmd)

	commandDefault := "sh"
	ExecCmd.Flags().String("command", commandDefault, fmt.Sprintf("exec command(default: %s)", commandDefault))

	waitDefault := 0
	ExecCmd.Flags().Int("wait", waitDefault, fmt.Sprintf("the number of seconds to wait for task to launch(default: %v)", waitDefault))

	inputFileDefault := ""
	ExecCmd.Flags().String("input-file", inputFileDefault, "input file path\nyou can make file, using exec make-input-file")
}
