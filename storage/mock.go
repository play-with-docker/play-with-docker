package storage

import (
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) SessionGet(id string) (*types.Session, error) {
	args := m.Called(id)
	return args.Get(0).(*types.Session), args.Error(1)
}
func (m *Mock) SessionGetAll() ([]*types.Session, error) {
	args := m.Called()
	return args.Get(0).([]*types.Session), args.Error(1)
}
func (m *Mock) SessionPut(session *types.Session) error {
	args := m.Called(session)
	return args.Error(0)
}
func (m *Mock) SessionDelete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *Mock) SessionCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}
func (m *Mock) InstanceGet(name string) (*types.Instance, error) {
	args := m.Called(name)
	return args.Get(0).(*types.Instance), args.Error(1)
}
func (m *Mock) InstancePut(instance *types.Instance) error {
	args := m.Called(instance)
	return args.Error(0)
}
func (m *Mock) InstanceDelete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}
func (m *Mock) InstanceCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}
func (m *Mock) InstanceFindBySessionId(sessionId string) ([]*types.Instance, error) {
	args := m.Called(sessionId)
	return args.Get(0).([]*types.Instance), args.Error(1)
}

func (m *Mock) WindowsInstanceGetAll() ([]*types.WindowsInstance, error) {
	args := m.Called()
	return args.Get(0).([]*types.WindowsInstance), args.Error(1)
}
func (m *Mock) WindowsInstancePut(instance *types.WindowsInstance) error {
	args := m.Called(instance)
	return args.Error(0)
}
func (m *Mock) WindowsInstanceDelete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *Mock) ClientGet(id string) (*types.Client, error) {
	args := m.Called(id)
	return args.Get(0).(*types.Client), args.Error(1)
}
func (m *Mock) ClientPut(client *types.Client) error {
	args := m.Called(client)
	return args.Error(0)
}
func (m *Mock) ClientDelete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *Mock) ClientCount() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}
func (m *Mock) ClientFindBySessionId(sessionId string) ([]*types.Client, error) {
	args := m.Called(sessionId)
	return args.Get(0).([]*types.Client), args.Error(1)
}
func (m *Mock) LoginRequestPut(loginRequest *types.LoginRequest) error {
	args := m.Called(loginRequest)
	return args.Error(0)
}
func (m *Mock) LoginRequestGet(id string) (*types.LoginRequest, error) {
	args := m.Called(id)
	return args.Get(0).(*types.LoginRequest), args.Error(1)
}
func (m *Mock) LoginRequestDelete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *Mock) UserFindByProvider(providerName, providerUserId string) (*types.User, error) {
	args := m.Called(providerName, providerUserId)
	return args.Get(0).(*types.User), args.Error(1)
}
func (m *Mock) UserPut(user *types.User) error {
	args := m.Called(user)
	return args.Error(0)
}
func (m *Mock) UserGet(id string) (*types.User, error) {
	args := m.Called(id)
	return args.Get(0).(*types.User), args.Error(1)
}
func (m *Mock) PlaygroundPut(playground *types.Playground) error {
	args := m.Called(playground)
	return args.Error(0)
}
func (m *Mock) PlaygroundGet(id string) (*types.Playground, error) {
	args := m.Called(id)
	return args.Get(0).(*types.Playground), args.Error(1)
}
func (m *Mock) PlaygroundGetAll() ([]*types.Playground, error) {
	args := m.Called()
	return args.Get(0).([]*types.Playground), args.Error(1)
}
