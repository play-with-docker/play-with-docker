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
	time.AfterFunc(1*time.Minute, func() {
		for _, i := range s.Instances {
			if err := DeleteContainer(i.Name); err != nil {
				log.Println(err)
			}
		}
		DeleteNetwork(s.Id)
	})

	if err := CreateNetwork(s.Id); err != nil {
		return nil, err
	}

	//TODO: Schedule deletion after an hour

	return s, nil
}

func GetSession(sessionId string) *types.Session {
	//TODO: Use redis
	s := sessions[sessionId]
	if instances[sessionId] != nil {
		s.Instances = instances[sessionId]
	}

	return s
}
