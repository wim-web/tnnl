package session_manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const CommandName = "session-manager-plugin"

type SessionResponse struct {
	SessionID  string `json:"SessionId"`
	StreamURL  string `json:"StreamUrl"`
	TokenValue string `json:"TokenValue"`
}

type Invocation struct {
	Response SessionResponse
	Region   string
	Target   string
}

func (i Invocation) arguments(profile, endpoint string) ([]string, error) {
	if strings.TrimSpace(i.Response.SessionID) == "" ||
		strings.TrimSpace(i.Response.StreamURL) == "" ||
		strings.TrimSpace(i.Response.TokenValue) == "" {
		return nil, errors.New("session response is missing id, stream URL, or token")
	}
	if strings.TrimSpace(i.Region) == "" {
		return nil, errors.New("AWS region is required for session-manager-plugin")
	}
	if strings.TrimSpace(i.Target) == "" {
		return nil, errors.New("session target is required for session-manager-plugin")
	}

	response, err := json.Marshal(i.Response)
	if err != nil {
		return nil, fmt.Errorf("encode plugin session response: %w", err)
	}
	request, err := json.Marshal(struct {
		Target string `json:"Target"`
	}{Target: i.Target})
	if err != nil {
		return nil, fmt.Errorf("encode plugin request: %w", err)
	}

	return []string{
		string(response),
		i.Region,
		"StartSession",
		profile,
		string(request),
		endpoint,
	}, nil
}

type Plugin interface {
	Run(context.Context, Invocation) error
}

type Runner struct {
	path     string
	profile  string
	endpoint string
}

type command interface {
	CombinedOutput() ([]byte, error)
}

type processFactory func(context.Context, string, ...string) command

type dependencies struct {
	lookPath       func(string) (string, error)
	commandContext processFactory
	preflightLimit time.Duration
}

func Preflight(ctx context.Context) (Plugin, error) {
	return preflight(ctx, dependencies{
		lookPath: exec.LookPath,
		commandContext: func(ctx context.Context, name string, args ...string) command {
			return exec.CommandContext(ctx, name, args...)
		},
		preflightLimit: 3 * time.Second,
	})
}

func preflight(ctx context.Context, deps dependencies) (*Runner, error) {
	path, err := deps.lookPath(CommandName)
	if err != nil {
		return nil, fmt.Errorf(
			"%s is required; install it and verify `%s --version`: %w",
			CommandName,
			CommandName,
			err,
		)
	}

	checkCtx, cancel := context.WithTimeout(ctx, deps.preflightLimit)
	defer cancel()

	output, err := deps.commandContext(checkCtx, path, "--version").CombinedOutput()
	if err != nil {
		processErr := fmt.Errorf(
			"%s --version failed: %s: %w",
			CommandName,
			strings.TrimSpace(string(output)),
			err,
		)
		if contextErr := checkCtx.Err(); contextErr != nil {
			return nil, errors.Join(processErr, contextErr)
		}
		return nil, processErr
	}
	if strings.TrimSpace(string(output)) == "" {
		return nil, fmt.Errorf("%s --version returned empty output", CommandName)
	}

	return &Runner{
		path:     path,
		profile:  firstEnvironment("AWS_PROFILE", "AWS_DEFAULT_PROFILE"),
		endpoint: firstEnvironment("AWS_ENDPOINT_URL_SSM", "AWS_ENDPOINT_URL"),
	}, nil
}

func firstEnvironment(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func (r *Runner) Run(ctx context.Context, invocation Invocation) error {
	arguments, err := invocation.arguments(r.profile, r.endpoint)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, r.path, arguments...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		runErr := fmt.Errorf("run %s: %w", CommandName, err)
		if contextErr := ctx.Err(); contextErr != nil {
			return errors.Join(runErr, contextErr)
		}
		return runErr
	}
	return nil
}
