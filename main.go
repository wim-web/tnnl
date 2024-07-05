package main

import (
	"github.com/wim-web/tnnl/cmd"
	_ "github.com/wim-web/tnnl/cmd/exec"
	_ "github.com/wim-web/tnnl/cmd/exec/make_input_file"
)

func main() {
	cmd.Execute()
}
