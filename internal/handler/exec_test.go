package handler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/internal/listview"
	"github.com/wim-web/tnnl/internal/session_manager"
)

const (
	handlerRegion        = "ap-northeast-1"
	handlerClusterARN    = "arn:aws:ecs:ap-northeast-1:123456789012:cluster/production"
	handlerFirstTaskARN  = "arn:aws:ecs:ap-northeast-1:123456789012:task/production/task-first"
	handlerSecondTaskARN = "arn:aws:ecs:ap-northeast-1:123456789012:task/production/task-second"
	handlerContainer     = "app"
	handlerSessionID     = "session-handler"
)

type handlerContextKey struct{}

type handlerPlugin struct {
	events     *[]string
	run        func(context.Context, session_manager.Invocation) error
	calls      int
	ctx        context.Context
	invocation session_manager.Invocation
}

func (p *handlerPlugin) Run(ctx context.Context, invocation session_manager.Invocation) error {
	p.calls++
	p.ctx = ctx
	p.invocation = invocation
	appendEvent(p.events, "plugin-run")
	if p.run != nil {
		return p.run(ctx, invocation)
	}
	return nil
}

type handlerECS struct {
	events *[]string

	listClustersOutput *ecs.ListClustersOutput
	listClustersErr    error
	listTasksOutput    *ecs.ListTasksOutput
	listTasksErr       error
	resolveOutput      *ecs.DescribeTasksOutput
	resolveErr         error
	executeOutput      *ecs.ExecuteCommandOutput
	executeErr         error
	refreshOutput      *ecs.DescribeTasksOutput
	refreshErr         error

	listClustersCalls int
	listClustersCtx   context.Context
	listClustersInput *ecs.ListClustersInput
	listTasksCalls    int
	listTasksCtx      context.Context
	listTasksInput    *ecs.ListTasksInput
	describeCalls     int
	describeContexts  []context.Context
	describeInputs    []*ecs.DescribeTasksInput
	executeCalls      int
	executeCtx        context.Context
	executeInput      *ecs.ExecuteCommandInput
}

func (f *handlerECS) ListClusters(ctx context.Context, in *ecs.ListClustersInput, _ ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	f.listClustersCalls++
	f.listClustersCtx = ctx
	f.listClustersInput = in
	appendEvent(f.events, "list-clusters")
	return f.listClustersOutput, f.listClustersErr
}

func (f *handlerECS) ListTasks(ctx context.Context, in *ecs.ListTasksInput, _ ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	f.listTasksCalls++
	f.listTasksCtx = ctx
	f.listTasksInput = in
	appendEvent(f.events, "list-tasks")
	return f.listTasksOutput, f.listTasksErr
}

func (f *handlerECS) DescribeTasks(ctx context.Context, in *ecs.DescribeTasksInput, _ ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	f.describeCalls++
	f.describeContexts = append(f.describeContexts, ctx)
	f.describeInputs = append(f.describeInputs, in)
	if f.describeCalls == 1 {
		appendEvent(f.events, "describe-targets")
		return f.resolveOutput, f.resolveErr
	}
	appendEvent(f.events, "describe-refresh")
	return f.refreshOutput, f.refreshErr
}

func (f *handlerECS) ExecuteCommand(ctx context.Context, in *ecs.ExecuteCommandInput, _ ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error) {
	f.executeCalls++
	f.executeCtx = ctx
	f.executeInput = in
	appendEvent(f.events, "execute-command")
	return f.executeOutput, f.executeErr
}

type handlerSSM struct {
	events *[]string

	startOutput    *ssm.StartSessionOutput
	startErr       error
	terminateErr   error
	startCalls     int
	startCtx       context.Context
	startInput     *ssm.StartSessionInput
	terminateCalls int
	terminateCtx   context.Context
	terminateInput *ssm.TerminateSessionInput
	onTerminate    func(context.Context)
}

func (f *handlerSSM) StartSession(ctx context.Context, in *ssm.StartSessionInput, _ ...func(*ssm.Options)) (*ssm.StartSessionOutput, error) {
	f.startCalls++
	f.startCtx = ctx
	f.startInput = in
	appendEvent(f.events, "start-session")
	return f.startOutput, f.startErr
}

func (f *handlerSSM) TerminateSession(ctx context.Context, in *ssm.TerminateSessionInput, _ ...func(*ssm.Options)) (*ssm.TerminateSessionOutput, error) {
	f.terminateCalls++
	f.terminateCtx = ctx
	f.terminateInput = in
	appendEvent(f.events, "terminate-session")
	if f.onTerminate != nil {
		f.onTerminate(ctx)
	}
	return &ssm.TerminateSessionOutput{}, f.terminateErr
}

func TestExecHandlerPreflightFailureStopsBeforeAWS(t *testing.T) {
	preflightErr := errors.New("preflight sentinel")
	var events []string
	deps := dependencies{
		preflight: func(context.Context) (session_manager.Plugin, error) {
			appendEvent(&events, "preflight")
			return nil, preflightErr
		},
		loadConfig: func(context.Context) (aws.Config, error) {
			t.Fatal("loadConfig called after preflight failure")
			return aws.Config{}, nil
		},
		newECS: func(aws.Config) ecsAPI { t.Fatal("newECS called"); return nil },
		newSSM: func(aws.Config) ssmAPI { t.Fatal("newSSM called"); return nil },
		choose: func(string, []listview.Option) (string, bool, error) {
			t.Fatal("chooser called")
			return "", false, nil
		},
		availablePort: func() (int, error) { t.Fatal("availablePort called"); return 0, nil },
	}

	err := execHandler(context.Background(), validExecHandlerInput(), deps)
	if !errors.Is(err, preflightErr) {
		t.Fatalf("execHandler() error = %v, want errors.Is(preflightErr)", err)
	}
	if !reflect.DeepEqual(events, []string{"preflight"}) {
		t.Fatalf("events = %#v, want preflight only", events)
	}
}

func TestExecHandlerSelectsExactSecondReplicaAndRunsInOrder(t *testing.T) {
	ctx := context.WithValue(context.Background(), handlerContextKey{}, "exec-context")
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)

	err := execHandler(ctx, validExecHandlerInput(), deps)
	if err != nil {
		t.Fatalf("execHandler() error = %v", err)
	}
	wantEvents := []string{
		"preflight", "load-config", "list-tasks", "describe-targets", "choose-task",
		"execute-command", "describe-refresh", "plugin-run",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", events, wantEvents)
	}
	if ecsClient.executeCalls != 1 || aws.ToString(ecsClient.executeInput.Task) != handlerSecondTaskARN {
		t.Fatalf("ExecuteCommand calls/task = %d/%q, want second replica %q", ecsClient.executeCalls, aws.ToString(ecsClient.executeInput.Task), handlerSecondTaskARN)
	}
	if aws.ToString(ecsClient.executeInput.Cluster) != handlerClusterARN ||
		aws.ToString(ecsClient.executeInput.Container) != handlerContainer ||
		aws.ToString(ecsClient.executeInput.Command) != "/bin/sh" || !ecsClient.executeInput.Interactive {
		t.Fatalf("ExecuteCommand input = %#v", ecsClient.executeInput)
	}
	if got := aws.ToString(ecsClient.listTasksInput.ServiceName); got != "service-web" {
		t.Fatalf("ListTasks service = %q, want service-web", got)
	}
	if got := plugin.invocation.Target; got != "ecs:production_task-second_runtime-refreshed" {
		t.Fatalf("plugin target = %q, want refreshed second replica target", got)
	}
	if plugin.invocation.Region != handlerRegion {
		t.Fatalf("plugin region = %q, want %q", plugin.invocation.Region, handlerRegion)
	}
	assertOriginalContext(t, ctx, ecsClient, plugin)
	if ssmClient.terminateCalls != 0 {
		t.Fatalf("TerminateSession calls = %d, want 0", ssmClient.terminateCalls)
	}
}

func TestExecHandlerPluginCancellationCleansUpWithDetachedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), handlerContextKey{}, "preserved"))
	defer cancel()
	var events []string
	ecsClient := newHandlerECS(&events)
	pluginErr := errors.New("plugin sentinel")
	cleanupErr := errors.New("cleanup sentinel")
	plugin := &handlerPlugin{events: &events, run: func(pluginCtx context.Context, _ session_manager.Invocation) error {
		if pluginCtx != ctx {
			t.Fatalf("plugin context = %v, want original %v", pluginCtx, ctx)
		}
		cancel()
		if !errors.Is(pluginCtx.Err(), context.Canceled) {
			t.Fatalf("plugin context error = %v, want canceled", pluginCtx.Err())
		}
		return errors.Join(pluginErr, pluginCtx.Err())
	}}
	ssmClient := &handlerSSM{
		events:       &events,
		terminateErr: cleanupErr,
		onTerminate: func(cleanupCtx context.Context) {
			if cleanupCtx.Err() != nil {
				t.Fatalf("cleanup context already canceled: %v", cleanupCtx.Err())
			}
			if got := cleanupCtx.Value(handlerContextKey{}); got != "preserved" {
				t.Fatalf("cleanup context value = %#v, want preserved", got)
			}
			if _, ok := cleanupCtx.Deadline(); !ok {
				t.Fatal("cleanup context has no deadline")
			}
		},
	}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)

	err := execHandler(ctx, validExecHandlerInput(), deps)
	for _, want := range []error{pluginErr, context.Canceled, cleanupErr} {
		if !errors.Is(err, want) {
			t.Fatalf("execHandler() error = %v, want errors.Is(%v)", err, want)
		}
	}
	if ssmClient.terminateCalls != 1 || aws.ToString(ssmClient.terminateInput.SessionId) != handlerSessionID {
		t.Fatalf("TerminateSession calls/input = %d/%#v", ssmClient.terminateCalls, ssmClient.terminateInput)
	}
	if len(events) < 2 || !reflect.DeepEqual(events[len(events)-2:], []string{"plugin-run", "terminate-session"}) {
		t.Fatalf("last events = %#v, want plugin then terminate", events)
	}
}

func TestExecHandlerViewCancellationCreatesNoRemoteSession(t *testing.T) {
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	deps.choose = func(_ string, options []listview.Option) (string, bool, error) {
		appendEvent(&events, "choose-task")
		if len(options) != 2 {
			t.Fatalf("task options = %#v, want 2", options)
		}
		return "", true, nil
	}

	if err := execHandler(context.Background(), validExecHandlerInput(), deps); err != nil {
		t.Fatalf("execHandler() error = %v, want nil on view cancellation", err)
	}
	if ecsClient.executeCalls != 0 || ssmClient.startCalls != 0 || ssmClient.terminateCalls != 0 || plugin.calls != 0 {
		t.Fatalf("remote calls after cancellation: execute=%d start=%d terminate=%d plugin=%d", ecsClient.executeCalls, ssmClient.startCalls, ssmClient.terminateCalls, plugin.calls)
	}
}

func TestExecHandlerPreservesConfigAndAPIErrors(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		configErr := errors.New("config sentinel")
		var events []string
		deps := dependencies{
			preflight: func(context.Context) (session_manager.Plugin, error) {
				appendEvent(&events, "preflight")
				return &handlerPlugin{}, nil
			},
			loadConfig: func(context.Context) (aws.Config, error) {
				appendEvent(&events, "load-config")
				return aws.Config{}, configErr
			},
			newECS: func(aws.Config) ecsAPI { t.Fatal("newECS called"); return nil },
			newSSM: func(aws.Config) ssmAPI { t.Fatal("newSSM called"); return nil },
		}
		err := execHandler(context.Background(), validExecHandlerInput(), deps)
		if !errors.Is(err, configErr) || !strings.Contains(err.Error(), "load AWS configuration") {
			t.Fatalf("execHandler() error = %v, want wrapped config error", err)
		}
		if !reflect.DeepEqual(events, []string{"preflight", "load-config"}) {
			t.Fatalf("events = %#v", events)
		}
	})

	t.Run("resolver API", func(t *testing.T) {
		apiErr := errors.New("list tasks sentinel")
		var events []string
		ecsClient := newHandlerECS(&events)
		ecsClient.listTasksErr = apiErr
		deps := handlerDependencies(t, &events, ecsClient, &handlerSSM{events: &events}, &handlerPlugin{events: &events})
		err := execHandler(context.Background(), validExecHandlerInput(), deps)
		if !errors.Is(err, apiErr) {
			t.Fatalf("execHandler() error = %v, want errors.Is(apiErr)", err)
		}
		if ecsClient.executeCalls != 0 {
			t.Fatalf("ExecuteCommand calls = %d, want 0", ecsClient.executeCalls)
		}
	})
}

func handlerDependencies(
	t *testing.T,
	events *[]string,
	ecsClient *handlerECS,
	ssmClient *handlerSSM,
	plugin *handlerPlugin,
) dependencies {
	t.Helper()
	return dependencies{
		preflight: func(ctx context.Context) (session_manager.Plugin, error) {
			appendEvent(events, "preflight")
			if ctx == nil {
				t.Fatal("preflight received nil context")
			}
			return plugin, nil
		},
		loadConfig: func(ctx context.Context) (aws.Config, error) {
			appendEvent(events, "load-config")
			if ctx == nil {
				t.Fatal("loadConfig received nil context")
			}
			return aws.Config{Region: handlerRegion}, nil
		},
		newECS: func(aws.Config) ecsAPI { return ecsClient },
		newSSM: func(aws.Config) ssmAPI { return ssmClient },
		choose: func(title string, options []listview.Option) (string, bool, error) {
			appendEvent(events, "choose-task")
			if !strings.Contains(strings.ToLower(title), "task") {
				t.Fatalf("chooser title = %q, want task", title)
			}
			if len(options) != 2 {
				t.Fatalf("task options = %#v, want two replicas", options)
			}
			if options[0].Value != handlerFirstTaskARN || options[1].Value != handlerSecondTaskARN {
				t.Fatalf("task values = %#v, want exact ARN order", options)
			}
			return options[1].Value, false, nil
		},
		availablePort: func() (int, error) {
			t.Fatal("availablePort called from exec handler")
			return 0, nil
		},
	}
}

func newHandlerECS(events *[]string) *handlerECS {
	first := readyHandlerTask(handlerFirstTaskARN, "runtime-first")
	second := readyHandlerTask(handlerSecondTaskARN, "runtime-second")
	return &handlerECS{
		events:          events,
		listTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{handlerFirstTaskARN, handlerSecondTaskARN}},
		resolveOutput:   &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{first, second}},
		executeOutput: &ecs.ExecuteCommandOutput{
			ClusterArn:    aws.String(handlerClusterARN),
			TaskArn:       aws.String(handlerSecondTaskARN),
			ContainerName: aws.String(handlerContainer),
			Session: &ecstypes.Session{
				SessionId:  aws.String(handlerSessionID),
				StreamUrl:  aws.String("wss://handler.example"),
				TokenValue: aws.String("handler-token"),
			},
		},
		refreshOutput: &ecs.DescribeTasksOutput{Tasks: []ecstypes.Task{
			readyHandlerTask(handlerSecondTaskARN, "runtime-refreshed"),
		}},
	}
}

func readyHandlerTask(taskARN, runtimeID string) ecstypes.Task {
	return ecstypes.Task{
		TaskArn:              aws.String(taskARN),
		Group:                aws.String("service:service-web"),
		LastStatus:           aws.String("RUNNING"),
		EnableExecuteCommand: true,
		Containers: []ecstypes.Container{{
			Name:       aws.String(handlerContainer),
			RuntimeId:  aws.String(runtimeID),
			LastStatus: aws.String("RUNNING"),
			ManagedAgents: []ecstypes.ManagedAgent{{
				Name:       ecstypes.ManagedAgentNameExecuteCommandAgent,
				LastStatus: aws.String("RUNNING"),
			}},
		}},
	}
}

func validExecHandlerInput() input.ExecInput {
	return input.ExecInput{
		EcsParameter: input.EcsParameter{Cluster: handlerClusterARN, Service: "service-web"},
		Cmd:          "/bin/sh",
		Wait:         0,
	}
}

func assertOriginalContext(t *testing.T, ctx context.Context, ecsClient *handlerECS, plugin *handlerPlugin) {
	t.Helper()
	if ecsClient.listTasksCtx != ctx || ecsClient.executeCtx != ctx || plugin.ctx != ctx {
		t.Fatalf("contexts list/execute/plugin = %v/%v/%v, want original %v", ecsClient.listTasksCtx, ecsClient.executeCtx, plugin.ctx, ctx)
	}
	for i, got := range ecsClient.describeContexts {
		if got != ctx {
			t.Fatalf("DescribeTasks context[%d] = %v, want original %v", i, got, ctx)
		}
	}
}

func appendEvent(events *[]string, event string) {
	if events != nil {
		*events = append(*events, event)
	}
}
