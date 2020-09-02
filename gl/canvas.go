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
