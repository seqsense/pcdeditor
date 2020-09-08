package gl

import (
	"syscall/js"
)

type PointerEvent struct {
	MouseEvent
	PointerId int
	IsPrimary bool
}

func parsePointerEvent(event js.Value) PointerEvent {
	return PointerEvent{
		MouseEvent: parseMouseEvent(event),
		PointerId:  event.Get("pointerId").Int(),
		IsPrimary:  event.Get("isPrimary").Bool(),
	}
}
