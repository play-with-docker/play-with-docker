package pwd

import "log"

type checkUsedPortsTask struct {
}

func (c checkUsedPortsTask) Run(i *Instance) error {
	if ports, err := i.docker.GetPorts(); err == nil {
		for _, p := range ports {
			i.setUsedPort(uint16(p))
		}
	} else {
		log.Println(err)
		return err
	}
	return nil
}
