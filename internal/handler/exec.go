package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/internal/target"
	"github.com/wim-web/tnnl/internal/view"
	"github.com/wim-web/tnnl/pkg/command"
)

func ExecHandler(ctx context.Context, in input.ExecInput) error {
	return execHandler(ctx, in, productionDependencies())
}

func execHandler(ctx context.Context, in input.ExecInput, deps dependencies) error {
	plugin, err := deps.preflight(ctx)
	if err != nil {
		return err
	}

	cfg, err := deps.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load AWS configuration: %w", err)
	}

	ecsClient := deps.newECS(cfg)
	resolved, quit, err := view.ResolveTarget(
		ctx,
		target.NewResolver(ecsClient),
		deps.choose,
		in.Cluster,
		in.Service,
		time.Duration(in.Wait)*time.Second,
	)
	if err != nil {
		return err
	}
	if quit {
		return nil
	}

	ssmClient := deps.newSSM(cfg)
	remote, err := command.StartExecSession(
		ctx,
		ecsClient,
		ssmClient,
		command.ExecTarget{
			Cluster:       resolved.ECSCluster,
			TaskARN:       resolved.TaskARN,
			ContainerName: resolved.ContainerName,
		},
		in.Cmd,
		cfg.Region,
	)
	if err != nil {
		return err
	}

	return remote.Run(ctx, plugin)
}
