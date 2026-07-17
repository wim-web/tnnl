package input

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func validatePort(name, value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s must be a decimal integer: %q", name, value)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535: %d", name, n)
	}
	return nil
}

func ValidateExec(v ExecInput) error {
	var errs []error
	if strings.TrimSpace(v.Cmd) == "" {
		errs = append(errs, errors.New("command is required"))
	}
	if v.Wait < 0 {
		errs = append(errs, errors.New("wait must be non-negative"))
	}
	return errors.Join(errs...)
}

func ValidatePortForward(v PortForwardInput) error {
	return errors.Join(
		validatePort("target port", v.TargetPortNumber, true),
		validatePort("local port", v.LocalPortNumber, false),
	)
}

func ValidateRemotePortForward(v RemotePortForwardInput) error {
	var hostErr error
	if strings.TrimSpace(v.Host) == "" {
		hostErr = errors.New("host is required")
	}
	return errors.Join(
		validatePort("remote port", v.RemotePortNumber, true),
		validatePort("local port", v.LocalPortNumber, false),
		hostErr,
	)
}
