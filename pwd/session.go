package pwd

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
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

	dockerClient, err := p.dockerFactory.GetForSession(s.Id)
	if err != nil {
		// We assume we are out of capacity
		return nil, fmt.Errorf("Out of capacity")
	}
	u, _ := url.Parse(dockerClient.GetDaemonHost())
	if u.Host == "" {
		s.Host = "localhost"
	} else {
		chunks := strings.Split(u.Host, ":")
		s.Host = chunks[0]
	}

	if err := dockerClient.CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING", err)
		return nil, err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	ip, err := dockerClient.ConnectNetwork(config.L2ContainerName, s.Id, s.PwdIpAddress)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)

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

	updatedSession, err := p.storage.SessionGet(s.Id)
	if err != nil {
		if storage.NotFound(err) {
			log.Printf("Session with id [%s] was not found in storage.\n", s.Id)
			return err
		} else {
			log.Printf("Couldn't close session. Got: %s\n", err)
			return err
		}
	}
	s = updatedSession

	log.Printf("Starting clean up of session [%s]\n", s.Id)
	g, _ := errgroup.WithContext(context.Background())
	for _, i := range s.Instances {
		i := i
		g.Go(func() error {
			return p.InstanceDelete(s, i)
		})
	}
	err = g.Wait()
	if err != nil {
		log.Println(err)
		return err
	}

	// Disconnect PWD daemon from the network
	dockerClient, err := p.dockerFactory.GetForSession(s.Id)
	if err != nil {
		log.Println(err)
		return err
	}
	if err := dockerClient.DisconnectNetwork(config.L2ContainerName, s.Id); err != nil {
		if !strings.Contains(err.Error(), "is not connected to the network") {
			log.Println("ERROR NETWORKING", err)
			return err
		}
	}
	log.Printf("Disconnected l2 from network [%s]\n", s.Id)
	if err := dockerClient.DeleteNetwork(s.Id); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			log.Println(err)
			return err
		}
	}

	err = p.storage.SessionDelete(s.Id)
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

	dockerClient, err := p.dockerFactory.GetForSession(s.Id)
	if err != nil {
		log.Println(err)
		return err
	}

	code, err := dockerClient.ExecAttach(i.Name, []string{"sh", "-c", cmd}, &w)
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
