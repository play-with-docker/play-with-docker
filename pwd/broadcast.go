package pwd

import (
	"net/http"

	"github.com/googollee/go-socket.io"
)

type BroadcastApi interface {
	BroadcastTo(sessionId, eventName string, args ...interface{})
	GetHandler() http.Handler
}

type broadcast struct {
	sio *socketio.Server
}

func (b *broadcast) BroadcastTo(sessionId, eventName string, args ...interface{}) {
	b.sio.BroadcastTo(sessionId, eventName, args...)
}

func (b *broadcast) GetHandler() http.Handler {
	return b.sio
}

func NewBroadcast(connectionEvent, errorEvent interface{}) (*broadcast, error) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		return nil, err
	}
	server.On("connection", connectionEvent)
	server.On("error", errorEvent)
	return &broadcast{sio: server}, nil
}
