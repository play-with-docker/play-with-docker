package pwd

import (
	"log"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkUsedPortsTask struct {
}

func (c checkUsedPortsTask) Run(i *types.Instance) error {
	if i.Docker == nil {
		return nil
	}
	if ports, err := i.Docker.GetPorts(); err == nil {
		for _, p := range ports {
			i.SetUsedPort(uint16(p))
		}
	} else {
		log.Println(err)
		return err
	}
	return nil
}
