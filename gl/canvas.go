package gl

import (
	"syscall/js"
)

type Canvas js.Value

func (c Canvas) ClientWidth() int {
	return js.Value(c).Get("clientWidth").Int()
}

func (c Canvas) ClientHeight() int {
	return js.Value(c).Get("clientHeight").Int()
}

func (c Canvas) Width() int {
	return js.Value(c).Get("width").Int()
}

func (c Canvas) Height() int {
	return js.Value(c).Get("height").Int()
}

func (c Canvas) SetWidth(width int) {
	js.Value(c).Set("width", width)
}

func (c Canvas) SetHeight(height int) {
	js.Value(c).Set("height", height)
}

func (c Canvas) OnWheel(cb func(WheelEvent)) {
	js.Value(c).Call("addEventListener", "wheel",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := args[0]
			cb(WheelEvent{
				UIEvent: UIEvent{
					Event: Event{
						event: event,
					},
				},
				DeltaX:    event.Get("deltaX").Float(),
				DeltaY:    event.Get("deltaY").Float(),
				DeltaZ:    event.Get("deltaZ").Float(),
				DeltaMode: DeltaMode(event.Get("deltaMode").Int()),
			})
			return nil
		}),
	)
}

func (c Canvas) OnMouseMove(cb func(MouseEvent)) {
	c.onMouse("mousemove", cb)
}

func (c Canvas) OnClick(cb func(MouseEvent)) {
	c.onMouse("click", cb)
}

func (c Canvas) onMouse(name string, cb func(MouseEvent)) {
	js.Value(c).Call("addEventListener", name,
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			event := args[0]
			cb(MouseEvent{
				UIEvent: UIEvent{
					Event: Event{
						event: event,
					},
				},
				ClientX: event.Get("clientX").Int(),
				ClientY: event.Get("clientY").Int(),
			})
			return nil
		}),
	)
}
