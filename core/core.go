package core

type Core interface {
	DeleteInstance(sessionId, instanceName string) error
	GetSession(sessionId string) (*Session, error)
	NewInstance(session *Session) (*Instance, error)
	NewSession() (*Session, error)
}

func New() Core {
	return nil
}
