package gl

import "fmt"

type Error struct {
	Context *WebGL
	Number  ErrorNumber
}

func (e *Error) Error() string {
	switch e.Number {
	case e.Context.INVALID_ENUM:
		return "invalid enum"
	case e.Context.INVALID_VALUE:
		return "invalid value"
	case e.Context.INVALID_OPERATION:
		return "invalid operation"
	case e.Context.INVALID_FRAMEBUFFER_OPERATION:
		return "invalid framebuffer operation"
	case e.Context.OUT_OF_MEMORY:
		return "out of memory"
	case e.Context.CONTEXT_LOST_WEBGL:
		return "context lost WebGL"
	default:
		return fmt.Sprintf("unknown %d", e.Number)
	}
}
