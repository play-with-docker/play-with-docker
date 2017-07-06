package pwd

import "github.com/play-with-docker/play-with-docker/pwd/types"

type mockStorage struct {
	sessionGet                 func(sessionId string) (*types.Session, error)
	sessionPut                 func(s *types.Session) error
	sessionCount               func() (int, error)
	sessionDelete              func(sessionId string) error
	instanceFindByAlias        func(sessionPrefix, alias string) (*types.Instance, error)
	instanceFindByIP           func(ip string) (*types.Instance, error)
	instanceFindByIPAndSession func(sessionPrefix, ip string) (*types.Instance, error)
	instanceCount              func() (int, error)
	clientCount                func() (int, error)
}

func (m *mockStorage) SessionGet(sessionId string) (*types.Session, error) {
	if m.sessionGet != nil {
		return m.sessionGet(sessionId)
	}
	return nil, nil
}
func (m *mockStorage) SessionPut(s *types.Session) error {
	if m.sessionPut != nil {
		return m.sessionPut(s)
	}
	return nil
}
func (m *mockStorage) SessionCount() (int, error) {
	if m.sessionCount != nil {
		return m.sessionCount()
	}
	return 0, nil
}
func (m *mockStorage) SessionDelete(sessionId string) error {
	if m.sessionDelete != nil {
		return m.sessionDelete(sessionId)
	}
	return nil
}
func (m *mockStorage) InstanceFindByAlias(sessionPrefix, alias string) (*types.Instance, error) {
	if m.instanceFindByAlias != nil {
		return m.instanceFindByAlias(sessionPrefix, alias)
	}
	return nil, nil
}
func (m *mockStorage) InstanceFindByIP(ip string) (*types.Instance, error) {
	if m.instanceFindByIP != nil {
		return m.instanceFindByIP(ip)
	}
	return nil, nil
}
func (m *mockStorage) InstanceFindByIPAndSession(sessionPrefix, ip string) (*types.Instance, error) {
	if m.instanceFindByIPAndSession != nil {
		return m.instanceFindByIPAndSession(sessionPrefix, ip)
	}
	return nil, nil
}
func (m *mockStorage) InstanceCount() (int, error) {
	if m.instanceCount != nil {
		return m.instanceCount()
	}
	return 0, nil
}
func (m *mockStorage) ClientCount() (int, error) {
	if m.clientCount != nil {
		return m.clientCount()
	}
	return 0, nil
}
