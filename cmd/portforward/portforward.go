package portforward

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/handler"
	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/pkg/port"
)

var localPortName = "local-port"
var targetPortName = "target-port"
var inputFileName = "input-file"

var PortforwardCmd = &cobra.Command{
	Use:   "portforward",
	Short: "like start-session --document-name AWS-StartPortForwardingSession",
	Run: func(cmd *cobra.Command, args []string) {
		var portforwardInput input.PortForwardInput

		inputFile, err := cmd.Flags().GetString(inputFileName)
		if err != nil {
			log.Fatalln(err)
		}

		if inputFile != "" {
			input.ReadInputFile(&portforwardInput, inputFile)
		}

		if portforwardInput.LocalPortNumber == "" {
			local, err := cmd.Flags().GetString(localPortName)
			if err != nil {
				log.Fatalln(err)
			}

			if local == "" {
				l, err := port.AvailablePort()
				if err != nil {
					log.Fatalln(err)
				}
				local = strconv.Itoa(l)
			}

			portforwardInput.LocalPortNumber = local
		}

		if portforwardInput.TargetPortNumber == "" {
			target, err := cmd.Flags().GetString(targetPortName)
			if err != nil {
				log.Fatalln(err)
			}

			portforwardInput.TargetPortNumber = target
		}

		errorMsgs := validateInput(portforwardInput)
		if len(errorMsgs) != 0 {
			log.Fatalln(strings.Join(errorMsgs, "\n"))
		}

		err = handler.PortforwardHandler(portforwardInput)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func validateInput(input input.PortForwardInput) (errorMsgs []string) {
	if input.TargetPortNumber == "" {
		errorMsgs = append(errorMsgs, fmt.Sprintf("%s is required", targetPortName))
	}

	return errorMsgs
}

func init() {
	cmd.RootCmd.AddCommand(PortforwardCmd)

	PortforwardCmd.Flags().StringP(localPortName, "l", "", "local port. if not specify, auto assigned")

	PortforwardCmd.Flags().StringP(targetPortName, "t", "", "target port")

	inputFileDefault := ""
	PortforwardCmd.Flags().String(inputFileName, inputFileDefault, "input file path\nyou can make file, using exec make-input-file")
}
