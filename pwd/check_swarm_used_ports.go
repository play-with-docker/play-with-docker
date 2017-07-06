package pwd

import (
	"fmt"
	"log"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkSwarmUsedPortsTask struct {
}

func (c checkSwarmUsedPortsTask) Run(i *types.Instance) error {
	if i.Docker == nil {
		return nil
	}
	if i.IsManager != nil && *i.IsManager {
		sessionPrefix := i.Session.Id[:8]
		// This is a swarm manager instance, then check for ports
		if hosts, ports, err := i.Docker.GetSwarmPorts(); err != nil {
			log.Println(err)
			return err
		} else {
			for _, host := range hosts {
				host = fmt.Sprintf("%s_%s", sessionPrefix, host)
				for _, port := range ports {
					if i.Session.Instances[host] != nil {
						i.Session.Instances[host].SetUsedPort(port)
					}
				}
			}
		}
	}
	return nil
}
