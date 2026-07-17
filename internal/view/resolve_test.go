package view

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/wim-web/tnnl/internal/listview"
	"github.com/wim-web/tnnl/internal/target"
)

const (
	viewClusterARN = "arn:aws:ecs:us-east-1:123456789012:cluster/production"
	viewFirstARN   = "arn:aws:ecs:us-east-1:123456789012:task/production/task-first"
	viewSecondARN  = "arn:aws:ecs:us-east-1:123456789012:task/production/task-second"
)

type fakeTargetResolver struct {
	clusters    []string
	clustersErr error
	tasks       []types.Task
	waitErr     error

	calls       []string
	waitCluster string
	waitService string
	waitMax     time.Duration
	waitClock   target.Clock
}

func (f *fakeTargetResolver) Clusters(context.Context) ([]string, error) {
	f.calls = append(f.calls, "clusters")
	return append([]string(nil), f.clusters...), f.clustersErr
}

func (f *fakeTargetResolver) WaitForEligibleTasks(
	_ context.Context,
	cluster string,
	service string,
	maxWait time.Duration,
	clock target.Clock,
) ([]types.Task, error) {
	f.calls = append(f.calls, "wait")
	f.waitCluster = cluster
	f.waitService = service
	f.waitMax = maxWait
	f.waitClock = clock
	return append([]types.Task(nil), f.tasks...), f.waitErr
}

func TestResolveTargetSelectsExactReadyTaskARN(t *testing.T) {
	first := viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime-first"))
	second := viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second"))
	resolver := &fakeTargetResolver{tasks: []types.Task{first, second}}
	chooseCalls := 0
	choose := func(title string, options []listview.Option) (string, bool, error) {
		chooseCalls++
		if !strings.Contains(strings.ToLower(title), "task") {
			t.Fatalf("chooser title = %q, want task title", title)
		}
		if len(options) != 2 {
			t.Fatalf("task option count = %d, want 2", len(options))
		}
		wantValues := []string{viewFirstARN, viewSecondARN}
		for i, option := range options {
			if option.Value != wantValues[i] {
				t.Errorf("task option %d value = %q, want full ARN %q", i, option.Value, wantValues[i])
			}
			if !strings.Contains(option.Label, []string{"task-first", "task-second"}[i]) {
				t.Errorf("task option %d label = %q, want short unique task ID", i, option.Label)
			}
		}
		return options[1].Value, false, nil
	}

	got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "payments", 9*time.Second)
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if quit {
		t.Fatal("ResolveTarget() quit = true, want false")
	}
	if chooseCalls != 1 {
		t.Fatalf("chooser call count = %d, want task chooser only", chooseCalls)
	}
	if got.TaskARN != viewSecondARN {
		t.Errorf("Resolved.TaskARN = %q, want %q", got.TaskARN, viewSecondARN)
	}
	if !reflect.DeepEqual(got.Task, second) {
		t.Errorf("Resolved.Task = %#v, want exact second task %#v", got.Task, second)
	}
	if got.TaskID != "task-second" {
		t.Errorf("Resolved.TaskID = %q, want %q", got.TaskID, "task-second")
	}
	if got.ContainerName != "app" || got.RuntimeID != "runtime-second" {
		t.Errorf("resolved container = (%q, %q), want (%q, %q)", got.ContainerName, got.RuntimeID, "app", "runtime-second")
	}
	if want := "ecs:production_task-second_runtime-second"; got.SSMTarget() != want {
		t.Errorf("Resolved.SSMTarget() = %q, want %q", got.SSMTarget(), want)
	}
}

func TestResolveTargetWaitsAfterClusterResolution(t *testing.T) {
	firstCluster := "arn:aws:ecs:us-east-1:123456789012:cluster/staging"
	resolver := &fakeTargetResolver{
		clusters: []string{firstCluster, viewClusterARN},
		tasks:    []types.Task{viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime"))},
	}
	choose := func(title string, options []listview.Option) (string, bool, error) {
		if !strings.Contains(strings.ToLower(title), "cluster") {
			t.Fatalf("chooser title = %q, want cluster title", title)
		}
		want := []listview.Option{
			{Label: "staging", Value: firstCluster},
			{Label: "production", Value: viewClusterARN},
		}
		if !reflect.DeepEqual(options, want) {
			t.Fatalf("cluster options = %#v, want %#v", options, want)
		}
		return options[1].Value, false, nil
	}

	got, quit, err := ResolveTarget(context.Background(), resolver, choose, "", "payments", 7*time.Second)
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if quit {
		t.Fatal("ResolveTarget() quit = true, want false")
	}
	if want := []string{"clusters", "wait"}; !reflect.DeepEqual(resolver.calls, want) {
		t.Fatalf("resolver call order = %v, want %v", resolver.calls, want)
	}
	if resolver.waitCluster != viewClusterARN {
		t.Errorf("WaitForEligibleTasks cluster = %q, want selected ARN %q", resolver.waitCluster, viewClusterARN)
	}
	if resolver.waitService != "payments" {
		t.Errorf("WaitForEligibleTasks service = %q, want %q", resolver.waitService, "payments")
	}
	if resolver.waitMax != 7*time.Second {
		t.Errorf("WaitForEligibleTasks max wait = %s, want %s", resolver.waitMax, 7*time.Second)
	}
	if resolver.waitClock == nil {
		t.Error("WaitForEligibleTasks clock = nil, want production clock")
	}
	if got.ECSCluster != viewClusterARN || got.ClusterName != "production" {
		t.Errorf("resolved cluster = (%q, %q), want (%q, %q)", got.ECSCluster, got.ClusterName, viewClusterARN, "production")
	}
}

func TestResolveTargetRejectsEmptyResourcesBeforeChooser(t *testing.T) {
	t.Run("no clusters", func(t *testing.T) {
		resolver := &fakeTargetResolver{}
		chooseCalls := 0
		choose := func(string, []listview.Option) (string, bool, error) {
			chooseCalls++
			return "", false, nil
		}

		got, quit, err := ResolveTarget(context.Background(), resolver, choose, "", "", 0)
		assertResolveError(t, got, quit, err, "cluster", "no")
		if chooseCalls != 0 {
			t.Fatalf("chooser call count = %d, want 0", chooseCalls)
		}
		if want := []string{"clusters"}; !reflect.DeepEqual(resolver.calls, want) {
			t.Fatalf("resolver calls = %v, want %v", resolver.calls, want)
		}
	})

	t.Run("no eligible tasks", func(t *testing.T) {
		resolver := &fakeTargetResolver{}
		chooseCalls := 0
		choose := func(string, []listview.Option) (string, bool, error) {
			chooseCalls++
			return "", false, nil
		}

		got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "", 0)
		assertResolveError(t, got, quit, err, "task", "eligible")
		if chooseCalls != 0 {
			t.Fatalf("chooser call count = %d, want 0", chooseCalls)
		}
	})

	t.Run("no eligible containers", func(t *testing.T) {
		noneligible := viewReadyTask(viewFirstARN, "service:payments", types.Container{
			Name:       aws.String("app"),
			LastStatus: aws.String("RUNNING"),
			RuntimeId:  aws.String(""),
		})
		resolver := &fakeTargetResolver{tasks: []types.Task{noneligible}}
		chooseCalls := 0
		choose := func(string, []listview.Option) (string, bool, error) {
			chooseCalls++
			return "", false, nil
		}

		got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "", 0)
		assertResolveError(t, got, quit, err, "container", "eligible")
		if chooseCalls != 0 {
			t.Fatalf("chooser call count = %d, want 0 before container chooser", chooseCalls)
		}
	})
}

func TestResolveTargetAutomaticSelection(t *testing.T) {
	t.Run("one task skips the task chooser", func(t *testing.T) {
		task := viewReadyTask(
			viewFirstARN,
			"service:payments",
			viewReadyContainer("web", "runtime-web"),
			viewReadyContainer("worker", "runtime-worker"),
		)
		resolver := &fakeTargetResolver{tasks: []types.Task{task}}
		chooseCalls := 0
		choose := func(title string, options []listview.Option) (string, bool, error) {
			chooseCalls++
			if !strings.Contains(strings.ToLower(title), "container") {
				t.Fatalf("chooser title = %q, one task should skip task chooser", title)
			}
			return options[1].Value, false, nil
		}

		got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "", 0)
		if err != nil || quit {
			t.Fatalf("ResolveTarget() = (%#v, %t, %v), want successful selection", got, quit, err)
		}
		if chooseCalls != 1 {
			t.Fatalf("chooser call count = %d, want container chooser only", chooseCalls)
		}
		if got.TaskARN != viewFirstARN || got.ContainerName != "worker" {
			t.Errorf("resolved target = (%q, %q), want (%q, %q)", got.TaskARN, got.ContainerName, viewFirstARN, "worker")
		}
	})

	t.Run("one eligible container skips the container chooser", func(t *testing.T) {
		ineligible := viewReadyContainer("pending", "runtime-pending")
		ineligible.ManagedAgents[0].LastStatus = aws.String("PENDING")
		first := viewReadyTask(
			viewFirstARN,
			"service:payments",
			ineligible,
			viewReadyContainer("app", "runtime-app"),
		)
		second := viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second"))
		resolver := &fakeTargetResolver{tasks: []types.Task{first, second}}
		chooseCalls := 0
		choose := func(title string, options []listview.Option) (string, bool, error) {
			chooseCalls++
			if !strings.Contains(strings.ToLower(title), "task") {
				t.Fatalf("chooser title = %q, one eligible container should skip container chooser", title)
			}
			return options[0].Value, false, nil
		}

		got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "", 0)
		if err != nil || quit {
			t.Fatalf("ResolveTarget() = (%#v, %t, %v), want successful selection", got, quit, err)
		}
		if chooseCalls != 1 {
			t.Fatalf("chooser call count = %d, want task chooser only", chooseCalls)
		}
		if got.ContainerName != "app" || got.RuntimeID != "runtime-app" {
			t.Errorf("resolved container = (%q, %q), want (%q, %q)", got.ContainerName, got.RuntimeID, "app", "runtime-app")
		}
	})
}

func TestResolveTargetUserCancellation(t *testing.T) {
	tests := []struct {
		name         string
		resolver     *fakeTargetResolver
		inputCluster string
		wantTitle    string
	}{
		{
			name:         "cluster chooser",
			resolver:     &fakeTargetResolver{clusters: []string{viewClusterARN}},
			inputCluster: "",
			wantTitle:    "cluster",
		},
		{
			name: "task chooser",
			resolver: &fakeTargetResolver{tasks: []types.Task{
				viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime-first")),
				viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second")),
			}},
			inputCluster: "production",
			wantTitle:    "task",
		},
		{
			name: "container chooser",
			resolver: &fakeTargetResolver{tasks: []types.Task{
				viewReadyTask(
					viewFirstARN,
					"service:payments",
					viewReadyContainer("web", "runtime-web"),
					viewReadyContainer("worker", "runtime-worker"),
				),
			}},
			inputCluster: "production",
			wantTitle:    "container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chooseCalls := 0
			choose := func(title string, _ []listview.Option) (string, bool, error) {
				chooseCalls++
				if !strings.Contains(strings.ToLower(title), tt.wantTitle) {
					t.Fatalf("chooser title = %q, want %q resource", title, tt.wantTitle)
				}
				return "", true, nil
			}

			got, quit, err := ResolveTarget(context.Background(), tt.resolver, choose, tt.inputCluster, "", 0)
			if err != nil {
				t.Fatalf("ResolveTarget() error = %v, want nil on user cancellation", err)
			}
			if !quit {
				t.Fatal("ResolveTarget() quit = false, want true")
			}
			if !reflect.DeepEqual(got, target.Resolved{}) {
				t.Fatalf("ResolveTarget() result = %#v, want zero value on cancellation", got)
			}
			if chooseCalls != 1 {
				t.Fatalf("chooser call count = %d, want 1", chooseCalls)
			}
			if tt.wantTitle == "cluster" && len(tt.resolver.calls) != 1 {
				t.Fatalf("resolver calls after cluster cancellation = %v, want no wait call", tt.resolver.calls)
			}
		})
	}
}

func TestResolveTargetPropagatesChooserError(t *testing.T) {
	chooseErr := errors.New("chooser sentinel")
	resolver := &fakeTargetResolver{tasks: []types.Task{
		viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime-first")),
		viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second")),
	}}
	choose := func(string, []listview.Option) (string, bool, error) {
		return "", false, chooseErr
	}

	got, quit, err := ResolveTarget(context.Background(), resolver, choose, "production", "", 0)
	if !errors.Is(err, chooseErr) {
		t.Fatalf("ResolveTarget() error = %v, want errors.Is(chooser sentinel)", err)
	}
	if quit {
		t.Fatal("ResolveTarget() quit = true, want false on chooser error")
	}
	if !reflect.DeepEqual(got, target.Resolved{}) {
		t.Fatalf("ResolveTarget() result = %#v, want zero value on chooser error", got)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "task") {
		t.Errorf("ResolveTarget() error = %q, want task context", err)
	}
}

func TestResolveTargetPreservesFullClusterARN(t *testing.T) {
	resolver := &fakeTargetResolver{
		tasks: []types.Task{viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime"))},
	}
	choose := func(string, []listview.Option) (string, bool, error) {
		t.Fatal("chooser called for one task and one container with explicit cluster")
		return "", false, nil
	}

	got, quit, err := ResolveTarget(context.Background(), resolver, choose, "  "+viewClusterARN+"  ", "payments", 4*time.Second)
	if err != nil || quit {
		t.Fatalf("ResolveTarget() = (%#v, %t, %v), want successful selection", got, quit, err)
	}
	if got.ECSCluster != viewClusterARN {
		t.Errorf("Resolved.ECSCluster = %q, want exact trimmed ARN %q", got.ECSCluster, viewClusterARN)
	}
	if got.ClusterName != "production" {
		t.Errorf("Resolved.ClusterName = %q, want %q", got.ClusterName, "production")
	}
	if resolver.waitCluster != viewClusterARN {
		t.Errorf("WaitForEligibleTasks cluster = %q, want exact ARN %q", resolver.waitCluster, viewClusterARN)
	}
	if wantCalls := []string{"wait"}; !reflect.DeepEqual(resolver.calls, wantCalls) {
		t.Errorf("resolver calls = %v, want %v", resolver.calls, wantCalls)
	}
}

func TestResolveTargetRejectsMalformedMetadata(t *testing.T) {
	badResourceARN := "arn:aws:ecs:us-east-1:123456789012:task/production/not-a-cluster"
	tests := []struct {
		name         string
		resolver     *fakeTargetResolver
		inputCluster string
		wantResource string
		wantWait     bool
	}{
		{
			name:         "input cluster",
			resolver:     &fakeTargetResolver{},
			inputCluster: badResourceARN,
			wantResource: "cluster",
		},
		{
			name:         "listed cluster",
			resolver:     &fakeTargetResolver{clusters: []string{badResourceARN}},
			inputCluster: "",
			wantResource: "cluster",
		},
		{
			name: "task ARN",
			resolver: &fakeTargetResolver{tasks: []types.Task{
				viewReadyTask("not-a-task-arn", "service:payments", viewReadyContainer("app", "runtime")),
			}},
			inputCluster: "production",
			wantResource: "task",
			wantWait:     true,
		},
		{
			name: "container name",
			resolver: &fakeTargetResolver{tasks: []types.Task{
				viewReadyTask(viewFirstARN, "service:payments", types.Container{
					Name:       aws.String("   "),
					LastStatus: aws.String("RUNNING"),
					RuntimeId:  aws.String("runtime"),
					ManagedAgents: []types.ManagedAgent{{
						Name:       types.ManagedAgentNameExecuteCommandAgent,
						LastStatus: aws.String("RUNNING"),
					}},
				}),
			}},
			inputCluster: "production",
			wantResource: "container",
			wantWait:     true,
		},
		{
			name: "container runtime ID",
			resolver: &fakeTargetResolver{tasks: []types.Task{
				viewReadyTask(viewFirstARN, "service:payments", types.Container{
					Name:       aws.String("app"),
					LastStatus: aws.String("RUNNING"),
					RuntimeId:  nil,
					ManagedAgents: []types.ManagedAgent{{
						Name:       types.ManagedAgentNameExecuteCommandAgent,
						LastStatus: aws.String("RUNNING"),
					}},
				}),
			}},
			inputCluster: "production",
			wantResource: "container",
			wantWait:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chooseCalls := 0
			choose := func(string, []listview.Option) (string, bool, error) {
				chooseCalls++
				return "", false, nil
			}

			got, quit, err := ResolveTarget(context.Background(), tt.resolver, choose, tt.inputCluster, "", 0)
			assertResolveError(t, got, quit, err, tt.wantResource)
			if chooseCalls != 0 {
				t.Fatalf("chooser call count = %d, want malformed metadata rejected first", chooseCalls)
			}
			waitCalled := false
			for _, call := range tt.resolver.calls {
				waitCalled = waitCalled || call == "wait"
			}
			if waitCalled != tt.wantWait {
				t.Fatalf("wait called = %t, want %t; calls = %v", waitCalled, tt.wantWait, tt.resolver.calls)
			}
		})
	}
}

func TestTaskOptionsPreserveExactTaskIdentity(t *testing.T) {
	first := viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime-first"))
	second := viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second"))

	got, err := taskOptions([]types.Task{first, second})
	if err != nil {
		t.Fatalf("taskOptions() error = %v", err)
	}
	want := []listview.Option{
		{Label: "service:payments task-first", Value: viewFirstARN},
		{Label: "service:payments task-second", Value: viewSecondARN},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("taskOptions() = %#v, want %#v", got, want)
	}
}

func TestTaskOptionsUsesFallbackLabelAndRejectsMalformedARN(t *testing.T) {
	t.Run("blank group uses task label", func(t *testing.T) {
		task := viewReadyTask(viewFirstARN, "   ", viewReadyContainer("app", "runtime"))
		got, err := taskOptions([]types.Task{task})
		if err != nil {
			t.Fatalf("taskOptions() error = %v", err)
		}
		if want := "task task-first"; got[0].Label != want {
			t.Fatalf("taskOptions()[0].Label = %q, want %q", got[0].Label, want)
		}
	})

	t.Run("malformed task ARN", func(t *testing.T) {
		_, err := taskOptions([]types.Task{{TaskArn: aws.String("malformed")}})
		if err == nil {
			t.Fatal("taskOptions() error = nil, want malformed task error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "task") {
			t.Fatalf("taskOptions() error = %q, want task context", err)
		}
	})
}

func TestTaskByARNUsesOnlyExactFullARN(t *testing.T) {
	first := viewReadyTask(viewFirstARN, "service:payments", viewReadyContainer("app", "runtime-first"))
	second := viewReadyTask(viewSecondARN, "service:payments", viewReadyContainer("app", "runtime-second"))
	tasks := []types.Task{first, second}

	got, err := taskByARN(tasks, viewSecondARN)
	if err != nil {
		t.Fatalf("taskByARN() error = %v", err)
	}
	if !reflect.DeepEqual(got, second) {
		t.Fatalf("taskByARN() = %#v, want exact second task %#v", got, second)
	}

	for _, selected := range []string{"service:payments", "task-second", viewSecondARN + " "} {
		_, err := taskByARN(tasks, selected)
		if err == nil {
			t.Errorf("taskByARN(%q) error = nil, want exact-identity error", selected)
			continue
		}
		if !strings.Contains(err.Error(), selected) || !strings.Contains(err.Error(), "no longer available") {
			t.Errorf("taskByARN(%q) error = %q, want selected value and availability context", selected, err)
		}
	}
}

func viewReadyTask(taskARN, group string, containers ...types.Container) types.Task {
	return types.Task{
		EnableExecuteCommand: true,
		LastStatus:           aws.String("RUNNING"),
		TaskArn:              aws.String(taskARN),
		Group:                aws.String(group),
		Containers:           containers,
	}
}

func viewReadyContainer(name, runtimeID string) types.Container {
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

func assertResolveError(t *testing.T, got target.Resolved, quit bool, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("ResolveTarget() error = nil, want error")
	}
	if quit {
		t.Fatal("ResolveTarget() quit = true, want false on error")
	}
	if !reflect.DeepEqual(got, target.Resolved{}) {
		t.Fatalf("ResolveTarget() result = %#v, want zero value on error", got)
	}
	for _, fragment := range fragments {
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fragment)) {
			t.Errorf("ResolveTarget() error = %q, want fragment %q", err, fragment)
		}
	}
}
