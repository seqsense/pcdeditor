package gl

type DeltaMode int

const (
	DOM_DELTA_PIXEL DeltaMode = 0x00
	DOM_DELTA_LINE  DeltaMode = 0x01
	DOM_DELTA_PAGE  DeltaMode = 0x02
)

type WheelEvent struct {
	MouseEvent
	DeltaX, DeltaY, DeltaZ float64
	DeltaMode              DeltaMode
}
