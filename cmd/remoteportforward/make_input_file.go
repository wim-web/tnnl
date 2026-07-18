package remoteportforward

import (
	"github.com/wim-web/tnnl/cmd/inputfile"
	"github.com/wim-web/tnnl/internal/input"
)

var MakeInputFileCmd = inputfile.New("remoteportforward", "remoteportforward-input.json", input.RemotePortForwardInput{})

func init() {
	RemoteportforwardCmd.AddCommand(MakeInputFileCmd)
}
