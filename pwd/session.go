package pwd

import (
	"fmt"
	"log"
	"math"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/twinj/uuid"
)

type sessionBuilderWriter struct {
	sessionId string
	broadcast BroadcastApi
}

func (s *sessionBuilderWriter) Write(p []byte) (n int, err error) {
	s.broadcast.BroadcastTo(s.sessionId, "session builder out", string(p))
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
	s.Id = uuid.NewV4().String()
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

	if err := p.docker.CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	if err := p.prepareSession(s); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := p.storage.SessionPut(s); err != nil {
		log.Println(err)
		return nil, err
	}

	p.setGauges()

	return s, nil
}

func (p *pwd) SessionClose(s *types.Session) error {
	defer observeAction("SessionClose", time.Now())

	s.Lock()
	defer s.Unlock()

	s.StopTicker()

	p.broadcast.BroadcastTo(s.Id, "session end")
	p.broadcast.BroadcastTo(s.Id, "disconnect")
	log.Printf("Starting clean up of session [%s]\n", s.Id)
	for _, i := range s.Instances {
		err := p.InstanceDelete(s, i)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	// Disconnect PWD daemon from the network
	if err := p.docker.DisconnectNetwork(config.PWDContainerName, s.Id); err != nil {
		if !strings.Contains(err.Error(), "is not connected to the network") {
			log.Println("ERROR NETWORKING")
			return err
		}
	}
	log.Printf("Disconnected pwd from network [%s]\n", s.Id)
	if err := p.docker.DeleteNetwork(s.Id); err != nil {
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
	p.broadcast.BroadcastTo(s.Id, "session ready", false)
	i, err := p.InstanceNew(s, InstanceConfig{ImageName: s.ImageName, Host: s.Host})
	if err != nil {
		log.Printf("Error creating instance for stack [%s]: %s\n", s.Stack, err)
		return err
	}
	err = p.InstanceUploadFromUrl(i, s.Stack)
	if err != nil {
		log.Printf("Error uploading stack file [%s]: %s\n", s.Stack, err)
		return err
	}

	fileName := path.Base(s.Stack)
	file := fmt.Sprintf("/var/run/pwd/uploads/%s", fileName)
	cmd := fmt.Sprintf("docker swarm init --advertise-addr eth0 && docker-compose -f %s pull && docker stack deploy -c %s %s", file, file, s.StackName)

	w := sessionBuilderWriter{sessionId: s.Id, broadcast: p.broadcast}
	code, err := p.docker.ExecAttach(i.Name, []string{"sh", "-c", cmd}, &w)
	if err != nil {
		log.Printf("Error executing stack [%s]: %s\n", s.Stack, err)
		return err
	}

	log.Printf("Stack execution finished with code %d\n", code)
	s.Ready = true
	p.broadcast.BroadcastTo(s.Id, "session ready", true)
	if err := p.storage.SessionPut(s); err != nil {
		return err
	}
	return nil
}

func (p *pwd) SessionGet(sessionId string) *types.Session {
	defer observeAction("SessionGet", time.Now())

	s, _ := p.storage.SessionGet(sessionId)

	if err := p.prepareSession(s); err != nil {
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
			instanceConf := InstanceConfig{
				ImageName: conf.Image,
				Hostname:  conf.Hostname,
				Host:      session.Host,
			}
			i, err := p.InstanceNew(session, instanceConf)
			if err != nil {
				return err
			}
			if i.Docker == nil {
				dock, err := p.docker.New(i.IP, i.Cert, i.Key)
				if err != nil {
					return err
				}
				i.Docker = dock
			}
			tkns, err := i.Docker.SwarmInit()
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
				instanceConf := InstanceConfig{
					ImageName: c.Image,
					Hostname:  c.Hostname,
				}
				i, err := p.InstanceNew(session, instanceConf)
				if err != nil {
					log.Println(err)
					return
				}
				if c.IsSwarmManager || c.IsSwarmWorker {
					// check if we have connection to the daemon, if not, create it
					if i.Docker == nil {
						dock, err := p.docker.New(i.IP, i.Cert, i.Key)
						if err != nil {
							log.Println(err)
							return
						}
						i.Docker = dock
					}
				}

				if firstSwarmManager != nil {
					if c.IsSwarmManager {
						// this is a swarm manager
						// cluster has already been initiated, join as manager
						err := i.Docker.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Manager)
						if err != nil {
							log.Println(err)
							return
						}
					}
					if c.IsSwarmWorker {
						// this is a swarm worker
						err := i.Docker.SwarmJoin(fmt.Sprintf("%s:2377", firstSwarmManager.IP), tokens.Worker)
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

// This function should be called any time a session needs to be prepared:
// 1. Like when it is created
// 2. When it was loaded from storage
func (p *pwd) prepareSession(session *types.Session) error {
	session.Lock()
	defer session.Unlock()

	if session.IsPrepared() {
		return nil
	}

	p.scheduleSessionClose(session)

	// Connect PWD daemon to the new network
	if err := p.connectToNetwork(session); err != nil {
		return err
	}

	// Schedule periodic tasks
	p.tasks.Schedule(session)

	for _, i := range session.Instances {
		// wire the session back to the instance
		i.Session = session
		go p.InstanceAttachTerminal(i)
	}
	session.SetPrepared()

	return nil
}

func (p *pwd) scheduleSessionClose(s *types.Session) {
	timeLeft := s.ExpiresAt.Sub(time.Now())
	s.SetClosingTimer(time.AfterFunc(timeLeft, func() {
		p.SessionClose(s)
	}))
}

func (p *pwd) connectToNetwork(s *types.Session) error {
	ip, err := p.docker.ConnectNetwork(config.PWDContainerName, s.Id, s.PwdIpAddress)
	if err != nil {
		log.Println("ERROR NETWORKING")
		return err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)
	return nil
}
