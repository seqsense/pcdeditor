package main

import (
	"math"

	webgl "github.com/seqsense/pcdeditor/gl"
)

type gestureMode int

const (
	gestureNone gestureMode = iota
	gestureRotate
	gestureWheel
	gestureMove
)

type gesture struct {
	pointers map[int]webgl.PointerEvent
	pointer0 webgl.PointerEvent

	onMouseUp   func(webgl.MouseEvent)
	onMouseMove func(webgl.MouseEvent)
	onMouseDown func(webgl.MouseEvent)
	onWheel     func(e webgl.WheelEvent)

	mode      gestureMode
	distance0 float64
}

func (g *gesture) pointerUp(e webgl.PointerEvent) {
	e.PreventDefault()
	e.StopPropagation()

	if _, ok := g.pointers[e.PointerId]; ok {
		delete(g.pointers, e.PointerId)
	}
	if len(g.pointers) == 0 {
		if e.IsPrimary {
			g.pointer0 = e
		}
		switch g.mode {
		case gestureRotate:
			g.onMouseUp(g.pointer0.MouseEvent)
		case gestureMove:
			g.pointer0.MouseEvent.Button = 1
			g.onMouseUp(g.pointer0.MouseEvent)
		}
		g.mode = gestureNone
	}
}

func (g *gesture) pointerMove(e webgl.PointerEvent) {
	e.PreventDefault()
	e.StopPropagation()
	if _, ok := g.pointers[e.PointerId]; !ok {
		return
	}
	g.pointers[e.PointerId] = e

	if g.mode == gestureNone {
		switch len(g.pointers) {
		case 1:
			g.onMouseDown(g.pointer0.MouseEvent)
			g.mode = gestureRotate
		case 2:
			g.mode = gestureWheel
		case 3:
			g.pointer0.MouseEvent.Button = 1
			g.onMouseDown(g.pointer0.MouseEvent)
			g.mode = gestureMove
		}
	}
	switch g.mode {
	case gestureRotate:
		if e.IsPrimary {
			g.onMouseMove(e.MouseEvent)
		}
	case gestureWheel:
		if len(g.pointers) != 2 {
			break
		}
		var pp []webgl.PointerEvent
		for id := range g.pointers {
			pp = append(pp, g.pointers[id])
		}
		d := math.Hypot(float64(pp[0].OffsetX-pp[1].OffsetX), float64(pp[0].OffsetY-pp[1].OffsetY))
		we := webgl.WheelEvent{
			MouseEvent: webgl.MouseEvent{
				UIEvent: webgl.UIEvent{
					Event: webgl.NewEvent("WheelEvent"),
				},
				AltKey:   e.AltKey,
				CtrlKey:  e.CtrlKey,
				ShiftKey: e.ShiftKey,
			},
			DeltaY: (g.distance0 - d) / 10,
		}
		g.onWheel(we)
		g.distance0 = d
	case gestureMove:
		if e.IsPrimary {
			e.MouseEvent.Button = 1
			g.onMouseMove(e.MouseEvent)
		}
	}
	if e.IsPrimary {
		g.pointer0 = e
	}
}

func (g *gesture) pointerDown(e webgl.PointerEvent) {
	e.PreventDefault()
	e.StopPropagation()
	g.pointers[e.PointerId] = e

	switch len(g.pointers) {
	case 1:
		g.pointer0 = e
	case 2:
		var pp []webgl.PointerEvent
		for id := range g.pointers {
			pp = append(pp, g.pointers[id])
		}
		g.distance0 = math.Hypot(float64(pp[0].OffsetX-pp[1].OffsetX), float64(pp[0].OffsetY-pp[1].OffsetY))
	case 3:
	}
}
