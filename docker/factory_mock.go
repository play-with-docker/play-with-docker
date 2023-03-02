package docker

import (
	"github.com/stretchr/testify/mock"
	"github.com/thebsdbox/play-with-docker/pwd/types"
)

type FactoryMock struct {
	mock.Mock
}

func (m *FactoryMock) GetForSession(session *types.Session) (DockerApi, error) {
	args := m.Called(session)
	return args.Get(0).(DockerApi), args.Error(1)
}

func (m *FactoryMock) GetForInstance(instance *types.Instance) (DockerApi, error) {
	args := m.Called(instance)
	return args.Get(0).(DockerApi), args.Error(1)
}
