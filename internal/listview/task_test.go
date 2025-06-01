package listview

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

// Mock ECSClient for task tests (can be merged with cluster_test's MockECSClient if desired)
type MockTaskECSClient struct {
	ListClustersFunc  func(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) // Added for ecsiface
	ListTasksFunc     func(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasksFunc func(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

func (m *MockTaskECSClient) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	if m.ListClustersFunc != nil {
		return m.ListClustersFunc(ctx, params, optFns...)
	}
	return nil, errors.New("ListClusters not implemented in MockTaskECSClient")
}

func (m *MockTaskECSClient) ListTasks(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
	if m.ListTasksFunc != nil {
		return m.ListTasksFunc(ctx, params, optFns...)
	}
	return nil, errors.New("ListTasksFunc not implemented")
}

func (m *MockTaskECSClient) DescribeTasks(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
	if m.DescribeTasksFunc != nil {
		return m.DescribeTasksFunc(ctx, params, optFns...)
	}
	return nil, errors.New("DescribeTasksFunc not implemented")
}

func TestSelectTaskView(t *testing.T) {
	originalRenderList := RenderList
	defer func() { RenderList = originalRenderList }()

	task1Arn := "arn:aws:ecs:us-east-1:123:task/cluster-name/task1id"
	task1Group := "service:task-def-1"
	task2Arn := "arn:aws:ecs:us-east-1:123:task/cluster-name/task2id"
	task2Group := "service:task-def-2"
	task3Arn := "arn:aws:ecs:us-east-1:123:task/cluster-name/task3id" // No EnableExecuteCommand
	task3Group := "service:task-def-3"

	baseTasks := []types.Task{
		{TaskArn: aws.String(task1Arn), Group: aws.String(task1Group), EnableExecuteCommand: true},
		{TaskArn: aws.String(task2Arn), Group: aws.String(task2Group), EnableExecuteCommand: true},
		{TaskArn: aws.String(task3Arn), Group: aws.String(task3Group), EnableExecuteCommand: false},
	}

	tests := []struct {
		name                        string
		cluster                     string
		inputService                string
		mockListTasksOutput         *ecs.ListTasksOutput
		mockListTasksError          error
		mockDescribeTasks           func(arns []string) (*ecs.DescribeTasksOutput, error) // More flexible mock for DescribeTasks
		mockRenderOutput            string
		mockRenderQuit              bool
		mockRenderError             error
		expectedSelectedTask        *types.Task // Check the selected task directly
		expectedQuit                bool
		expectedErrorMsg            string
		expectedTaskArnsForDescribe []string
		expectedRenderListCallCount int
	}{
		{
			name:             "Successful selection - multiple tasks",
			cluster:          "test-cluster",
			inputService:     "test-service",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task1Arn, task2Arn, task3Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				assert.ElementsMatch(t, []string{task1Arn, task2Arn, task3Arn}, arns)
				return &ecs.DescribeTasksOutput{Tasks: baseTasks}, nil
			},
			mockRenderOutput: task1Group, // User selects task1
			expectedSelectedTask: &baseTasks[0],
			expectedTaskArnsForDescribe: []string{task1Arn, task2Arn, task3Arn},
			expectedRenderListCallCount: 1,
		},
		{
			name:             "Successful selection - only one executable task",
			cluster:          "test-cluster",
			inputService:     "", // No specific service, list all tasks in cluster
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task1Arn, task3Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				assert.ElementsMatch(t, []string{task1Arn, task3Arn}, arns)
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{baseTasks[0], baseTasks[2]}}, nil
			},
			expectedSelectedTask: &baseTasks[0], // Auto-selected
			expectedTaskArnsForDescribe: []string{task1Arn, task3Arn},
			expectedRenderListCallCount: 0, // RenderList should not be called
		},
		{
			name:             "RenderList returns quit",
			cluster:          "test-cluster",
			inputService:     "test-service",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task1Arn, task2Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{baseTasks[0], baseTasks[1]}}, nil
			},
			mockRenderQuit:   true,
			expectedQuit:     true,
			expectedTaskArnsForDescribe: []string{task1Arn, task2Arn},
			expectedRenderListCallCount: 1,
		},
		{
			name:             "RenderList returns error",
			cluster:          "test-cluster",
			inputService:     "test-service",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task1Arn, task2Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{baseTasks[0], baseTasks[1]}}, nil
			},
			mockRenderError:  errors.New("render error"),
			expectedErrorMsg: "render error",
			expectedTaskArnsForDescribe: []string{task1Arn, task2Arn},
			expectedRenderListCallCount: 1,
		},
		{
			name:               "ListTasks error",
			cluster:            "test-cluster",
			inputService:       "test-service",
			mockListTasksError: errors.New("list tasks error"),
			expectedErrorMsg:   "list tasks error",
			expectedRenderListCallCount: 0,
		},
		{
			name:             "DescribeTasks error",
			cluster:          "test-cluster",
			inputService:     "test-service",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task1Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				return nil, errors.New("describe tasks error")
			},
			expectedErrorMsg: "describe tasks error",
			expectedTaskArnsForDescribe: []string{task1Arn},
			expectedRenderListCallCount: 0,
		},
		{
			name:             "No tasks found by ListTasks",
			cluster:          "test-cluster",
			inputService:     "test-service",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				assert.Empty(t, arns) // DescribeTasks should be called with empty list
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{}}, nil
			},
			mockRenderQuit: true, // Simulate user quitting from empty list
			expectedQuit:   true,
			expectedTaskArnsForDescribe: []string{},
			expectedRenderListCallCount: 1,
		},
		{
			name:             "No executable tasks found",
			cluster:          "test-cluster",
			inputService:     "",
			mockListTasksOutput: &ecs.ListTasksOutput{TaskArns: []string{task3Arn}},
			mockDescribeTasks: func(arns []string) (*ecs.DescribeTasksOutput, error) {
				return &ecs.DescribeTasksOutput{Tasks: []types.Task{baseTasks[2]}}, nil
			},
			mockRenderQuit: true,
			expectedQuit:   true,
			expectedTaskArnsForDescribe: []string{task3Arn},
			expectedRenderListCallCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderListCallCount := 0
			RenderList = func(title string, l []string) (string, bool, error) {
				renderListCallCount++
				expectedNames := []string{}
				// Determine expected names based on which tasks are executable from baseTasks
				// that are also in tt.expectedTaskArnsForDescribe (or all if not specified for describe)

				tempDescribeArns := tt.expectedTaskArnsForDescribe
				if tempDescribeArns == nil && tt.mockListTasksOutput != nil { // If nil, assume all from ListTasks
					tempDescribeArns = tt.mockListTasksOutput.TaskArns
				}


				if tempDescribeArns != nil {
					describedTasksMap := make(map[string]types.Task)
					if tt.mockDescribeTasks != nil {
						// Simulate the describeTasks call to get the tasks that would be processed
						// This is a bit complex because mockDescribeTasks is itself a function.
						// For simplicity in this mock, we'll assume baseTasks contains all possible tasks.
						for _, taskArn := range tempDescribeArns {
							for _, bt := range baseTasks {
								if *bt.TaskArn == taskArn {
									describedTasksMap[taskArn] = bt
									break
								}
							}
						}
					}

					for _, arn := range tempDescribeArns {
						task, ok := describedTasksMap[arn]
						if ok && task.EnableExecuteCommand {
							expectedNames = append(expectedNames, *task.Group)
						}
					}
				}


				if tt.expectedRenderListCallCount > 0 { // Only assert if RenderList is expected to be called
					if len(expectedNames) > 0 {
						assert.ElementsMatch(t, expectedNames, l, "RenderList called with unexpected list of names")
					} else {
						assert.Empty(t, l, "RenderList should be called with an empty list")
					}
					assert.Equal(t, "Select a Task", title)
				}
				return tt.mockRenderOutput, tt.mockRenderQuit, tt.mockRenderError
			}

			mockClient := &MockTaskECSClient{
				ListTasksFunc: func(ctx context.Context, params *ecs.ListTasksInput, optFns ...func(*ecs.Options)) (*ecs.ListTasksOutput, error) {
					assert.Equal(t, tt.cluster, *params.Cluster)
					if tt.inputService != "" {
						assert.Equal(t, tt.inputService, *params.ServiceName)
					} else {
						assert.Nil(t, params.ServiceName)
					}
					return tt.mockListTasksOutput, tt.mockListTasksError
				},
				DescribeTasksFunc: func(ctx context.Context, params *ecs.DescribeTasksInput, optFns ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error) {
					assert.Equal(t, tt.cluster, *params.Cluster)
					if tt.expectedTaskArnsForDescribe != nil {
						assert.ElementsMatch(t, tt.expectedTaskArnsForDescribe, params.Tasks)
					} else {
						assert.Empty(t, params.Tasks)
					}
					return tt.mockDescribeTasks(params.Tasks)
				},
			}

			selectedTask, quit, err := SelectTaskView(mockClient, tt.cluster, tt.inputService)

			assert.Equal(t, tt.expectedRenderListCallCount, renderListCallCount, "RenderList call count mismatch")

			if tt.expectedErrorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				// If an error occurs, quit should generally be false, unless the error originates from RenderList itself
				// which might also set quit to true. The current test structure implies quit is false on error
				// from AWS calls.
				if tt.mockRenderError != nil && tt.expectedQuit {
					assert.True(t, quit)
				} else if tt.mockRenderError != nil && !tt.expectedQuit {
					assert.False(t, quit)
				} else if tt.mockRenderError == nil {
					assert.False(t, quit)
				}


			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedQuit, quit)
				if !quit && tt.expectedSelectedTask != nil {
					assert.NotNil(t, selectedTask)
					assert.Equal(t, *tt.expectedSelectedTask.TaskArn, *selectedTask.TaskArn)
					assert.Equal(t, *tt.expectedSelectedTask.Group, *selectedTask.Group)
					assert.Equal(t, tt.expectedSelectedTask.EnableExecuteCommand, selectedTask.EnableExecuteCommand)
				} else if !quit && tt.expectedSelectedTask == nil {
					// This case could mean no task was expected to be selected (e.g. user quit from an empty list)
					// or it's a test setup error. Given the test cases, this path might not be hit if quit is false.
					assert.Nil(t, selectedTask)
				}
			}
		})
	}
}
