package services

import (
	"github.com/twinj/uuid"
	"github.com/xetorthio/play-with-docker/types"
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
