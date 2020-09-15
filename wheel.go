package main

import (
	"time"
)

const (
	binaryDetectCnt = 4
	initialMaxDelta = 10
)

type wheelType int

const (
	wheelTypeNone wheelType = iota
	wheelTypeBinary
	wheelTypeContinuous
)

type wheelNormalizer struct {
	init     bool
	eventCnt int

	wheelType wheelType
	maxDelta  float64

	binaryCnt int
	binaryAbs float64

	timePrev time.Time
	dSum     float64
}

func (n *wheelNormalizer) Normalize(d float64) (float64, bool) {
	if n.eventCnt > binaryDetectCnt {
		n.init = true
	} else {
		n.eventCnt++
	}

	dAbs := d
	if dAbs < 0 {
		dAbs = -d
	}
	if dAbs == 0 {
		return 0, n.init
	}

	if n.binaryAbs == dAbs {
		n.binaryCnt++
	} else {
		n.binaryCnt = 0
	}
	n.binaryAbs = dAbs

	typePrev := n.wheelType
	if n.binaryCnt > binaryDetectCnt {
		n.wheelType = wheelTypeBinary
		if n.binaryAbs != dAbs {
			n.maxDelta = initialMaxDelta
		}
	} else {
		n.wheelType = wheelTypeContinuous
	}

	if n.wheelType != typePrev {
		n.maxDelta = initialMaxDelta
	}

	now := time.Now()
	dt := now.Sub(n.timePrev).Seconds()
	if dt > 0 {
		if dt > 0.1 {
			dt = 0.1
		}

		n.dSum += d
		dps := n.dSum / dt
		n.dSum = 0
		n.timePrev = now

		dpsAbs := dps
		if dpsAbs < 0 {
			dpsAbs = -dps
		}

		if n.maxDelta < dpsAbs {
			// LPF to suppress spikes
			n.maxDelta = n.maxDelta*0.5 + dpsAbs*0.5
		}
		n.maxDelta *= 0.95
	} else {
		n.dSum += d
	}

	if n.maxDelta < 1 {
		n.maxDelta = 1
	}
	if n.wheelType == wheelTypeBinary {
		if d < 0 {
			return -1, n.init
		}
		return 1, n.init
	}
	return d * 250 / n.maxDelta, n.init
}
