package pwd

import (
	"context"
	"fmt"
	"testing"
	"time"

	dtypes "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thebsdbox/play-with-docker/config"
	"github.com/thebsdbox/play-with-docker/docker"
	"github.com/thebsdbox/play-with-docker/event"
	"github.com/thebsdbox/play-with-docker/id"
	"github.com/thebsdbox/play-with-docker/provisioner"
	"github.com/thebsdbox/play-with-docker/pwd/types"
	"github.com/thebsdbox/play-with-docker/router"
	"github.com/thebsdbox/play-with-docker/storage"
)

func TestInstanceResizeTerminal(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	s := &types.Session{Id: "aaaabbbbcccc"}
	_d.On("ContainerResize", "foobar", uint(24), uint(80)).Return(nil)
	_s.On("SessionGet", "aaaabbbbcccc").Return(s, nil)
	_f.On("GetForSession", s).Return(_d, nil)

	p := NewPWD(_f, _e, _s, sp, ipf)

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
	_g := &id.MockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", mock.AnythingOfType("*types.Session")).Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("ClientCount").Return(0, nil)
	_s.On("InstanceCount").Return(0, nil)
	_s.On("InstanceFindBySessionId", "aaaabbbbcccc").Return([]*types.Instance{}, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	playground := &types.Playground{Id: "foobar", DefaultDinDInstanceImage: "franela/dind"}

	_s.On("PlaygroundGet", "foobar").Return(playground, nil)

	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:        fmt.Sprintf("%s_aaaabbbbcccc", session.Id[:8]),
		Hostname:    "node1",
		IP:          "10.0.0.1",
		RoutableIP:  "10.0.0.1",
		Image:       "franela/dind",
		SessionId:   session.Id,
		SessionHost: session.Host,
		ProxyHost:   router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    false,
		HostFQDN:      "something.play-with-docker.com",
		Networks:      []string{session.Id},
	}
	_d.On("ContainerCreate", expectedContainerOpts).Return(nil)
	_d.On("ContainerIPs", expectedInstance.Name).Return(map[string]string{session.Id: "10.0.0.1"}, nil)
	_s.On("InstancePut", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_aaaabbbbcccc", "10.0.0.1", "node1", "ip10-0-0-1-aaaabbbbcccc"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{PlaygroundFQDN: "something.play-with-docker.com"})
	assert.Nil(t, err)

	assert.Equal(t, expectedInstance, *instance)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestInstanceNew_Privileged(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", mock.AnythingOfType("*types.Session")).Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("ClientCount").Return(0, nil)
	_s.On("InstanceCount").Return(0, nil)
	_s.On("InstanceFindBySessionId", "aaaabbbbcccc").Return([]*types.Instance{}, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	playground := &types.Playground{Id: "foobar"}
	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)

	assert.Nil(t, err)

	// Switch to unsafe mode in order to test custom networks below
	//
	// TODO: move config away from being a global in order that we don't
	// have to hack setting the context in this way.
	config.Unsafe = true
	defer func() {
		config.Unsafe = false
	}()

	expectedInstance := types.Instance{
		Name:        fmt.Sprintf("%s_aaaabbbbcccc", session.Id[:8]),
		Hostname:    "node1",
		IP:          "10.0.0.1",
		RoutableIP:  "10.0.0.1",
		Image:       "redis",
		SessionId:   session.Id,
		SessionHost: session.Host,
		ProxyHost:   router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    true,
		Envs:          []string{"HELLO=WORLD"},
		Networks:      []string{session.Id, "arpanet"},
	}
	_d.On("ContainerCreate", expectedContainerOpts).Return(nil)
	_d.On("ContainerIPs", expectedInstance.Name).Return(map[string]string{session.Id: "10.0.0.1"}, nil)
	_s.On("InstancePut", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_aaaabbbbcccc", "10.0.0.1", "node1", "ip10-0-0-1-aaaabbbbcccc"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{ImageName: "redis", Envs: []string{"HELLO=WORLD"}, Networks: []string{"arpanet"}, Privileged: true})
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
	_g := &id.MockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", mock.AnythingOfType("*types.Session")).Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("ClientCount").Return(0, nil)
	_s.On("InstanceCount").Return(0, nil)
	_s.On("InstanceFindBySessionId", "aaaabbbbcccc").Return([]*types.Instance{}, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	playground := &types.Playground{Id: "foobar"}
	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)

	assert.Nil(t, err)

	// Switch to unsafe mode in order to test custom networks below
	//
	// TODO: move config away from being a global in order that we don't
	// have to hack setting the context in this way.
	config.Unsafe = true
	defer func() {
		config.Unsafe = false
	}()

	expectedInstance := types.Instance{
		Name:        fmt.Sprintf("%s_aaaabbbbcccc", session.Id[:8]),
		Hostname:    "node1",
		IP:          "10.0.0.1",
		RoutableIP:  "10.0.0.1",
		Image:       "redis",
		SessionId:   session.Id,
		SessionHost: session.Host,
		ProxyHost:   router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    false,
		Envs:          []string{"HELLO=WORLD"},
		Networks:      []string{session.Id, "arpanet"},
	}
	_d.On("ContainerCreate", expectedContainerOpts).Return(nil)
	_d.On("ContainerIPs", expectedInstance.Name).Return(map[string]string{session.Id: "10.0.0.1"}, nil)
	_s.On("InstancePut", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_aaaabbbbcccc", "10.0.0.1", "node1", "ip10-0-0-1-aaaabbbbcccc"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{ImageName: "redis", Envs: []string{"HELLO=WORLD"}, Networks: []string{"arpanet"}})
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
	_g := &id.MockGenerator{}
	_e := &event.Mock{}

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", mock.AnythingOfType("*types.Session")).Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("ClientCount").Return(0, nil)
	_s.On("InstanceCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	playground := &types.Playground{Id: "foobar"}
	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, err)

	expectedInstance := types.Instance{
		Name:        fmt.Sprintf("%s_aaaabbbbcccc", session.Id[:8]),
		Hostname:    "redis-master",
		IP:          "10.0.0.1",
		RoutableIP:  "10.0.0.1",
		Image:       "redis",
		SessionHost: session.Host,
		SessionId:   session.Id,
		ProxyHost:   router.EncodeHost(session.Id, "10.0.0.1", router.HostOpts{}),
	}
	expectedContainerOpts := docker.CreateContainerOpts{
		Image:         expectedInstance.Image,
		SessionId:     session.Id,
		ContainerName: expectedInstance.Name,
		Hostname:      expectedInstance.Hostname,
		ServerCert:    nil,
		ServerKey:     nil,
		CACert:        nil,
		Privileged:    false,
		Networks:      []string{session.Id},
	}

	_d.On("ContainerCreate", expectedContainerOpts).Return(nil)
	_d.On("ContainerIPs", expectedInstance.Name).Return(map[string]string{session.Id: "10.0.0.1"}, nil)
	_s.On("InstancePut", mock.AnythingOfType("*types.Instance")).Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_aaaabbbbcccc", "10.0.0.1", "redis-master", "ip10-0-0-1-aaaabbbbcccc"}).Return()

	instance, err := p.InstanceNew(session, types.InstanceConfig{ImageName: "redis", Hostname: "redis-master"})

	assert.Nil(t, err)

	assert.Equal(t, expectedInstance, *instance)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
