package handlers

import (
	"io"

	"golang.org/x/net/context"
	"golang.org/x/net/websocket"

	"github.com/franela/play-with-docker/services"
	"github.com/go-zoo/bone"
)

// Echo the data received on the WebSocket.
func Exec(ws *websocket.Conn) {
	sessionId := bone.GetValue(ws.Request(), "sessionId")
	instanceId := bone.GetValue(ws.Request(), "instanceId")

	ctx := context.Background()

	session := services.GetSession(sessionId)
	instance := services.GetInstance(session, instanceId)

	if instance.ExecId == "" {
		execId, err := services.CreateExecConnection(instance.Name, ctx)
		if err != nil {
			return
		}
		instance.ExecId = execId
	}
	conn, err := services.AttachExecConnection(instance.ExecId, ctx)
	if err != nil {
		return
	}

	defer conn.Close()
	go func() {
		io.Copy(ws, conn.Reader)
	}()
	go func() {
		io.Copy(conn.Conn, ws)
	}()
	select {
	case <-ctx.Done():
	}
}
