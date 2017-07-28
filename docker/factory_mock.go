package docker

import "github.com/stretchr/testify/mock"

type FactoryMock struct {
	mock.Mock
}

func (m *FactoryMock) GetForSession(sessionId string) (DockerApi, error) {
	args := m.Called(sessionId)
	return args.Get(0).(DockerApi), args.Error(1)
}

func (m *FactoryMock) GetForInstance(sessionId, instanceName string) (DockerApi, error) {
	args := m.Called(sessionId, instanceName)
	return args.Get(0).(DockerApi), args.Error(1)
}
