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
