package docker

import (
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/mock"
)

type FactoryMock struct {
	mock.Mock
}

func (m *FactoryMock) GetForSession(sessionId string) (DockerApi, error) {
	args := m.Called(sessionId)
	return args.Get(0).(DockerApi), args.Error(1)
}

func (m *FactoryMock) GetForInstance(instance *types.Instance) (DockerApi, error) {
	args := m.Called(instance)
	return args.Get(0).(DockerApi), args.Error(1)
}
