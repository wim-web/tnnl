package portforward

import (
	"github.com/wim-web/tnnl/cmd/inputfile"
	"github.com/wim-web/tnnl/internal/input"
)

var MakeInputFileCmd = inputfile.New("portforward", "portforward-input.json", input.PortForwardInput{})

func init() {
	PortforwardCmd.AddCommand(MakeInputFileCmd)
}
