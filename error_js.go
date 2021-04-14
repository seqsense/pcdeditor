package main

import (
	"errors"
	"syscall/js"
)

var errContextLostEvent = errors.New("received context lost event")

func errorToJS(err error) js.Value {
	return js.Global().Get("Error").New(err.Error())
}
