package listview

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	// "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

// MockECSClient is a mock implementation of an ECS client.
type MockECSClient struct {
	ListClustersFunc  func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListTasksFunc     func(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasksFunc func(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

func (m *MockECSClient) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	if m.ListClustersFunc != nil {
		return m.ListClustersFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func (m *MockECSClient) ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	if m.ListTasksFunc != nil {
		return m.ListTasksFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func (m *MockECSClient) DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	if m.DescribeTasksFunc != nil {
		return m.DescribeTasksFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func TestSelectClusterView(t *testing.T) {
	// Store original RenderList and restore it after the test
	originalRenderList := RenderList
	defer func() { RenderList = originalRenderList }()

	tests := []struct {
		name             string
		clusterArns      []string
		mockRenderOutput string
		mockRenderQuit   bool
		mockRenderError  error
		expectedCluster  string
		expectedQuit     bool
		expectedErrorMsg string
	}{
		{
			name:        "Successful selection",
			clusterArns: []string{"arn:aws:ecs:us-east-1:123456789012:cluster/cluster1", "arn:aws:ecs:us-east-1:123456789012:cluster/cluster2"},
			mockRenderOutput: "cluster1",
			mockRenderQuit:   false,
			mockRenderError:  nil,
			expectedCluster:  "cluster1",
			expectedQuit:     false,
			expectedErrorMsg: "",
		},
		{
			name:             "RenderList returns quit",
			clusterArns:      []string{"arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"},
			mockRenderOutput: "",
			mockRenderQuit:   true,
			mockRenderError:  nil,
			expectedCluster:  "",
			expectedQuit:     true,
			expectedErrorMsg: "",
		},
		{
			name:             "RenderList returns error",
			clusterArns:      []string{"arn:aws:ecs:us-east-1:123456789012:cluster/cluster1"},
			mockRenderOutput: "",
			mockRenderQuit:   false,
			mockRenderError:  assert.AnError,
			expectedCluster:  "",
			expectedQuit:     false,
			expectedErrorMsg: assert.AnError.Error(),
		},
		{
			name:             "No clusters found",
			clusterArns:      []string{},
			mockRenderOutput: "",
			mockRenderQuit:   false,
			mockRenderError:  nil,
			expectedCluster:  "",
			expectedQuit:     false,
			expectedErrorMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockECSClient{
				ListClustersFunc: func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
					return &ecs.ListClustersOutput{
						ClusterArns: tt.clusterArns,
					}, nil
				},
			}

			RenderList = func(title string, l []string) (string, bool, error) {
				expectedNames := []string{}
				for _, arn := range tt.clusterArns {
					v := strings.Split(arn, "/")
					expectedNames = append(expectedNames, v[1])
				}
				assert.Equal(t, expectedNames, l, "RenderList called with unexpected list of names")
				assert.Equal(t, "Select a cluster", title, "RenderList called with unexpected title")
				return tt.mockRenderOutput, tt.mockRenderQuit, tt.mockRenderError
			}

			clusterName, quit, err := SelectClusterView(mockClient)

			assert.Equal(t, tt.expectedCluster, clusterName)
			assert.Equal(t, tt.expectedQuit, quit)

			if tt.expectedErrorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSelectClusterView_ListClustersError(t *testing.T) {
	mockClient := &MockECSClient{
		ListClustersFunc: func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
			return nil, assert.AnError
		},
	}

	originalRenderList := RenderList
	defer func() { RenderList = originalRenderList }()
	RenderList = func(title string, l []string) (string, bool, error) {
		t.Fatal("RenderList should not be called when ListClusters fails")
		return "", false, nil
	}


	_, quit, err := SelectClusterView(mockClient)

	assert.False(t, quit)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), assert.AnError.Error())
}
