package pwd

import (
	"fmt"
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInstanceResizeTerminal(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}

	_d.On("ContainerResize", "foobar", uint(24), uint(80)).Return(nil)
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)

	p := NewPWD(_f, _e, _s)

	err := p.InstanceResizeTerminal(&types.Instance{Name: "foobar", SessionId: "aaaabbbbcccc"}, 24, 80)
	assert.Nil(t, err)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestInstanceNew(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("CreateNetwork", "aaaabbbbcccc").Return(nil)
	_d.On("ConnectNetwork", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s)
	p.generator = _g

	session, err := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_node1", session.Id[:8]),
		Hostname:     "node1",
		IP:           "10.0.0.1",
		Image:        config.GetDindImageName(),
		IsDockerHost: true,
		SessionId:    session.Id,
		Session:      session,
		ProxyHost:    router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
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
	_d.On("CreateContainer", expectedContainerOpts).Return("10.0.0.1", nil)
	_s.On("InstanceCreate", "aaaabbbbcccc", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_node1", "10.0.0.1", "node1"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{Host: "something.play-with-docker.com"})
	assert.Nil(t, err)

	assert.Equal(t, expectedInstance, *instance)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestInstanceNew_WithNotAllowedImage(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("CreateNetwork", "aaaabbbbcccc").Return(nil)
	_d.On("ConnectNetwork", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s)
	p.generator = _g

	session, err := p.SessionNew(time.Hour, "", "", "")

	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_node1", session.Id[:8]),
		Hostname:     "node1",
		IP:           "10.0.0.1",
		Image:        "redis",
		SessionId:    session.Id,
		IsDockerHost: false,
		Session:      session,
		ProxyHost:    router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
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
	_d.On("CreateContainer", expectedContainerOpts).Return("10.0.0.1", nil)
	_s.On("InstanceCreate", "aaaabbbbcccc", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_node1", "10.0.0.1", "node1"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{ImageName: "redis"})
	assert.Nil(t, err)

	assert.Equal(t, expectedInstance, *instance)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestInstanceNew_WithCustomHostname(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("CreateNetwork", "aaaabbbbcccc").Return(nil)
	_d.On("ConnectNetwork", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s)
	p.generator = _g

	session, err := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:         fmt.Sprintf("%s_redis-master", session.Id[:8]),
		Hostname:     "redis-master",
		IP:           "10.0.0.1",
		Image:        "redis",
		IsDockerHost: false,
		Session:      session,
		SessionId:    session.Id,
		ProxyHost:    router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
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

	_d.On("CreateContainer", expectedContainerOpts).Return("10.0.0.1", nil)
	_s.On("InstanceCreate", "aaaabbbbcccc", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_redis-master", "10.0.0.1", "redis-master"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{ImageName: "redis", Hostname: "redis-master"})

	assert.Nil(t, err)

	assert.Equal(t, expectedInstance, *instance)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
