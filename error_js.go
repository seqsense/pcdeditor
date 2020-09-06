package main

import (
	"syscall/js"
)

func errorToJS(err error) js.Value {
	return js.Global().Get("Error").New(err.Error())
}
