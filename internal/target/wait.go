package target

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const eligibleTasksPollInterval = 2 * time.Second

// Clock is the time boundary used while waiting for eligible ECS tasks.
type Clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

type realClock struct{}

// RealClock returns a clock backed by the system clock and context-aware timers.
func RealClock() Clock {
	return realClock{}
}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// WaitForEligibleTasks polls until the selected cluster contains at least one
// task ready for an ECS target operation or maxWait expires.
func (r *Resolver) WaitForEligibleTasks(
	ctx context.Context,
	cluster string,
	service string,
	maxWait time.Duration,
	clock Clock,
) ([]types.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("wait for eligible ECS task: %w", err)
	}

	deadline := clock.Now().Add(maxWait)
	waitCtx := ctx
	cancel := func() {}
	if maxWait > 0 {
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
	}
	defer cancel()

	firstLookup := true
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("wait for eligible ECS task: %w", err)
		}
		if !firstLookup && !clock.Now().Before(deadline) {
			return nil, noEligibleTasksError(cluster, service, maxWait)
		}

		tasks, err := r.EligibleTasks(waitCtx, cluster, service)
		if err != nil {
			if parentErr := ctx.Err(); parentErr != nil {
				return nil, fmt.Errorf("wait for eligible ECS task: %w", parentErr)
			}
			if maxWait > 0 && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return nil, noEligibleTasksError(cluster, service, maxWait)
			}
			return nil, err
		}
		if len(tasks) > 0 {
			return tasks, nil
		}
		if maxWait == 0 || !clock.Now().Before(deadline) {
			return nil, noEligibleTasksError(cluster, service, maxWait)
		}

		remaining := deadline.Sub(clock.Now())
		delay := min(eligibleTasksPollInterval, remaining)
		if err := clock.Sleep(waitCtx, delay); err != nil {
			if parentErr := ctx.Err(); parentErr != nil {
				return nil, fmt.Errorf("wait for eligible ECS task: %w", parentErr)
			}
			if maxWait > 0 && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
				return nil, noEligibleTasksError(cluster, service, maxWait)
			}
			return nil, fmt.Errorf("wait for eligible ECS task: %w", err)
		}
		firstLookup = false
	}
}

func noEligibleTasksError(cluster, service string, maxWait time.Duration) error {
	scope := fmt.Sprintf("cluster %q", cluster)
	if service != "" {
		scope += fmt.Sprintf(" service %q", service)
	}
	return fmt.Errorf(
		"no eligible ECS task became ready in %s within %s: readiness requires a RUNNING task with execute command enabled and a RUNNING container, ExecuteCommandAgent, and non-empty runtime ID",
		scope,
		maxWait,
	)
}
