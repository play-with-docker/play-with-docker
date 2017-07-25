package provider

import (
	"sync"

	"github.com/docker/docker/client"
	"github.com/play-with-docker/play-with-docker/docker"
)

type localSessionProvider struct {
	rw sync.Mutex

	docker docker.DockerApi
}

func (p *localSessionProvider) GetDocker(sessionId string) (docker.DockerApi, error) {
	p.rw.Lock()
	defer p.rw.Unlock()

	if p.docker != nil {
		return p.docker, nil
	}

	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	d := docker.NewDocker(c)

	p.docker = d
	return d, nil
}

func NewLocalSessionProvider() *localSessionProvider {
	return &localSessionProvider{}
}
