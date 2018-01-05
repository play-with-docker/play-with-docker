package pwd

import (
	"context"
	"testing"
	"time"

	dtypes "docker.io/go-docker/api/types"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/id"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClientNew(t *testing.T) {
	_s := &storage.Mock{}
	_f := &docker.FactoryMock{}
	_g := &id.MockGenerator{}
	_d := &docker.Mock{}
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
	_s.On("InstanceCount").Return(0, nil)
	_s.On("ClientCount").Return(1, nil)
	_s.On("ClientPut", mock.AnythingOfType("*types.Client")).Return(nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	playground := &types.Playground{Id: "foobar"}

	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, err)

	client := p.ClientNew("foobar", session)

	assert.Equal(t, types.Client{Id: "foobar", SessionId: session.Id, ViewPort: types.ViewPort{Cols: 0, Rows: 0}}, *client)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestClientCount(t *testing.T) {
	_s := &storage.Mock{}
	_f := &docker.FactoryMock{}
	_g := &id.MockGenerator{}
	_d := &docker.Mock{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", mock.AnythingOfType("*types.Session")).Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("ClientPut", mock.AnythingOfType("*types.Client")).Return(nil)
	_s.On("ClientCount").Return(1, nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(-1, nil)
	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g
	playground := &types.Playground{Id: "foobar"}

	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, err)

	p.ClientNew("foobar", session)

	assert.Equal(t, 1, p.ClientCount())

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestClientResizeViewPort(t *testing.T) {
	_s := &storage.Mock{}
	_f := &docker.FactoryMock{}
	_g := &id.MockGenerator{}
	_d := &docker.Mock{}
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
	_s.On("InstanceCount").Return(0, nil)
	_s.On("InstanceFindBySessionId", "aaaabbbbcccc").Return([]*types.Instance{}, nil)
	_s.On("ClientPut", mock.AnythingOfType("*types.Client")).Return(nil)
	_s.On("ClientCount").Return(1, nil)
	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	_e.M.On("Emit", event.INSTANCE_VIEWPORT_RESIZE, "aaaabbbbcccc", []interface{}{uint(80), uint(24)}).Return()
	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g
	playground := &types.Playground{Id: "foobar"}

	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	session, err := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, err)
	client := p.ClientNew("foobar", session)
	_s.On("ClientFindBySessionId", "aaaabbbbcccc").Return([]*types.Client{client}, nil)

	p.ClientResizeViewPort(client, 80, 24)

	assert.Equal(t, types.ViewPort{Cols: 80, Rows: 24}, client.ViewPort)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
