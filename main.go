package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wim-web/tnnl/cmd"
	_ "github.com/wim-web/tnnl/cmd/exec"
	_ "github.com/wim-web/tnnl/cmd/portforward"
	_ "github.com/wim-web/tnnl/cmd/remoteportforward"
	_ "github.com/wim-web/tnnl/cmd/update"
)

//go:embed .version
var version string

func main() {
	cmd.Version = strings.TrimSpace(version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
