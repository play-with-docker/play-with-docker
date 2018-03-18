package task

import (
	"context"
	"log"

	"docker.io/go-docker/api/types/swarm"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkSwarmStatus struct {
	event   event.EventApi
	factory docker.FactoryApi
}

var CheckSwarmStatusEvent event.EventType

func init() {
	CheckSwarmStatusEvent = event.EventType("instance docker swarm status")
}

func (t *checkSwarmStatus) Name() string {
	return "CheckSwarmStatus"
}

func (t *checkSwarmStatus) Run(ctx context.Context, instance *types.Instance) error {
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
	status.Instance = instance.Name

	t.event.Emit(CheckSwarmStatusEvent, instance.SessionId, status)
	return nil
}

func NewCheckSwarmStatus(e event.EventApi, f docker.FactoryApi) *checkSwarmStatus {
	return &checkSwarmStatus{event: e, factory: f}
}

func getDockerSwarmStatus(ctx context.Context, client docker.DockerApi) (ClusterStatus, error) {
	status := ClusterStatus{}
	info, err := client.DaemonInfo()
	if err != nil {
		return status, err
	}

	if info.Swarm.LocalNodeState != swarm.LocalNodeStateInactive && info.Swarm.LocalNodeState != swarm.LocalNodeStateLocked {
		status.IsManager = info.Swarm.ControlAvailable
		status.IsWorker = !info.Swarm.ControlAvailable
	}

	return status, nil
}
