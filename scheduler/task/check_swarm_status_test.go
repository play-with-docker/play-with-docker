package task

import (
	"context"
	"testing"

	dockerTypes "docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/swarm"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/assert"
)

func TestCheckSwarmStatus_Name(t *testing.T) {
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	task := NewCheckSwarmStatus(e, f)

	assert.Equal(t, "CheckSwarmStatus", task.Name())
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckSwarmStatus_RunWhenInactive(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "node1",
		SessionId: "aaabbbccc",
	}
	infoInactive := dockerTypes.Info{
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateInactive,
		},
	}

	f.On("GetForInstance", i).Return(d, nil)
	d.On("DaemonInfo").Return(infoInactive, nil)
	e.M.On("Emit", CheckSwarmStatusEvent, "aaabbbccc", []interface{}{ClusterStatus{IsManager: false, IsWorker: false, Instance: "node1"}}).Return()

	task := NewCheckSwarmStatus(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckSwarmStatus_RunWhenLocked(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "node1",
		SessionId: "aaabbbccc",
	}
	infoLocked := dockerTypes.Info{
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateLocked,
		},
	}

	f.On("GetForInstance", i).Return(d, nil)
	d.On("DaemonInfo").Return(infoLocked, nil)
	e.M.On("Emit", CheckSwarmStatusEvent, "aaabbbccc", []interface{}{ClusterStatus{IsManager: false, IsWorker: false, Instance: "node1"}}).Return()

	task := NewCheckSwarmStatus(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckSwarmStatus_RunWhenManager(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "node1",
		SessionId: "aaabbbccc",
	}
	infoLocked := dockerTypes.Info{
		Swarm: swarm.Info{
			LocalNodeState:   swarm.LocalNodeStateActive,
			ControlAvailable: true,
		},
	}

	f.On("GetForInstance", i).Return(d, nil)
	d.On("DaemonInfo").Return(infoLocked, nil)
	e.M.On("Emit", CheckSwarmStatusEvent, "aaabbbccc", []interface{}{ClusterStatus{IsManager: true, IsWorker: false, Instance: "node1"}}).Return()

	task := NewCheckSwarmStatus(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckSwarmStatus_RunWhenWorker(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "node1",
		SessionId: "aaabbbccc",
	}
	infoLocked := dockerTypes.Info{
		Swarm: swarm.Info{
			LocalNodeState:   swarm.LocalNodeStateActive,
			ControlAvailable: false,
		},
	}

	f.On("GetForInstance", i).Return(d, nil)
	d.On("DaemonInfo").Return(infoLocked, nil)
	e.M.On("Emit", CheckSwarmStatusEvent, "aaabbbccc", []interface{}{ClusterStatus{IsManager: false, IsWorker: true, Instance: "node1"}}).Return()

	task := NewCheckSwarmStatus(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}
