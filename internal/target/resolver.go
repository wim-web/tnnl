package target

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const describeTasksLimit = 100

// ECSAPI is the subset of the ECS client used to resolve executable targets.
type ECSAPI interface {
	ListClusters(context.Context, *ecs.ListClustersInput, ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListTasks(context.Context, *ecs.ListTasksInput, ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput, ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

// Resolver retrieves ECS resources that are eligible for a target operation.
type Resolver struct {
	client ECSAPI
}

// NewResolver creates a Resolver backed by client.
func NewResolver(client ECSAPI) *Resolver {
	return &Resolver{client: client}
}

// Clusters returns every ECS cluster ARN in API response order.
func (r *Resolver) Clusters(ctx context.Context) ([]string, error) {
	var (
		clusters  []string
		nextToken *string
	)
	seenTokens := make(map[string]struct{})

	for {
		output, err := r.client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: nextToken})
		if err != nil {
			return nil, fmt.Errorf("list ECS clusters: %w", err)
		}
		if output == nil {
			return nil, fmt.Errorf("list ECS clusters: nil response")
		}
		clusters = append(clusters, output.ClusterArns...)
		token := aws.ToString(output.NextToken)
		if token == "" {
			return clusters, nil
		}
		if _, seen := seenTokens[token]; seen {
			return nil, fmt.Errorf("list ECS clusters: repeated pagination token %q", token)
		}
		seenTokens[token] = struct{}{}
		nextToken = output.NextToken
	}
}

// EligibleTasks returns every task in cluster that is ready for an ECS target
// operation. When serviceName is non-empty, only that service is queried.
func (r *Resolver) EligibleTasks(ctx context.Context, cluster, serviceName string) ([]types.Task, error) {
	arns, err := r.taskARNs(ctx, cluster, serviceName)
	if err != nil {
		return nil, err
	}
	if len(arns) == 0 {
		return nil, nil
	}

	tasks, err := r.describeTasks(ctx, cluster, arns)
	if err != nil {
		return nil, err
	}

	eligible := make([]types.Task, 0, len(tasks))
	for _, task := range tasks {
		if IsEligibleTask(task) {
			eligible = append(eligible, task)
		}
	}
	return eligible, nil
}

func (r *Resolver) taskARNs(ctx context.Context, cluster, serviceName string) ([]string, error) {
	var (
		arns      []string
		nextToken *string
	)
	seenARNs := make(map[string]struct{})
	seenTokens := make(map[string]struct{})

	for {
		input := &ecs.ListTasksInput{
			Cluster:       aws.String(cluster),
			DesiredStatus: types.DesiredStatusRunning,
			NextToken:     nextToken,
		}
		if serviceName != "" {
			input.ServiceName = aws.String(serviceName)
		}

		output, err := r.client.ListTasks(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("list ECS tasks in cluster %q: %w", cluster, err)
		}
		if output == nil {
			return nil, fmt.Errorf("list ECS tasks in cluster %q: nil response", cluster)
		}
		for _, arn := range output.TaskArns {
			if _, exists := seenARNs[arn]; exists {
				continue
			}
			seenARNs[arn] = struct{}{}
			arns = append(arns, arn)
		}
		token := aws.ToString(output.NextToken)
		if token == "" {
			return arns, nil
		}
		if _, seen := seenTokens[token]; seen {
			return nil, fmt.Errorf("list ECS tasks in cluster %q: repeated pagination token %q", cluster, token)
		}
		seenTokens[token] = struct{}{}
		nextToken = output.NextToken
	}
}

func (r *Resolver) describeTasks(ctx context.Context, cluster string, arns []string) ([]types.Task, error) {
	var tasks []types.Task
	for start := 0; start < len(arns); start += describeTasksLimit {
		end := min(start+describeTasksLimit, len(arns))
		batch := append([]string(nil), arns[start:end]...)
		output, err := r.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(cluster),
			Tasks:   batch,
		})
		if err != nil {
			return nil, fmt.Errorf("describe ECS tasks in cluster %q: %w", cluster, err)
		}
		if output == nil {
			return nil, fmt.Errorf("describe ECS tasks in cluster %q: nil response", cluster)
		}
		if err := describeFailuresError(output.Failures); err != nil {
			return nil, err
		}
		tasks = append(tasks, output.Tasks...)
	}
	return tasks, nil
}

func describeFailuresError(failures []types.Failure) error {
	errs := make([]error, 0, len(failures))
	for _, failure := range failures {
		errs = append(errs, fmt.Errorf(
			"describe ECS task %s: %s: %s",
			failureValue(failure.Arn, "<unknown ARN>"),
			failureValue(failure.Reason, "<unknown reason>"),
			failureValue(failure.Detail, "<no detail>"),
		))
	}
	return errors.Join(errs...)
}

func failureValue(value *string, placeholder string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return placeholder
	}
	return *value
}
