package remoteportforward

import (
	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/input"
)

var MakeInputFileCmd = &cobra.Command{
	Use:   "make-input-file",
	Short: "make input file skelton for exec",
	Run: func(cmd *cobra.Command, args []string) {
		input.MakeInputFile(input.RemotePortForwardInput{}, "remoteportforward-input.json")
	},
}

func init() {
	RemoteportforwardCmd.AddCommand(MakeInputFileCmd)
}
