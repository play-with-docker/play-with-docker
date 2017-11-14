package storage

import (
	"errors"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

var NotFoundError = errors.New("NotFound")

func NotFound(e error) bool {
	return e == NotFoundError
}

type StorageApi interface {
	SessionGet(id string) (*types.Session, error)
	SessionGetAll() ([]*types.Session, error)
	SessionPut(session *types.Session) error
	SessionDelete(id string) error
	SessionCount() (int, error)

	InstanceGet(name string) (*types.Instance, error)
	InstancePut(instance *types.Instance) error
	InstanceDelete(name string) error
	InstanceCount() (int, error)
	InstanceFindBySessionId(sessionId string) ([]*types.Instance, error)

	WindowsInstanceGetAll() ([]*types.WindowsInstance, error)
	WindowsInstancePut(instance *types.WindowsInstance) error
	WindowsInstanceDelete(id string) error

	ClientGet(id string) (*types.Client, error)
	ClientPut(client *types.Client) error
	ClientDelete(id string) error
	ClientCount() (int, error)
	ClientFindBySessionId(sessionId string) ([]*types.Client, error)

	LoginRequestPut(loginRequest *types.LoginRequest) error
	LoginRequestGet(id string) (*types.LoginRequest, error)
	LoginRequestDelete(id string) error

	UserFindByProvider(providerName, providerUserId string) (*types.User, error)
	UserPut(user *types.User) error
	UserGet(id string) (*types.User, error)

	PlaygroundPut(playground *types.Playground) error
	PlaygroundGet(id string) (*types.Playground, error)
	PlaygroundGetAll() ([]*types.Playground, error)
}
