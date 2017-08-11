package task

import (
	"context"
	"testing"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/assert"
)

func TestCheckPorts_Name(t *testing.T) {
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	task := NewCheckPorts(e, f)

	assert.Equal(t, "CheckPorts", task.Name())
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCheckPorts_Run(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}

	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "aaaabbbb_node1",
		SessionId: "aaaabbbbcccc",
	}

	d.On("GetPorts").Return([]uint16{8080, 9090}, nil)
	f.On("GetForInstance", i).Return(d, nil)
	e.M.On("Emit", CheckPortsEvent, "aaaabbbbcccc", []interface{}{DockerPorts{Instance: "aaaabbbb_node1", Ports: []int{8080, 9090}}}).Return()

	task := NewCheckPorts(e, f)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}
