package ebiten

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
)

// The video options Gamma and Contrast of the escape menu (11 steps, 5 is
// neutral). UNVERIFIED: the original changes the display palette through
// DirectDraw/D3D gamma ramps; here the finished frame is passed through a
// colour matrix. The matrix is linear, so gamma is the straight line through
// the origin and the gamma curve's value at one half (the exponent runs from
// 2 at step 0 to 0.5 at step 10, so higher steps are brighter); contrast
// scales the distance from mid grey from 0.5 to 1.5 times.
const (
	adjustSteps   = 10
	adjustNeutral = 5
)

// adjustMatrix returns the colour matrix of the options and whether it
// differs from the identity.
func adjustMatrix(gamma, contrast int) (m ebiten.ColorM, active bool) {
	if gamma == adjustNeutral && contrast == adjustNeutral {
		return m, false
	}

	exponent := math.Pow(2, float64(adjustNeutral-gamma)/float64(adjustNeutral)) // 2 .. 0.5
	gain := math.Pow(0.5, exponent-1)                                            // 0.5^(g-1): 0.5 .. 1.41
	k := 1 + float64(contrast-adjustNeutral)*0.1                                 // 0.5 .. 1.5

	m.Scale(gain, gain, gain, 1)
	m.Scale(k, k, k, 1)
	m.Translate(0.5*(1-k), 0.5*(1-k), 0.5*(1-k), 0)

	return m, true
}

// SetColorAdjust sets the gamma and contrast steps (0..10, 5 = unchanged).
func (r *Renderer) SetColorAdjust(gamma, contrast int) {
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}

		if v > adjustSteps {
			return adjustSteps
		}

		return v
	}

	r.gamma, r.contrast = clamp(gamma), clamp(contrast)
}

// ColorAdjust returns the steps set.
func (r *Renderer) ColorAdjust() (gamma, contrast int) { return r.gamma, r.contrast }

// AdjustColors passes everything drawn on the surface so far through the
// gamma / contrast matrix (nothing happens at the neutral steps).
func (r *Renderer) AdjustColors(target d2interface.Surface) {
	m, active := adjustMatrix(r.gamma, r.contrast)

	s, ok := target.(*ebitenSurface)
	if !active || !ok {
		return
	}

	w, h := s.image.Size()
	if r.adjustTmp == nil {
		r.adjustTmp = ebiten.NewImage(w, h)
	} else if tw, th := r.adjustTmp.Size(); tw != w || th != h {
		r.adjustTmp = ebiten.NewImage(w, h)
	}

	r.adjustTmp.Clear()
	r.adjustTmp.DrawImage(s.image, nil)
	s.image.Clear()
	s.image.DrawImage(r.adjustTmp, &ebiten.DrawImageOptions{ColorM: m})
}
