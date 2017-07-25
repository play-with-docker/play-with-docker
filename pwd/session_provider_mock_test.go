package pwd

import "github.com/play-with-docker/play-with-docker/docker"

type mockSessionProvider struct {
	docker docker.DockerApi
}

func (p *mockSessionProvider) GetDocker(sessionId string) (docker.DockerApi, error) {
	return p.docker, nil
}
