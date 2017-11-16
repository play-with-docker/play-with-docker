package task

import (
	"context"
	"log"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/k8s"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type checkK8sClusterStatusTask struct {
	event   event.EventApi
	factory k8s.FactoryApi
}

var CheckK8sStatusEvent event.EventType

func init() {
	CheckK8sStatusEvent = event.EventType("instance k8s status")
}

func NewCheckK8sClusterStatus(e event.EventApi, f k8s.FactoryApi) *checkK8sClusterStatusTask {
	return &checkK8sClusterStatusTask{event: e, factory: f}
}

func (c *checkK8sClusterStatusTask) Name() string {
	return "CheckK8sClusterStatus"
}

func (c checkK8sClusterStatusTask) Run(ctx context.Context, i *types.Instance) error {
	status := ClusterStatus{Instance: i.Name}

	kc, err := c.factory.GetKubeletForInstance(i)
	if err != nil {
		log.Println(err)
		c.event.Emit(CheckSwarmStatusEvent, i.SessionId, status)
		return err
	}

	if isManager, err := kc.IsManager(); err != nil {
		c.event.Emit(CheckSwarmStatusEvent, i.SessionId, status)
		return err
	} else if !isManager {
		// Not a manager node, nothing to do for this task
		status.IsWorker = true
	} else {
		status.IsManager = true
	}

	c.event.Emit(CheckK8sStatusEvent, i.SessionId, status)

	return nil
}
