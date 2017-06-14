package pwd

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/stretchr/testify/assert"
)

func TestSessionNew(t *testing.T) {
	sessions = map[string]*Session{}

	config.PWDContainerName = "pwd"
	var connectContainerName, connectNetworkName, connectIP string
	createdNetworkId := ""
	saveCalled := false
	expectedSessions := map[string]*Session{}

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
	storage.save = func() error {
		saveCalled = true
		return nil
	}

	p := NewPWD(docker, tasks, broadcast, storage)

	before := time.Now()

	s, e := p.SessionNew(time.Hour, "", "")
	expectedSessions[s.Id] = s

	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.Equal(t, "pwd", s.StackName)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.Equal(t, s.Id, createdNetworkId)
	assert.True(t, s.Ready)

	s, _ = p.SessionNew(time.Hour, "stackPath", "stackName")
	expectedSessions[s.Id] = s

	assert.Equal(t, "stackPath", s.Stack)
	assert.Equal(t, "stackName", s.StackName)
	assert.False(t, s.Ready)

	assert.NotNil(t, s.closingTimer)

	assert.Equal(t, config.PWDContainerName, connectContainerName)
	assert.Equal(t, s.Id, connectNetworkName)
	assert.Empty(t, connectIP)

	assert.Equal(t, "10.0.0.1", s.PwdIpAddress)

	assert.Equal(t, s, scheduledSession)

	assert.Equal(t, expectedSessions, sessions)
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
	s, e := p.SessionNew(time.Hour, "", "")
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
	assert.Equal(t, Instance{
		Name:         manager1,
		Image:        "franela/dind",
		Hostname:     "manager1",
		IP:           "10.0.0.1",
		Alias:        "",
		IsDockerHost: true,
		session:      s,
		conn:         manager1Received.conn,
		docker:       manager1Received.docker,
	}, manager1Received)

	manager2 := fmt.Sprintf("%s_manager2", s.Id[:8])
	manager2Received := *s.Instances[manager2]
	assert.Equal(t, Instance{
		Name:         manager2,
		Image:        "franela/dind",
		Hostname:     "manager2",
		IP:           "10.0.0.2",
		Alias:        "",
		IsDockerHost: true,
		session:      s,
		conn:         manager2Received.conn,
		docker:       manager2Received.docker,
	}, manager2Received)

	manager3 := fmt.Sprintf("%s_manager3", s.Id[:8])
	manager3Received := *s.Instances[manager3]
	assert.Equal(t, Instance{
		Name:         manager3,
		Image:        "franela/dind:overlay2-dev",
		Hostname:     "manager3",
		IP:           "10.0.0.3",
		Alias:        "",
		IsDockerHost: true,
		session:      s,
		conn:         manager3Received.conn,
		docker:       manager3Received.docker,
	}, manager3Received)

	worker1 := fmt.Sprintf("%s_worker1", s.Id[:8])
	worker1Received := *s.Instances[worker1]
	assert.Equal(t, Instance{
		Name:         worker1,
		Image:        "franela/dind",
		Hostname:     "worker1",
		IP:           "10.0.0.4",
		Alias:        "",
		IsDockerHost: true,
		session:      s,
		conn:         worker1Received.conn,
		docker:       worker1Received.docker,
	}, worker1Received)

	other := fmt.Sprintf("%s_other", s.Id[:8])
	otherReceived := *s.Instances[other]
	assert.Equal(t, Instance{
		Name:         other,
		Image:        "franela/dind",
		Hostname:     "other",
		IP:           "10.0.0.5",
		Alias:        "",
		IsDockerHost: true,
		session:      s,
		conn:         otherReceived.conn,
		docker:       otherReceived.docker,
	}, otherReceived)

	assert.True(t, swarmInitOnMaster1)
	assert.True(t, manager2JoinedHasManager)
	assert.True(t, manager3JoinedHasManager)
	assert.True(t, worker1JoinedHasWorker)
}

func TestSessionLoadAndPrepare(t *testing.T) {
	config.PWDContainerName = "pwd"
	lock := sync.Mutex{}
	var s1NetworkConnect []string
	var s2NetworkConnect []string

	wg := sync.WaitGroup{}
	wg.Add(3)
	connectedInstances := []string{}
	sessions = map[string]*Session{}
	i1 := &Instance{
		Image:        "dind",
		Name:         "session1_i1",
		Hostname:     "i1",
		IP:           "10.0.0.10",
		IsDockerHost: true,
	}
	i2 := &Instance{
		Image:        "dind",
		Name:         "session1_i2",
		Hostname:     "i1",
		IP:           "10.0.0.11",
		IsDockerHost: true,
	}
	i3 := &Instance{
		Image:        "dind",
		Name:         "session1_i3",
		Hostname:     "i1",
		IP:           "10.0.0.12",
		IsDockerHost: true,
	}
	s1 := &Session{
		Id:           "session1",
		Instances:    map[string]*Instance{"session1_i1": i1},
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		PwdIpAddress: "10.0.0.1",
		Ready:        true,
		Stack:        "",
		StackName:    "",
	}
	s2 := &Session{
		Id:           "session2",
		Instances:    map[string]*Instance{"session1_i2": i2, "session1_i3": i3},
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		PwdIpAddress: "10.0.0.2",
		Ready:        true,
		Stack:        "",
		StackName:    "",
	}

	dock := &mockDocker{}
	dock.createAttachConnection = func(instanceName string) (net.Conn, error) {
		lock.Lock()
		defer lock.Unlock()
		connectedInstances = append(connectedInstances, instanceName)
		wg.Done()
		return &mockConn{}, nil
	}
	dock.connectNetwork = func(container, network, ip string) (string, error) {
		if s1.Id == network {
			s1NetworkConnect = []string{container, network, ip}
		} else if s2.Id == network {
			s2NetworkConnect = []string{container, network, ip}
		}
		return ip, nil
	}
	tasks := &mockTasks{}
	tasks.schedule = func(s *Session) {
		s.ticker = time.NewTicker(1 * time.Second)
	}
	broadcast := &mockBroadcast{}
	storage := &mockStorage{}

	storage.load = func() error {
		sessions = map[string]*Session{"session1": s1, "session2": s2}
		return nil
	}

	p := NewPWD(dock, tasks, broadcast, storage)

	err := p.SessionLoadAndPrepare()
	assert.Nil(t, err)
	assert.Len(t, sessions, 2)
	assert.NotNil(t, s1.closingTimer)
	assert.NotNil(t, s2.closingTimer)
	assert.NotNil(t, s1.ticker)
	assert.NotNil(t, s2.ticker)

	assert.Equal(t, []string{"pwd", s1.Id, s1.PwdIpAddress}, s1NetworkConnect)
	assert.Equal(t, []string{"pwd", s2.Id, s2.PwdIpAddress}, s2NetworkConnect)

	wg.Wait()
	assert.Subset(t, connectedInstances, []string{i1.Name, i2.Name, i3.Name})
}
