package services

import (
	"log"
	"time"

	"github.com/franela/play-with-docker/types"
	"github.com/twinj/uuid"
)

var sessions map[string]*types.Session

func init() {
	sessions = make(map[string]*types.Session)
}

func NewSession() (*types.Session, error) {
	s := &types.Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*types.Instance{}

	//TODO: Store in something like redis
	sessions[s.Id] = s

	// Schedule cleanup of the session
	time.AfterFunc(1*time.Hour, func() {
		s = GetSession(s.Id)
		log.Printf("Starting clean up of session [%s]\n", s.Id)
		for _, i := range s.Instances {
			i.Conn.Close()
			if err := DeleteContainer(i.Name); err != nil {
				log.Println(err)
			}
		}
		if err := DeleteNetwork(s.Id); err != nil {
			log.Println(err)
		}
		delete(sessions, s.Id)
		log.Printf("Cleaned up session [%s]\n", s.Id)
	})

	if err := CreateNetwork(s.Id); err != nil {
		return nil, err
	}

	return s, nil
}

func GetSession(sessionId string) *types.Session {
	//TODO: Use redis
	s, found := sessions[sessionId]
	if found {
		s.Instances = instances[sessionId]
		return s
	} else {
		return nil
	}
}
