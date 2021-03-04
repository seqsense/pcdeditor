package gl

import (
	"syscall/js"
)

type WebGLContextEvent struct {
	Event
	StatusMessage string
}

func parseWebGLContextEvent(event js.Value) WebGLContextEvent {
	return WebGLContextEvent{
		Event:         Event{event: event},
		StatusMessage: event.Get("statusMessage").String(),
	}
}
