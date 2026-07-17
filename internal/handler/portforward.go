package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/internal/target"
	"github.com/wim-web/tnnl/internal/view"
	"github.com/wim-web/tnnl/pkg/command"
)

func PortforwardHandler(ctx context.Context, in input.PortForwardInput) error {
	return portForwardHandler(ctx, in, productionDependencies())
}

func portForwardHandler(ctx context.Context, in input.PortForwardInput, deps dependencies) error {
	params := map[string][]string{
		"portNumber":      {in.TargetPortNumber},
		"localPortNumber": {in.LocalPortNumber},
	}
	return portforwardHandler(ctx, command.PORT_FORWARD_DOCUMENT_NAME, params, in.EcsParameter, deps)
}

func RemotePortforwardHandler(ctx context.Context, in input.RemotePortForwardInput) error {
	return remotePortForwardHandler(ctx, in, productionDependencies())
}

func remotePortForwardHandler(ctx context.Context, in input.RemotePortForwardInput, deps dependencies) error {
	params := map[string][]string{
		"portNumber":      {in.RemotePortNumber},
		"localPortNumber": {in.LocalPortNumber},
		"host":            {in.Host},
	}
	return portforwardHandler(ctx, command.REMOTE_PORT_FORWARD_DOCUMENT_NAME, params, in.EcsParameter, deps)
}

func portforwardHandler(
	ctx context.Context,
	doc command.DocumentName,
	parameters map[string][]string,
	ecsParam input.EcsParameter,
	deps dependencies,
) error {
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
		ecsParam.Cluster,
		ecsParam.Service,
		0,
	)
	if err != nil {
		return err
	}
	if quit {
		return nil
	}

	params := cloneParameters(parameters)
	localPort := firstParameter(params, "localPortNumber")
	if strings.TrimSpace(localPort) == "" {
		allocated, err := deps.availablePort()
		if err != nil {
			return fmt.Errorf("allocate local port: %w", err)
		}
		if allocated < 1 || allocated > 65535 {
			return fmt.Errorf("allocate local port: returned invalid port %d", allocated)
		}
		params["localPortNumber"] = []string{strconv.Itoa(allocated)}
	}

	ssmClient := deps.newSSM(cfg)
	remote, err := command.StartPortForwardSession(
		ctx,
		ssmClient,
		command.PortTarget{SSMTarget: resolved.SSMTarget()},
		cfg.Region,
		doc,
		params,
	)
	if err != nil {
		return err
	}

	return remote.Run(ctx, plugin)
}

func cloneParameters(parameters map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(parameters))
	for name, values := range parameters {
		cloned[name] = append([]string(nil), values...)
	}
	return cloned
}

func firstParameter(parameters map[string][]string, name string) string {
	values := parameters[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
