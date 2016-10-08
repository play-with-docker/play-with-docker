package services

import "github.com/xetorthio/play-with-docker/types"

var instances map[string]map[string]*types.Instance

func init() {
	instances = make(map[string]map[string]*types.Instance)
}

func NewInstance(session *types.Session) (*types.Instance, error) {
	//TODO: Validate that a session can only have 10 instances

	//TODO: Create in redis

	instance, err := CreateInstance(session.Id)

	if err != nil {
		return nil, err
	}

	if instances[session.Id] == nil {
		instances[session.Id] = make(map[string]*types.Instance)
	}
	instances[session.Id][instance.Name] = instance

	return instance, nil
}

func GetInstance(session *types.Session, instanceId string) *types.Instance {
	//TODO: Use redis
	i := instances[session.Id][instanceId]
	return i
}
func DeleteInstance(session *types.Session, instance *types.Instance) error {
	//TODO: Use redis
	delete(instances[session.Id], instance.Name)
	return DeleteContainer(instance.Name)
}
