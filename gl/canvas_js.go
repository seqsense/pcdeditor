package gl

import (
	"syscall/js"
)

type Canvas js.Value

func (c Canvas) Focus() {
	js.Value(c).Call("focus")
}

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
				MouseEvent: parseMouseEvent(event),
				DeltaX:     event.Get("deltaX").Float(),
				DeltaY:     event.Get("deltaY").Float(),
				DeltaZ:     event.Get("deltaZ").Float(),
				DeltaMode:  DeltaMode(event.Get("deltaMode").Int()),
			})
			return nil
		}),
	)
}

func (c Canvas) OnMouseMove(cb func(MouseEvent)) {
	c.onMouse("mousemove", cb)
}

func (c Canvas) OnMouseDown(cb func(MouseEvent)) {
	c.onMouse("mousedown", cb)
}

func (c Canvas) OnMouseUp(cb func(MouseEvent)) {
	c.onMouse("mouseup", cb)
}

func (c Canvas) OnClick(cb func(MouseEvent)) {
	c.onMouse("click", cb)
}

func (c Canvas) OnContextMenu(cb func(MouseEvent)) {
	c.onMouse("contextmenu", cb)
}

func (c Canvas) onMouse(name string, cb func(MouseEvent)) {
	js.Value(c).Call("addEventListener", name,
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			cb(parseMouseEvent(args[0]))
			return nil
		}),
	)
}

func (c Canvas) OnKeyDown(cb func(KeyboardEvent)) {
	c.onKey("keydown", cb)
}

func (c Canvas) OnKeyPress(cb func(KeyboardEvent)) {
	c.onKey("keypress", cb)
}

func (c Canvas) OnKeyUp(cb func(KeyboardEvent)) {
	c.onKey("keyup", cb)
}

func (c Canvas) onKey(name string, cb func(KeyboardEvent)) {
	js.Value(c).Call("addEventListener", name,
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			cb(parseKeyboardEvent(args[0]))
			return nil
		}),
	)
}

func (c Canvas) OnPointerMove(cb func(PointerEvent)) {
	c.onPointer("pointermove", cb)
}

func (c Canvas) OnPointerDown(cb func(PointerEvent)) {
	c.onPointer("pointerdown", cb)
}

func (c Canvas) OnPointerUp(cb func(PointerEvent)) {
	c.onPointer("pointerup", cb)
}

func (c Canvas) OnPointerOut(cb func(PointerEvent)) {
	c.onPointer("pointerout", cb)
}

func (c Canvas) onPointer(name string, cb func(PointerEvent)) {
	js.Value(c).Call("addEventListener", name,
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			cb(parsePointerEvent(args[0]))
			return nil
		}),
	)
}

func (c Canvas) OnWebGLContextLost(cb func(WebGLContextEvent)) {
	c.onWebGLContext("webglcontextlost", cb)
}

func (c Canvas) OnWebGLContextRestored(cb func(WebGLContextEvent)) {
	c.onWebGLContext("webglcontextrestored", cb)
}

func (c Canvas) onWebGLContext(name string, cb func(WebGLContextEvent)) {
	js.Value(c).Call("addEventListener", name,
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			cb(parseWebGLContextEvent(args[0]))
			return nil
		}),
	)
}
