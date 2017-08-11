package docker

import "github.com/play-with-docker/play-with-docker/pwd/types"

type FactoryApi interface {
	GetForSession(sessionId string) (DockerApi, error)
	GetForInstance(instance *types.Instance) (DockerApi, error)
}
