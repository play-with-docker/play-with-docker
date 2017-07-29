package storage

import "github.com/play-with-docker/play-with-docker/pwd/types"

const notFound = "NotFound"

func NotFound(e error) bool {
	return e.Error() == notFound
}

type StorageApi interface {
	SessionGet(string) (*types.Session, error)
	SessionPut(*types.Session) error
	SessionCount() (int, error)
	SessionDelete(string) error
	SessionGetAll() (map[string]*types.Session, error)

	InstanceGet(sessionId, name string) (*types.Instance, error)
	InstanceFindByIP(session, ip string) (*types.Instance, error)
	InstanceCreate(sessionId string, instance *types.Instance) error
	InstanceDelete(sessionId, instanceName string) error
	InstanceCount() (int, error)
}
