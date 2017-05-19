package pwd

import (
	"log"
	"time"

	"github.com/franela/play-with-docker.old/config"
	"github.com/twinj/uuid"
)

func (p *pwd) NewSession(duration time.Duration, stack, stackName string) (*Session, error) {
	s := &Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*Instance{}
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(duration)
	/*
		if stack == "" {
			s.Ready = true
		}
		s.Stack = stack
	*/
	log.Printf("NewSession id=[%s]\n", s.Id)

	if err := p.docker.CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	s.Prepare()

	return s, nil
}

// This function should be called any time a session needs to be prepared:
// 1. Like when it is created
// 2. When it was loaded from storage
func (s *Session) Prepare() error {
	s.scheduleSessionClose()

	// Connect PWD daemon to the new network
	s.connectToNetwork()

	return nil
}

func (s *Session) scheduleSessionClose() {
	timeLeft := s.ExpiresAt.Sub(time.Now())
	s.closingTimer = time.AfterFunc(timeLeft, func() {
		s.Close()
	})
}

func (s *Session) Close() {
}

func (s *Session) connectToNetwork() {
	ip, err := ConnectNetwork(config.PWDContainerName, s.Id, "")
	if err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)
}
