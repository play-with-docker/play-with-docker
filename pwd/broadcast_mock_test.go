package pwd

import "net/http"

type mockBroadcast struct {
}

func (m *mockBroadcast) BroadcastTo(sessionId, eventName string, args ...interface{}) {
}
func (m *mockBroadcast) GetHandler() http.Handler {
	return nil
}
