package pwd

import (
	"log"

	"github.com/docker/docker/api/types/swarm"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkSwarmStatusTask struct {
}

func (c checkSwarmStatusTask) Run(i *types.Instance) error {
	if i.Docker == nil {
		return nil
	}
	if info, err := i.Docker.GetDaemonInfo(); err == nil {
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
