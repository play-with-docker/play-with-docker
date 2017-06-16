package pwd

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
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

	err := p.InstanceResizeTerminal(&types.Instance{Name: "foobar"}, 24, 80)

	assert.Nil(t, err)
	assert.Equal(t, "foobar", resizedInstanceName)
	assert.Equal(t, uint(24), resizedRows)
	assert.Equal(t, uint(80), resizedCols)
}

func TestInstanceNew(t *testing.T) {
	containerOpts := docker.CreateContainerOpts{}
	dock := &mockDocker{}
	dock.createContainer = func(opts docker.CreateContainerOpts) (string, error) {
		containerOpts = opts
		return "10.0.0.1", nil
	}

	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")

	assert.Nil(t, err)

	instance, err := p.InstanceNew(session, InstanceConfig{Host: "something.play-with-docker.com"})

	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_node1", session.Id[:8]),
		Hostname:     "node1",
		IP:           "10.0.0.1",
		Alias:        "",
		Image:        config.GetDindImageName(),
		IsDockerHost: true,
		Session:      session,
	}

	assert.Equal(t, expectedInstance, *instance)

	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		PwdIpAddress:  session.PwdIpAddress,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    true,
		HostFQDN:      "something.play-with-docker.com",
	}
	assert.Equal(t, expectedContainerOpts, containerOpts)
}

func TestInstanceNew_Concurrency(t *testing.T) {
	i := 0
	dock := &mockDocker{}
	dock.createContainer = func(opts docker.CreateContainerOpts) (string, error) {
		time.Sleep(time.Second)
		i++
		return fmt.Sprintf("10.0.0.%d", i), nil
	}

	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")

	assert.Nil(t, err)

	var instance1 *types.Instance
	var instance2 *types.Instance

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		instance, err := p.InstanceNew(session, InstanceConfig{})
		assert.Nil(t, err)
		instance1 = instance
	}()
	go func() {
		defer wg.Done()
		instance, err := p.InstanceNew(session, InstanceConfig{})
		assert.Nil(t, err)
		instance2 = instance
	}()
	wg.Wait()

	assert.Subset(t, []string{"node1", "node2"}, []string{instance1.Hostname, instance2.Hostname})
}

func TestInstanceNew_WithNotAllowedImage(t *testing.T) {
	containerOpts := docker.CreateContainerOpts{}
	dock := &mockDocker{}
	dock.createContainer = func(opts docker.CreateContainerOpts) (string, error) {
		containerOpts = opts
		return "10.0.0.1", nil
	}

	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")

	assert.Nil(t, err)

	instance, err := p.InstanceNew(session, InstanceConfig{ImageName: "redis"})

	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_node1", session.Id[:8]),
		Hostname:     "node1",
		IP:           "10.0.0.1",
		Alias:        "",
		Image:        "redis",
		IsDockerHost: false,
		Session:      session,
	}

	assert.Equal(t, expectedInstance, *instance)

	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		PwdIpAddress:  session.PwdIpAddress,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    false,
	}
	assert.Equal(t, expectedContainerOpts, containerOpts)
}

func TestInstanceNew_WithCustomHostname(t *testing.T) {
	containerOpts := docker.CreateContainerOpts{}
	dock := &mockDocker{}
	dock.createContainer = func(opts docker.CreateContainerOpts) (string, error) {
		containerOpts = opts
		return "10.0.0.1", nil
	}

	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")

	assert.Nil(t, err)

	instance, err := p.InstanceNew(session, InstanceConfig{ImageName: "redis", Hostname: "redis-master"})

	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_redis-master", session.Id[:8]),
		Hostname:     "redis-master",
		IP:           "10.0.0.1",
		Alias:        "",
		Image:        "redis",
		IsDockerHost: false,
		Session:      session,
	}

	assert.Equal(t, expectedInstance, *instance)

	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		PwdIpAddress:  session.PwdIpAddress,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    false,
	}
	assert.Equal(t, expectedContainerOpts, containerOpts)
}

func TestInstanceAllowedImages(t *testing.T) {
	dock := &mockDocker{}
	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)

	expectedImages := []string{config.GetDindImageName(), "franela/dind:overlay2-dev", "franela/ucp:2.4.1"}

	assert.Equal(t, expectedImages, p.InstanceAllowedImages())
}
