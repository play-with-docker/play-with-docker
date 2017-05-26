package pwd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceResizeTerminal(t *testing.T) {
	resizedInstanceName := ""
	resizedRows := uint(0)
	resizedCols := uint(0)

	docker := &mockDocker{}
	docker.containerResize = func(name string, rows, cols uint) error {
		resizedInstanceName = name
		resizedRows = rows
		resizedCols = cols

		return nil
	}

	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(docker, tasks, broadcast, storage)

	err := p.InstanceResizeTerminal(&Instance{Name: "foobar"}, 24, 80)

	assert.Nil(t, err)
	assert.Equal(t, "foobar", resizedInstanceName)
	assert.Equal(t, uint(24), resizedRows)
	assert.Equal(t, uint(80), resizedCols)
}
