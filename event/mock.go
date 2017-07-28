package event

import "github.com/stretchr/testify/mock"

type Mock struct {
	M mock.Mock
}

func (m *Mock) Emit(name EventType, sessionId string, args ...interface{}) {
	m.M.Called(name, sessionId, args)
}

func (m *Mock) On(name EventType, handler Handler) {
	m.M.Called(name, handler)
}

func (m *Mock) OnAny(handler AnyHandler) {
	m.M.Called(handler)
}
