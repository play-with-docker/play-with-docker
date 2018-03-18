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

func TestCheckSwarmPorts_Name(t *testing.T) {
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	task := NewCheckSwarmPorts(e, f)

	assert.Equal(t, "CheckSwarmPorts", task.Name())
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckSwarmPorts_RunWhenManager(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "aaaabbbb_node1",
		SessionId: "aaaabbbbcccc",
		Hostname:  "node1",
	}
	info := dockerTypes.Info{
		Swarm: swarm.Info{
			LocalNodeState:   swarm.LocalNodeStateActive,
			ControlAvailable: true,
		},
	}

	f.On("GetForInstance", i).Return(d, nil)
	d.On("DaemonInfo").Return(info, nil)
	d.On("GetSwarmPorts").Return([]string{"aaaabbbb_node1", "aaaabbbb_node2"}, []uint16{8080, 9090}, nil)
	e.M.On("Emit", CheckSwarmPortsEvent, "aaaabbbbcccc", []interface{}{ClusterPorts{Manager: i.Name, Instances: []string{i.Name, "aaaabbbb_node2"}, Ports: []int{8080, 9090}}}).Return()

	task := NewCheckSwarmPorts(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}
