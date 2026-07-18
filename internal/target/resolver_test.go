package target

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type resolverContextMarkerKey struct{}

type listClustersCall struct {
	input  ecs.ListClustersInput
	marker any
}

type listTasksCall struct {
	input  ecs.ListTasksInput
	marker any
}

type describeTasksCall struct {
	input  ecs.DescribeTasksInput
	marker any
}

type fakeECS struct {
	clusterPages          map[string]*ecs.ListClustersOutput
	clusterErrors         map[string]error
	nilClusterOutputs     map[string]bool
	taskPages             map[string]*ecs.ListTasksOutput
	taskErrors            map[string]error
	nilTaskOutputs        map[string]bool
	describe              func(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	callLimitError        error
	listClustersCallLimit int
	listTasksCallLimit    int

	listClustersCalls  []listClustersCall
	listTasksCalls     []listTasksCall
	describeTasksCalls []describeTasksCall
}

var _ ECSAPI = (*fakeECS)(nil)

func (f *fakeECS) ListClusters(
	ctx context.Context,
	input *ecs.ListClustersInput,
	_ ...func(*ecs.Options),
) (*ecs.ListClustersOutput, error) {
	inputCopy := ecs.ListClustersInput{}
	if input != nil {
		inputCopy = *input
	}
	f.listClustersCalls = append(f.listClustersCalls, listClustersCall{
		input:  inputCopy,
		marker: ctx.Value(resolverContextMarkerKey{}),
	})
	if f.listClustersCallLimit > 0 && len(f.listClustersCalls) > f.listClustersCallLimit {
		return nil, f.callLimitError
	}

	token := aws.ToString(inputCopy.NextToken)
	if err := f.clusterErrors[token]; err != nil {
		return nil, err
	}
	if f.nilClusterOutputs[token] {
		return nil, nil
	}
	if page := f.clusterPages[token]; page != nil {
		return page, nil
	}
	return &ecs.ListClustersOutput{}, nil
}

func (f *fakeECS) ListTasks(
	ctx context.Context,
	input *ecs.ListTasksInput,
	_ ...func(*ecs.Options),
) (*ecs.ListTasksOutput, error) {
	inputCopy := ecs.ListTasksInput{}
	if input != nil {
		inputCopy = *input
	}
	f.listTasksCalls = append(f.listTasksCalls, listTasksCall{
		input:  inputCopy,
		marker: ctx.Value(resolverContextMarkerKey{}),
	})
	if f.listTasksCallLimit > 0 && len(f.listTasksCalls) > f.listTasksCallLimit {
		return nil, f.callLimitError
	}

	token := aws.ToString(inputCopy.NextToken)
	if err := f.taskErrors[token]; err != nil {
		return nil, err
	}
	if f.nilTaskOutputs[token] {
		return nil, nil
	}
	if page := f.taskPages[token]; page != nil {
		return page, nil
	}
	return &ecs.ListTasksOutput{}, nil
}

func (f *fakeECS) DescribeTasks(
	ctx context.Context,
	input *ecs.DescribeTasksInput,
	_ ...func(*ecs.Options),
) (*ecs.DescribeTasksOutput, error) {
	inputCopy := ecs.DescribeTasksInput{}
	if input != nil {
		inputCopy = *input
		inputCopy.Tasks = append([]string(nil), input.Tasks...)
	}
	f.describeTasksCalls = append(f.describeTasksCalls, describeTasksCall{
		input:  inputCopy,
		marker: ctx.Value(resolverContextMarkerKey{}),
	})

	if f.describe != nil {
		return f.describe(ctx, input)
	}
	return &ecs.DescribeTasksOutput{}, nil
}

func TestResolverClustersPaginatesInOrder(t *testing.T) {
	client := &fakeECS{
		clusterPages: map[string]*ecs.ListClustersOutput{
			"": {
				ClusterArns: []string{"cluster-a", "cluster-b"},
				NextToken:   aws.String("clusters-page-2"),
			},
			"clusters-page-2": {
				ClusterArns: []string{"cluster-c"},
				NextToken:   aws.String(""),
			},
		},
	}
	ctx := context.WithValue(context.Background(), resolverContextMarkerKey{}, "cluster-marker")

	got, err := NewResolver(client).Clusters(ctx)
	if err != nil {
		t.Fatalf("Clusters() error = %v", err)
	}
	if want := []string{"cluster-a", "cluster-b", "cluster-c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Clusters() = %#v, want %#v", got, want)
	}
	if got := len(client.listClustersCalls); got != 2 {
		t.Fatalf("ListClusters call count = %d, want 2", got)
	}
	if client.listClustersCalls[0].input.NextToken != nil {
		t.Fatalf("first ListClusters NextToken = %q, want nil", aws.ToString(client.listClustersCalls[0].input.NextToken))
	}
	if got := aws.ToString(client.listClustersCalls[1].input.NextToken); got != "clusters-page-2" {
		t.Fatalf("second ListClusters NextToken = %q, want %q", got, "clusters-page-2")
	}
	assertResolverCallMarkers(t, client, "cluster-marker")
}

func TestResolverClustersRejectsRepeatedPaginationTokens(t *testing.T) {
	tests := []struct {
		name      string
		pages     map[string]*ecs.ListClustersOutput
		wantCalls int
		wantToken string
	}{
		{
			name: "same token repeats",
			pages: map[string]*ecs.ListClustersOutput{
				"":       {NextToken: aws.String("repeat")},
				"repeat": {NextToken: aws.String("repeat")},
			},
			wantCalls: 2,
			wantToken: "repeat",
		},
		{
			name: "tokens form a cycle",
			pages: map[string]*ecs.ListClustersOutput{
				"":       {NextToken: aws.String("page-a")},
				"page-a": {NextToken: aws.String("page-b")},
				"page-b": {NextToken: aws.String("page-a")},
			},
			wantCalls: 3,
			wantToken: "page-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callLimitError := errors.New("ListClusters fake call limit reached")
			client := &fakeECS{
				clusterPages:          tt.pages,
				callLimitError:        callLimitError,
				listClustersCallLimit: tt.wantCalls,
			}

			_, err := NewResolver(client).Clusters(context.Background())
			assertResolverPaginationError(t, err, callLimitError, "list ECS clusters", tt.wantToken)
			if got := len(client.listClustersCalls); got != tt.wantCalls {
				t.Errorf("ListClusters call count = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestResolverEligibleTasksPaginatesAndOrderedDeduplicates(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
	}{
		{name: "with service", serviceName: "payments"},
		{name: "without service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeECS{
				taskPages: map[string]*ecs.ListTasksOutput{
					"": {
						TaskArns:  []string{"task-a", "task-b"},
						NextToken: aws.String("tasks-page-2"),
					},
					"tasks-page-2": {
						TaskArns:  []string{"task-b", "task-c"},
						NextToken: aws.String(""),
					},
				},
				describe: describeEveryInputTaskAsEligible,
			}
			ctx := context.WithValue(context.Background(), resolverContextMarkerKey{}, "task-marker")

			got, err := NewResolver(client).EligibleTasks(ctx, "production", tt.serviceName)
			if err != nil {
				t.Fatalf("EligibleTasks() error = %v", err)
			}
			if want := []string{"task-a", "task-b", "task-c"}; !reflect.DeepEqual(resolverTaskARNs(got), want) {
				t.Fatalf("EligibleTasks() ARNs = %#v, want %#v", resolverTaskARNs(got), want)
			}

			if got := len(client.listTasksCalls); got != 2 {
				t.Fatalf("ListTasks call count = %d, want 2", got)
			}
			for i, call := range client.listTasksCalls {
				if got := aws.ToString(call.input.Cluster); got != "production" {
					t.Errorf("ListTasks call %d Cluster = %q, want %q", i, got, "production")
				}
				if got := call.input.DesiredStatus; got != types.DesiredStatusRunning {
					t.Errorf("ListTasks call %d DesiredStatus = %q, want %q", i, got, types.DesiredStatusRunning)
				}
				if tt.serviceName == "" {
					if call.input.ServiceName != nil {
						t.Errorf("ListTasks call %d ServiceName = %q, want nil", i, aws.ToString(call.input.ServiceName))
					}
				} else if got := aws.ToString(call.input.ServiceName); got != tt.serviceName {
					t.Errorf("ListTasks call %d ServiceName = %q, want %q", i, got, tt.serviceName)
				}
			}
			if client.listTasksCalls[0].input.NextToken != nil {
				t.Errorf("first ListTasks NextToken = %q, want nil", aws.ToString(client.listTasksCalls[0].input.NextToken))
			}
			if got := aws.ToString(client.listTasksCalls[1].input.NextToken); got != "tasks-page-2" {
				t.Errorf("second ListTasks NextToken = %q, want %q", got, "tasks-page-2")
			}

			if got := len(client.describeTasksCalls); got != 1 {
				t.Fatalf("DescribeTasks call count = %d, want 1", got)
			}
			if want := []string{"task-a", "task-b", "task-c"}; !reflect.DeepEqual(client.describeTasksCalls[0].input.Tasks, want) {
				t.Errorf("DescribeTasks Tasks = %#v, want %#v", client.describeTasksCalls[0].input.Tasks, want)
			}
			assertResolverCallMarkers(t, client, "task-marker")
		})
	}
}

func TestResolverEligibleTasksRejectsRepeatedPaginationTokens(t *testing.T) {
	tests := []struct {
		name      string
		pages     map[string]*ecs.ListTasksOutput
		wantCalls int
		wantToken string
	}{
		{
			name: "same token repeats",
			pages: map[string]*ecs.ListTasksOutput{
				"":       {NextToken: aws.String("repeat")},
				"repeat": {NextToken: aws.String("repeat")},
			},
			wantCalls: 2,
			wantToken: "repeat",
		},
		{
			name: "tokens form a cycle",
			pages: map[string]*ecs.ListTasksOutput{
				"":       {NextToken: aws.String("page-a")},
				"page-a": {NextToken: aws.String("page-b")},
				"page-b": {NextToken: aws.String("page-a")},
			},
			wantCalls: 3,
			wantToken: "page-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callLimitError := errors.New("ListTasks fake call limit reached")
			client := &fakeECS{
				taskPages:          tt.pages,
				callLimitError:     callLimitError,
				listTasksCallLimit: tt.wantCalls,
			}

			_, err := NewResolver(client).EligibleTasks(context.Background(), "cyclic-cluster", "")
			assertResolverPaginationError(
				t,
				err,
				callLimitError,
				"list ECS tasks",
				"cyclic-cluster",
				tt.wantToken,
			)
			if got := len(client.listTasksCalls); got != tt.wantCalls {
				t.Errorf("ListTasks call count = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestResolverEligibleTasksChunksDescribeRequestsAndPreservesResponseOrder(t *testing.T) {
	arns := make([]string, 201)
	for i := range arns {
		arns[i] = fmt.Sprintf("task-%03d", i)
	}

	client := &fakeECS{
		taskPages: map[string]*ecs.ListTasksOutput{
			"": {TaskArns: arns},
		},
		describe: func(_ context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
			tasks := make([]types.Task, 0, len(input.Tasks))
			for i := len(input.Tasks) - 1; i >= 0; i-- {
				tasks = append(tasks, resolverEligibleTask(input.Tasks[i]))
			}
			return &ecs.DescribeTasksOutput{Tasks: tasks}, nil
		},
	}
	ctx := context.WithValue(context.Background(), resolverContextMarkerKey{}, "batch-marker")

	got, err := NewResolver(client).EligibleTasks(ctx, "batch-cluster", "")
	if err != nil {
		t.Fatalf("EligibleTasks() error = %v", err)
	}
	if gotCalls := len(client.describeTasksCalls); gotCalls != 3 {
		t.Fatalf("DescribeTasks call count = %d, want 3", gotCalls)
	}

	wantBatchLengths := []int{100, 100, 1}
	var wantOrder []string
	for i, call := range client.describeTasksCalls {
		if gotLength := len(call.input.Tasks); gotLength != wantBatchLengths[i] {
			t.Errorf("DescribeTasks call %d task count = %d, want %d", i, gotLength, wantBatchLengths[i])
		}
		start := i * 100
		end := min(start+100, len(arns))
		if want := arns[start:end]; !reflect.DeepEqual(call.input.Tasks, want) {
			t.Errorf("DescribeTasks call %d Tasks = %#v, want %#v", i, call.input.Tasks, want)
		}
		if gotCluster := aws.ToString(call.input.Cluster); gotCluster != "batch-cluster" {
			t.Errorf("DescribeTasks call %d Cluster = %q, want %q", i, gotCluster, "batch-cluster")
		}
		for j := end - 1; j >= start; j-- {
			wantOrder = append(wantOrder, arns[j])
		}
	}
	if gotOrder := resolverTaskARNs(got); !reflect.DeepEqual(gotOrder, wantOrder) {
		t.Fatalf("EligibleTasks() response order = %#v, want %#v", gotOrder, wantOrder)
	}
	assertResolverCallMarkers(t, client, "batch-marker")
}

func TestResolverDescribeTaskBatchesUseIndependentSlices(t *testing.T) {
	arns := make([]string, 201)
	for i := range arns {
		arns[i] = fmt.Sprintf("task-%03d", i)
	}

	client := &fakeECS{
		taskPages: map[string]*ecs.ListTasksOutput{
			"": {TaskArns: arns},
		},
		describe: func(_ context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
			if len(input.Tasks) > 0 && input.Tasks[0] == "task-000" {
				input.Tasks = append(input.Tasks, "client-appended-task")
			}
			return &ecs.DescribeTasksOutput{}, nil
		},
	}

	_, err := NewResolver(client).EligibleTasks(context.Background(), "batch-cluster", "")
	if err != nil {
		t.Fatalf("EligibleTasks() error = %v", err)
	}
	if got := len(client.describeTasksCalls); got != 3 {
		t.Fatalf("DescribeTasks call count = %d, want 3", got)
	}
	if got := client.describeTasksCalls[1].input.Tasks[0]; got != "task-100" {
		t.Fatalf("second DescribeTasks batch starts with %q, want %q", got, "task-100")
	}
}

func TestResolverEligibleTasksReturnsJoinedDescribeFailures(t *testing.T) {
	client := &fakeECS{
		taskPages: map[string]*ecs.ListTasksOutput{
			"": {TaskArns: []string{"task-ok", "task-missing", "task-stopped"}},
		},
		describe: func(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
			return &ecs.DescribeTasksOutput{
				Tasks: []types.Task{resolverEligibleTask("task-ok")},
				Failures: []types.Failure{
					{
						Arn:    aws.String("task-missing"),
						Reason: aws.String("MISSING"),
						Detail: aws.String("task could not be found"),
					},
					{
						Arn:    aws.String("task-stopped"),
						Reason: aws.String("INACTIVE"),
						Detail: aws.String("task is no longer available"),
					},
				},
			}, nil
		},
	}

	got, err := NewResolver(client).EligibleTasks(context.Background(), "production", "")
	if err == nil {
		t.Fatal("EligibleTasks() error = nil, want describe failure")
	}
	if len(got) != 0 {
		t.Errorf("EligibleTasks() returned %d partial tasks, want none on failure", len(got))
	}
	for _, fragment := range []string{
		"describe ECS task task-missing: MISSING: task could not be found",
		"describe ECS task task-stopped: INACTIVE: task is no longer available",
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("EligibleTasks() error = %q, want fragment %q", err, fragment)
		}
	}
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		t.Fatalf("EligibleTasks() error type = %T, want errors.Join-compatible error", err)
	}
	if gotFailures := len(joined.Unwrap()); gotFailures != 2 {
		t.Errorf("joined failure count = %d, want 2", gotFailures)
	}
}

func TestResolverEligibleTasksFormatsBlankDescribeFailureFields(t *testing.T) {
	client := &fakeECS{
		taskPages: map[string]*ecs.ListTasksOutput{
			"": {TaskArns: []string{"task-unknown"}},
		},
		describe: func(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
			return &ecs.DescribeTasksOutput{Failures: []types.Failure{{
				Reason: aws.String("   "),
			}}}, nil
		},
	}

	_, err := NewResolver(client).EligibleTasks(context.Background(), "production", "")
	if err == nil {
		t.Fatal("EligibleTasks() error = nil, want describe failure")
	}
	if want := "describe ECS task <unknown ARN>: <unknown reason>: <no detail>"; !strings.Contains(err.Error(), want) {
		t.Fatalf("EligibleTasks() error = %q, want fragment %q", err, want)
	}
}

func TestResolverEligibleTasksSkipsDescribeForEmptyTaskPages(t *testing.T) {
	client := &fakeECS{
		taskPages: map[string]*ecs.ListTasksOutput{
			"": {
				NextToken: aws.String("empty-page-2"),
			},
			"empty-page-2": {},
		},
	}

	got, err := NewResolver(client).EligibleTasks(context.Background(), "empty-cluster", "")
	if err != nil {
		t.Fatalf("EligibleTasks() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("EligibleTasks() returned %d tasks, want empty", len(got))
	}
	if gotCalls := len(client.listTasksCalls); gotCalls != 2 {
		t.Errorf("ListTasks call count = %d, want 2", gotCalls)
	}
	if gotCalls := len(client.describeTasksCalls); gotCalls != 0 {
		t.Errorf("DescribeTasks call count = %d, want 0", gotCalls)
	}
}

func TestResolverAPIErrorsPreserveCauseAndContext(t *testing.T) {
	sentinel := errors.New("sentinel ECS error")

	t.Run("list clusters", func(t *testing.T) {
		client := &fakeECS{clusterErrors: map[string]error{"": sentinel}}
		_, err := NewResolver(client).Clusters(context.Background())
		assertResolverAPIError(t, err, sentinel, "list ECS clusters")
	})

	t.Run("list tasks", func(t *testing.T) {
		client := &fakeECS{taskErrors: map[string]error{"": sentinel}}
		_, err := NewResolver(client).EligibleTasks(context.Background(), "critical-cluster", "")
		assertResolverAPIError(t, err, sentinel, "list ECS tasks", "critical-cluster")
	})

	t.Run("describe tasks", func(t *testing.T) {
		client := &fakeECS{
			taskPages: map[string]*ecs.ListTasksOutput{
				"": {TaskArns: []string{"task-a"}},
			},
			describe: func(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
				return nil, sentinel
			},
		}
		_, err := NewResolver(client).EligibleTasks(context.Background(), "critical-cluster", "")
		assertResolverAPIError(t, err, sentinel, "describe ECS tasks", "critical-cluster")
	})
}

func TestResolverRejectsNilAPIOutputs(t *testing.T) {
	tests := []struct {
		name          string
		client        *fakeECS
		call          func(*Resolver) error
		wantFragments []string
	}{
		{
			name: "list clusters",
			client: &fakeECS{
				nilClusterOutputs: map[string]bool{"": true},
			},
			call: func(resolver *Resolver) error {
				_, err := resolver.Clusters(context.Background())
				return err
			},
			wantFragments: []string{"list ECS clusters", "nil response"},
		},
		{
			name: "list tasks",
			client: &fakeECS{
				nilTaskOutputs: map[string]bool{"": true},
			},
			call: func(resolver *Resolver) error {
				_, err := resolver.EligibleTasks(context.Background(), "nil-output-cluster", "")
				return err
			},
			wantFragments: []string{"list ECS tasks", "nil-output-cluster", "nil response"},
		},
		{
			name: "describe tasks",
			client: &fakeECS{
				taskPages: map[string]*ecs.ListTasksOutput{
					"": {TaskArns: []string{"task-a"}},
				},
				describe: func(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
					return nil, nil
				},
			},
			call: func(resolver *Resolver) error {
				_, err := resolver.EligibleTasks(context.Background(), "nil-output-cluster", "")
				return err
			},
			wantFragments: []string{"describe ECS tasks", "nil-output-cluster", "nil response"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := captureResolverPanic(func() error {
				return tt.call(NewResolver(tt.client))
			})
			if err == nil {
				t.Fatal("error = nil, want nil API response error")
			}
			for _, fragment := range tt.wantFragments {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("error = %q, want fragment %q", err, fragment)
				}
			}
		})
	}
}

func describeEveryInputTaskAsEligible(_ context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	tasks := make([]types.Task, 0, len(input.Tasks))
	for _, arn := range input.Tasks {
		tasks = append(tasks, resolverEligibleTask(arn))
	}
	return &ecs.DescribeTasksOutput{Tasks: tasks}, nil
}

func resolverEligibleTask(arn string) types.Task {
	return types.Task{
		EnableExecuteCommand: true,
		LastStatus:           aws.String("RUNNING"),
		TaskArn:              aws.String(arn),
		Containers: []types.Container{{
			Name:       aws.String("application"),
			LastStatus: aws.String("RUNNING"),
			RuntimeId:  aws.String("runtime-id"),
			ManagedAgents: []types.ManagedAgent{{
				Name:       types.ManagedAgentNameExecuteCommandAgent,
				LastStatus: aws.String("RUNNING"),
			}},
		}},
	}
}

func resolverTaskARNs(tasks []types.Task) []string {
	arns := make([]string, 0, len(tasks))
	for _, task := range tasks {
		arns = append(arns, aws.ToString(task.TaskArn))
	}
	return arns
}

func assertResolverCallMarkers(t *testing.T, client *fakeECS, want any) {
	t.Helper()
	for i, call := range client.listClustersCalls {
		if call.marker != want {
			t.Errorf("ListClusters call %d context marker = %#v, want %#v", i, call.marker, want)
		}
	}
	for i, call := range client.listTasksCalls {
		if call.marker != want {
			t.Errorf("ListTasks call %d context marker = %#v, want %#v", i, call.marker, want)
		}
	}
	for i, call := range client.describeTasksCalls {
		if call.marker != want {
			t.Errorf("DescribeTasks call %d context marker = %#v, want %#v", i, call.marker, want)
		}
	}
}

func assertResolverAPIError(t *testing.T, got, wantCause error, wantFragments ...string) {
	t.Helper()
	if got == nil {
		t.Fatal("error = nil, want API error")
	}
	if !errors.Is(got, wantCause) {
		t.Errorf("errors.Is(%v, sentinel) = false", got)
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(got.Error(), fragment) {
			t.Errorf("error = %q, want fragment %q", got, fragment)
		}
	}
}

func assertResolverPaginationError(t *testing.T, got, callLimitError error, wantFragments ...string) {
	t.Helper()
	if got == nil {
		t.Fatal("error = nil, want repeated pagination token error")
	}
	if errors.Is(got, callLimitError) {
		t.Errorf("error = %v, resolver reached fake call limit before detecting repeated token", got)
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(got.Error(), fragment) {
			t.Errorf("error = %q, want fragment %q", got, fragment)
		}
	}
}

func captureResolverPanic(call func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return call()
}
