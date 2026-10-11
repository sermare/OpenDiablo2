package d2maprenderer

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

func TestScreenShake(t *testing.T) {
	var s screenShake

	top := func(n int) int { return n - 1 } // always +m

	if dx, dy := s.offset(top); dx != 0 || dy != 0 {
		t.Fatal("idle shake moved the view")
	}

	s = screenShake{shake: d2monster.StompShake(8, 5, 20, 15), on: true}
	s.advance(0.3) // inside the hold

	if dx, dy := s.offset(top); dx != 8 || dy != 8 {
		t.Errorf("hold offset (%d,%d)", dx, dy)
	}

	s.advance(2) // past ramp+hold+decay (1.6 s)

	if s.on {
		t.Error("shake did not end")
	}

	if dx, _ := s.offset(top); dx != 0 {
		t.Error("ended shake moved the view")
	}
}
