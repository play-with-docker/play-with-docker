package event

type EventType string

const INSTANCE_VIEWPORT_RESIZE EventType = "instance viewport resize"
const INSTANCE_DELETE EventType = "instance delete"
const INSTANCE_NEW EventType = "instance new"
const SESSION_END EventType = "session end"
const SESSION_READY EventType = "session ready"

type Handler func(args ...interface{})

type EventApi interface {
	Emit(name EventType, args ...interface{})
	On(name EventType, handler Handler)
}
