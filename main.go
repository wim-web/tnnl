package main

import (
	"github.com/wim-web/tnnl/cmd"
	_ "github.com/wim-web/tnnl/cmd/exec"
	_ "github.com/wim-web/tnnl/cmd/portforward"
	_ "github.com/wim-web/tnnl/cmd/remoteportforward"
)

func main() {
	cmd.Execute()
}
