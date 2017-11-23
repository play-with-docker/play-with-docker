package pwd

import (
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/id"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPlaygroundNew(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	var nilArgs []interface{}
	_e.M.On("Emit", event.PLAYGROUND_NEW, uuid.NewV5(uuid.NamespaceOID, "localhost").String(), nilArgs).Return()
	_s.On("PlaygroundPut", mock.AnythingOfType("*types.Playground")).Return(nil)

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	expectedPlayground := types.Playground{Domain: "localhost", DefaultDinDInstanceImage: "franela/dind", AllowWindowsInstances: false, DefaultSessionDuration: time.Hour * 3, Extras: types.PlaygroundExtras{"foo": "bar"}}
	playground, e := p.PlaygroundNew(expectedPlayground)
	assert.Nil(t, e)
	assert.NotNil(t, playground)

	expectedPlayground.Id = uuid.NewV5(uuid.NamespaceOID, "localhost").String()
	assert.Equal(t, expectedPlayground, *playground)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestPlaygroundGet(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}

	var nilArgs []interface{}
	_e.M.On("Emit", event.PLAYGROUND_NEW, uuid.NewV5(uuid.NamespaceOID, "localhost").String(), nilArgs).Return()
	_s.On("PlaygroundPut", mock.AnythingOfType("*types.Playground")).Return(nil)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	expectedPlayground := types.Playground{Domain: "localhost", DefaultDinDInstanceImage: "franela/dind", AllowWindowsInstances: false, DefaultSessionDuration: time.Hour * 3, Extras: types.PlaygroundExtras{"foo": "bar"}}
	playground, e := p.PlaygroundNew(expectedPlayground)
	assert.Nil(t, e)
	assert.NotNil(t, playground)

	_s.On("PlaygroundGet", playground.Id).Return(playground, nil)

	playground2 := p.PlaygroundGet(playground.Id)
	assert.NotNil(t, playground2)

	assert.Equal(t, *playground, *playground2)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestPlaygroundFindByDomain(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}

	var nilArgs []interface{}
	_e.M.On("Emit", event.PLAYGROUND_NEW, uuid.NewV5(uuid.NamespaceOID, "localhost").String(), nilArgs).Return()
	_s.On("PlaygroundPut", mock.AnythingOfType("*types.Playground")).Return(nil)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	expectedPlayground := types.Playground{Domain: "localhost", DefaultDinDInstanceImage: "franela/dind", AllowWindowsInstances: false, DefaultSessionDuration: time.Hour * 3, Extras: types.PlaygroundExtras{"foo": "bar"}}
	playground, e := p.PlaygroundNew(expectedPlayground)
	assert.Nil(t, e)
	assert.NotNil(t, playground)

	_s.On("PlaygroundGet", playground.Id).Return(playground, nil)

	playground2 := p.PlaygroundFindByDomain("localhost")
	assert.NotNil(t, playground2)

	assert.Equal(t, *playground, *playground2)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestPlaygroundList(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &id.MockGenerator{}
	_e := &event.Mock{}

	var nilArgs []interface{}
	_e.M.On("Emit", event.PLAYGROUND_NEW, uuid.NewV5(uuid.NamespaceOID, "localhost1").String(), nilArgs).Return()
	_e.M.On("Emit", event.PLAYGROUND_NEW, uuid.NewV5(uuid.NamespaceOID, "localhost2").String(), nilArgs).Return()
	_s.On("PlaygroundPut", mock.AnythingOfType("*types.Playground")).Return(nil)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_g, _f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	pd := types.Playground{Domain: "localhost1", DefaultDinDInstanceImage: "franela/dind", AllowWindowsInstances: false, DefaultSessionDuration: time.Hour * 3, Extras: types.PlaygroundExtras{"foo": "bar"}}
	p1, e := p.PlaygroundNew(pd)
	assert.Nil(t, e)
	assert.NotNil(t, p1)

	pd = types.Playground{Domain: "localhost2", DefaultDinDInstanceImage: "franela/dind", AllowWindowsInstances: false, DefaultSessionDuration: time.Hour * 3, Extras: types.PlaygroundExtras{"foo": "bar"}}
	p2, e := p.PlaygroundNew(pd)
	assert.Nil(t, e)
	assert.NotNil(t, p2)

	_s.On("PlaygroundGetAll").Return([]*types.Playground{p1, p2}, nil)

	received, err := p.PlaygroundList()
	assert.Nil(t, err)
	assert.NotNil(t, received)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
