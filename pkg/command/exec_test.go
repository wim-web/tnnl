package command

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wim-web/tnnl/internal/session_manager"
)

const (
	execCallerCluster   = "caller-cluster"
	execCallerTask      = "caller-task"
	execCallerContainer = "caller-container"
	execClusterARN      = "arn:aws:ecs:ap-northeast-1:123456789012:cluster/response-cluster"
	execTaskARN         = "arn:aws:ecs:ap-northeast-1:123456789012:task/response-cluster/response-task-id"
	execContainer       = "response-container"
	execSessionID       = "ecs-session-123"
	execStreamURL       = "wss://ecs-session.example"
	execToken           = "ecs-token"
	execRuntimeID       = "fresh-runtime-id"
)

type execContextKey struct{}

type fakeExecSessionAPI struct {
	executeOutput  *ecs.ExecuteCommandOutput
	executeErr     error
	describeOutput *ecs.DescribeTasksOutput
	describeErr    error

	executeCalls  int
	executeCtx    context.Context
	executeInput  *ecs.ExecuteCommandInput
	describeCalls int
	describeCtx   context.Context
	describeInput *ecs.DescribeTasksInput
}

func (f *fakeExecSessionAPI) ExecuteCommand(
	ctx context.Context,
	input *ecs.ExecuteCommandInput,
	_ ...func(*ecs.Options),
) (*ecs.ExecuteCommandOutput, error) {
	f.executeCalls++
	f.executeCtx = ctx
	f.executeInput = input
	return f.executeOutput, f.executeErr
}

func (f *fakeExecSessionAPI) DescribeTasks(
	ctx context.Context,
	input *ecs.DescribeTasksInput,
	_ ...func(*ecs.Options),
) (*ecs.DescribeTasksOutput, error) {
	f.describeCalls++
	f.describeCtx = ctx
	f.describeInput = input
	return f.describeOutput, f.describeErr
}

func TestStartExecSessionUsesAuthoritativeResponseAndFreshRuntimeID(t *testing.T) {
	ctx := context.WithValue(context.Background(), execContextKey{}, "caller-context")
	ecsClient := &fakeExecSessionAPI{
		executeOutput: validExecuteCommandOutput(),
		describeOutput: &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{
			{
				TaskArn: aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task/response-cluster/decoy-task"),
				Containers: []ecstypes.Container{
					{Name: aws.String(execContainer), RuntimeId: aws.String("wrong-task-runtime")},
				},
			},
			{
				TaskArn: aws.String(execTaskARN),
				Containers: []ecstypes.Container{
					{Name: aws.String("decoy-container"), RuntimeId: aws.String("wrong-container-runtime")},
					{Name: aws.String(execContainer), RuntimeId: aws.String(execRuntimeID)},
				},
			},
		}},
	}
	ssmClient := &fakeSessionAPI{}
	target := ExecTarget{
		Cluster:       execCallerCluster,
		TaskARN:       execCallerTask,
		ContainerName: execCallerContainer,
	}

	session, err := StartExecSession(ctx, ecsClient, ssmClient, target, "/bin/sh", "ap-northeast-1")
	if err != nil {
		t.Fatalf("StartExecSession() error = %v, want nil", err)
	}

	if ecsClient.executeCalls != 1 {
		t.Fatalf("ExecuteCommand calls = %d, want 1", ecsClient.executeCalls)
	}
	if ecsClient.executeCtx != ctx {
		t.Fatalf("ExecuteCommand context = %v, want original caller context %v", ecsClient.executeCtx, ctx)
	}
	if got := ecsClient.executeInput; got == nil ||
		aws.ToString(got.Cluster) != execCallerCluster ||
		aws.ToString(got.Task) != execCallerTask ||
		aws.ToString(got.Container) != execCallerContainer ||
		aws.ToString(got.Command) != "/bin/sh" ||
		!got.Interactive {
		t.Fatalf("ExecuteCommand input = %#v, want exact caller target, command, and Interactive=true", got)
	}

	if ecsClient.describeCalls != 1 {
		t.Fatalf("DescribeTasks calls = %d, want 1", ecsClient.describeCalls)
	}
	if ecsClient.describeCtx != ctx {
		t.Fatalf("DescribeTasks context = %v, want original caller context %v", ecsClient.describeCtx, ctx)
	}
	if got := ecsClient.describeInput; got == nil ||
		aws.ToString(got.Cluster) != execClusterARN ||
		!reflect.DeepEqual(got.Tasks, []string{execTaskARN}) {
		t.Fatalf("DescribeTasks input = %#v, want response cluster %q and task %q", got, execClusterARN, execTaskARN)
	}

	wantInvocation := session_manager.Invocation{
		Response: session_manager.SessionResponse{
			SessionID:  execSessionID,
			StreamURL:  execStreamURL,
			TokenValue: execToken,
		},
		Region: "ap-northeast-1",
		Target: "ecs:response-cluster_response-task-id_" + execRuntimeID,
	}
	if session.ID != execSessionID {
		t.Fatalf("RemoteSession.ID = %q, want %q", session.ID, execSessionID)
	}
	if !reflect.DeepEqual(session.Invocation, wantInvocation) {
		t.Fatalf("RemoteSession.Invocation = %#v, want %#v", session.Invocation, wantInvocation)
	}
	if session.cleanupTimeout != 5*time.Second {
		t.Fatalf("RemoteSession cleanup timeout = %v, want 5s", session.cleanupTimeout)
	}
	if ssmClient.terminateCalls != 0 {
		t.Fatalf("TerminateSession calls during successful construction = %d, want 0", ssmClient.terminateCalls)
	}

	terminateCtx := context.WithValue(context.Background(), execContextKey{}, "terminate-context")
	if err := session.terminate(terminateCtx, session.ID); err != nil {
		t.Fatalf("RemoteSession terminate() error = %v, want nil", err)
	}
	assertTerminateCall(t, ssmClient, terminateCtx, execSessionID)
}

func TestStartExecSessionTerminatesExactSessionWhenPluginFails(t *testing.T) {
	ctx := context.WithValue(context.Background(), execContextKey{}, "preserved-for-cleanup")
	ecsClient := &fakeExecSessionAPI{
		executeOutput:  validExecuteCommandOutput(),
		describeOutput: validDescribeTasksOutput(),
	}
	ssmClient := &fakeSessionAPI{}
	session, err := StartExecSession(ctx, ecsClient, ssmClient, validExecTarget(), "/bin/sh", "ap-northeast-1")
	if err != nil {
		t.Fatalf("StartExecSession() error = %v, want nil", err)
	}
	pluginErr := errors.New("plugin failed")

	err = session.Run(ctx, pluginFunc(func(pluginCtx context.Context, invocation session_manager.Invocation) error {
		if pluginCtx != ctx {
			t.Fatalf("plugin context = %v, want original caller context %v", pluginCtx, ctx)
		}
		if !reflect.DeepEqual(invocation, session.Invocation) {
			t.Fatalf("plugin invocation = %#v, want %#v", invocation, session.Invocation)
		}
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("RemoteSession.Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if ssmClient.terminateCalls != 1 {
		t.Fatalf("TerminateSession calls = %d, want 1", ssmClient.terminateCalls)
	}
	if got := aws.ToString(ssmClient.terminateInput.SessionId); got != execSessionID {
		t.Fatalf("TerminateSession session ID = %q, want %q", got, execSessionID)
	}
	if got := ssmClient.terminateCtx.Value(execContextKey{}); got != "preserved-for-cleanup" {
		t.Fatalf("TerminateSession context marker = %#v, want preserved caller value", got)
	}
}

func TestStartExecSessionRejectsInvalidInputBeforeExecuteCommand(t *testing.T) {
	tests := []struct {
		name    string
		target  ExecTarget
		command string
		region  string
		want    string
	}{
		{name: "cluster", target: ExecTarget{Cluster: " ", TaskARN: execCallerTask, ContainerName: execCallerContainer}, command: "/bin/sh", region: "ap-northeast-1", want: "cluster"},
		{name: "task ARN", target: ExecTarget{Cluster: execCallerCluster, TaskARN: "\t", ContainerName: execCallerContainer}, command: "/bin/sh", region: "ap-northeast-1", want: "task"},
		{name: "container", target: ExecTarget{Cluster: execCallerCluster, TaskARN: execCallerTask, ContainerName: "\n"}, command: "/bin/sh", region: "ap-northeast-1", want: "container"},
		{name: "command", target: validExecTarget(), command: "  ", region: "ap-northeast-1", want: "command"},
		{name: "region", target: validExecTarget(), command: "/bin/sh", region: "\t", want: "region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ecsClient := &fakeExecSessionAPI{}
			ssmClient := &fakeSessionAPI{}

			_, err := StartExecSession(context.Background(), ecsClient, ssmClient, tt.target, tt.command, tt.region)

			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("StartExecSession() error = %v, want actionable %q context", err, tt.want)
			}
			if ecsClient.executeCalls != 0 || ecsClient.describeCalls != 0 {
				t.Fatalf("ECS calls = execute %d, describe %d; want no remote call", ecsClient.executeCalls, ecsClient.describeCalls)
			}
			if ssmClient.terminateCalls != 0 {
				t.Fatalf("TerminateSession calls = %d, want 0", ssmClient.terminateCalls)
			}
		})
	}
}

func TestStartExecSessionWrapsExecuteCommandError(t *testing.T) {
	apiErr := errors.New("execute sentinel")
	ctx := context.WithValue(context.Background(), execContextKey{}, "execute-context")
	ecsClient := &fakeExecSessionAPI{executeErr: apiErr}
	ssmClient := &fakeSessionAPI{}

	_, err := StartExecSession(ctx, ecsClient, ssmClient, validExecTarget(), "/bin/sh", "ap-northeast-1")

	if !errors.Is(err, apiErr) {
		t.Fatalf("StartExecSession() error = %v, want errors.Is(execute sentinel)", err)
	}
	for _, want := range []string{"ExecuteCommand", execCallerCluster, execCallerTask, execCallerContainer} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("StartExecSession() error = %q, want context %q", err, want)
		}
	}
	if ecsClient.executeCtx != ctx {
		t.Fatalf("ExecuteCommand context = %v, want original caller context %v", ecsClient.executeCtx, ctx)
	}
	if ecsClient.describeCalls != 0 || ssmClient.terminateCalls != 0 {
		t.Fatalf("calls after ExecuteCommand failure = describe %d, terminate %d; want 0, 0", ecsClient.describeCalls, ssmClient.terminateCalls)
	}
}

func TestStartExecSessionValidatesExecuteCommandResponseAndCleansUp(t *testing.T) {
	tests := []struct {
		name          string
		output        *ecs.ExecuteCommandOutput
		wantTerminate bool
	}{
		{name: "nil output", output: nil},
		{name: "nil session", output: executeOutputWithSession(nil)},
		{name: "missing session ID", output: executeOutputWithSession(&ecstypes.Session{StreamUrl: aws.String(execStreamURL), TokenValue: aws.String(execToken)})},
		{name: "blank session ID", output: executeOutputWithSession(&ecstypes.Session{SessionId: aws.String(" \t"), StreamUrl: aws.String(execStreamURL), TokenValue: aws.String(execToken)})},
		{name: "missing stream URL", output: executeOutputWithSession(&ecstypes.Session{SessionId: aws.String(execSessionID), TokenValue: aws.String(execToken)}), wantTerminate: true},
		{name: "missing token", output: executeOutputWithSession(&ecstypes.Session{SessionId: aws.String(execSessionID), StreamUrl: aws.String(execStreamURL)}), wantTerminate: true},
		{name: "missing cluster ARN", output: executeOutputWithIdentifiers("", execTaskARN, execContainer), wantTerminate: true},
		{name: "missing task ARN", output: executeOutputWithIdentifiers(execClusterARN, "", execContainer), wantTerminate: true},
		{name: "missing container name", output: executeOutputWithIdentifiers(execClusterARN, execTaskARN, " \n"), wantTerminate: true},
		{name: "invalid cluster ARN", output: executeOutputWithIdentifiers("arn:aws:s3:ap-northeast-1:123456789012:cluster/not-ecs", execTaskARN, execContainer), wantTerminate: true},
		{name: "invalid task ARN", output: executeOutputWithIdentifiers(execClusterARN, "arn:aws:ecs:ap-northeast-1:123456789012:service/not-a-task", execContainer), wantTerminate: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ecsClient := &fakeExecSessionAPI{executeOutput: tt.output}
			ssmClient := &fakeSessionAPI{}
			ctx := context.WithValue(context.Background(), execContextKey{}, tt.name)

			_, err := StartExecSession(ctx, ecsClient, ssmClient, validExecTarget(), "/bin/sh", "ap-northeast-1")

			if !errors.Is(err, errInvalidSessionResponse) {
				t.Fatalf("StartExecSession() error = %v, want errors.Is(errInvalidSessionResponse)", err)
			}
			if !strings.Contains(err.Error(), "ExecuteCommand") {
				t.Fatalf("StartExecSession() error = %q, want ExecuteCommand response context", err)
			}
			wantCalls := 0
			if tt.wantTerminate {
				wantCalls = 1
			}
			if ssmClient.terminateCalls != wantCalls {
				t.Fatalf("TerminateSession calls = %d, want %d", ssmClient.terminateCalls, wantCalls)
			}
			if tt.wantTerminate {
				if got := aws.ToString(ssmClient.terminateInput.SessionId); got != execSessionID {
					t.Fatalf("TerminateSession session ID = %q, want %q", got, execSessionID)
				}
				if got := ssmClient.terminateCtx.Value(execContextKey{}); got != tt.name {
					t.Fatalf("TerminateSession context marker = %#v, want %q", got, tt.name)
				}
			}
			if ecsClient.describeCalls != 0 {
				t.Fatalf("DescribeTasks calls = %d, want 0 after invalid ExecuteCommand response", ecsClient.describeCalls)
			}
		})
	}
}

func TestStartExecSessionValidatesDescribeTasksResponseAndCleansUp(t *testing.T) {
	describeErr := errors.New("describe sentinel")
	tests := []struct {
		name           string
		describeOutput *ecs.DescribeTasksOutput
		describeErr    error
		wantError      error
	}{
		{name: "API error", describeErr: describeErr, wantError: describeErr},
		{name: "nil output", wantError: errInvalidSessionResponse},
		{
			name: "reported failure",
			describeOutput: &ecs.DescribeTasksOutput{
				Failures: []ecstypes.Failure{{Arn: aws.String(execTaskARN), Reason: aws.String("MISSING")}},
				Tasks:    validDescribeTasksOutput().Tasks,
			},
			wantError: errInvalidSessionResponse,
		},
		{
			name: "missing exact task",
			describeOutput: &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{{
				TaskArn:    aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task/response-cluster/other-task"),
				Containers: []ecstypes.Container{{Name: aws.String(execContainer), RuntimeId: aws.String(execRuntimeID)}},
			}}},
			wantError: errInvalidSessionResponse,
		},
		{
			name: "missing exact container",
			describeOutput: &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{{
				TaskArn:    aws.String(execTaskARN),
				Containers: []ecstypes.Container{{Name: aws.String("other-container"), RuntimeId: aws.String(execRuntimeID)}},
			}}},
			wantError: errInvalidSessionResponse,
		},
		{
			name: "blank runtime ID",
			describeOutput: &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{{
				TaskArn:    aws.String(execTaskARN),
				Containers: []ecstypes.Container{{Name: aws.String(execContainer), RuntimeId: aws.String(" \t")}},
			}}},
			wantError: errInvalidSessionResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ecsClient := &fakeExecSessionAPI{
				executeOutput:  validExecuteCommandOutput(),
				describeOutput: tt.describeOutput,
				describeErr:    tt.describeErr,
			}
			ssmClient := &fakeSessionAPI{}
			ctx := context.WithValue(context.Background(), execContextKey{}, tt.name)

			_, err := StartExecSession(ctx, ecsClient, ssmClient, validExecTarget(), "/bin/sh", "ap-northeast-1")

			if !errors.Is(err, tt.wantError) {
				t.Fatalf("StartExecSession() error = %v, want errors.Is(%v)", err, tt.wantError)
			}
			if !strings.Contains(err.Error(), "DescribeTasks") {
				t.Fatalf("StartExecSession() error = %q, want DescribeTasks context", err)
			}
			if ecsClient.describeCalls != 1 || ecsClient.describeCtx != ctx {
				t.Fatalf("DescribeTasks calls/context = %d/%v, want 1/original %v", ecsClient.describeCalls, ecsClient.describeCtx, ctx)
			}
			if ssmClient.terminateCalls != 1 {
				t.Fatalf("TerminateSession calls = %d, want 1", ssmClient.terminateCalls)
			}
			if got := aws.ToString(ssmClient.terminateInput.SessionId); got != execSessionID {
				t.Fatalf("TerminateSession session ID = %q, want %q", got, execSessionID)
			}
			if got := ssmClient.terminateCtx.Value(execContextKey{}); got != tt.name {
				t.Fatalf("TerminateSession context marker = %#v, want %q", got, tt.name)
			}
		})
	}
}

func TestStartExecSessionJoinsValidationAndTerminateErrors(t *testing.T) {
	cleanupErr := errors.New("terminate sentinel")
	ecsClient := &fakeExecSessionAPI{
		executeOutput: executeOutputWithSession(&ecstypes.Session{
			SessionId:  aws.String(execSessionID),
			TokenValue: aws.String(execToken),
		}),
	}
	ssmClient := &fakeSessionAPI{terminateErr: cleanupErr}

	_, err := StartExecSession(context.Background(), ecsClient, ssmClient, validExecTarget(), "/bin/sh", "ap-northeast-1")

	if !errors.Is(err, errInvalidSessionResponse) {
		t.Fatalf("StartExecSession() error = %v, want validation sentinel", err)
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("StartExecSession() error = %v, want cleanup sentinel", err)
	}
	if ssmClient.terminateCalls != 1 || aws.ToString(ssmClient.terminateInput.SessionId) != execSessionID {
		t.Fatalf("TerminateSession calls/input = %d/%#v, want one call for %q", ssmClient.terminateCalls, ssmClient.terminateInput, execSessionID)
	}
}

func validExecTarget() ExecTarget {
	return ExecTarget{
		Cluster:       execCallerCluster,
		TaskARN:       execCallerTask,
		ContainerName: execCallerContainer,
	}
}

func validExecuteCommandOutput() *ecs.ExecuteCommandOutput {
	return executeOutputWithSession(&ecstypes.Session{
		SessionId:  aws.String(execSessionID),
		StreamUrl:  aws.String(execStreamURL),
		TokenValue: aws.String(execToken),
	})
}

func executeOutputWithSession(session *ecstypes.Session) *ecs.ExecuteCommandOutput {
	return &ecs.ExecuteCommandOutput{
		ClusterArn:    aws.String(execClusterARN),
		TaskArn:       aws.String(execTaskARN),
		ContainerName: aws.String(execContainer),
		Session:       session,
	}
}

func executeOutputWithIdentifiers(clusterARN, taskARN, containerName string) *ecs.ExecuteCommandOutput {
	output := validExecuteCommandOutput()
	output.ClusterArn = aws.String(clusterARN)
	output.TaskArn = aws.String(taskARN)
	output.ContainerName = aws.String(containerName)
	return output
}

func validDescribeTasksOutput() *ecs.DescribeTasksOutput {
	return &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{{
		TaskArn: aws.String(execTaskARN),
		Containers: []ecstypes.Container{{
			Name:      aws.String(execContainer),
			RuntimeId: aws.String(execRuntimeID),
		}},
	}}}
}
