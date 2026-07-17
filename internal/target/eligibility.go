package target

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const runningStatus = "RUNNING"

// EligibleContainers returns the containers ready for ECS Exec or port forwarding.
func EligibleContainers(task types.Task) []types.Container {
	if !task.EnableExecuteCommand ||
		strings.TrimSpace(aws.ToString(task.LastStatus)) != runningStatus ||
		strings.TrimSpace(aws.ToString(task.TaskArn)) == "" {
		return nil
	}

	var eligible []types.Container
	for _, container := range task.Containers {
		if strings.TrimSpace(aws.ToString(container.Name)) == "" ||
			strings.TrimSpace(aws.ToString(container.LastStatus)) != runningStatus ||
			strings.TrimSpace(aws.ToString(container.RuntimeId)) == "" ||
			!hasRunningExecuteCommandAgent(container.ManagedAgents) {
			continue
		}
		eligible = append(eligible, container)
	}
	return eligible
}

// IsEligibleTask reports whether a task contains at least one ready container.
func IsEligibleTask(task types.Task) bool {
	return len(EligibleContainers(task)) > 0
}

func hasRunningExecuteCommandAgent(agents []types.ManagedAgent) bool {
	for _, agent := range agents {
		if agent.Name == types.ManagedAgentNameExecuteCommandAgent &&
			strings.TrimSpace(aws.ToString(agent.LastStatus)) == runningStatus {
			return true
		}
	}
	return false
}
