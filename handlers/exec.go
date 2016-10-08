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
	id := bone.GetValue(ws.Request(), "id")
	ctx := context.Background()
	conn, err := services.GetExecConnection(id, ctx)
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
	//io.Copy(ws, os.Stdout)
	//go func() {
	//io.Copy(*conn, ws)
	//}()
}
