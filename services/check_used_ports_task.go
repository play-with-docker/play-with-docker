package services

type checkUsedPortsTask struct {
}

func (c checkUsedPortsTask) Run(i *Instance) {
	if ports, err := GetUsedPorts(i); err == nil {
		for _, p := range ports {
			i.setUsedPort(p)
		}
	}
}
