package main

import (
	_ "embed"
	"strings"

	"github.com/wim-web/tnnl/cmd"
	_ "github.com/wim-web/tnnl/cmd/exec"
	_ "github.com/wim-web/tnnl/cmd/portforward"
	_ "github.com/wim-web/tnnl/cmd/remoteportforward"
)

//go:embed .version
var version string

func main() {
	cmd.Version = strings.TrimSpace(version)
	cmd.Execute()
}
