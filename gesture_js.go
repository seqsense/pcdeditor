package main

import (
	"context"
	"math"
	"time"

	webgl "github.com/seqsense/webgl-go"
)

const (
	doubleTapInterval = 250 * time.Millisecond
	tapMaxEndDuration = 150 * time.Millisecond
)

type gestureMode int

const (
	gestureNone gestureMode = iota
	gestureRotate
	gestureWheel
	gestureDrag
	gestureMove
)

type gesture struct {
	pointer0  webgl.TouchEvent
	primaryID int

	onClick     func(webgl.MouseEvent)
	onMouseUp   func(webgl.MouseEvent)
	onMouseDrag func(webgl.MouseEvent)
	onMouseDown func(webgl.MouseEvent)
	onWheel     func(e webgl.WheelEvent)

	mode      gestureMode
	distance0 float64

	lastStart   time.Time
	tapCnt      int
	clickCancel func()
}

func (g *gesture) fromLastEnd(now time.Time) time.Duration {
	return now.Sub(g.lastStart)
}

func (g *gesture) touchEnd(e webgl.TouchEvent) {
	e.PreventDefault()
	e.StopPropagation()

	now := time.Now()

	switch g.mode {
	case gestureNone:
		if g.fromLastEnd(now) < tapMaxEndDuration {
			ctx, cancel := context.WithCancel(context.Background())
			g.clickCancel = cancel
			go func() {
				defer cancel()
				select {
				case <-time.After(doubleTapInterval):
					g.onClick(touchToMouse(g.pointer0, 0))
				case <-ctx.Done():
				}
			}()
		}
	case gestureRotate:
		g.onMouseUp(touchToMouse(g.pointer0, 0))
	case gestureDrag:
		g.onMouseUp(touchToMouse(g.pointer0, 1))
	}
	g.mode = gestureNone
}

func (g *gesture) touchMove(e webgl.TouchEvent) {
	e.PreventDefault()
	e.StopPropagation()

	now := time.Now()

	if g.mode == gestureNone {
		n := len(e.Touches)
		if g.tapCnt == 1 {
			n = 3
		}
		switch n {
		case 1:
			if g.fromLastEnd(now) > doubleTapInterval {
				g.onMouseDown(touchToMouse(g.pointer0, 0))
				g.mode = gestureRotate
			}
		case 2:
			g.mode = gestureWheel
		case 3:
			g.onMouseDown(touchToMouse(g.pointer0, 1))
			g.mode = gestureDrag
		}
	}
	switch g.mode {
	case gestureRotate:
		g.onMouseDrag(touchToMouse(e, 0))
	case gestureWheel:
		if len(e.Touches) != 2 {
			break
		}
		d := math.Hypot(
			float64(e.Touches[0].ClientX-e.Touches[1].ClientX),
			float64(e.Touches[0].ClientY-e.Touches[1].ClientY),
		)
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
	case gestureDrag:
		g.onMouseDrag(touchToMouse(e, 1))
	}
	if len(e.Touches) > 0 {
		g.pointer0 = e
	}
}

func (g *gesture) touchStart(e webgl.TouchEvent) {
	e.PreventDefault()
	e.StopPropagation()

	now := time.Now()

	if g.clickCancel != nil {
		g.clickCancel()
		g.clickCancel = nil
	}

	switch len(e.Touches) {
	case 1:
		if g.fromLastEnd(now) < doubleTapInterval {
			g.tapCnt++
		} else {
			g.tapCnt = 0
		}
		g.lastStart = now
	case 2:
		g.distance0 = math.Hypot(
			float64(e.Touches[0].ClientX-e.Touches[1].ClientX),
			float64(e.Touches[0].ClientY-e.Touches[1].ClientY),
		)
	case 3:
	}
}

func touchToMouse(e webgl.TouchEvent, button webgl.MouseButton) webgl.MouseEvent {
	return webgl.MouseEvent{
		UIEvent:  e.UIEvent,
		OffsetX:  e.Touches[0].ClientX,
		OffsetY:  e.Touches[0].ClientY,
		Button:   button,
		AltKey:   e.AltKey,
		CtrlKey:  e.CtrlKey,
		ShiftKey: e.ShiftKey,
	}
}
