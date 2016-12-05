package services

import (
	"log"

	"github.com/docker/docker/api/types/swarm"
)

type checkSwarmStatusTask struct {
}

func (c checkSwarmStatusTask) Run(i *Instance) {
	if info, err := GetDaemonInfo(i); err == nil {
		if info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive && info.Swarm.LocalNodeState != swarm.LocalNodeStateLocked {
			i.IsManager = &info.Swarm.ControlAvailable
		} else {
			i.IsManager = nil
		}
	} else {
		log.Println(err)
	}

}
