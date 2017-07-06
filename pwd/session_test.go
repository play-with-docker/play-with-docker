package pwd

import (
	"fmt"
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/assert"
)

func TestSessionNew(t *testing.T) {
	config.PWDContainerName = "pwd"
	var connectContainerName, connectNetworkName, connectIP string
	createdNetworkId := ""
	saveCalled := false
	expectedSessions := map[string]*types.Session{}

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

	var scheduledSession *types.Session
	tasks := &mockTasks{}
	tasks.schedule = func(s *types.Session) {
		scheduledSession = s
	}

	broadcast := &mockBroadcast{}
	storage := &mockStorage{}
	storage.sessionPut = func(s *types.Session) error {
		saveCalled = true
		return nil
	}

	p := NewPWD(docker, tasks, broadcast, storage)

	before := time.Now()

	s, e := p.SessionNew(time.Hour, "", "", "")
	expectedSessions[s.Id] = s

	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.Equal(t, "pwd", s.StackName)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.Equal(t, s.Id, createdNetworkId)
	assert.True(t, s.Ready)

	s, _ = p.SessionNew(time.Hour, "stackPath", "stackName", "imageName")
	expectedSessions[s.Id] = s

	assert.Equal(t, "stackPath", s.Stack)
	assert.Equal(t, "stackName", s.StackName)
	assert.Equal(t, "imageName", s.ImageName)
	assert.False(t, s.Ready)

	assert.NotNil(t, s.ClosingTimer())

	assert.Equal(t, config.PWDContainerName, connectContainerName)
	assert.Equal(t, s.Id, connectNetworkName)
	assert.Empty(t, connectIP)

	assert.Equal(t, "10.0.0.1", s.PwdIpAddress)

	assert.Equal(t, s, scheduledSession)

	assert.True(t, saveCalled)
}

func TestSessionSetup(t *testing.T) {
	swarmInitOnMaster1 := false
	manager2JoinedHasManager := false
	manager3JoinedHasManager := false
	worker1JoinedHasWorker := false

	dock := &mockDocker{}
	dock.createContainer = func(opts docker.CreateContainerOpts) (string, error) {
		if opts.Hostname == "manager1" {
			return "10.0.0.1", nil
		} else if opts.Hostname == "manager2" {
			return "10.0.0.2", nil
		} else if opts.Hostname == "manager3" {
			return "10.0.0.3", nil
		} else if opts.Hostname == "worker1" {
			return "10.0.0.4", nil
		} else if opts.Hostname == "other" {
			return "10.0.0.5", nil
		} else {
			assert.Fail(t, "Should not have reached here")
		}
		return "", nil
	}
	dock.new = func(ip string, cert, key []byte) (docker.DockerApi, error) {
		if ip == "10.0.0.1" {
			return &mockDocker{
				swarmInit: func() (*docker.SwarmTokens, error) {
					swarmInitOnMaster1 = true
					return &docker.SwarmTokens{Worker: "worker-join-token", Manager: "manager-join-token"}, nil
				},
			}, nil
		}
		if ip == "10.0.0.2" {
			return &mockDocker{
				swarmInit: func() (*docker.SwarmTokens, error) {
					assert.Fail(t, "Shouldn't have reached here.")
					return nil, nil
				},
				swarmJoin: func(addr, token string) error {
					if addr == "10.0.0.1:2377" && token == "manager-join-token" {
						manager2JoinedHasManager = true
						return nil
					}
					assert.Fail(t, "Shouldn't have reached here.")
					return nil
				},
			}, nil
		}
		if ip == "10.0.0.3" {
			return &mockDocker{
				swarmInit: func() (*docker.SwarmTokens, error) {
					assert.Fail(t, "Shouldn't have reached here.")
					return nil, nil
				},
				swarmJoin: func(addr, token string) error {
					if addr == "10.0.0.1:2377" && token == "manager-join-token" {
						manager3JoinedHasManager = true
						return nil
					}
					assert.Fail(t, "Shouldn't have reached here.")
					return nil
				},
			}, nil
		}
		if ip == "10.0.0.4" {
			return &mockDocker{
				swarmInit: func() (*docker.SwarmTokens, error) {
					assert.Fail(t, "Shouldn't have reached here.")
					return nil, nil
				},
				swarmJoin: func(addr, token string) error {
					if addr == "10.0.0.1:2377" && token == "worker-join-token" {
						worker1JoinedHasWorker = true
						return nil
					}
					assert.Fail(t, "Shouldn't have reached here.")
					return nil
				},
			}, nil
		}
		assert.Fail(t, "Shouldn't have reached here.")
		return nil, nil
	}
	tasks := &mockTasks{}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	p := NewPWD(dock, tasks, broadcast, storage)
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

	manager1 := fmt.Sprintf("%s_manager1", s.Id[:8])
	manager1Received := *s.Instances[manager1]
	assert.Equal(t, types.Instance{
		Name:         manager1,
		Image:        "franela/dind",
		Hostname:     "manager1",
		IP:           "10.0.0.1",
		Alias:        "",
		IsDockerHost: true,
		Session:      s,
		Terminal:     manager1Received.Terminal,
		Docker:       manager1Received.Docker,
	}, manager1Received)

	manager2 := fmt.Sprintf("%s_manager2", s.Id[:8])
	manager2Received := *s.Instances[manager2]
	assert.Equal(t, types.Instance{
		Name:         manager2,
		Image:        "franela/dind",
		Hostname:     "manager2",
		IP:           "10.0.0.2",
		Alias:        "",
		IsDockerHost: true,
		Session:      s,
		Terminal:     manager2Received.Terminal,
		Docker:       manager2Received.Docker,
	}, manager2Received)

	manager3 := fmt.Sprintf("%s_manager3", s.Id[:8])
	manager3Received := *s.Instances[manager3]
	assert.Equal(t, types.Instance{
		Name:         manager3,
		Image:        "franela/dind:overlay2-dev",
		Hostname:     "manager3",
		IP:           "10.0.0.3",
		Alias:        "",
		IsDockerHost: true,
		Session:      s,
		Terminal:     manager3Received.Terminal,
		Docker:       manager3Received.Docker,
	}, manager3Received)

	worker1 := fmt.Sprintf("%s_worker1", s.Id[:8])
	worker1Received := *s.Instances[worker1]
	assert.Equal(t, types.Instance{
		Name:         worker1,
		Image:        "franela/dind",
		Hostname:     "worker1",
		IP:           "10.0.0.4",
		Alias:        "",
		IsDockerHost: true,
		Session:      s,
		Terminal:     worker1Received.Terminal,
		Docker:       worker1Received.Docker,
	}, worker1Received)

	other := fmt.Sprintf("%s_other", s.Id[:8])
	otherReceived := *s.Instances[other]
	assert.Equal(t, types.Instance{
		Name:         other,
		Image:        "franela/dind",
		Hostname:     "other",
		IP:           "10.0.0.5",
		Alias:        "",
		IsDockerHost: true,
		Session:      s,
		Terminal:     otherReceived.Terminal,
		Docker:       otherReceived.Docker,
	}, otherReceived)

	assert.True(t, swarmInitOnMaster1)
	assert.True(t, manager2JoinedHasManager)
	assert.True(t, manager3JoinedHasManager)
	assert.True(t, worker1JoinedHasWorker)
}
