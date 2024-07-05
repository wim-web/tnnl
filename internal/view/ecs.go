package view

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wim-web/tnnl/internal/listview"
)

func Cluster2Task2Container(
	ecsService *ecs.Client,
	inputCluster string,
	inputService string,
) (string, types.Task, types.Container, bool, error) {
	var clusterName string
	var task types.Task
	var container types.Container

	if inputCluster != "" {
		clusterName = inputCluster
	} else {
		var quit bool
		var err error
		clusterName, quit, err = listview.SelectClusterView(ecsService)

		if quit {
			return clusterName, task, container, true, nil
		}
		if err != nil {
			return clusterName, task, container, false, err
		}
	}

	task, quit, err := listview.SelectTaskView(ecsService, clusterName, inputService)

	if quit {
		return clusterName, task, container, true, nil
	}
	if err != nil {
		return clusterName, task, container, false, err
	}

	container, quit, err = listview.SelectContainerView(task, true)

	if quit {
		return clusterName, task, container, true, nil
	}
	if err != nil {
		return clusterName, task, container, false, err
	}

	return clusterName, task, container, false, nil
}
