package pwd

import (
	"testing"
	"time"

	dtypes "github.com/docker/docker/api/types"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSessionNew(t *testing.T) {
	config.PWDContainerName = "pwd"

	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_f))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("CreateNetwork", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("GetDaemonHost").Return("localhost")
	_d.On("ConnectNetwork", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	before := time.Now()

	s, e := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.Equal(t, "pwd", s.StackName)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.True(t, s.Ready)

	s, _ = p.SessionNew(time.Hour, "stackPath", "stackName", "imageName")

	assert.Equal(t, "stackPath", s.Stack)
	assert.Equal(t, "stackName", s.StackName)
	assert.Equal(t, "imageName", s.ImageName)
	assert.Equal(t, "localhost", s.Host)
	assert.False(t, s.Ready)

	assert.Equal(t, "10.0.0.1", s.PwdIpAddress)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

func TestSessionSetup(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_f))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("CreateNetwork", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("GetDaemonHost").Return("localhost")
	_d.On("ConnectNetwork", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("InstanceCreate", "aaaabbbbcccc", mock.AnythingOfType("*types.Instance")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", PwdIpAddress: "10.0.0.1", ContainerName: "aaaabbbb_manager1", Hostname: "manager1", Privileged: true, HostFQDN: "localhost", Networks: map[string]string{"aaaabbbbcccc": "manager1"}}).Return("10.0.0.2", nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmInit").Return(&docker.SwarmTokens{Manager: "managerToken", Worker: "workerToken"}, nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager1", "10.0.0.2", "manager1", "ip10-0-0-2-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", PwdIpAddress: "10.0.0.1", ContainerName: "aaaabbbb_manager2", Hostname: "manager2", Privileged: true, Networks: map[string]string{"aaaabbbbcccc": "manager2"}}).Return("10.0.0.3", nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "managerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager2", "10.0.0.3", "manager2", "ip10-0-0-3-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind:overlay2-dev", SessionId: "aaaabbbbcccc", PwdIpAddress: "10.0.0.1", ContainerName: "aaaabbbb_manager3", Hostname: "manager3", Privileged: true, Networks: map[string]string{"aaaabbbbcccc": "manager3"}}).Return("10.0.0.4", nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "managerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager3", "10.0.0.4", "manager3", "ip10-0-0-4-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", PwdIpAddress: "10.0.0.1", ContainerName: "aaaabbbb_worker1", Hostname: "worker1", Privileged: true, Networks: map[string]string{"aaaabbbbcccc": "worker1"}}).Return("10.0.0.5", nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "workerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_worker1", "10.0.0.5", "worker1", "ip10-0-0-5-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", PwdIpAddress: "10.0.0.1", ContainerName: "aaaabbbb_other", Hostname: "other", Privileged: true, Networks: map[string]string{"aaaabbbbcccc": "other"}}).Return("10.0.0.6", nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_other", "10.0.0.6", "other", "ip10-0-0-6-aaaabbbbcccc"}).Return()

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g
	s, e := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, e)

	err := p.SessionSetup(s, SessionSetupConf{
		Instances: []SessionSetupInstanceConf{
			{
				Image:          "franela/dind",
				IsSwarmManager: true,
				Hostname:       "manager1",
			},
			{
				IsSwarmManager: true,
				Hostname:       "manager2",
			},
			{
				Image:          "franela/dind:overlay2-dev",
				IsSwarmManager: true,
				Hostname:       "manager3",
			},
			{
				IsSwarmWorker: true,
				Hostname:      "worker1",
			},
			{
				Hostname: "other",
			},
		},
	})
	assert.Nil(t, err)

	assert.Equal(t, 5, len(s.Instances))

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
