package session_manager

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestMakeStartSessionCmd(t *testing.T) {
	ctx := context.Background()
	responseJSON := `{"SessionId": "test-session-id", "TokenValue": "test-token", "StreamUrl": "wss://example.com/stream"}`
	region := "us-east-1"

	cmd := MakeStartSessionCmd(ctx, responseJSON, region)

	// 1. Check the command path
	// The actual path might be resolved by the system, so we check if SESSION_MANAGER_COMMAND is part of it.
	if !strings.Contains(cmd.Path, SESSION_MANAGER_COMMAND) && cmd.Path != SESSION_MANAGER_COMMAND {
		// If cmd.Path is just the command name, it means it will be looked up in PATH.
		// If it's a full path, it should contain the command name.
		t.Errorf("Expected command to be '%s' or contain it, got '%s'", SESSION_MANAGER_COMMAND, cmd.Path)
	}


	// 2. Check the arguments
	// Args[0] is the command itself
	expectedArgs := []string{
		SESSION_MANAGER_COMMAND, // This will be cmd.Args[0] if Path is just the command name
		responseJSON,
		region,
		"StartSession", // OperationName
	}

	// If cmd.Path is a fully resolved path, then cmd.Args[0] will be that path.
	// We should compare starting from cmd.Args[1] if cmd.Path is not SESSION_MANAGER_COMMAND or
	// ensure the first argument in expectedArgs matches cmd.Args[0] if we are sure about how Path is resolved.
	// For simplicity, let's assume cmd.Args[0] will contain SESSION_MANAGER_COMMAND if Path is resolved,
	// or be SESSION_MANAGER_COMMAND if not.

	// A more robust way to check arguments:
	// Check if cmd.Args[0] ends with SESSION_MANAGER_COMMAND
	if !strings.HasSuffix(cmd.Args[0], SESSION_MANAGER_COMMAND) {
		t.Errorf("Expected cmd.Args[0] to be or end with '%s', got '%s'", SESSION_MANAGER_COMMAND, cmd.Args[0])
	}

	// Compare the rest of the arguments
	if !reflect.DeepEqual(cmd.Args[1:], expectedArgs[1:]) {
		t.Errorf("Expected args (excluding command) to be %v, got %v", expectedArgs[1:], cmd.Args[1:])
	}


	// 3. Check Stdout, Stdin, Stderr (they should be nil by default if not set)
	//    Context is passed to exec.CommandContext, but not directly retrievable from exec.Cmd for comparison.
	if cmd.Stdout != nil {
		t.Errorf("Expected Stdout to be nil, got %v", cmd.Stdout)
	}
	if cmd.Stdin != nil {
		t.Errorf("Expected Stdin to be nil, got %v", cmd.Stdin)
	}
	if cmd.Stderr != nil {
		t.Errorf("Expected Stderr to be nil, got %v", cmd.Stderr)
	}
}

func TestMakeStartSessionCmd_EmptyResponse(t *testing.T) {
	ctx := context.Background()
	responseJSON := "" // Empty response
	region := "us-west-2"

	cmd := MakeStartSessionCmd(ctx, responseJSON, region)

	expectedArgs := []string{
		SESSION_MANAGER_COMMAND,
		responseJSON,
		region,
		"StartSession",
	}

	if !strings.HasSuffix(cmd.Args[0], SESSION_MANAGER_COMMAND) {
		t.Errorf("Expected cmd.Args[0] to be or end with '%s', got '%s'", SESSION_MANAGER_COMMAND, cmd.Args[0])
	}
	if !reflect.DeepEqual(cmd.Args[1:], expectedArgs[1:]) {
		t.Errorf("Expected args (excluding command) for empty response to be %v, got %v", expectedArgs[1:], cmd.Args[1:])
	}
}

func TestMakeStartSessionCmd_EmptyRegion(t *testing.T) {
	ctx := context.Background()
	responseJSON := `{"SessionId": "another-id"}`
	region := "" // Empty region

	cmd := MakeStartSessionCmd(ctx, responseJSON, region)

	expectedArgs := []string{
		SESSION_MANAGER_COMMAND,
		responseJSON,
		region,
		"StartSession",
	}

	if !strings.HasSuffix(cmd.Args[0], SESSION_MANAGER_COMMAND) {
		t.Errorf("Expected cmd.Args[0] to be or end with '%s', got '%s'", SESSION_MANAGER_COMMAND, cmd.Args[0])
	}
	if !reflect.DeepEqual(cmd.Args[1:], expectedArgs[1:]) {
		t.Errorf("Expected args (excluding command) for empty region to be %v, got %v", expectedArgs[1:], cmd.Args[1:])
	}
}
