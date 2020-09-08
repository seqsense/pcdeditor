package gl

import (
	"syscall/js"
)

type UIEvent struct {
	Event
}

type Event struct {
	event js.Value
}

func (e Event) PreventDefault() {
	e.event.Call("preventDefault")
}

func (e Event) StopPropagation() {
	e.event.Call("stopPropagation")
}

func NewEvent(typ string) Event {
	return Event{
		event: js.Global().Get("Event").New(typ),
	}
}
