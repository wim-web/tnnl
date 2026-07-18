package command

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/session_manager"
)

const (
	portTarget    = "ecs:cluster_task_runtime"
	portRegion    = "ap-northeast-1"
	portSessionID = "port-session-123"
	portStreamURL = "wss://port-session.example"
	portToken     = "port-token"
)

type portContextKey struct{}

type fakeSessionAPI struct {
	startOutput  *ssm.StartSessionOutput
	startErr     error
	terminateErr error

	startCalls     int
	startCtx       context.Context
	startInput     *ssm.StartSessionInput
	terminateCalls int
	terminateCtx   context.Context
	terminateInput *ssm.TerminateSessionInput
}

func (f *fakeSessionAPI) StartSession(
	ctx context.Context,
	input *ssm.StartSessionInput,
	_ ...func(*ssm.Options),
) (*ssm.StartSessionOutput, error) {
	f.startCalls++
	f.startCtx = ctx
	f.startInput = input
	return f.startOutput, f.startErr
}

func (f *fakeSessionAPI) TerminateSession(
	ctx context.Context,
	input *ssm.TerminateSessionInput,
	_ ...func(*ssm.Options),
) (*ssm.TerminateSessionOutput, error) {
	f.terminateCalls++
	f.terminateCtx = ctx
	f.terminateInput = input
	return &ssm.TerminateSessionOutput{SessionId: input.SessionId}, f.terminateErr
}

func TestStartPortForwardSessionUsesCallerContextAndExactInput(t *testing.T) {
	ctx := context.WithValue(context.Background(), portContextKey{}, "caller-context")
	params := map[string][]string{
		"portNumber":      {"5432"},
		"localPortNumber": {"15432"},
	}
	client := &fakeSessionAPI{startOutput: validStartSessionOutput()}

	session, err := StartPortForwardSession(
		ctx,
		client,
		PortTarget{SSMTarget: portTarget},
		portRegion,
		PORT_FORWARD_DOCUMENT_NAME,
		params,
	)
	if err != nil {
		t.Fatalf("StartPortForwardSession() error = %v, want nil", err)
	}

	if client.startCalls != 1 {
		t.Fatalf("StartSession calls = %d, want 1", client.startCalls)
	}
	if client.startCtx != ctx {
		t.Fatalf("StartSession context = %v, want original caller context %v", client.startCtx, ctx)
	}
	if got := client.startInput; got == nil ||
		aws.ToString(got.Target) != portTarget ||
		aws.ToString(got.DocumentName) != string(PORT_FORWARD_DOCUMENT_NAME) ||
		!reflect.DeepEqual(got.Parameters, params) {
		t.Fatalf("StartSession input = %#v, want exact target/document/parameters", got)
	}

	wantInvocation := session_manager.Invocation{
		Response: session_manager.SessionResponse{
			SessionID:  portSessionID,
			StreamURL:  portStreamURL,
			TokenValue: portToken,
		},
		Region: portRegion,
		Target: portTarget,
	}
	if session.ID != portSessionID {
		t.Fatalf("RemoteSession.ID = %q, want %q", session.ID, portSessionID)
	}
	if !reflect.DeepEqual(session.Invocation, wantInvocation) {
		t.Fatalf("RemoteSession.Invocation = %#v, want %#v", session.Invocation, wantInvocation)
	}
	if session.cleanupTimeout != 5*time.Second {
		t.Fatalf("RemoteSession cleanup timeout = %v, want 5s", session.cleanupTimeout)
	}
	if client.terminateCalls != 0 {
		t.Fatalf("TerminateSession calls during successful construction = %d, want 0", client.terminateCalls)
	}
}

func TestStartPortForwardSessionTerminatesExactSessionWhenPluginFails(t *testing.T) {
	ctx := context.WithValue(context.Background(), portContextKey{}, "preserved-for-cleanup")
	client := &fakeSessionAPI{startOutput: validStartSessionOutput()}
	session, err := StartPortForwardSession(
		ctx,
		client,
		PortTarget{SSMTarget: portTarget},
		portRegion,
		REMOTE_PORT_FORWARD_DOCUMENT_NAME,
		map[string][]string{"host": {"db.example"}},
	)
	if err != nil {
		t.Fatalf("StartPortForwardSession() error = %v, want nil", err)
	}
	pluginErr := errors.New("plugin failed")

	err = session.Run(ctx, pluginFunc(func(pluginCtx context.Context, invocation session_manager.Invocation) error {
		if pluginCtx != ctx {
			t.Fatalf("plugin context = %v, want original caller context %v", pluginCtx, ctx)
		}
		if !reflect.DeepEqual(invocation, session.Invocation) {
			t.Fatalf("plugin invocation = %#v, want %#v", invocation, session.Invocation)
		}
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("RemoteSession.Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if client.terminateCalls != 1 {
		t.Fatalf("TerminateSession calls = %d, want 1", client.terminateCalls)
	}
	if got := aws.ToString(client.terminateInput.SessionId); got != portSessionID {
		t.Fatalf("TerminateSession session ID = %q, want %q", got, portSessionID)
	}
	if got := client.terminateCtx.Value(portContextKey{}); got != "preserved-for-cleanup" {
		t.Fatalf("TerminateSession context marker = %#v, want preserved caller value", got)
	}
}

func TestStartPortForwardSessionRejectsInvalidInputBeforeStartSession(t *testing.T) {
	tests := []struct {
		name   string
		target PortTarget
		region string
		doc    DocumentName
		want   string
	}{
		{name: "target", target: PortTarget{SSMTarget: " \t"}, region: portRegion, doc: PORT_FORWARD_DOCUMENT_NAME, want: "target"},
		{name: "region", target: PortTarget{SSMTarget: portTarget}, region: "\n", doc: PORT_FORWARD_DOCUMENT_NAME, want: "region"},
		{name: "document", target: PortTarget{SSMTarget: portTarget}, region: portRegion, doc: DocumentName("  "), want: "document"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeSessionAPI{}

			_, err := StartPortForwardSession(context.Background(), client, tt.target, tt.region, tt.doc, nil)

			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tt.want) {
				t.Fatalf("StartPortForwardSession() error = %v, want actionable %q context", err, tt.want)
			}
			if client.startCalls != 0 || client.terminateCalls != 0 {
				t.Fatalf("SSM calls = start %d, terminate %d; want 0, 0", client.startCalls, client.terminateCalls)
			}
		})
	}
}

func TestStartPortForwardSessionWrapsStartSessionError(t *testing.T) {
	apiErr := errors.New("start sentinel")
	ctx := context.WithValue(context.Background(), portContextKey{}, "start-context")
	client := &fakeSessionAPI{startErr: apiErr}

	_, err := StartPortForwardSession(
		ctx,
		client,
		PortTarget{SSMTarget: portTarget},
		portRegion,
		PORT_FORWARD_DOCUMENT_NAME,
		nil,
	)

	if !errors.Is(err, apiErr) {
		t.Fatalf("StartPortForwardSession() error = %v, want errors.Is(start sentinel)", err)
	}
	for _, want := range []string{"StartSession", portTarget} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("StartPortForwardSession() error = %q, want context %q", err, want)
		}
	}
	if client.startCtx != ctx {
		t.Fatalf("StartSession context = %v, want original caller context %v", client.startCtx, ctx)
	}
	if client.terminateCalls != 0 {
		t.Fatalf("TerminateSession calls = %d, want 0", client.terminateCalls)
	}
}

func TestStartPortForwardSessionRejectsNilResponseWithoutTermination(t *testing.T) {
	client := &fakeSessionAPI{}

	_, err := StartPortForwardSession(
		context.Background(),
		client,
		PortTarget{SSMTarget: portTarget},
		portRegion,
		PORT_FORWARD_DOCUMENT_NAME,
		nil,
	)

	if !errors.Is(err, errInvalidSessionResponse) {
		t.Fatalf("StartPortForwardSession() error = %v, want validation sentinel", err)
	}
	if client.terminateCalls != 0 {
		t.Fatalf("TerminateSession calls = %d, want 0 without a session ID", client.terminateCalls)
	}
}

func TestStartPortForwardSessionRejectsMissingSessionIDWithoutTermination(t *testing.T) {
	tests := []struct {
		name string
		id   *string
	}{
		{name: "missing", id: nil},
		{name: "blank", id: aws.String(" \t")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeSessionAPI{startOutput: &ssm.StartSessionOutput{
				SessionId:  tt.id,
				StreamUrl:  aws.String(portStreamURL),
				TokenValue: aws.String(portToken),
			}}

			_, err := StartPortForwardSession(
				context.Background(),
				client,
				PortTarget{SSMTarget: portTarget},
				portRegion,
				PORT_FORWARD_DOCUMENT_NAME,
				nil,
			)

			if !errors.Is(err, errInvalidSessionResponse) {
				t.Fatalf("StartPortForwardSession() error = %v, want validation sentinel", err)
			}
			if client.terminateCalls != 0 {
				t.Fatalf("TerminateSession calls = %d, want 0 without a nonblank session ID", client.terminateCalls)
			}
		})
	}
}

func TestStartPortForwardSessionTerminatesWhenStartedResponseIsInvalid(t *testing.T) {
	tests := []struct {
		name   string
		output *ssm.StartSessionOutput
	}{
		{
			name: "missing stream URL",
			output: &ssm.StartSessionOutput{
				SessionId:  aws.String(portSessionID),
				TokenValue: aws.String(portToken),
			},
		},
		{
			name: "missing token",
			output: &ssm.StartSessionOutput{
				SessionId: aws.String(portSessionID),
				StreamUrl: aws.String(portStreamURL),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), portContextKey{}, tt.name)
			client := &fakeSessionAPI{startOutput: tt.output}

			_, err := StartPortForwardSession(
				ctx,
				client,
				PortTarget{SSMTarget: portTarget},
				portRegion,
				PORT_FORWARD_DOCUMENT_NAME,
				nil,
			)

			if !errors.Is(err, errInvalidSessionResponse) {
				t.Fatalf("StartPortForwardSession() error = %v, want validation sentinel", err)
			}
			if client.terminateCalls != 1 {
				t.Fatalf("TerminateSession calls = %d, want 1", client.terminateCalls)
			}
			if got := aws.ToString(client.terminateInput.SessionId); got != portSessionID {
				t.Fatalf("TerminateSession session ID = %q, want %q", got, portSessionID)
			}
			if got := client.terminateCtx.Value(portContextKey{}); got != tt.name {
				t.Fatalf("TerminateSession context marker = %#v, want %q", got, tt.name)
			}
		})
	}
}

func TestStartPortForwardSessionJoinsValidationAndTerminateErrors(t *testing.T) {
	cleanupErr := errors.New("terminate sentinel")
	client := &fakeSessionAPI{
		startOutput: &ssm.StartSessionOutput{
			SessionId:  aws.String(portSessionID),
			TokenValue: aws.String(portToken),
		},
		terminateErr: cleanupErr,
	}

	_, err := StartPortForwardSession(
		context.Background(),
		client,
		PortTarget{SSMTarget: portTarget},
		portRegion,
		PORT_FORWARD_DOCUMENT_NAME,
		nil,
	)

	if !errors.Is(err, errInvalidSessionResponse) {
		t.Fatalf("StartPortForwardSession() error = %v, want validation sentinel", err)
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("StartPortForwardSession() error = %v, want cleanup sentinel", err)
	}
	if client.terminateCalls != 1 || aws.ToString(client.terminateInput.SessionId) != portSessionID {
		t.Fatalf("TerminateSession calls/input = %d/%#v, want one call for %q", client.terminateCalls, client.terminateInput, portSessionID)
	}
}

func validStartSessionOutput() *ssm.StartSessionOutput {
	return &ssm.StartSessionOutput{
		SessionId:  aws.String(portSessionID),
		StreamUrl:  aws.String(portStreamURL),
		TokenValue: aws.String(portToken),
	}
}

func assertTerminateCall(t *testing.T, client *fakeSessionAPI, wantCtx context.Context, wantID string) {
	t.Helper()
	if client.terminateCalls != 1 {
		t.Fatalf("TerminateSession calls = %d, want 1", client.terminateCalls)
	}
	if client.terminateCtx != wantCtx {
		t.Fatalf("TerminateSession context = %v, want exact context %v", client.terminateCtx, wantCtx)
	}
	if got := aws.ToString(client.terminateInput.SessionId); got != wantID {
		t.Fatalf("TerminateSession session ID = %q, want %q", got, wantID)
	}
}
