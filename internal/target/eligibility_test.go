package target

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestEligibleTaskRequiresReadyTaskAndContainer(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.Task)
		want   bool
	}{
		{
			name: "fully ready task",
			want: true,
		},
		{
			name: "execute command disabled",
			mutate: func(task *types.Task) {
				task.EnableExecuteCommand = false
			},
		},
		{
			name: "task pending",
			mutate: func(task *types.Task) {
				task.LastStatus = aws.String("PENDING")
			},
		},
		{
			name: "task status padded with whitespace",
			mutate: func(task *types.Task) {
				task.LastStatus = aws.String(" RUNNING ")
			},
		},
		{
			name: "missing task ARN",
			mutate: func(task *types.Task) {
				task.TaskArn = aws.String("   ")
			},
		},
		{
			name: "missing container name",
			mutate: func(task *types.Task) {
				task.Containers[0].Name = aws.String("   ")
			},
		},
		{
			name: "container stopped",
			mutate: func(task *types.Task) {
				task.Containers[0].LastStatus = aws.String("STOPPED")
			},
		},
		{
			name: "container status padded with whitespace",
			mutate: func(task *types.Task) {
				task.Containers[0].LastStatus = aws.String(" RUNNING ")
			},
		},
		{
			name: "missing runtime ID",
			mutate: func(task *types.Task) {
				task.Containers[0].RuntimeId = aws.String("   ")
			},
		},
		{
			name: "missing execute command agent",
			mutate: func(task *types.Task) {
				task.Containers[0].ManagedAgents = nil
			},
		},
		{
			name: "execute command agent pending",
			mutate: func(task *types.Task) {
				task.Containers[0].ManagedAgents[0].LastStatus = aws.String("PENDING")
			},
		},
		{
			name: "execute command agent status padded with whitespace",
			mutate: func(task *types.Task) {
				task.Containers[0].ManagedAgents[0].LastStatus = aws.String(" RUNNING ")
			},
		},
		{
			name: "nil pointers",
			mutate: func(task *types.Task) {
				task.LastStatus = nil
				task.TaskArn = nil
				task.Containers = []types.Container{{
					ManagedAgents: []types.ManagedAgent{{
						Name: types.ManagedAgentNameExecuteCommandAgent,
					}},
				}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := fullyReadyTask()
			if tt.mutate != nil {
				tt.mutate(&task)
			}

			if got := IsEligibleTask(task); got != tt.want {
				t.Fatalf("IsEligibleTask() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEligibleContainersReturnsOnlyReadyContainers(t *testing.T) {
	ready := readyContainer("ready", "runtime-ready")
	missingName := readyContainer("missing-name", "runtime-missing-name")
	missingName.Name = nil
	stopped := readyContainer("stopped", "runtime-stopped")
	stopped.LastStatus = aws.String("STOPPED")
	missingRuntimeID := readyContainer("missing-runtime", "runtime-missing")
	missingRuntimeID.RuntimeId = nil
	missingAgent := readyContainer("missing-agent", "runtime-missing-agent")
	missingAgent.ManagedAgents = nil
	pendingAgent := readyContainer("pending-agent", "runtime-pending-agent")
	pendingAgent.ManagedAgents[0].LastStatus = aws.String("PENDING")
	nilPointers := types.Container{
		ManagedAgents: []types.ManagedAgent{{
			Name: types.ManagedAgentNameExecuteCommandAgent,
		}},
	}

	task := fullyReadyTask()
	task.Containers = []types.Container{
		missingName,
		stopped,
		ready,
		missingRuntimeID,
		missingAgent,
		pendingAgent,
		nilPointers,
	}

	want := []types.Container{ready}
	if got := EligibleContainers(task); !reflect.DeepEqual(got, want) {
		t.Fatalf("EligibleContainers() = %#v, want %#v", got, want)
	}
}

func fullyReadyTask() types.Task {
	return types.Task{
		EnableExecuteCommand: true,
		LastStatus:           aws.String("RUNNING"),
		TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/cluster/abc"),
		Containers:           []types.Container{readyContainer("app", "runtime-app")},
	}
}

func readyContainer(name, runtimeID string) types.Container {
	return types.Container{
		Name:       aws.String(name),
		LastStatus: aws.String("RUNNING"),
		RuntimeId:  aws.String(runtimeID),
		ManagedAgents: []types.ManagedAgent{{
			Name:       types.ManagedAgentNameExecuteCommandAgent,
			LastStatus: aws.String("RUNNING"),
		}},
	}
}
