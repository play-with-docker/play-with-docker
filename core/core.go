package core

import "net/http"

type Core interface {
	DeleteInstance(sessionId, instanceName string) error
	GetSession(sessionId string) (*Session, error)
	GetInstance(sessionId, instanceName string) (*Instance, error)
	NewInstance(session *Session) (*Instance, error)
	NewSession() (*Session, error)
	SetInstanceCertificate(sessionId, instanceName string, cert, key []byte) error
	NewHTTPDirector() func(*http.Request)
	NewDockerDaemonDirector() func(*http.Request)
}

func New() Core {
	return nil
}
