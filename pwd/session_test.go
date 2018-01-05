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

func TestSessionNew(t *testing.T) {
	config.PWDContainerName = "pwd"

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
	_s.On("InstanceCount").Return(0, nil)
	_s.On("ClientCount").Return(0, nil)

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g

	before := time.Now()

	playground := &types.Playground{Id: "foobar"}
	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	s, e := p.SessionNew(context.Background(), sConfig)
	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.Equal(t, "pwd", s.StackName)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.True(t, s.Ready)

	sConfig = types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "stackPath", StackName: "stackName", ImageName: "imageName"}
	s, _ = p.SessionNew(context.Background(), sConfig)

	assert.Equal(t, "stackPath", s.Stack)
	assert.Equal(t, "stackName", s.StackName)
	assert.Equal(t, "imageName", s.ImageName)
	assert.Equal(t, "localhost", s.Host)
	assert.Equal(t, playground.Id, s.PlaygroundId)
	assert.False(t, s.Ready)

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}

/*

************************** Not sure how to test this as it can pick any manager as the first node in the swarm cluster.


func TestSessionSetup(t *testing.T) {
	_d := &docker.Mock{}
	_f := &docker.FactoryMock{}
	_s := &storage.Mock{}
	_g := &mockGenerator{}
	_e := &event.Mock{}
	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(_f, _s), provisioner.NewDinD(_f, _s))
	sp := provisioner.NewOverlaySessionProvisioner(_f)

	_g.On("NewId").Return("aaaabbbbcccc")
	_f.On("GetForSession", "aaaabbbbcccc").Return(_d, nil)
	_d.On("NetworkCreate", "aaaabbbbcccc", dtypes.NetworkCreate{Attachable: true, Driver: "overlay"}).Return(nil)
	_d.On("DaemonHost").Return("localhost")
	_d.On("NetworkConnect", config.L2ContainerName, "aaaabbbbcccc", "").Return("10.0.0.1", nil)
	_s.On("SessionPut", mock.AnythingOfType("*types.Session")).Return(nil)
	_s.On("InstancePut", mock.AnythingOfType("*types.Instance")).Return(nil)
	_s.On("SessionCount").Return(1, nil)
	_s.On("ClientCount").Return(1, nil)
	_s.On("InstanceCount").Return(0, nil)
	_s.On("InstanceFindBySessionId", "aaaabbbbcccc").Return([]*types.Instance{}, nil)

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", ContainerName: "aaaabbbb_manager1", Hostname: "manager1", Privileged: true, HostFQDN: "localhost", Networks: []string{"aaaabbbbcccc"}}).Return(nil)
	_d.On("ContainerIPs", "aaaabbbb_manager1").Return(map[string]string{"aaaabbbbcccc": "10.0.0.2"}, nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmInit").Return(&docker.SwarmTokens{Manager: "managerToken", Worker: "workerToken"}, nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager1", "10.0.0.2", "manager1", "ip10-0-0-2-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", ContainerName: "aaaabbbb_manager2", Hostname: "manager2", Privileged: true, HostFQDN: "localhost", Networks: []string{"aaaabbbbcccc"}}).Return(nil)
	_d.On("ContainerIPs", "aaaabbbb_manager2").Return(map[string]string{"aaaabbbbcccc": "10.0.0.3"}, nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "managerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager2", "10.0.0.3", "manager2", "ip10-0-0-3-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind:overlay2-dev", SessionId: "aaaabbbbcccc", ContainerName: "aaaabbbb_manager3", Hostname: "manager3", Privileged: true, HostFQDN: "localhost", Networks: []string{"aaaabbbbcccc"}}).Return(nil)
	_d.On("ContainerIPs", "aaaabbbb_manager3").Return(map[string]string{"aaaabbbbcccc": "10.0.0.4"}, nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "managerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_manager3", "10.0.0.4", "manager3", "ip10-0-0-4-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", ContainerName: "aaaabbbb_worker1", Hostname: "worker1", Privileged: true, HostFQDN: "localhost", Networks: []string{"aaaabbbbcccc"}}).Return(nil)
	_d.On("ContainerIPs", "aaaabbbb_worker1").Return(map[string]string{"aaaabbbbcccc": "10.0.0.5"}, nil)
	_f.On("GetForInstance", mock.AnythingOfType("*types.Instance")).Return(_d, nil)
	_d.On("SwarmJoin", "10.0.0.2:2377", "workerToken").Return(nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_worker1", "10.0.0.5", "worker1", "ip10-0-0-5-aaaabbbbcccc"}).Return()

	_d.On("CreateContainer", docker.CreateContainerOpts{Image: "franela/dind", SessionId: "aaaabbbbcccc", ContainerName: "aaaabbbb_other", Hostname: "other", Privileged: true, HostFQDN: "localhost", Networks: []string{"aaaabbbbcccc"}}).Return(nil)
	_d.On("ContainerIPs", "aaaabbbb_other").Return(map[string]string{"aaaabbbbcccc": "10.0.0.6"}, nil)
	_e.M.On("Emit", event.INSTANCE_NEW, "aaaabbbbcccc", []interface{}{"aaaabbbb_other", "10.0.0.6", "other", "ip10-0-0-6-aaaabbbbcccc"}).Return()

	var nilArgs []interface{}
	_e.M.On("Emit", event.SESSION_NEW, "aaaabbbbcccc", nilArgs).Return()

	p := NewPWD(_f, _e, _s, sp, ipf)
	p.generator = _g
	sConfig := types.SessionConfig{Playground: playground, UserId: "", Duration: time.Hour, Stack: "", StackName: "", ImageName: ""}
	s, e := p.SessionNew(context.Background(), sConfig)
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

	_d.AssertExpectations(t)
	_f.AssertExpectations(t)
	_s.AssertExpectations(t)
	_g.AssertExpectations(t)
	_e.M.AssertExpectations(t)
}
*/
