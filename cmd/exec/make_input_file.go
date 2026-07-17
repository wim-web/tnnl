package exec

import (
	"github.com/wim-web/tnnl/cmd/inputfile"
	"github.com/wim-web/tnnl/internal/input"
)

var MakeInputFileCmd = inputfile.New("exec", "exec-input.json", input.ExecInput{})

func init() {
	ExecCmd.AddCommand(MakeInputFileCmd)
}
