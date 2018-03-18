package event

type EventType string

func (e EventType) String() string {
	return string(e)
}

var (
	INSTANCE_VIEWPORT_RESIZE = EventType("instance viewport resize")
	INSTANCE_DELETE          = EventType("instance delete")
	INSTANCE_NEW             = EventType("instance new")
	INSTANCE_STATS           = EventType("instance stats")
	SESSION_NEW              = EventType("session new")
	SESSION_END              = EventType("session end")
	SESSION_READY            = EventType("session ready")
	SESSION_BUILDER_OUT      = EventType("session builder out")
	PLAYGROUND_NEW           = EventType("playground_new")
)

type Handler func(id string, args ...interface{})
type AnyHandler func(eventType EventType, id string, args ...interface{})

type EventApi interface {
	Emit(name EventType, id string, args ...interface{})
	On(name EventType, handler Handler)
	OnAny(handler AnyHandler)
}
