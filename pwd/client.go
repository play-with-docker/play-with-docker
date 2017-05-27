package pwd

import "log"

type Client struct {
	Id       string
	viewPort ViewPort
	session  *Session
}

type ViewPort struct {
	Rows uint
	Cols uint
}

func (p *pwd) ClientNew(id string, session *Session) *Client {
	c := &Client{Id: id, session: session}
	session.clients = append(session.clients, c)
	return c
}

func (p *pwd) ClientResizeViewPort(c *Client, cols, rows uint) {
	c.viewPort.Rows = rows
	c.viewPort.Cols = cols

	p.notifyClientSmallestViewPort(c.session)
}

func (p *pwd) ClientClose(client *Client) {
	// Client has disconnected. Remove from session and recheck terminal sizes.
	session := client.session
	for i, cl := range session.clients {
		if cl.Id == client.Id {
			session.clients = append(session.clients[:i], session.clients[i+1:]...)
			break
		}
	}
	if len(session.clients) > 0 {
		p.notifyClientSmallestViewPort(session)
	}
	setGauges()
}

func (p *pwd) notifyClientSmallestViewPort(session *Session) {
	vp := p.SessionGetSmallestViewPort(session)
	// Resize all terminals in the session
	p.broadcast.BroadcastTo(session.Id, "viewport resize", vp.Cols, vp.Rows)
	for _, instance := range session.Instances {
		err := p.InstanceResizeTerminal(instance, vp.Rows, vp.Cols)
		if err != nil {
			log.Println("Error resizing terminal", err)
		}
	}
}
