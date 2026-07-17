package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wim-web/tnnl/internal/session_manager"
	"github.com/wim-web/tnnl/internal/target"
)

type ExecSessionAPI interface {
	ExecuteCommand(context.Context, *ecs.ExecuteCommandInput, ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput, ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

type ExecTarget struct {
	Cluster       string
	TaskARN       string
	ContainerName string
}

func StartExecSession(
	ctx context.Context,
	ecsClient ExecSessionAPI,
	ssmClient SessionAPI,
	execTarget ExecTarget,
	command string,
	region string,
) (RemoteSession, error) {
	if err := validateExecSessionInput(execTarget, command, region); err != nil {
		return RemoteSession{}, err
	}

	output, err := ecsClient.ExecuteCommand(ctx, &ecs.ExecuteCommandInput{
		Cluster:     aws.String(execTarget.Cluster),
		Task:        aws.String(execTarget.TaskARN),
		Container:   aws.String(execTarget.ContainerName),
		Command:     aws.String(command),
		Interactive: true,
	})
	if err != nil {
		return RemoteSession{}, fmt.Errorf(
			"ExecuteCommand for cluster %q task %q container %q: %w",
			execTarget.Cluster,
			execTarget.TaskARN,
			execTarget.ContainerName,
			err,
		)
	}
	if output == nil {
		return RemoteSession{}, invalidSessionResponse("ExecuteCommand", "output is nil")
	}
	if output.Session == nil {
		return RemoteSession{}, invalidSessionResponse("ExecuteCommand", "session is nil")
	}

	sessionID, err := requiredSessionResponseValue("ExecuteCommand", "session ID", output.Session.SessionId)
	if err != nil {
		return RemoteSession{}, err
	}
	terminate := terminateSessionFunc(ssmClient)
	cleanup := func(primary error) error {
		return cleanupCreatedSession(ctx, sessionID, remoteSessionCleanupTimeout, terminate, primary)
	}

	streamURL, err := requiredSessionResponseValue("ExecuteCommand", "stream URL", output.Session.StreamUrl)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}
	tokenValue, err := requiredSessionResponseValue("ExecuteCommand", "token", output.Session.TokenValue)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}
	clusterARN, err := requiredSessionResponseValue("ExecuteCommand", "cluster ARN", output.ClusterArn)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}
	taskARN, err := requiredSessionResponseValue("ExecuteCommand", "task ARN", output.TaskArn)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}
	containerName, err := requiredSessionResponseValue("ExecuteCommand", "container name", output.ContainerName)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}

	clusterName, err := target.ClusterName(clusterARN)
	if err != nil {
		return RemoteSession{}, cleanup(invalidSessionResponse(
			"ExecuteCommand",
			fmt.Sprintf("invalid returned cluster ARN %q: %v", clusterARN, err),
		))
	}
	taskID, err := target.TaskID(taskARN)
	if err != nil {
		return RemoteSession{}, cleanup(invalidSessionResponse(
			"ExecuteCommand",
			fmt.Sprintf("invalid returned task ARN %q: %v", taskARN, err),
		))
	}

	describeOutput, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterARN),
		Tasks:   []string{taskARN},
	})
	if err != nil {
		return RemoteSession{}, cleanup(fmt.Errorf(
			"DescribeTasks for task %q in cluster %q: %w",
			taskARN,
			clusterARN,
			err,
		))
	}

	runtimeID, err := runtimeIDFromDescribeTasks(describeOutput, taskARN, containerName)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}

	return RemoteSession{
		ID: sessionID,
		Invocation: session_manager.Invocation{
			Response: session_manager.SessionResponse{
				SessionID:  sessionID,
				StreamURL:  streamURL,
				TokenValue: tokenValue,
			},
			Region: region,
			Target: fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID),
		},
		terminate:      terminate,
		cleanupTimeout: remoteSessionCleanupTimeout,
	}, nil
}

func validateExecSessionInput(execTarget ExecTarget, command, region string) error {
	if strings.TrimSpace(execTarget.Cluster) == "" {
		return fmt.Errorf("exec target cluster is required")
	}
	if strings.TrimSpace(execTarget.TaskARN) == "" {
		return fmt.Errorf("exec target task ARN is required")
	}
	if strings.TrimSpace(execTarget.ContainerName) == "" {
		return fmt.Errorf("exec target container name is required")
	}
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("exec command is required")
	}
	if strings.TrimSpace(region) == "" {
		return fmt.Errorf("AWS region is required for exec session")
	}
	return nil
}

func runtimeIDFromDescribeTasks(output *ecs.DescribeTasksOutput, taskARN, containerName string) (string, error) {
	if output == nil {
		return "", invalidSessionResponse("DescribeTasks", "output is nil")
	}
	if len(output.Failures) > 0 {
		failure := output.Failures[0]
		return "", invalidSessionResponse(
			"DescribeTasks",
			fmt.Sprintf(
				"reported %d failure(s), first failure arn=%q reason=%q detail=%q",
				len(output.Failures),
				aws.ToString(failure.Arn),
				aws.ToString(failure.Reason),
				aws.ToString(failure.Detail),
			),
		)
	}

	var describedTask *ecstypes.Task
	for i := range output.Tasks {
		if aws.ToString(output.Tasks[i].TaskArn) == taskARN {
			describedTask = &output.Tasks[i]
			break
		}
	}
	if describedTask == nil {
		return "", invalidSessionResponse(
			"DescribeTasks",
			fmt.Sprintf("exact returned task %q is missing", taskARN),
		)
	}

	var describedContainer *ecstypes.Container
	for i := range describedTask.Containers {
		if aws.ToString(describedTask.Containers[i].Name) == containerName {
			describedContainer = &describedTask.Containers[i]
			break
		}
	}
	if describedContainer == nil {
		return "", invalidSessionResponse(
			"DescribeTasks",
			fmt.Sprintf("exact returned container %q is missing from task %q", containerName, taskARN),
		)
	}

	runtimeID := aws.ToString(describedContainer.RuntimeId)
	if strings.TrimSpace(runtimeID) == "" {
		return "", invalidSessionResponse(
			"DescribeTasks",
			fmt.Sprintf("runtime ID is missing for container %q in task %q", containerName, taskARN),
		)
	}
	return runtimeID, nil
}

func ExecCommand(ctx context.Context, c *ecs.Client, cluster string, task string, command string, container *string, region string) (*exec.Cmd, error) {
	input := &ecs.ExecuteCommandInput{
		Cluster:     aws.String(cluster),
		Task:        aws.String(task),
		Interactive: *aws.Bool(true),
		Command:     aws.String(command),
		Container:   container,
	}

	res, err := c.ExecuteCommand(context.Background(), input)

	if err != nil {
		return nil, err
	}

	r, err := json.Marshal(res.Session)

	if err != nil {
		return nil, err
	}

	cmd := session_manager.MakeStartSessionCmd(ctx, string(r), region)

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	return cmd, nil
}
