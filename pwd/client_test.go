package pwd

import (
	"sync"
	"testing"
	"time"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/assert"
)

func TestClientNew(t *testing.T) {
	d := &mockDocker{}
	tasks := &mockTasks{}
	e := event.NewLocalBroker()
	storage := &mockStorage{}
	sp := &mockSessionProvider{docker: d}

	p := NewPWD(sp, tasks, e, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, err)

	client := p.ClientNew("foobar", session)

	assert.Equal(t, types.Client{Id: "foobar", Session: session, ViewPort: types.ViewPort{Cols: 0, Rows: 0}}, *client)
	assert.Contains(t, session.Clients, client)
}
func TestClientCount(t *testing.T) {
	d := &mockDocker{}
	tasks := &mockTasks{}
	e := event.NewLocalBroker()
	storage := &mockStorage{}
	sp := &mockSessionProvider{docker: d}

	p := NewPWD(sp, tasks, e, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, err)

	p.ClientNew("foobar", session)

	assert.Equal(t, 1, p.ClientCount())
}

func TestClientResizeViewPort(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	d := &mockDocker{}
	tasks := &mockTasks{}
	e := event.NewLocalBroker()
	sp := &mockSessionProvider{docker: d}

	broadcastedSessionId := ""
	broadcastedArgs := []interface{}{}

	e.On(event.INSTANCE_VIEWPORT_RESIZE, func(sessionId string, args ...interface{}) {
		broadcastedSessionId = sessionId
		broadcastedArgs = args
		wg.Done()
	})

	storage := &mockStorage{}

	p := NewPWD(sp, tasks, e, storage)

	session, err := p.SessionNew(time.Hour, "", "", "")
	assert.Nil(t, err)
	client := p.ClientNew("foobar", session)

	p.ClientResizeViewPort(client, 80, 24)
	wg.Wait()

	assert.Equal(t, types.ViewPort{Cols: 80, Rows: 24}, client.ViewPort)
	assert.Equal(t, session.Id, broadcastedSessionId)
	assert.Equal(t, uint(80), broadcastedArgs[0])
	assert.Equal(t, uint(24), broadcastedArgs[1])
}
