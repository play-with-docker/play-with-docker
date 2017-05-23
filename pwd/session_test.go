package pwd

import (
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/stretchr/testify/assert"
)

func TestSessionNew(t *testing.T) {
	config.PWDContainerName = "pwd"
	var connectContainerName, connectNetworkName, connectIP string
	createdNetworkId := ""

	docker := &mockDocker{}
	docker.createNetwork = func(id string) error {
		createdNetworkId = id
		return nil
	}
	docker.connectNetwork = func(containerName, networkName, ip string) (string, error) {
		connectContainerName = containerName
		connectNetworkName = networkName
		connectIP = ip
		return "10.0.0.1", nil
	}

	var scheduledSession *Session
	tasks := &mockTasks{}
	tasks.schedule = func(s *Session) {
		scheduledSession = s
	}

	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(docker, tasks, broadcast, storage)

	before := time.Now()

	s, e := p.SessionNew(time.Hour, "", "")

	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.Equal(t, s.Id, createdNetworkId)
	assert.True(t, s.Ready)

	s, _ = p.SessionNew(time.Hour, "stackPath", "stackName")

	assert.Equal(t, "stackPath", s.Stack)
	assert.Equal(t, "stackName", s.StackName)
	assert.False(t, s.Ready)

	assert.NotNil(t, s.closingTimer)

	assert.Equal(t, config.PWDContainerName, connectContainerName)
	assert.Equal(t, s.Id, connectNetworkName)
	assert.Empty(t, connectIP)

	assert.Equal(t, "10.0.0.1", s.PwdIpAddress)

	assert.Equal(t, s, scheduledSession)
}
