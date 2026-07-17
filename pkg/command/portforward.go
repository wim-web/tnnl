package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/session_manager"
)

const remoteSessionCleanupTimeout = 5 * time.Second

var errInvalidSessionResponse = errors.New("invalid remote session response")

type SessionAPI interface {
	StartSession(context.Context, *ssm.StartSessionInput, ...func(*ssm.Options)) (*ssm.StartSessionOutput, error)
	TerminateSession(context.Context, *ssm.TerminateSessionInput, ...func(*ssm.Options)) (*ssm.TerminateSessionOutput, error)
}

type PortTarget struct {
	SSMTarget string
}

type DocumentName string

const (
	PORT_FORWARD_DOCUMENT_NAME        DocumentName = "AWS-StartPortForwardingSession"
	REMOTE_PORT_FORWARD_DOCUMENT_NAME DocumentName = "AWS-StartPortForwardingSessionToRemoteHost"
)

func StartPortForwardSession(
	ctx context.Context,
	ssmClient SessionAPI,
	portTarget PortTarget,
	region string,
	doc DocumentName,
	params map[string][]string,
) (RemoteSession, error) {
	if strings.TrimSpace(portTarget.SSMTarget) == "" {
		return RemoteSession{}, fmt.Errorf("SSM target is required for port-forward session")
	}
	if strings.TrimSpace(region) == "" {
		return RemoteSession{}, fmt.Errorf("AWS region is required for port-forward session")
	}
	if strings.TrimSpace(string(doc)) == "" {
		return RemoteSession{}, fmt.Errorf("SSM document name is required for port-forward session")
	}

	output, err := ssmClient.StartSession(ctx, &ssm.StartSessionInput{
		Target:       aws.String(portTarget.SSMTarget),
		DocumentName: aws.String(string(doc)),
		Parameters:   params,
	})
	if err != nil {
		return RemoteSession{}, fmt.Errorf("StartSession for target %q: %w", portTarget.SSMTarget, err)
	}
	if output == nil {
		return RemoteSession{}, invalidSessionResponse("StartSession", "output is nil")
	}

	sessionID, err := requiredSessionResponseValue("StartSession", "session ID", output.SessionId)
	if err != nil {
		return RemoteSession{}, err
	}
	terminate := terminateSessionFunc(ssmClient)
	cleanup := func(primary error) error {
		return cleanupCreatedSession(ctx, sessionID, remoteSessionCleanupTimeout, terminate, primary)
	}

	streamURL, err := requiredSessionResponseValue("StartSession", "stream URL", output.StreamUrl)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}
	tokenValue, err := requiredSessionResponseValue("StartSession", "token", output.TokenValue)
	if err != nil {
		return RemoteSession{}, cleanup(err)
	}

	return RemoteSession{
		ID: sessionID,
		Invocation: session_manager.Invocation{
			Response: session_manager.SessionResponse{
				SessionID:  sessionID,
				StreamURL:  streamURL,
				TokenValue: tokenValue,
			},
			Region: region,
			Target: portTarget.SSMTarget,
		},
		terminate:      terminate,
		cleanupTimeout: remoteSessionCleanupTimeout,
	}, nil
}

func invalidSessionResponse(operation, detail string) error {
	return fmt.Errorf("%s response: %w: %s", operation, errInvalidSessionResponse, detail)
}

func requiredSessionResponseValue(operation, field string, value *string) (string, error) {
	result := aws.ToString(value)
	if strings.TrimSpace(result) == "" {
		return "", invalidSessionResponse(operation, fmt.Sprintf("%s is missing or blank", field))
	}
	return result, nil
}

func terminateSessionFunc(ssmClient SessionAPI) terminateFunc {
	return func(ctx context.Context, sessionID string) error {
		_, err := ssmClient.TerminateSession(ctx, &ssm.TerminateSessionInput{
			SessionId: aws.String(sessionID),
		})
		return err
	}
}
