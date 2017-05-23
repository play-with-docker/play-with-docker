package pwd

import (
	"fmt"
	"log"
)

type checkSwarmUsedPortsTask struct {
}

func (c checkSwarmUsedPortsTask) Run(i *Instance) error {
	if i.IsManager != nil && *i.IsManager {
		sessionPrefix := i.session.Id[:8]
		// This is a swarm manager instance, then check for ports
		if hosts, ports, err := i.docker.GetSwarmPorts(); err != nil {
			log.Println(err)
			return err
		} else {
			for _, host := range hosts {
				host = fmt.Sprintf("%s_%s", sessionPrefix, host)
				for _, port := range ports {
					if i.session.Instances[host] != nil {
						i.session.Instances[host].setUsedPort(port)
					}
				}
			}
		}
	}
	return nil
}
