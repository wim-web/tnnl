package target

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// ClusterName returns the short cluster name from a short name or ECS ARN.
func ClusterName(input string) (string, error) {
	original := input
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("cluster identifier %q is empty", original)
	}

	if strings.HasPrefix(input, "arn:") {
		parsed, err := arn.Parse(input)
		if err != nil {
			return "", fmt.Errorf("invalid cluster ARN %q: %w", original, err)
		}
		if parsed.Service != "ecs" {
			return "", fmt.Errorf("cluster ARN %q is not an ECS ARN", original)
		}

		parts := strings.Split(parsed.Resource, "/")
		if len(parts) != 2 || parts[0] != "cluster" {
			return "", fmt.Errorf("cluster ARN %q must have resource cluster/<name>", original)
		}
		name, err := finalSegment(parts)
		if err != nil {
			return "", fmt.Errorf("invalid cluster ARN %q: %w", original, err)
		}
		return name, nil
	}

	if strings.Contains(input, "/") {
		return "", fmt.Errorf("cluster name %q must not contain a slash", original)
	}
	return input, nil
}

// TaskID returns the task ID from a short or long ECS task resource or ARN.
func TaskID(input string) (string, error) {
	original := input
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("task identifier %q is empty", original)
	}

	resource := input
	if strings.HasPrefix(input, "arn:") {
		parsed, err := arn.Parse(input)
		if err != nil {
			return "", fmt.Errorf("invalid task ARN %q: %w", original, err)
		}
		if parsed.Service != "ecs" {
			return "", fmt.Errorf("task ARN %q is not an ECS ARN", original)
		}
		resource = parsed.Resource
	}

	if !strings.HasPrefix(resource, "task/") {
		return "", fmt.Errorf("task identifier %q must have resource task/<id> or task/<cluster>/<id>", original)
	}
	parts := strings.Split(resource, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return "", fmt.Errorf("task identifier %q must have two or three path components", original)
	}
	if len(parts) == 3 && strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("task identifier %q has an empty cluster segment", original)
	}

	id, err := finalSegment(parts)
	if err != nil {
		return "", fmt.Errorf("invalid task identifier %q: %w", original, err)
	}
	return id, nil
}

func finalSegment(parts []string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("resource path has no components")
	}

	segment := strings.TrimSpace(parts[len(parts)-1])
	if segment == "" {
		return "", fmt.Errorf("resource path has an empty final segment")
	}
	return segment, nil
}
