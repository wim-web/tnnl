package target

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type waitContextKey struct{}

type fakeWaitClock struct {
	now     time.Time
	sleeps  []time.Duration
	onSleep func(context.Context, time.Duration) error
}

func (c *fakeWaitClock) Now() time.Time {
	return c.now
}

func (c *fakeWaitClock) Sleep(ctx context.Context, delay time.Duration) error {
	c.sleeps = append(c.sleeps, delay)
	c.now = c.now.Add(delay)
	if c.onSleep != nil {
		return c.onSleep(ctx, delay)
	}
	return nil
}

type waitECS struct {
	listTasks        func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error)
	describeTasks    func(context.Context, *ecs.DescribeTasksInput, int) (*ecs.DescribeTasksOutput, error)
	listContexts     []context.Context
	describeContexts []context.Context
}

var _ ECSAPI = (*waitECS)(nil)

func (f *waitECS) ListClusters(
	context.Context,
	*ecs.ListClustersInput,
	...func(*ecs.Options),
) (*ecs.ListClustersOutput, error) {
	return &ecs.ListClustersOutput{}, nil
}

func (f *waitECS) ListTasks(
	ctx context.Context,
	input *ecs.ListTasksInput,
	_ ...func(*ecs.Options),
) (*ecs.ListTasksOutput, error) {
	call := len(f.listContexts)
	f.listContexts = append(f.listContexts, ctx)
	if f.listTasks != nil {
		return f.listTasks(ctx, input, call)
	}
	return &ecs.ListTasksOutput{}, nil
}

func (f *waitECS) DescribeTasks(
	ctx context.Context,
	input *ecs.DescribeTasksInput,
	_ ...func(*ecs.Options),
) (*ecs.DescribeTasksOutput, error) {
	call := len(f.describeContexts)
	f.describeContexts = append(f.describeContexts, ctx)
	if f.describeTasks != nil {
		return f.describeTasks(ctx, input, call)
	}

	tasks := make([]types.Task, 0, len(input.Tasks))
	for _, taskARN := range input.Tasks {
		tasks = append(tasks, waitReadyTask(taskARN))
	}
	return &ecs.DescribeTasksOutput{Tasks: tasks}, nil
}

func TestWaitForEligibleTasks(t *testing.T) {
	t.Run("zero wait performs one ready lookup with the caller context", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/ready"
		client := &waitECS{
			listTasks: func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error) {
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
		}
		clock := &fakeWaitClock{now: time.Now()}
		ctx := context.WithValue(context.Background(), waitContextKey{}, "caller")

		got, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", 0, clock)
		if err != nil {
			t.Fatalf("WaitForEligibleTasks() error = %v", err)
		}
		if gotARN := aws.ToString(got[0].TaskArn); gotARN != taskARN {
			t.Fatalf("WaitForEligibleTasks() task ARN = %q, want %q", gotARN, taskARN)
		}
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
		if client.listContexts[0] != ctx {
			t.Errorf("zero-wait API context differs from caller context")
		}
		if _, ok := client.listContexts[0].Deadline(); ok {
			t.Error("zero-wait API context has a deadline, want caller context unchanged")
		}
		if len(clock.sleeps) != 0 {
			t.Fatalf("Sleep call count = %d, want 0", len(clock.sleeps))
		}
	})

	t.Run("zero wait performs one empty lookup and returns details", func(t *testing.T) {
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}
		ctx := context.Background()

		_, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", 0, clock)
		assertNoEligibleTasksError(t, err, "production", "0s", "eligible", "ready")
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
		if client.listContexts[0] != ctx {
			t.Errorf("zero-wait API context differs from caller context")
		}
		if _, ok := client.listContexts[0].Deadline(); ok {
			t.Error("zero-wait API context has a deadline, want caller context unchanged")
		}
	})

	t.Run("pending then ready returns the fresh final tasks", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/fresh"
		client := &waitECS{
			listTasks: func(_ context.Context, _ *ecs.ListTasksInput, call int) (*ecs.ListTasksOutput, error) {
				if call == 0 {
					return &ecs.ListTasksOutput{}, nil
				}
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
		}
		clock := &fakeWaitClock{now: time.Now()}

		got, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "payments", 5*time.Second, clock)
		if err != nil {
			t.Fatalf("WaitForEligibleTasks() error = %v", err)
		}
		want := []types.Task{waitReadyTask(taskARN)}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("WaitForEligibleTasks() = %#v, want fresh response %#v", got, want)
		}
		if calls := len(client.listContexts); calls != 2 {
			t.Fatalf("EligibleTasks lookup count = %d, want 2", calls)
		}
		if calls := len(client.describeContexts); calls != 1 {
			t.Fatalf("DescribeTasks call count = %d, want 1", calls)
		}
		if wantSleeps := []time.Duration{2 * time.Second}; !reflect.DeepEqual(clock.sleeps, wantSleeps) {
			t.Fatalf("Sleep durations = %v, want %v", clock.sleeps, wantSleeps)
		}
	})

	t.Run("timeout names cluster service duration and readiness", func(t *testing.T) {
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}

		_, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "payments", 3*time.Second, clock)
		assertNoEligibleTasksError(t, err, "production", "payments", "3s", "eligible", "ready")
		if calls := len(client.listContexts); calls != 2 {
			t.Fatalf("EligibleTasks lookup count = %d, want 2", calls)
		}
		if wantSleeps := []time.Duration{2 * time.Second, time.Second}; !reflect.DeepEqual(clock.sleeps, wantSleeps) {
			t.Fatalf("Sleep durations = %v, want %v", clock.sleeps, wantSleeps)
		}
	})

	t.Run("API sentinel returns immediately and remains discoverable", func(t *testing.T) {
		apiErr := errors.New("list tasks sentinel")
		client := &waitECS{
			listTasks: func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error) {
				return nil, apiErr
			},
		}
		clock := &fakeWaitClock{now: time.Now()}

		_, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "payments", time.Minute, clock)
		if !errors.Is(err, apiErr) {
			t.Fatalf("WaitForEligibleTasks() error = %v, want errors.Is(API sentinel)", err)
		}
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
		if len(clock.sleeps) != 0 {
			t.Fatalf("Sleep call count = %d, want 0", len(clock.sleeps))
		}
	})

	t.Run("negative wait is rejected before an API lookup", func(t *testing.T) {
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}

		_, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "", -time.Second, clock)
		if err == nil {
			t.Fatal("WaitForEligibleTasks() error = nil, want negative-wait validation error")
		}
		for _, fragment := range []string{"non-negative", "-1s"} {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("WaitForEligibleTasks() error = %q, want fragment %q", err, fragment)
			}
		}
		if calls := len(client.listContexts); calls != 0 {
			t.Fatalf("EligibleTasks lookup count = %d, want 0", calls)
		}
		if len(clock.sleeps) != 0 {
			t.Fatalf("Sleep call count = %d, want 0", len(clock.sleeps))
		}
	})

	t.Run("already canceled parent returns context canceled without lookup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}

		_, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", time.Minute, clock)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForEligibleTasks() error = %v, want errors.Is(context.Canceled)", err)
		}
		if calls := len(client.listContexts); calls != 0 {
			t.Fatalf("EligibleTasks lookup count = %d, want 0 for canceled parent", calls)
		}
	})

	t.Run("parent cancellation during sleep remains discoverable", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}
		clock.onSleep = func(sleepCtx context.Context, _ time.Duration) error {
			cancel()
			return sleepCtx.Err()
		}

		_, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", time.Minute, clock)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForEligibleTasks() error = %v, want errors.Is(context.Canceled)", err)
		}
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
	})

	t.Run("parent cancellation before a ready API response wins over success", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/ready"
		ctx, cancel := context.WithCancel(context.Background())
		client := &waitECS{
			listTasks: func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error) {
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
			describeTasks: func(_ context.Context, _ *ecs.DescribeTasksInput, _ int) (*ecs.DescribeTasksOutput, error) {
				cancel()
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{waitReadyTask(taskARN)}}, nil
			},
		}
		clock := &fakeWaitClock{now: time.Now()}

		got, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", 5*time.Second, clock)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForEligibleTasks() = (%#v, %v), want errors.Is(context.Canceled)", got, err)
		}
	})

	t.Run("child deadline before a ready API response wins over success", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/ready"
		client := &waitECS{
			listTasks: func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error) {
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
			describeTasks: func(ctx context.Context, _ *ecs.DescribeTasksInput, _ int) (*ecs.DescribeTasksOutput, error) {
				<-ctx.Done()
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{waitReadyTask(taskARN)}}, nil
			},
		}
		clock := &fakeWaitClock{now: time.Now()}

		got, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "", 20*time.Millisecond, clock)
		if len(got) != 0 {
			t.Fatalf("WaitForEligibleTasks() returned %d tasks after deadline, want none", len(got))
		}
		assertNoEligibleTasksError(t, err, "production", "20ms", "eligible", "ready")
	})

	t.Run("fake deadline before a ready API response wins over success", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/ready"
		maxWait := 5 * time.Second
		clock := &fakeWaitClock{now: time.Now()}
		client := &waitECS{
			listTasks: func(context.Context, *ecs.ListTasksInput, int) (*ecs.ListTasksOutput, error) {
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
			describeTasks: func(_ context.Context, _ *ecs.DescribeTasksInput, _ int) (*ecs.DescribeTasksOutput, error) {
				clock.now = clock.now.Add(maxWait)
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{waitReadyTask(taskARN)}}, nil
			},
		}

		got, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "production", "", maxWait, clock)
		if len(got) != 0 {
			t.Fatalf("WaitForEligibleTasks() returned %d tasks after fake deadline, want none", len(got))
		}
		assertNoEligibleTasksError(t, err, "production", "5s", "eligible", "ready")
	})

	t.Run("sleep reaching the deadline prevents another lookup", func(t *testing.T) {
		client := &waitECS{}
		clock := &fakeWaitClock{now: time.Now()}

		_, err := NewResolver(client).WaitForEligibleTasks(context.Background(), "boundary", "", 2*time.Second, clock)
		assertNoEligibleTasksError(t, err, "boundary", "2s", "eligible", "ready")
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
		if wantSleeps := []time.Duration{2 * time.Second}; !reflect.DeepEqual(clock.sleeps, wantSleeps) {
			t.Fatalf("Sleep durations = %v, want %v", clock.sleeps, wantSleeps)
		}
	})

	t.Run("positive wait uses a real API timeout independent of the fake clock epoch", func(t *testing.T) {
		const taskARN = "arn:aws:ecs:us-east-1:123456789012:task/production/ready"
		maxWait := 5 * time.Second
		client := &waitECS{
			listTasks: func(ctx context.Context, _ *ecs.ListTasksInput, _ int) (*ecs.ListTasksOutput, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return &ecs.ListTasksOutput{TaskArns: []string{taskARN}}, nil
			},
			describeTasks: func(ctx context.Context, _ *ecs.DescribeTasksInput, _ int) (*ecs.DescribeTasksOutput, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{waitReadyTask(taskARN)}}, nil
			},
		}
		clock := &fakeWaitClock{now: time.Unix(123, 0)}
		ctx := context.Background()

		before := time.Now()
		got, err := NewResolver(client).WaitForEligibleTasks(ctx, "production", "", maxWait, clock)
		after := time.Now()
		if err != nil {
			t.Fatalf("WaitForEligibleTasks() error = %v", err)
		}
		if gotARN := aws.ToString(got[0].TaskArn); gotARN != taskARN {
			t.Fatalf("WaitForEligibleTasks() task ARN = %q, want %q", gotARN, taskARN)
		}
		if calls := len(client.listContexts); calls != 1 {
			t.Fatalf("EligibleTasks lookup count = %d, want 1", calls)
		}
		deadline, ok := client.listContexts[0].Deadline()
		if !ok {
			t.Fatal("positive-wait API context has no deadline")
		}
		if deadline.Before(before.Add(maxWait)) {
			t.Fatalf("API context deadline = %v, earlier than lower bound %v", deadline, before.Add(maxWait))
		}
		if deadline.After(after.Add(maxWait)) {
			t.Fatalf("API context deadline = %v, later than upper bound %v", deadline, after.Add(maxWait))
		}
	})
}

func waitReadyTask(taskARN string) types.Task {
	return types.Task{
		EnableExecuteCommand: true,
		LastStatus:           aws.String("RUNNING"),
		TaskArn:              aws.String(taskARN),
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

func assertNoEligibleTasksError(t *testing.T, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("WaitForEligibleTasks() error = nil, want no-eligible-task error")
	}
	for _, fragment := range fragments {
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fragment)) {
			t.Errorf("WaitForEligibleTasks() error = %q, want fragment %q", err, fragment)
		}
	}
}
