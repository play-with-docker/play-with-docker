package pwd

import (
	"fmt"
	"log"
	"math"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

var preparedSessions = map[string]bool{}

type sessionBuilderWriter struct {
	sessionId string
	event     event.EventApi
}

func (s *sessionBuilderWriter) Write(p []byte) (n int, err error) {
	s.event.Emit(event.SESSION_BUILDER_OUT, s.sessionId, string(p))
	return len(p), nil
}

type SessionSetupConf struct {
	Instances []SessionSetupInstanceConf `json:"instances"`
}

type SessionSetupInstanceConf struct {
	Image          string `json:"image"`
	Hostname       string `json:"hostname"`
	IsSwarmManager bool   `json:"is_swarm_manager"`
	IsSwarmWorker  bool   `json:"is_swarm_worker"`
}

func (p *pwd) SessionNew(duration time.Duration, stack, stackName, imageName string) (*types.Session, error) {
	defer observeAction("SessionNew", time.Now())

	s := &types.Session{}
	s.Id = p.generator.NewId()
	s.Instances = map[string]*types.Instance{}
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(duration)
	s.Ready = true
	s.Stack = stack

	if s.Stack != "" {
		s.Ready = false
	}
	if stackName == "" {
		stackName = "pwd"
	}
	s.StackName = stackName
	s.ImageName = imageName

	log.Printf("NewSession id=[%s]\n", s.Id)

	if err := p.docker(s.Id).CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	if err := p.connectToNetwork(s); err != nil {
		return nil, err
	}

	if err := p.storage.SessionPut(s); err != nil {
		log.Println(err)
		return nil, err
	}

	p.setGauges()
	p.event.Emit(event.SESSION_NEW, s.Id)

	return s, nil
}

func (p *pwd) SessionClose(s *types.Session) error {
	defer observeAction("SessionClose", time.Now())

	s.Lock()
	defer s.Unlock()

	log.Printf("Starting clean up of session [%s]\n", s.Id)
	for _, i := range s.Instances {
		err := p.InstanceDelete(s, i)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	// Disconnect PWD daemon from the network
	if err := p.docker(s.Id).DisconnectNetwork(config.L2ContainerName, s.Id); err != nil {
		if !strings.Contains(err.Error(), "is not connected to the network") {
			log.Println("ERROR NETWORKING")
			return err
		}
	}
	log.Printf("Disconnected pwd from network [%s]\n", s.Id)
	if err := p.docker(s.Id).DeleteNetwork(s.Id); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			log.Println(err)
			return err
		}
	}

	err := p.storage.SessionDelete(s.Id)
	if err != nil {
		return err
	}

	log.Printf("Cleaned up session [%s]\n", s.Id)
	p.setGauges()
	p.event.Emit(event.SESSION_END, s.Id)
	return nil

}

func (p *pwd) SessionGetSmallestViewPort(s *types.Session) types.ViewPort {
	defer observeAction("SessionGetSmallestViewPort", time.Now())

	minRows := s.Clients[0].ViewPort.Rows
	minCols := s.Clients[0].ViewPort.Cols

	for _, c := range s.Clients {
		minRows = uint(math.Min(float64(minRows), float64(c.ViewPort.Rows)))
		minCols = uint(math.Min(float64(minCols), float64(c.ViewPort.Cols)))
	}

	return types.ViewPort{Rows: minRows, Cols: minCols}
}

func (p *pwd) SessionDeployStack(s *types.Session) error {
	defer observeAction("SessionDeployStack", time.Now())

	if s.Ready {
		// a stack was already deployed on this session, just ignore
		return nil
	}

	s.Ready = false
	p.event.Emit(event.SESSION_READY, s.Id, false)
	i, err := p.InstanceNew(s, types.InstanceConfig{ImageName: s.ImageName, Host: s.Host})
	if err != nil {
		log.Printf("Error creating instance for stack [%s]: %s\n", s.Stack, err)
		return err
	}

	_, fileName := filepath.Split(s.Stack)
	err = p.InstanceUploadFromUrl(i, fileName, "/var/run/pwd/uploads", s.Stack)
	if err != nil {
		log.Printf("Error uploading stack file [%s]: %s\n", s.Stack, err)
		return err
	}

	fileName = path.Base(s.Stack)
	file := fmt.Sprintf("/var/run/pwd/uploads/%s", fileName)
	cmd := fmt.Sprintf("docker swarm init --advertise-addr eth0 && docker-compose -f %s pull && docker stack deploy -c %s %s", file, file, s.StackName)

	w := sessionBuilderWriter{sessionId: s.Id, event: p.event}
	code, err := p.docker(s.Id).ExecAttach(i.Name, []string{"sh", "-c", cmd}, &w)
	if err != nil {
		log.Printf("Error executing stack [%s]: %s\n", s.Stack, err)
		return err
	}

	log.Printf("Stack execution finished with code %d\n", code)
	s.Ready = true
	p.event.Emit(event.SESSION_READY, s.Id, true)
	if err := p.storage.SessionPut(s); err != nil {
		return err
	}
	return nil
}

func (p *pwd) SessionGet(sessionId string) *types.Session {
	defer observeAction("SessionGet", time.Now())

	s, err := p.storage.SessionGet(sessionId)

	if err != nil {
		log.Println(err)
		return nil
	}

	return s
}

func (p *pwd) SessionSetup(session *types.Session, conf SessionSetupConf) error {
	defer observeAction("SessionSetup", time.Now())
	var tokens *docker.SwarmTokens = nil
	var firstSwarmManager *types.Instance = nil

	// first look for a swarm manager and create it
	for _, conf := range conf.Instances {
		if conf.IsSwarmManager {
			instanceConf := types.InstanceConfig{
				ImageName: conf.Image,
				Hostname:  conf.Hostname,
				Host:      session.Host,
			}
			i, err := p.InstanceNew(session, instanceConf)
			if err != nil {
				return err
			}
			dockerClient, err := p.dockerFactory.GetForInstance(session.Id, i.Name)
			if err != nil {
				return err
			}
			tkns, err := dockerClient.SwarmInit()
			if err != nil {
				return err
			}
			tokens = tkns
			firstSwarmManager = i
			break
		}
	}

	// now create the rest in parallel

	wg := sync.WaitGroup{}
	for _, c := range conf.Instances {
		if firstSwarmManager != nil && c.Hostname != firstSwarmManager.Hostname {
			wg.Add(1)
			go func(c SessionSetupInstanceConf) {
				defer wg.Done()
				instanceConf := types.InstanceConfig{
					ImageName: c.Image,
					Hostname:  c.Hostname,
				}
				i, err := p.InstanceNew(session, instanceConf)
				if err != nil {
					log.Println(err)
					return
				}

				if firstSwarmManager != nil {
					if c.IsSwarmManager {
						dockerClient, err := p.dockerFactory.GetForInstance(session.Id, i.Name)
						if err != nil {
							log.Println(err)
							return
						}
						// this is a swarm manager
						// cluster has already been initiated, join as manager
						err = dockerClient.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Manager)
						if err != nil {
							log.Println(err)
							return
						}
					}
					if c.IsSwarmWorker {
						dockerClient, err := p.dockerFactory.GetForInstance(session.Id, i.Name)
						if err != nil {
							log.Println(err)
							return
						}
						// this is a swarm worker
						err = dockerClient.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Worker)
						if err != nil {
							log.Println(err)
							return
						}
					}
				}
			}(c)
		}
	}
	wg.Wait()

	return nil
}

/*
// This function should be called any time a session needs to be prepared:
// 1. Like when it is created
// 2. When it was loaded from storage
func (p *pwd) prepareSession(session *types.Session) (bool, error) {
	session.Lock()
	defer session.Unlock()

	if isSessionPrepared(session.Id) {
		return false, nil
	}

	// Connect PWD daemon to the new network
	if err := p.connectToNetwork(session); err != nil {
		return false, err
	}

	for _, i := range session.Instances {
		// wire the session back to the instance
		i.Session = session
		go p.InstanceAttachTerminal(i)
	}
	preparedSessions[session.Id] = true

	return true, nil
}
*/

func (p *pwd) connectToNetwork(s *types.Session) error {
	ip, err := p.docker(s.Id).ConnectNetwork(config.L2ContainerName, s.Id, s.PwdIpAddress)
	if err != nil {
		log.Println("ERROR NETWORKING")
		return err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)
	return nil
}
