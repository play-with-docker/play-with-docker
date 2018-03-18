package task

import (
	"context"
	"log"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkSwarmPorts struct {
	event   event.EventApi
	factory docker.FactoryApi
}

var CheckSwarmPortsEvent event.EventType

func init() {
	CheckSwarmPortsEvent = event.EventType("instance docker swarm ports")
}

func (t *checkSwarmPorts) Name() string {
	return "CheckSwarmPorts"
}

func (t *checkSwarmPorts) Run(ctx context.Context, instance *types.Instance) error {
	dockerClient, err := t.factory.GetForInstance(instance)
	if err != nil {
		log.Println(err)
		return err
	}

	status, err := getDockerSwarmStatus(ctx, dockerClient)
	if err != nil {
		log.Println(err)
		return err
	}

	if !status.IsManager {
		return nil
	}

	hosts, ps, err := dockerClient.GetSwarmPorts()
	if err != nil {
		log.Println(err)
		return err
	}
	ports := make([]int, len(ps))
	for i, port := range ps {
		ports[i] = int(port)
	}

	t.event.Emit(CheckSwarmPortsEvent, instance.SessionId, ClusterPorts{Manager: instance.Name, Instances: hosts, Ports: ports})
	return nil
}

func NewCheckSwarmPorts(e event.EventApi, f docker.FactoryApi) *checkSwarmPorts {
	return &checkSwarmPorts{event: e, factory: f}
}
