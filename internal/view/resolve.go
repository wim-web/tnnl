package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wim-web/tnnl/internal/listview"
	"github.com/wim-web/tnnl/internal/target"
)

const (
	clusterChoiceTitle   = "Select an ECS cluster"
	taskChoiceTitle      = "Select an ECS task"
	containerChoiceTitle = "Select an ECS container"
)

type targetResolver interface {
	Clusters(context.Context) ([]string, error)
	WaitForEligibleTasks(context.Context, string, string, time.Duration, target.Clock) ([]types.Task, error)
}

// Choose presents typed options and returns the selected option value.
type Choose func(string, []listview.Option) (string, bool, error)

// ResolveTarget resolves an exact eligible ECS task and container.
func ResolveTarget(
	ctx context.Context,
	resolver targetResolver,
	choose Choose,
	inputCluster string,
	inputService string,
	maxWait time.Duration,
) (target.Resolved, bool, error) {
	var resolved target.Resolved

	ecsCluster := strings.TrimSpace(inputCluster)
	if ecsCluster == "" {
		clusters, err := resolver.Clusters(ctx)
		if err != nil {
			return resolved, false, fmt.Errorf("resolve ECS clusters: %w", err)
		}
		options, err := clusterOptions(clusters)
		if err != nil {
			return resolved, false, fmt.Errorf("prepare ECS cluster choices: %w", err)
		}
		selected, quit, err := chooseOption(clusterChoiceTitle, options, false, choose)
		if err != nil {
			return resolved, false, fmt.Errorf("select ECS cluster: %w", err)
		}
		if quit {
			return target.Resolved{}, true, nil
		}
		if !hasOptionValue(options, selected) {
			return resolved, false, fmt.Errorf("selected ECS cluster %q is no longer available", selected)
		}
		ecsCluster = selected
	}

	clusterName, err := target.ClusterName(ecsCluster)
	if err != nil {
		return resolved, false, fmt.Errorf("resolve ECS cluster: %w", err)
	}

	tasks, err := resolver.WaitForEligibleTasks(ctx, ecsCluster, inputService, maxWait, target.RealClock())
	if err != nil {
		return resolved, false, fmt.Errorf("resolve eligible ECS tasks in cluster %q: %w", ecsCluster, err)
	}
	taskChoices, err := taskOptions(tasks)
	if err != nil {
		return resolved, false, fmt.Errorf("prepare ECS task choices: %w", err)
	}
	selectedTaskARN, quit, err := chooseOption(taskChoiceTitle, taskChoices, true, choose)
	if err != nil {
		return resolved, false, fmt.Errorf("select ECS task: %w", err)
	}
	if quit {
		return target.Resolved{}, true, nil
	}
	selectedTask, err := taskByARN(tasks, selectedTaskARN)
	if err != nil {
		return resolved, false, fmt.Errorf("resolve selected ECS task: %w", err)
	}

	eligibleContainers := target.EligibleContainers(selectedTask)
	containerChoices, err := containerOptions(eligibleContainers)
	if err != nil {
		return resolved, false, fmt.Errorf("prepare ECS container choices: %w", err)
	}
	selectedContainerName, quit, err := chooseOption(containerChoiceTitle, containerChoices, true, choose)
	if err != nil {
		return resolved, false, fmt.Errorf("select ECS container: %w", err)
	}
	if quit {
		return target.Resolved{}, true, nil
	}
	selectedContainer, err := containerByName(eligibleContainers, selectedContainerName)
	if err != nil {
		return resolved, false, fmt.Errorf("resolve selected ECS container: %w", err)
	}

	taskARN := strings.TrimSpace(aws.ToString(selectedTask.TaskArn))
	if taskARN == "" {
		return resolved, false, fmt.Errorf("resolve selected ECS task: task ARN is empty")
	}
	taskID, err := target.TaskID(taskARN)
	if err != nil {
		return resolved, false, fmt.Errorf("resolve selected ECS task ID: %w", err)
	}
	containerName := strings.TrimSpace(aws.ToString(selectedContainer.Name))
	if containerName == "" {
		return resolved, false, fmt.Errorf("resolve selected ECS container: container name is empty")
	}
	runtimeID := strings.TrimSpace(aws.ToString(selectedContainer.RuntimeId))
	if runtimeID == "" {
		return resolved, false, fmt.Errorf("resolve selected ECS container: runtime ID is empty")
	}

	return target.Resolved{
		ECSCluster:    ecsCluster,
		ClusterName:   clusterName,
		Task:          selectedTask,
		TaskARN:       taskARN,
		TaskID:        taskID,
		Container:     selectedContainer,
		ContainerName: containerName,
		RuntimeID:     runtimeID,
	}, false, nil
}

func chooseOption(title string, options []listview.Option, auto bool, choose Choose) (string, bool, error) {
	if len(options) == 0 {
		return "", false, fmt.Errorf("%s: no eligible items", title)
	}
	if auto && len(options) == 1 {
		return options[0].Value, false, nil
	}
	return choose(title, options)
}

func hasOptionValue(options []listview.Option, selected string) bool {
	for _, option := range options {
		if option.Value == selected {
			return true
		}
	}
	return false
}

func clusterOptions(clusters []string) ([]listview.Option, error) {
	options := make([]listview.Option, 0, len(clusters))
	for _, cluster := range clusters {
		name, err := target.ClusterName(cluster)
		if err != nil {
			return nil, err
		}
		options = append(options, listview.Option{Label: name, Value: cluster})
	}
	return options, nil
}

func taskOptions(tasks []types.Task) ([]listview.Option, error) {
	options := make([]listview.Option, 0, len(tasks))
	for _, task := range tasks {
		arnValue := strings.TrimSpace(aws.ToString(task.TaskArn))
		id, err := target.TaskID(arnValue)
		if err != nil {
			return nil, err
		}
		group := strings.TrimSpace(aws.ToString(task.Group))
		if group == "" {
			group = "task"
		}
		options = append(options, listview.Option{
			Label: fmt.Sprintf("%s %s", group, id),
			Value: arnValue,
		})
	}
	return options, nil
}

func taskByARN(tasks []types.Task, selected string) (types.Task, error) {
	for _, task := range tasks {
		if aws.ToString(task.TaskArn) == selected {
			return task, nil
		}
	}
	return types.Task{}, fmt.Errorf("selected ECS task %q is no longer available", selected)
}

func containerOptions(containers []types.Container) ([]listview.Option, error) {
	options := make([]listview.Option, 0, len(containers))
	for _, container := range containers {
		name := aws.ToString(container.Name)
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("ECS container name is empty")
		}
		options = append(options, listview.Option{Label: name, Value: name})
	}
	return options, nil
}

func containerByName(containers []types.Container, selected string) (types.Container, error) {
	for _, container := range containers {
		if aws.ToString(container.Name) == selected {
			return container, nil
		}
	}
	return types.Container{}, fmt.Errorf("selected ECS container %q is no longer available", selected)
}
