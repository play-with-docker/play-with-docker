package event

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalBroker_On(t *testing.T) {
	broker := NewLocalBroker()

	called := 0
	receivedSessionId := ""
	receivedArgs := []interface{}{}

	wg := sync.WaitGroup{}
	wg.Add(1)

	broker.On(INSTANCE_NEW, func(sessionId string, args ...interface{}) {
		called++
		receivedSessionId = sessionId
		receivedArgs = args
		wg.Done()
	})
	broker.Emit(SESSION_READY, "1")
	broker.Emit(INSTANCE_NEW, "2", "foo", "bar")

	wg.Wait()

	assert.Equal(t, 1, called)
	assert.Equal(t, "2", receivedSessionId)
	assert.Equal(t, []interface{}{"foo", "bar"}, receivedArgs)
}

func TestLocalBroker_OnAny(t *testing.T) {
	broker := NewLocalBroker()

	var receivedEvent EventType
	receivedSessionId := ""
	receivedArgs := []interface{}{}

	wg := sync.WaitGroup{}
	wg.Add(1)

	broker.OnAny(func(eventType EventType, sessionId string, args ...interface{}) {
		receivedSessionId = sessionId
		receivedArgs = args
		receivedEvent = eventType
		wg.Done()
	})
	broker.Emit(SESSION_READY, "1")

	wg.Wait()

	var expectedArgs []interface{}
	assert.Equal(t, SESSION_READY, receivedEvent)
	assert.Equal(t, "1", receivedSessionId)
	assert.Equal(t, expectedArgs, receivedArgs)
}
