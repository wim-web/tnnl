package remoteportforward

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
var remotePortName = "remote-port"
var hostName = "host"
var inputFileName = "input-file"

var RemoteportforwardCmd = &cobra.Command{
	Use:   "remoteportforward",
	Short: "like start-session --document-name AWS-StartPortForwardingSessionToRemote",
	Run: func(cmd *cobra.Command, args []string) {
		var remotePortforwardInput input.RemotePortForwardInput

		inputFile, err := cmd.Flags().GetString(inputFileName)
		if err != nil {
			log.Fatalln(err)
		}

		if inputFile != "" {
			input.ReadInputFile(&remotePortforwardInput, inputFile)
		}

		if remotePortforwardInput.RemotePortNumber == "" {
			remote, err := cmd.Flags().GetString("remote-port")
			if err != nil {
				log.Fatalln(err)
			}

			remotePortforwardInput.RemotePortNumber = remote
		}

		if remotePortforwardInput.LocalPortNumber == "" {
			local, err := cmd.Flags().GetString("local-port")
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

			remotePortforwardInput.LocalPortNumber = local
		}

		if remotePortforwardInput.Host == "" {
			host, err := cmd.Flags().GetString("host")
			if err != nil {
				log.Fatalln(err)
			}

			remotePortforwardInput.Host = host
		}

		errorMsgs := validateInput(remotePortforwardInput)
		if len(errorMsgs) != 0 {
			log.Fatalln(strings.Join(errorMsgs, "\n"))
		}

		err = handler.RemotePortforwardHandler(remotePortforwardInput)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func validateInput(input input.RemotePortForwardInput) (errorMsgs []string) {
	if input.RemotePortNumber == "" {
		errorMsgs = append(errorMsgs, fmt.Sprintf("%s is required", remotePortName))
	}

	if input.Host == "" {
		errorMsgs = append(errorMsgs, fmt.Sprintf("%s is required", hostName))
	}

	return errorMsgs
}

func init() {
	cmd.RootCmd.AddCommand(RemoteportforwardCmd)

	RemoteportforwardCmd.Flags().StringP(localPortName, "l", "", "local port. if not specify, auto assigned")

	RemoteportforwardCmd.Flags().StringP(remotePortName, "r", "", "remote port")

	RemoteportforwardCmd.Flags().String(hostName, "", "host")

	inputFileDefault := ""
	RemoteportforwardCmd.Flags().String(inputFileName, inputFileDefault, "input file path\nyou can make file, using exec make-input-file")
}
