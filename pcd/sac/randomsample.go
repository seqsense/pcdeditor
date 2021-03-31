package sac

import (
	"math/rand"
)

func NewRandomSampler(n int) Sampler {
	if n < 0x8000000 {
		return &randomSampler31{int32(n)}
	}
	return &randomSampler63{int64(n)}
}

type randomSampler31 struct {
	n int32
}

func (s *randomSampler31) Sample() int {
	return int(rand.Int31n(s.n))
}

type randomSampler63 struct {
	n int64
}

func (s *randomSampler63) Sample() int {
	return int(rand.Int63n(s.n))
}
