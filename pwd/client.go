package pwd

import (
	"log"
	"time"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func (p *pwd) ClientNew(id string, session *types.Session) *types.Client {
	defer observeAction("ClientNew", time.Now())
	c := &types.Client{Id: id, SessionId: session.Id}
	if err := p.storage.ClientPut(c); err != nil {
		log.Println("Error saving client", err)
	}
	return c
}

func (p *pwd) ClientResizeViewPort(c *types.Client, cols, rows uint) {
	defer observeAction("ClientResizeViewPort", time.Now())
	c.ViewPort.Rows = rows
	c.ViewPort.Cols = cols

	if err := p.storage.ClientPut(c); err != nil {
		log.Println("Error saving client", err)
		return
	}
	p.notifyClientSmallestViewPort(c.SessionId)
}

func (p *pwd) ClientClose(client *types.Client) {
	defer observeAction("ClientClose", time.Now())
	// Client has disconnected. Remove from session and recheck terminal sizes.
	if err := p.storage.ClientDelete(client.Id); err != nil {
		log.Println("Error deleting client", err)
		return
	}
	p.notifyClientSmallestViewPort(client.SessionId)
}

func (p *pwd) ClientCount() int {
	count, err := p.storage.ClientCount()
	if err != nil {
		log.Println("Error counting clients", err)
		return 0
	}
	return count
}

func (p *pwd) notifyClientSmallestViewPort(sessionId string) {
	instances, err := p.storage.InstanceFindBySessionId(sessionId)
	if err != nil {
		log.Printf("Error finding instances for session [%s]. Got: %v\n", sessionId, err)
		return
	}

	vp := p.SessionGetSmallestViewPort(sessionId)
	// Resize all terminals in the session
	for _, instance := range instances {
		err := p.InstanceResizeTerminal(instance, vp.Rows, vp.Cols)
		if err != nil {
			log.Println("Error resizing terminal", err)
		}
	}
	p.event.Emit(event.INSTANCE_VIEWPORT_RESIZE, sessionId, vp.Cols, vp.Rows)
}
