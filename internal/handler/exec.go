package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/wim-web/tnnl/internal/view"
	"github.com/wim-web/tnnl/pkg/command"
	"golang.org/x/sync/errgroup"
)

func ExecHandler(cmd string, wait int, inputCluster string, inputService string) error {
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		return err
	}

	ecsService := ecs.NewFromConfig(cfg)

	cluster, task, container, quit, err := view.Cluster2Task2Container(ecsService, inputCluster, inputService)

	if quit {
		return nil
	}
	if err != nil {
		return err
	}

	if wait > 0 {
		eg, ctx := errgroup.WithContext(context.Background())
		ctx, cancel := context.WithCancel(ctx)

		eg.Go(func() error {
			err := ecs.NewTasksRunningWaiter(ecsService).Wait(
				ctx,
				&ecs.DescribeTasksInput{
					Cluster: &cluster,
					Tasks:   []string{*task.TaskArn},
				},
				time.Duration(wait)*time.Second,
			)
			if err != nil {
				return err
			}
			cancel()
			return nil
		})

		eg.Go(func() error {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					fmt.Printf(".")
				}
			}
		})

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	exeCmd, err := command.ExecCommand(
		context.Background(),
		ecsService,
		cluster,
		*task.TaskArn,
		cmd,
		container.Name,
		cfg.Region,
	)

	if err != nil {
		return err
	}

	return exeCmd.Run()
}
