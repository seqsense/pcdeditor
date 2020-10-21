package main

type cursor string

const (
	cursorAuto      cursor = "auto"
	cursorDefault   cursor = "default"
	cursorPointer   cursor = "pointer"
	cursorCrosshair cursor = "crosshair"
	cursorMove      cursor = "move"
	cursorText      cursor = "text"
	cursorWait      cursor = "wait"
	cursorHelp      cursor = "help"
	cursorNResize   cursor = "n-resize"
	cursorSResize   cursor = "s-resize"
	cursorWResize   cursor = "w-resize"
	cursorEResize   cursor = "e-resize"
	cursorNEResize  cursor = "ne-resize"
	cursorNWResize  cursor = "nw-resize"
	cursorSEResize  cursor = "se-resize"
	cursorSWResize  cursor = "sw-resize"
	cursorProgress  cursor = "progress"
)

func (pe *pcdeditor) SetCursor(c cursor) {
	pe.canvas.Get("style").Set("cursor", string(c))
}
