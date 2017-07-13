package event

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalBroker(t *testing.T) {
	broker := NewLocalBroker()

	called := 0
	receivedArgs := []interface{}{}

	wg := sync.WaitGroup{}
	wg.Add(1)

	broker.On(INSTANCE_NEW, func(args ...interface{}) {
		called++
		receivedArgs = args
		wg.Done()
	})
	broker.Emit(SESSION_READY)
	broker.Emit(INSTANCE_NEW, "foo", "bar")

	wg.Wait()

	assert.Equal(t, 1, called)
	assert.Equal(t, []interface{}{"foo", "bar"}, receivedArgs)
}
