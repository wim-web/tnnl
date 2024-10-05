package handler

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/internal/view"
	"github.com/wim-web/tnnl/pkg/command"
)

func PortforwardHandler(input input.PortForwardInput) error {
	params := map[string][]string{
		"portNumber":      {input.TargetPortNumber},
		"localPortNumber": {input.LocalPortNumber},
	}

	return portforwardHandler(command.PORT_FORWARD_DOCUMENT_NAME, params, input.EcsParameter)
}

func RemotePortforwardHandler(input input.RemotePortForwardInput) error {
	params := map[string][]string{
		"portNumber":      {input.RemotePortNumber},
		"localPortNumber": {input.LocalPortNumber},
		"host":            {input.Host},
	}

	return portforwardHandler(command.REMOTE_PORT_FORWARD_DOCUMENT_NAME, params, input.EcsParameter)
}

func portforwardHandler(doc command.DocumentName, params map[string][]string, ecsParam input.EcsParameter) error {
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		return err
	}

	ssmService := ssm.NewFromConfig(cfg)
	ecsService := ecs.NewFromConfig(cfg)

	cluster, task, container, quit, err := view.Cluster2Task2Container(ecsService, ecsParam.Cluster, ecsParam.Service)

	if quit {
		return nil
	}
	if err != nil {
		return err
	}

	taskId := strings.Split(*task.TaskArn, "/")[2]

	cmd, err := command.PortForwardCommand(
		context.Background(),
		ssmService,
		cluster,
		taskId,
		*container.RuntimeId,
		cfg.Region,
		doc,
		params,
	)

	if err != nil {
		return err
	}

	return cmd.Run()
}
