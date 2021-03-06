package main

import (
	"testing"
	"time"
)

func TestWheelNormalizer(t *testing.T) {
	interval := 10 * time.Millisecond
	testCases := map[string]struct {
		pre       []float64
		input     []float64
		expected  []float64
		tolerance float64
	}{
		"BinaryWheel1": {
			pre:      []float64{1, 1, -1, 0, -1, -1},
			input:    []float64{1, -1, 0},
			expected: []float64{1, -1, 0},
		},
		"BinaryWheel10": {
			pre:      []float64{10, 10, -10, 0, -10, -10},
			input:    []float64{10, -10, 0},
			expected: []float64{1, -1, 0},
		},
		"AnalogWheel3": {
			pre:       []float64{2, 4, 3, 0, -1, 2},
			input:     []float64{3, -2, 0},
			expected:  []float64{3, -2, 0},
			tolerance: 1,
		},
		"AnalogWheel30": {
			pre:       []float64{20, 40, 30, 0, -10, 20},
			input:     []float64{30, -20, 0},
			expected:  []float64{3, -2, 0},
			tolerance: 1,
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			wn := &wheelNormalizer{}
			tick := time.NewTicker(interval)
			defer tick.Stop()

			for _, v := range tt.pre {
				<-tick.C
				wn.Normalize(v)
			}
			for i, v := range tt.input {
				<-tick.C
				o, ok := wn.Normalize(v)
				if !ok {
					t.Error("Normalizer should be ready")
					continue
				}
				if o < tt.expected[i]-tt.tolerance || tt.expected[i]+tt.tolerance < o {
					t.Errorf("Expected: %f, got: %f", tt.expected[i], o)
					continue
				}
			}
		})
	}
}
