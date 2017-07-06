package storage

import "github.com/play-with-docker/play-with-docker/pwd/types"

const notFound = "NotFound"

func NotFound(e error) bool {
	return e.Error() == notFound
}

type StorageApi interface {
	SessionGet(sessionId string) (*types.Session, error)
	SessionPut(*types.Session) error
	SessionCount() (int, error)
	SessionDelete(sessionId string) error

	InstanceFindByAlias(sessionPrefix, alias string) (*types.Instance, error)
	// Should have the session id too, soon
	InstanceFindByIP(ip string) (*types.Instance, error)
	InstanceFindByIPAndSession(sessionPrefix, ip string) (*types.Instance, error)
	InstanceCount() (int, error)

	ClientCount() (int, error)
}
