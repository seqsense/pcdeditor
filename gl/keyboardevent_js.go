package gl

import (
	"syscall/js"
)

type KeyboardEvent struct {
	UIEvent
	Code     string
	Key      string
	AltKey   bool
	CtrlKey  bool
	ShiftKey bool
}

func parseKeyboardEvent(event js.Value) KeyboardEvent {
	return KeyboardEvent{
		UIEvent: UIEvent{
			Event: Event{
				event: event,
			},
		},
		Code:     event.Get("code").String(),
		Key:      event.Get("key").String(),
		AltKey:   event.Get("altKey").Bool(),
		CtrlKey:  event.Get("ctrlKey").Bool(),
		ShiftKey: event.Get("shiftKey").Bool(),
	}
}
