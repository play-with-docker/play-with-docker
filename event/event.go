package event

type EventType string

func (e EventType) String() string {
	return string(e)
}

const INSTANCE_VIEWPORT_RESIZE EventType = "instance viewport resize"
const INSTANCE_DELETE EventType = "instance delete"
const INSTANCE_NEW EventType = "instance new"
const INSTANCE_STATS EventType = "instance stats"
const INSTANCE_TERMINAL_OUT EventType = "instance terminal out"
const SESSION_END EventType = "session end"
const SESSION_READY EventType = "session ready"
const SESSION_BUILDER_OUT EventType = "session builder out"

type Handler func(sessionId string, args ...interface{})
type AnyHandler func(eventType EventType, sessionId string, args ...interface{})

type EventApi interface {
	Emit(name EventType, sessionId string, args ...interface{})
	On(name EventType, handler Handler)
	OnAny(handler AnyHandler)
}
