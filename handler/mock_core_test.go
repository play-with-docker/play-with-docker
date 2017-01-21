package handler

import "github.com/franela/play-with-docker/core"

type mockCore struct {
	deleteInstance         func(sessionId, instanceName string) error
	getSession             func(sessionId string) (*core.Session, error)
	getInstance            func(sessionId, instanceName string) (*core.Instance, error)
	newInstance            func(session *core.Session) (*core.Instance, error)
	newSession             func() (*core.Session, error)
	setInstanceCertificate func(sessionId, instanceName string, cert, key []byte) error
}

func (m *mockCore) DeleteInstance(sessionId, instanceName string) error {
	return m.deleteInstance(sessionId, instanceName)
}

func (m *mockCore) GetSession(sessionId string) (*core.Session, error) {
	return m.getSession(sessionId)
}

func (m *mockCore) GetInstance(sessionId, instanceName string) (*core.Instance, error) {
	return m.getInstance(sessionId, instanceName)
}

func (m *mockCore) NewInstance(s *core.Session) (*core.Instance, error) {
	return m.newInstance(s)
}

func (m *mockCore) NewSession() (*core.Session, error) {
	return m.newSession()
}

func (m *mockCore) SetInstanceCertificate(sessionId, instanceName string, cert, key []byte) error {
	return m.setInstanceCertificate(sessionId, instanceName, cert, key)
}
