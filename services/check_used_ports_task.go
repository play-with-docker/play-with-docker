package services

import "log"

type checkUsedPortsTask struct {
}

func (c checkUsedPortsTask) Run(i *Instance) error {
	if ports, err := GetUsedPorts(i); err == nil {
		for _, p := range ports {
			i.setUsedPort(p)
		}
	} else {
		log.Println(err)
		return err
	}
	return nil
}
