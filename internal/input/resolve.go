package input

import "strings"

func ResolveExec(path string, overrides ExecOverrides) (ExecInput, error) {
	resolved := ExecInput{Cmd: "sh", Wait: 0}
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil {
			return ExecInput{}, err
		}
	}
	if overrides.Command != nil {
		resolved.Cmd = *overrides.Command
	}
	if overrides.Wait != nil {
		resolved.Wait = *overrides.Wait
	}
	normalizeExec(&resolved)
	if err := ValidateExec(resolved); err != nil {
		return ExecInput{}, err
	}
	return resolved, nil
}

func ResolvePortForward(path string, overrides PortForwardOverrides) (PortForwardInput, error) {
	var resolved PortForwardInput
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil {
			return PortForwardInput{}, err
		}
	}
	if overrides.TargetPort != nil {
		resolved.TargetPortNumber = *overrides.TargetPort
	}
	if overrides.LocalPort != nil {
		resolved.LocalPortNumber = *overrides.LocalPort
	}
	normalizeECS(&resolved.EcsParameter)
	resolved.TargetPortNumber = strings.TrimSpace(resolved.TargetPortNumber)
	resolved.LocalPortNumber = strings.TrimSpace(resolved.LocalPortNumber)
	if err := ValidatePortForward(resolved); err != nil {
		return PortForwardInput{}, err
	}
	return resolved, nil
}

func ResolveRemotePortForward(path string, overrides RemotePortForwardOverrides) (RemotePortForwardInput, error) {
	var resolved RemotePortForwardInput
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil {
			return RemotePortForwardInput{}, err
		}
	}
	if overrides.RemotePort != nil {
		resolved.RemotePortNumber = *overrides.RemotePort
	}
	if overrides.LocalPort != nil {
		resolved.LocalPortNumber = *overrides.LocalPort
	}
	if overrides.Host != nil {
		resolved.Host = *overrides.Host
	}
	normalizeECS(&resolved.EcsParameter)
	resolved.RemotePortNumber = strings.TrimSpace(resolved.RemotePortNumber)
	resolved.LocalPortNumber = strings.TrimSpace(resolved.LocalPortNumber)
	resolved.Host = strings.TrimSpace(resolved.Host)
	if err := ValidateRemotePortForward(resolved); err != nil {
		return RemotePortForwardInput{}, err
	}
	return resolved, nil
}

func normalizeECS(value *EcsParameter) {
	value.Cluster = strings.TrimSpace(value.Cluster)
	value.Service = strings.TrimSpace(value.Service)
}

func normalizeExec(value *ExecInput) {
	normalizeECS(&value.EcsParameter)
	value.Cmd = strings.TrimSpace(value.Cmd)
}
