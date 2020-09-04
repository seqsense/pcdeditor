package gl

import (
	"syscall/js"
)

type MouseButton int

const (
	MouseButtonNull MouseButton = -1
)

type MouseEvent struct {
	UIEvent
	OffsetX, OffsetY int
	Button           MouseButton
	AltKey           bool
	CtrlKey          bool
	ShiftKey         bool
}

func parseMouseEvent(event js.Value) MouseEvent {
	b := MouseButtonNull
	button := event.Get("button")
	if !button.IsNull() {
		b = MouseButton(button.Int())
	}
	return MouseEvent{
		UIEvent: UIEvent{
			Event: Event{
				event: event,
			},
		},
		OffsetX:  event.Get("offsetX").Int(),
		OffsetY:  event.Get("offsetY").Int(),
		Button:   b,
		AltKey:   event.Get("altKey").Bool(),
		CtrlKey:  event.Get("ctrlKey").Bool(),
		ShiftKey: event.Get("shiftKey").Bool(),
	}
}
