package sac

import (
	"github.com/seqsense/pcdeditor/mat"
)

type Sampler interface {
	Sample() int
}

type Model interface {
	NumRange() (min, max int)
	Fit([]int) (ModelCoefficients, bool)
}

type ModelCoefficients interface {
	Evaluate() int
	Inliers(float32) []int
	IsIn(mat.Vec3, float32) bool
}

type SAC struct {
	Sampler Sampler
	Model   Model

	bestCoeff ModelCoefficients
}

func New(s Sampler, m Model) *SAC {
	return &SAC{Sampler: s, Model: m}
}

func (s *SAC) Compute(n int) bool {
	var bestCoeff ModelCoefficients
	var bestE int

	num, _ := s.Model.NumRange()
	ids := make([]int, num)

	for i := 0; i < n; i++ {
		for j := 0; j < num; j++ {
			ids[j] = s.Sampler.Sample()
		}
		coeff, ok := s.Model.Fit(ids)
		if !ok {
			continue
		}
		e := coeff.Evaluate()
		if e > bestE {
			bestE = e
			bestCoeff = coeff
		}
	}
	if bestCoeff == nil {
		return false
	}
	s.bestCoeff = bestCoeff
	return true
}

func (s *SAC) Coefficients() ModelCoefficients {
	return s.bestCoeff
}
