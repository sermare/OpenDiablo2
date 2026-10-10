package d2maprenderer

import (
	"math"
	"testing"
)

func TestRoofRegion(t *testing.T) {
	roofs := map[[2]int]bool{{1, 1}: true, {2, 1}: true, {2, 2}: true, {8, 8}: true}
	has := func(x, y int) bool { return roofs[[2]int{x, y}] }

	if r := roofRegion(has, 0, 0); r != nil {
		t.Errorf("hero without roof: %v", r)
	}

	r := roofRegion(has, 2, 2)
	if len(r) != 3 || !r[[2]int{1, 1}] || r[[2]int{8, 8}] {
		t.Errorf("blob = %v, want the 3 connected tiles only", r)
	}
}

func TestFadeAlphaAndRoofAlpha(t *testing.T) {
	if a := fadeAlpha(1, 0, 0.25, 0.5); math.Abs(a-0.5) > 1e-9 {
		t.Errorf("half fade = %v", a)
	}

	if a := fadeAlpha(0.1, 0, 1, 0.5); a != 0 {
		t.Errorf("clamp = %v", a)
	}

	l := newLighting()
	l.elapsed = 0.5

	if a := l.roofAlpha(wallKey{1, 1, 0}, true); a != 0 {
		t.Errorf("covered roof after 500 ms = %v, want 0", a)
	}

	if a := l.roofAlpha(wallKey{1, 1, 0}, false); a != 1 {
		t.Errorf("roof back after 500 ms = %v, want 1", a)
	}

	if len(l.fades) != 0 {
		t.Error("finished fade not forgotten")
	}
}
