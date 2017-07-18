package event

import "sync"

type localBroker struct {
	sync.Mutex

	handlers    map[EventType][]Handler
	anyHandlers []AnyHandler
}

func NewLocalBroker() *localBroker {
	return &localBroker{handlers: map[EventType][]Handler{}, anyHandlers: []AnyHandler{}}
}

func (b *localBroker) On(name EventType, handler Handler) {
	b.Lock()
	defer b.Unlock()

	if b.handlers[name] == nil {
		b.handlers[name] = []Handler{}
	}
	b.handlers[name] = append(b.handlers[name], handler)
}

func (b *localBroker) OnAny(handler AnyHandler) {
	b.Lock()
	defer b.Unlock()

	b.anyHandlers = append(b.anyHandlers, handler)
}

func (b *localBroker) Emit(name EventType, sessionId string, args ...interface{}) {
	go func() {
		b.Lock()
		defer b.Unlock()

		for _, handler := range b.anyHandlers {
			handler(name, sessionId, args...)
		}
		if b.handlers[name] != nil {
			for _, handler := range b.handlers[name] {
				handler(sessionId, args...)
			}
		}
	}()
}
