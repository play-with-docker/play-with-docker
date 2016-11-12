package services

import "github.com/googollee/go-socket.io"

type ViewPort struct {
	Rows uint
	Cols uint
}

type Client struct {
	SO       socketio.Socket
	ViewPort ViewPort
}

func (c *Client) ResizeViewPort(cols, rows uint) {
	c.ViewPort.Rows = rows
	c.ViewPort.Cols = cols
}

func NewClient(so socketio.Socket, session *Session) *Client {
	so.Join(session.Id)

	c := &Client{SO: so}

	so.On("session close", func() {
		CloseSession(session)
	})

	so.On("terminal in", func(name, data string) {
		// User wrote something on the terminal. Need to write it to the instance terminal
		instance := GetInstance(session, name)
		if instance != nil && len(data) > 0 {
			instance.Conn.Conn.Write([]byte(data))
		}
	})

	so.On("viewport resize", func(cols, rows uint) {
		// User resized his viewport
		c.ResizeViewPort(cols, rows)
		vp := session.GetSmallestViewPort()
		// Resize all terminals in the session
		wsServer.BroadcastTo(session.Id, "viewport resize", vp.Cols, vp.Rows)
		for _, instance := range session.Instances {
			instance.ResizeTerminal(vp.Cols, vp.Rows)
		}
	})
	so.On("disconnection", func() {
		// Client has disconnected. Remove from session and recheck terminal sizes.
		for i, cl := range session.Clients {
			if cl.SO.Id() == c.SO.Id() {
				session.Clients = append(session.Clients[:i], session.Clients[i+1:]...)
				break
			}
		}
		if len(session.Clients) > 0 {
			vp := session.GetSmallestViewPort()
			// Resize all terminals in the session
			wsServer.BroadcastTo(session.Id, "viewport resize", vp.Cols, vp.Rows)
			for _, instance := range session.Instances {
				instance.ResizeTerminal(vp.Cols, vp.Rows)
			}
		}
	})

	return c
}
