package provider

import "github.com/play-with-docker/play-with-docker/docker"

type InstanceProvider interface {
}

type SessionProvider interface {
	GetDocker(sessionId string) (docker.DockerApi, error)
}
