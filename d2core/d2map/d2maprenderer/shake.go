package d2maprenderer

import (
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

// screenShake is the state of the view shake (GFX_StartScreenShake 0x4727c0 /
// GFX_UpdateScreenShake 0x472a80): see d2monster.Shake for the profile.
type screenShake struct {
	shake d2monster.Shake
	ms    float64
	on    bool
}

// StartShake begins a screen shake; a shake with no hold time is ignored, like
// the exe does. A new shake replaces the running one.
func (mr *MapRenderer) StartShake(s d2monster.Shake) {
	if s.HoldMs == 0 {
		return
	}

	mr.shake = screenShake{shake: s, on: true}
}

// advance moves the shake clock by elapsed seconds.
func (s *screenShake) advance(elapsed float64) {
	if !s.on {
		return
	}

	s.ms += elapsed * 1000
	if !s.shake.Active(int(s.ms)) {
		s.on = false
	}
}

// offset is the pixel displacement for this frame: a random value in
// [-m, m] on each axis, m from the profile (zero when idle).
func (s *screenShake) offset(rnd func(n int) int) (dx, dy int) {
	if !s.on {
		return 0, 0
	}

	m := s.shake.Magnitude(int(s.ms))
	if m <= 0 {
		return 0, 0
	}

	return rnd(2*m+1) - m, rnd(2*m+1) - m
}

var shakeRand = rand.New(rand.NewSource(1)) //nolint:gosec // cosmetic
