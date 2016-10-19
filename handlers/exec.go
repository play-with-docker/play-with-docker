package handlers

import (
	"io"

	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
	"golang.org/x/text/encoding"

	"github.com/franela/play-with-docker/cookoo"
	"github.com/franela/play-with-docker/services"
	"github.com/go-zoo/bone"
	"github.com/twinj/uuid"
)

// Echo the data received on the WebSocket.
func Exec(ws *websocket.Conn) {
	sessionId := bone.GetValue(ws.Request(), "sessionId")
	instanceName := bone.GetValue(ws.Request(), "instanceName")

	ctx := context.Background()

	session := services.GetSession(sessionId)
	instance := services.GetInstance(session, instanceName)

	if instance.Stdout == nil {
		id, err := services.CreateExecConnection(instance.Name, ctx)
		if err != nil {
			return
		}
		conn, err := services.AttachExecConnection(id, ctx)
		if err != nil {
			return
		}

		encoder := encoding.Replacement.NewEncoder()
		instance.Conn = conn
		instance.Stdout = &cookoo.MultiWriter{}
		instance.Stdout.Init()
		u1 := uuid.NewV4()
		instance.Stdout.AddWriter(u1.String(), ws)
		go func() {
			io.Copy(encoder.Writer(instance.Stdout), instance.Conn.Reader)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		go func() {
			io.Copy(instance.Conn.Conn, ws)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		select {
		case <-ctx.Done():
		}
	} else {
		u1 := uuid.NewV4()
		instance.Stdout.AddWriter(u1.String(), ws)

		go func() {
			io.Copy(instance.Conn.Conn, ws)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		select {
		case <-ctx.Done():
		}
	}

}
