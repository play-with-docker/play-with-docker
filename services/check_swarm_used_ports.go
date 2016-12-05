package services

import "log"

type checkSwarmUsedPortsTask struct {
}

func (c checkSwarmUsedPortsTask) Run(i *Instance) {
	if i.IsManager != nil && *i.IsManager {
		// This is a swarm manager instance, then check for ports
		if err := SetInstanceSwarmPorts(i); err != nil {
			log.Println(err)
		}
	}
}
