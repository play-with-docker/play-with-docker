package task

import (
	"context"
	"log"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type DockerPorts struct {
	Instance string `json:"instance"`
	Ports    []int  `json:"ports"`
}

type checkPorts struct {
	event   event.EventApi
	factory docker.FactoryApi
}

var CheckPortsEvent event.EventType

func init() {
	CheckPortsEvent = event.EventType("instance docker ports")
}

func (t *checkPorts) Name() string {
	return "CheckPorts"
}

func (t *checkPorts) Run(ctx context.Context, instance *types.Instance) error {
	dockerClient, err := t.factory.GetForInstance(instance)
	if err != nil {
		log.Println(err)
		return err
	}

	ps, err := dockerClient.GetPorts()
	if err != nil {
		log.Println(err)
		return err
	}
	ports := make([]int, len(ps))
	for i, port := range ps {
		ports[i] = int(port)
	}

	t.event.Emit(CheckPortsEvent, instance.SessionId, DockerPorts{Instance: instance.Name, Ports: ports})
	return nil
}

func NewCheckPorts(e event.EventApi, f docker.FactoryApi) *checkPorts {
	return &checkPorts{event: e, factory: f}
}
