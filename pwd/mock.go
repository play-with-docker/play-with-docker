package pwd

import (
	"context"
	"io"
	"net"

	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) SessionNew(ctx context.Context, config types.SessionConfig) (*types.Session, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*types.Session), args.Error(1)
}

func (m *Mock) SessionClose(session *types.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *Mock) SessionGetSmallestViewPort(sessionId string) types.ViewPort {
	args := m.Called(sessionId)
	return args.Get(0).(types.ViewPort)
}

func (m *Mock) SessionDeployStack(session *types.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *Mock) SessionGet(id string) (*types.Session, error) {
	args := m.Called(id)
	return args.Get(0).(*types.Session), args.Error(1)
}

func (m *Mock) SessionSetup(session *types.Session, conf SessionSetupConf) error {
	args := m.Called(session, conf)
	return args.Error(0)
}

func (m *Mock) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {
	args := m.Called(session, conf)
	return args.Get(0).(*types.Instance), args.Error(1)
}

func (m *Mock) InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error {
	args := m.Called(instance, cols, rows)
	return args.Error(0)
}

func (m *Mock) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	args := m.Called(instance)
	return args.Get(0).(net.Conn), args.Error(1)
}

func (m *Mock) InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error {
	args := m.Called(instance, fileName, dest, url)
	return args.Error(0)
}

func (m *Mock) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	args := m.Called(instance, fileName, dest, reader)
	return args.Error(0)
}

func (m *Mock) InstanceGet(session *types.Session, name string) *types.Instance {
	args := m.Called(session, name)
	return args.Get(0).(*types.Instance)
}
func (m *Mock) InstanceFindBySession(session *types.Session) ([]*types.Instance, error) {
	args := m.Called(session)
	return args.Get(0).([]*types.Instance), args.Error(1)
}

func (m *Mock) InstanceDelete(session *types.Session, instance *types.Instance) error {
	args := m.Called(session, instance)
	return args.Error(0)
}

func (m *Mock) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	args := m.Called(instance, cmd)
	return args.Int(0), args.Error(1)
}

func (m *Mock) InstanceFSTree(instance *types.Instance) (io.Reader, error) {
	args := m.Called(instance)
	return args.Get(0).(io.Reader), args.Error(1)
}

func (m *Mock) InstanceFile(instance *types.Instance, filePath string) (io.Reader, error) {
	args := m.Called(instance, filePath)
	return args.Get(0).(io.Reader), args.Error(1)
}

func (m *Mock) ClientNew(id string, session *types.Session) *types.Client {
	args := m.Called(id, session)
	return args.Get(0).(*types.Client)
}

func (m *Mock) ClientResizeViewPort(client *types.Client, cols, rows uint) {
	m.Called(client, cols, rows)
}

func (m *Mock) ClientClose(client *types.Client) {
	m.Called(client)
}

func (m *Mock) ClientCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *Mock) UserNewLoginRequest(providerName string) (*types.LoginRequest, error) {
	args := m.Called(providerName)
	return args.Get(0).(*types.LoginRequest), args.Error(1)
}

func (m *Mock) UserGetLoginRequest(id string) (*types.LoginRequest, error) {
	args := m.Called(id)
	return args.Get(0).(*types.LoginRequest), args.Error(1)
}

func (m *Mock) UserLogin(loginRequest *types.LoginRequest, user *types.User) (*types.User, error) {
	args := m.Called(loginRequest, user)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *Mock) UserGet(id string) (*types.User, error) {
	args := m.Called(id)
	return args.Get(0).(*types.User), args.Error(1)
}

func (m *Mock) PlaygroundNew(playground types.Playground) (*types.Playground, error) {
	args := m.Called(playground)
	return args.Get(0).(*types.Playground), args.Error(1)
}

func (m *Mock) PlaygroundGet(id string) *types.Playground {
	args := m.Called(id)
	return args.Get(0).(*types.Playground)
}

func (m *Mock) PlaygroundFindByDomain(domain string) *types.Playground {
	args := m.Called(domain)
	return args.Get(0).(*types.Playground)
}
func (m *Mock) PlaygroundList() ([]*types.Playground, error) {
	args := m.Called()
	return args.Get(0).([]*types.Playground), args.Error(1)
}
