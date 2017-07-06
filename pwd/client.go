package pwd

import (
	"log"
	"time"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func (p *pwd) ClientNew(id string, session *types.Session) *types.Client {
	defer observeAction("ClientNew", time.Now())
	c := &types.Client{Id: id, Session: session}
	session.Clients = append(session.Clients, c)
	return c
}

func (p *pwd) ClientResizeViewPort(c *types.Client, cols, rows uint) {
	defer observeAction("ClientResizeViewPort", time.Now())
	c.ViewPort.Rows = rows
	c.ViewPort.Cols = cols

	p.notifyClientSmallestViewPort(c.Session)
}

func (p *pwd) ClientClose(client *types.Client) {
	defer observeAction("ClientClose", time.Now())
	// Client has disconnected. Remove from session and recheck terminal sizes.
	session := client.Session
	for i, cl := range session.Clients {
		if cl.Id == client.Id {
			session.Clients = append(session.Clients[:i], session.Clients[i+1:]...)
			break
		}
	}
	if len(session.Clients) > 0 {
		p.notifyClientSmallestViewPort(session)
	}
	p.setGauges()
}

func (p *pwd) notifyClientSmallestViewPort(session *types.Session) {
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
