package event

import "sync"

type localBroker struct {
	sync.Mutex

	handlers map[EventType][]Handler
}

func NewLocalBroker() *localBroker {
	return &localBroker{handlers: map[EventType][]Handler{}}
}

func (b *localBroker) On(name EventType, handler Handler) {
	b.Lock()
	defer b.Unlock()

	if b.handlers[name] == nil {
		b.handlers[name] = []Handler{}
	}
	b.handlers[name] = append(b.handlers[name], handler)
}

func (b *localBroker) Emit(name EventType, args ...interface{}) {
	b.Lock()
	defer b.Unlock()

	if b.handlers[name] != nil {
		for _, handler := range b.handlers[name] {
			handler(args...)
		}
	}
}
