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

type PortForwardInput struct {
	EcsParameter
	TargetPortNumber string `json:"target_port_number"`
	LocalPortNumber  string `json:"local_port_number"`
}

type RemotePortForwardInput struct {
	EcsParameter
	RemotePortNumber string `json:"remote_port_number"`
	LocalPortNumber  string `json:"local_port_number"`
	Host             string `json:"host"`
}
