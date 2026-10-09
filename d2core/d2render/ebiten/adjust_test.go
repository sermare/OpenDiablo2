package ebiten

import (
	"image/color"
	"testing"
)

func TestAdjustMatrix(t *testing.T) {
	if _, active := adjustMatrix(5, 5); active {
		t.Fatal("neutral steps must not adjust")
	}

	lum := func(gamma, contrast int, c float64) float64 {
		m, _ := adjustMatrix(gamma, contrast)
		v := uint8(c * 255)
		r, _, _, _ := m.Apply(color.RGBA{R: v, G: v, B: v, A: 255}).RGBA()

		return float64(r) / 65535
	}

	if lum(0, 5, 0.5) >= lum(5, 5, 0.5) || lum(10, 5, 0.5) <= 0.5 {
		t.Errorf("gamma: step 0 %.3f step 10 %.3f", lum(0, 5, 0.5), lum(10, 5, 0.5))
	}

	if lum(5, 0, 0.8) >= 0.8 || lum(5, 10, 0.8) <= 0.8 || lum(5, 10, 0.2) >= 0.2 {
		t.Errorf("contrast: low %.3f high %.3f", lum(5, 0, 0.8), lum(5, 10, 0.8))
	}

	if d := lum(5, 10, 0.5) - 0.5; d > 0.01 || d < -0.01 {
		t.Errorf("contrast must keep mid grey: %.3f", lum(5, 10, 0.5))
	}
}

func TestSetColorAdjustClamps(t *testing.T) {
	r := &Renderer{}
	r.SetColorAdjust(-3, 40)

	if g, c := r.ColorAdjust(); g != 0 || c != 10 {
		t.Errorf("got %d,%d", g, c)
	}
}
