package services

type checkUsedPortsTask struct {
}

func (c checkUsedPortsTask) Run(i *Instance) {
	if ports, err := GetUsedPorts(i); err == nil {
		i.Ports = ports
	}
}
