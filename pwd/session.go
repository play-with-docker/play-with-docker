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

type Session struct {
	rw           sync.Mutex
	Id           string               `json:"id"`
	Instances    map[string]*Instance `json:"instances"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	PwdIpAddress string               `json:"pwd_ip_address"`
	Ready        bool                 `json:"ready"`
	Stack        string               `json:"stack"`
	StackName    string               `json:"stack_name"`
	closingTimer *time.Timer          `json:"-"`
	scheduled    bool                 `json:"-"`
	clients      []*Client            `json:"-"`
	ticker       *time.Ticker         `json:"-"`
}

func (p *pwd) SessionNew(duration time.Duration, stack, stackName string) (*Session, error) {
	s := &Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*Instance{}
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

	sessions[s.Id] = s
	if err := p.storage.Save(); err != nil {
		log.Println(err)
		return nil, err
	}

	setGauges()

	return s, nil
}

func (p *pwd) SessionClose(s *Session) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}
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
	delete(sessions, s.Id)

	// We store sessions as soon as we delete one
	if err := p.storage.Save(); err != nil {
		return err
	}
	setGauges()
	log.Printf("Cleaned up session [%s]\n", s.Id)
	return nil

}

func (p *pwd) SessionGetSmallestViewPort(s *Session) ViewPort {
	minRows := s.clients[0].viewPort.Rows
	minCols := s.clients[0].viewPort.Cols

	for _, c := range s.clients {
		minRows = uint(math.Min(float64(minRows), float64(c.viewPort.Rows)))
		minCols = uint(math.Min(float64(minCols), float64(c.viewPort.Cols)))
	}

	return ViewPort{Rows: minRows, Cols: minCols}
}

func (p *pwd) SessionDeployStack(s *Session) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.Ready {
		// a stack was already deployed on this session, just ignore
		return nil
	}

	s.Ready = false
	p.broadcast.BroadcastTo(s.Id, "session ready", false)
	i, err := p.InstanceNew(s, InstanceConfig{})
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
	if err := p.storage.Save(); err != nil {
		return err
	}
	return nil
}

func (p *pwd) SessionGet(sessionId string) *Session {
	s := sessions[sessionId]
	return s
}

func (p *pwd) SessionLoadAndPrepare() error {
	err := p.storage.Load()
	if err != nil {
		return err
	}

	for _, s := range sessions {
		err := p.prepareSession(s)
		if err != nil {
			return err
		}
		for _, i := range s.Instances {
			// wire the session back to the instance
			i.session = s
			go p.InstanceAttachTerminal(i)
		}
		// Connect PWD daemon to the new network
		if s.PwdIpAddress == "" {
			return fmt.Errorf("Cannot load stored sessions as they don't have the pwd ip address stored with them")
		}
	}

	setGauges()

	return nil
}

// This function should be called any time a session needs to be prepared:
// 1. Like when it is created
// 2. When it was loaded from storage
func (p *pwd) prepareSession(session *Session) error {
	p.scheduleSessionClose(session)

	// Connect PWD daemon to the new network
	if err := p.connectToNetwork(session); err != nil {
		return nil
	}

	// Schedule periodic tasks
	p.tasks.Schedule(session)

	return nil
}

func (p *pwd) scheduleSessionClose(s *Session) {
	timeLeft := s.ExpiresAt.Sub(time.Now())
	s.closingTimer = time.AfterFunc(timeLeft, func() {
		p.SessionClose(s)
	})
}

func (p *pwd) connectToNetwork(s *Session) error {
	ip, err := p.docker.ConnectNetwork(config.PWDContainerName, s.Id, s.PwdIpAddress)
	if err != nil {
		log.Println("ERROR NETWORKING")
		return err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)
	return nil
}
