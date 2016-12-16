package services

import (
	"log"

	"github.com/googollee/go-socket.io"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	clientsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "clients",
		Help: "Clients",
	})
)

func init() {
	prometheus.MustRegister(clientsGauge)
}

type ViewPort struct {
	Rows uint
	Cols uint
}

type Client struct {
	Id       string
	so       socketio.Socket
	ViewPort ViewPort
}

func (c *Client) ResizeViewPort(cols, rows uint) {
	c.ViewPort.Rows = rows
	c.ViewPort.Cols = cols
}

func NewClient(so socketio.Socket, session *Session) *Client {
	clientsGauge.Inc()
	so.Join(session.Id)

	c := &Client{so: so, Id: so.Id()}

	so.On("session close", func() {
		CloseSession(session)
	})

	so.On("terminal in", func(name, data string) {
		// User wrote something on the terminal. Need to write it to the instance terminal
		instance := GetInstance(session, name)
		if instance != nil && instance.conn != nil && len(data) > 0 {
			instance.conn.Conn.Write([]byte(data))
		}
	})

	so.On("viewport resize", func(cols, rows uint) {
		// User resized his viewport
		c.ResizeViewPort(cols, rows)
		vp := session.GetSmallestViewPort()
		// Resize all terminals in the session
		wsServer.BroadcastTo(session.Id, "viewport resize", vp.Cols, vp.Rows)
		for _, instance := range session.Instances {
			err := instance.ResizeTerminal(vp.Cols, vp.Rows)
			if err != nil {
				log.Println("Error resizing terminal", err)
			}
		}
	})

	so.On("disconnection", func() {
		clientsGauge.Dec()
		// Client has disconnected. Remove from session and recheck terminal sizes.
		for i, cl := range session.clients {
			if cl.Id == c.Id {
				session.clients = append(session.clients[:i], session.clients[i+1:]...)
				break
			}
		}
		if len(session.clients) > 0 {
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
