package target

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// Resolved is a fully validated ECS target ready for Session Manager.
type Resolved struct {
	ECSCluster    string
	ClusterName   string
	Task          types.Task
	TaskARN       string
	TaskID        string
	Container     types.Container
	ContainerName string
	RuntimeID     string
}

// SSMTarget returns the ECS target identifier expected by Session Manager.
func (r Resolved) SSMTarget() string {
	return fmt.Sprintf("ecs:%s_%s_%s", r.ClusterName, r.TaskID, r.RuntimeID)
}
