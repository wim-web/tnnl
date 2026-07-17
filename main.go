package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wim-web/tnnl/cmd"
	_ "github.com/wim-web/tnnl/cmd/exec"
	_ "github.com/wim-web/tnnl/cmd/portforward"
	_ "github.com/wim-web/tnnl/cmd/remoteportforward"
	_ "github.com/wim-web/tnnl/cmd/update"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
