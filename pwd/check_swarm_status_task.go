package pwd

import (
	"log"

	"github.com/docker/docker/api/types/swarm"
)

type checkSwarmStatusTask struct {
}

func (c checkSwarmStatusTask) Run(i *Instance) error {
	if i.docker == nil {
		return nil
	}
	if info, err := i.docker.GetDaemonInfo(); err == nil {
		if info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive && info.Swarm.LocalNodeState != swarm.LocalNodeStateLocked {
			i.IsManager = &info.Swarm.ControlAvailable
		} else {
			i.IsManager = nil
		}
	} else {
		log.Println(err)
		return err
	}
	return nil
}
