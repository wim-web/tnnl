package input

import (
	"strings"
	"testing"
)

func TestValidatePortForward(t *testing.T) {
	tests := []struct {
		name    string
		input   PortForwardInput
		wantErr string
	}{
		{
			name:    "missing target",
			input:   PortForwardInput{},
			wantErr: "target port is required",
		},
		{
			name:    "non-decimal target",
			input:   PortForwardInput{TargetPortNumber: "abc"},
			wantErr: `target port must be a decimal integer: "abc"`,
		},
		{
			name:    "target above range",
			input:   PortForwardInput{TargetPortNumber: "65536"},
			wantErr: "target port must be between 1 and 65535",
		},
		{
			name:    "target below range",
			input:   PortForwardInput{TargetPortNumber: "0"},
			wantErr: "target port must be between 1 and 65535",
		},
		{
			name:    "negative-sign target",
			input:   PortForwardInput{TargetPortNumber: "-1"},
			wantErr: "target port must be a decimal integer",
		},
		{
			name:    "positive-sign target",
			input:   PortForwardInput{TargetPortNumber: "+80"},
			wantErr: "target port must be a decimal integer",
		},
		{
			name:    "malformed signed target",
			input:   PortForwardInput{TargetPortNumber: "--1"},
			wantErr: "target port must be a decimal integer",
		},
		{
			name:    "space-padded target",
			input:   PortForwardInput{TargetPortNumber: " 80 "},
			wantErr: "target port must be a decimal integer",
		},
		{
			name:    "fractional target",
			input:   PortForwardInput{TargetPortNumber: "80.5"},
			wantErr: "target port must be a decimal integer",
		},
		{
			name:    "digit-only overflow",
			input:   PortForwardInput{TargetPortNumber: strings.Repeat("9", 100)},
			wantErr: "target port must be between 1 and 65535",
		},
		{
			name:    "local below range",
			input:   PortForwardInput{TargetPortNumber: "80", LocalPortNumber: "0"},
			wantErr: "local port must be between 1 and 65535",
		},
		{
			name:    "non-decimal local",
			input:   PortForwardInput{TargetPortNumber: "80", LocalPortNumber: "8e3"},
			wantErr: "local port must be a decimal integer",
		},
		{
			name:  "lower bound with default local",
			input: PortForwardInput{TargetPortNumber: "1"},
		},
		{
			name:  "upper bounds",
			input: PortForwardInput{TargetPortNumber: "65535", LocalPortNumber: "65535"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortForward(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidatePortForward() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidatePortForward() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePortForwardJoinsTargetAndLocalErrors(t *testing.T) {
	err := ValidatePortForward(PortForwardInput{
		TargetPortNumber: "abc",
		LocalPortNumber:  "0",
	})
	if err == nil {
		t.Fatal("ValidatePortForward() error = nil, want joined errors")
	}
	for _, want := range []string{"target port must be a decimal integer", "local port must be between 1 and 65535"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidatePortForward() error = %q, want substring %q", err, want)
		}
	}
}

func TestValidateExecRejectsNegativeWaitAndBlankCommand(t *testing.T) {
	err := ValidateExec(ExecInput{Cmd: "  ", Wait: -1})
	if err == nil {
		t.Fatal("ValidateExec() error = nil, want joined errors")
	}
	for _, want := range []string{"command is required", "wait must be non-negative"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateExec() error = %q, want substring %q", err, want)
		}
	}
}

func TestValidateExecAcceptsCommandAndNonNegativeWait(t *testing.T) {
	if err := ValidateExec(ExecInput{Cmd: " sh ", Wait: 0}); err != nil {
		t.Fatalf("ValidateExec() error = %v, want nil", err)
	}
}

func TestValidateRemotePortForward(t *testing.T) {
	tests := []struct {
		name      string
		input     RemotePortForwardInput
		wantParts []string
	}{
		{
			name:  "valid lower bounds with default local",
			input: RemotePortForwardInput{RemotePortNumber: "1", Host: "localhost"},
		},
		{
			name:  "valid upper bounds",
			input: RemotePortForwardInput{RemotePortNumber: "65535", LocalPortNumber: "65535", Host: "example.com"},
		},
		{
			name:  "blank host",
			input: RemotePortForwardInput{RemotePortNumber: "22", Host: "  "},
			wantParts: []string{
				"host is required",
			},
		},
		{
			name: "all invalid",
			input: RemotePortForwardInput{
				RemotePortNumber: "abc",
				LocalPortNumber:  "0",
			},
			wantParts: []string{
				"remote port must be a decimal integer",
				"local port must be between 1 and 65535",
				"host is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRemotePortForward(tt.input)
			if len(tt.wantParts) == 0 {
				if err != nil {
					t.Fatalf("ValidateRemotePortForward() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("ValidateRemotePortForward() error = nil, want validation error")
			}
			for _, want := range tt.wantParts {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("ValidateRemotePortForward() error = %q, want substring %q", err, want)
				}
			}
		})
	}
}
