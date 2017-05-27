package pwd

import "net/http"

type mockBroadcast struct {
	broadcastTo func(sessionId, eventName string, args ...interface{})
	getHandler  func() http.Handler
}

func (m *mockBroadcast) BroadcastTo(sessionId, eventName string, args ...interface{}) {
	if m.broadcastTo != nil {
		m.broadcastTo(sessionId, eventName, args...)
	}
}
func (m *mockBroadcast) GetHandler() http.Handler {
	if m.getHandler != nil {
		return m.getHandler()
	}
	return nil
}
