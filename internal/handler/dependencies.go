package handler

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/listview"
	"github.com/wim-web/tnnl/internal/session_manager"
	"github.com/wim-web/tnnl/internal/target"
	"github.com/wim-web/tnnl/internal/view"
	"github.com/wim-web/tnnl/pkg/command"
	"github.com/wim-web/tnnl/pkg/port"
)

type ecsAPI interface {
	target.ECSAPI
	command.ExecSessionAPI
}

type ssmAPI interface {
	command.SessionAPI
}

type dependencies struct {
	loadConfig    func(context.Context) (aws.Config, error)
	newECS        func(aws.Config) ecsAPI
	newSSM        func(aws.Config) ssmAPI
	preflight     func(context.Context) (session_manager.Plugin, error)
	choose        view.Choose
	availablePort func() (int, error)
}

func productionDependencies() dependencies {
	return dependencies{
		loadConfig: func(ctx context.Context) (aws.Config, error) {
			return config.LoadDefaultConfig(ctx)
		},
		newECS: func(cfg aws.Config) ecsAPI {
			return ecs.NewFromConfig(cfg)
		},
		newSSM: func(cfg aws.Config) ssmAPI {
			return ssm.NewFromConfig(cfg)
		},
		preflight:     session_manager.Preflight,
		choose:        listview.RenderOptions,
		availablePort: port.AvailablePort,
	}
}
