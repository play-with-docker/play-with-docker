package storage

import (
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) SessionGet(sessionId string) (*types.Session, error) {
	args := m.Called(sessionId)
	return args.Get(0).(*types.Session), args.Error(1)
}

func (m *Mock) SessionPut(session *types.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *Mock) SessionCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *Mock) SessionDelete(sessionId string) error {
	args := m.Called(sessionId)
	return args.Error(0)
}

func (m *Mock) SessionGetAll() (map[string]*types.Session, error) {
	args := m.Called()
	return args.Get(0).(map[string]*types.Session), args.Error(1)
}

func (m *Mock) InstanceGet(sessionId, name string) (*types.Instance, error) {
	args := m.Called(sessionId, name)
	return args.Get(0).(*types.Instance), args.Error(1)
}

func (m *Mock) InstanceGetAllWindows() ([]*types.WindowsInstance, error) {
	args := m.Called()
	return args.Get(0).([]*types.WindowsInstance), args.Error(1)
}

func (m *Mock) InstanceFindByIP(sessionId, ip string) (*types.Instance, error) {
	args := m.Called(sessionId, ip)
	return args.Get(0).(*types.Instance), args.Error(1)
}

func (m *Mock) InstanceCreate(sessionId string, instance *types.Instance) error {
	args := m.Called(sessionId, instance)
	return args.Error(0)
}

func (m *Mock) InstanceCreateWindows(instance *types.WindowsInstance) error {
	args := m.Called(instance)
	return args.Error(0)
}

func (m *Mock) InstanceDelete(sessionId, instanceName string) error {
	args := m.Called(sessionId, instanceName)
	return args.Error(0)
}

func (m *Mock) InstanceDeleteWindows(sessionId, instanceId string) error {
	args := m.Called(sessionId, instanceId)
	return args.Error(0)
}

func (m *Mock) InstanceCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}
