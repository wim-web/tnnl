package input

type EcsParameter struct {
	Cluster string `json:"cluster"`
	Service string `json:"service"`
}

type ExecInput struct {
	EcsParameter
	Cmd  string `json:"command"`
	Wait int    `json:"wait"`
}

type ExecOverrides struct {
	Command *string
	Wait    *int
}

type PortForwardInput struct {
	EcsParameter
	TargetPortNumber string `json:"target_port_number"`
	LocalPortNumber  string `json:"local_port_number"`
}

type PortForwardOverrides struct {
	TargetPort *string
	LocalPort  *string
}

type RemotePortForwardInput struct {
	EcsParameter
	RemotePortNumber string `json:"remote_port_number"`
	LocalPortNumber  string `json:"local_port_number"`
	Host             string `json:"host"`
}

type RemotePortForwardOverrides struct {
	RemotePort *string
	LocalPort  *string
	Host       *string
}
