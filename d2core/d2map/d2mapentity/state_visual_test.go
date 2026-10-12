package d2mapentity

import (
	"image/color"
	"testing"
)

func TestStateTintedAndMul(t *testing.T) {
	var s StateTinted
	if _, _, _, on := s.StateTint(); on {
		t.Fatal("tint on by default")
	}

	s.SetStateTint(0.5, 0.75, 1, true)

	r, g, b, on := s.StateTint()
	if !on || r != 0.5 || g != 0.75 || b != 1 {
		t.Fatalf("tint %v %v %v %v", r, g, b, on)
	}

	got := MulStateTint(color.RGBA{R: 200, G: 100, B: 80, A: 255}, r, g, b)
	if got != (color.RGBA{R: 100, G: 75, B: 80, A: 255}) {
		t.Fatalf("mul: %v", got)
	}

	s.SetStateTint(1, 1, 1, false)

	if _, _, _, on := s.StateTint(); on {
		t.Fatal("tint not cleared")
	}
}
