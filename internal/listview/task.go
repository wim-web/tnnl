package listview

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go/aws"
)

type tasks []types.Task

func (ts tasks) onlyEnableExecuteCommand() tasks {
	filtered := tasks{}

	for _, t := range ts {
		if t.EnableExecuteCommand {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

func (ts tasks) names() (taskList []string) {
	for _, t := range ts {
		taskList = append(taskList, *t.Group)
	}
	return
}

func (ts tasks) findByName(name string) (types.Task, bool) {
	for _, t := range ts {
		if *t.Group == name {
			return t, true
		}
	}
	return types.Task{}, false
}

func SelectTaskView(c *ecs.Client, cluster string, inputService string) (types.Task, bool, error) {
	var task types.Task
	var listTasksInput *ecs.ListTasksInput

	if inputService != "" {
		listTasksInput = &ecs.ListTasksInput{
			Cluster:       aws.String(cluster),
			DesiredStatus: types.DesiredStatusRunning,
			ServiceName:   &inputService,
		}
	} else {
		listTasksInput = &ecs.ListTasksInput{
			Cluster:       aws.String(cluster),
			DesiredStatus: types.DesiredStatusRunning,
		}
	}

	ltRes, err := c.ListTasks(context.Background(), listTasksInput)

	if err != nil {
		return task, false, err
	}

	dtRes, err := c.DescribeTasks(context.Background(), &ecs.DescribeTasksInput{
		Tasks:   ltRes.TaskArns,
		Cluster: &cluster,
	})

	if err != nil {
		return task, false, err
	}

	tasks := tasks(dtRes.Tasks).onlyEnableExecuteCommand()

	if len(tasks) == 1 {
		return tasks[0], false, nil
	}

	taskName, quit, err := RenderList("Select a Task", tasks.names())

	if quit {
		return task, true, nil
	}

	if err != nil {
		return task, false, err
	}

	task, _ = tasks.findByName(taskName)

	return task, false, nil
}
