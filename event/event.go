package event

var events []string

type EventType int

func (e EventType) String() string {
	return events[int(e)]
}

func ciota(s string) EventType {
	events = append(events, s)
	return EventType(len(events) - 1)
}

func FindEventType(name string) (EventType, bool) {
	for i, event := range events {
		if event == name {
			return EventType(i), true
		}
	}
	return EventType(-1), false
}

var (
	INSTANCE_VIEWPORT_RESIZE = ciota("instance viewport resize")
	INSTANCE_DELETE          = ciota("instance delete")
	INSTANCE_NEW             = ciota("instance new")
	INSTANCE_STATS           = ciota("instance stats")
	INSTANCE_TERMINAL_OUT    = ciota("instance terminal out")
	SESSION_END              = ciota("session end")
	SESSION_READY            = ciota("session ready")
	SESSION_BUILDER_OUT      = ciota("session builder out")
)

type Handler func(sessionId string, args ...interface{})
type AnyHandler func(eventType EventType, sessionId string, args ...interface{})

type EventApi interface {
	Emit(name EventType, sessionId string, args ...interface{})
	On(name EventType, handler Handler)
	OnAny(handler AnyHandler)
}
