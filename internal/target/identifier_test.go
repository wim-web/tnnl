package target

import (
	"strings"
	"testing"
)

func TestClusterName(t *testing.T) {
	t.Run("accepts short names and ECS ARNs", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "short name",
				input: "cluster-name",
				want:  "cluster-name",
			},
			{
				name:  "trimmed short name",
				input: "  cluster-name  ",
				want:  "cluster-name",
			},
			{
				name:  "ECS ARN",
				input: "arn:aws:ecs:us-east-1:123456789012:cluster/cluster-name",
				want:  "cluster-name",
			},
			{
				name:  "trimmed ECS ARN",
				input: "  arn:aws:ecs:us-east-1:123456789012:cluster/cluster-name  ",
				want:  "cluster-name",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := ClusterName(tt.input)
				if err != nil {
					t.Fatalf("ClusterName(%q) error = %v, want nil", tt.input, err)
				}
				if got != tt.want {
					t.Fatalf("ClusterName(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("rejects malformed identifiers", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{name: "empty", input: ""},
			{name: "whitespace", input: "   "},
			{name: "slash in short name", input: "cluster/name"},
			{name: "malformed ARN", input: "arn:aws:ecs"},
			{name: "non-ECS ARN", input: "arn:aws:ssm:us-east-1:123456789012:cluster/cluster-name"},
			{name: "wrong ARN resource type", input: "arn:aws:ecs:us-east-1:123456789012:task/cluster-name"},
			{name: "missing ARN path component", input: "arn:aws:ecs:us-east-1:123456789012:cluster"},
			{name: "extra ARN path component", input: "arn:aws:ecs:us-east-1:123456789012:cluster/cluster-name/extra"},
			{name: "empty ARN final segment", input: "arn:aws:ecs:us-east-1:123456789012:cluster/"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ClusterName(tt.input)
				requireIdentifierError(t, tt.input, err)
			})
		}
	})
}

func TestTaskID(t *testing.T) {
	t.Run("accepts short and long ECS task identifiers", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "short resource",
				input: "task/abc",
				want:  "abc",
			},
			{
				name:  "long resource",
				input: "task/cluster/abc",
				want:  "abc",
			},
			{
				name:  "short ARN",
				input: "arn:aws:ecs:us-east-1:123456789012:task/abc",
				want:  "abc",
			},
			{
				name:  "long ARN",
				input: "arn:aws:ecs:us-east-1:123456789012:task/cluster/abc",
				want:  "abc",
			},
			{
				name:  "trimmed long ARN",
				input: "  arn:aws:ecs:us-east-1:123456789012:task/cluster/abc  ",
				want:  "abc",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := TaskID(tt.input)
				if err != nil {
					t.Fatalf("TaskID(%q) error = %v, want nil", tt.input, err)
				}
				if got != tt.want {
					t.Fatalf("TaskID(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("rejects malformed identifiers", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{name: "empty", input: ""},
			{name: "whitespace", input: "   "},
			{name: "missing resource prefix", input: "abc"},
			{name: "missing resource path", input: "task"},
			{name: "wrong resource type", input: "service/abc"},
			{name: "empty short final segment", input: "task/"},
			{name: "empty long final segment", input: "task/cluster/"},
			{name: "missing long middle segment", input: "task//abc"},
			{name: "extra resource path component", input: "task/cluster/abc/extra"},
			{name: "malformed ARN", input: "arn:aws:ecs"},
			{name: "non-ECS ARN", input: "arn:aws:ssm:us-east-1:123456789012:task/abc"},
			{name: "wrong ARN resource type", input: "arn:aws:ecs:us-east-1:123456789012:service/cluster/abc"},
			{name: "missing ARN path component", input: "arn:aws:ecs:us-east-1:123456789012:task"},
			{name: "extra ARN path component", input: "arn:aws:ecs:us-east-1:123456789012:task/cluster/abc/extra"},
			{name: "empty ARN final segment", input: "arn:aws:ecs:us-east-1:123456789012:task/cluster/"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := TaskID(tt.input)
				requireIdentifierError(t, tt.input, err)
			})
		}
	})
}

func requireIdentifierError(t *testing.T, input string, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("identifier %q error = nil, want validation error", input)
	}
	if input == "" {
		if !strings.Contains(strings.ToLower(err.Error()), "empty") {
			t.Fatalf("identifier %q error = %q, want explicit empty-input error", input, err)
		}
		return
	}
	if !strings.Contains(err.Error(), input) {
		t.Fatalf("identifier %q error = %q, want original input included", input, err)
	}
}
