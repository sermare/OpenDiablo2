package d2gamescreen

import "testing"

func TestGroundProgressRestartsTheGiveUpClock(t *testing.T) {
	var g groundState

	target := new(int)

	g.progress(target, 40) // first sight of the target
	g.elapsed = 9

	g.progress(target, 39.8) // 0.2 tiles closer is no headway
	if g.elapsed != 9 {
		t.Errorf("a step of 0.2 tiles restarted the clock: elapsed %v", g.elapsed)
	}

	g.progress(target, 38) // 2 tiles closer
	if g.elapsed != 0 || g.best != 38 {
		t.Errorf("headway did not restart the clock: elapsed %v best %v", g.elapsed, g.best)
	}

	g.elapsed = 5
	g.progress(target, 45) // pushed back: no restart, the best stays
	if g.elapsed != 5 || g.best != 38 {
		t.Errorf("retreat changed the clock: elapsed %v best %v", g.elapsed, g.best)
	}

	g.progress(new(int), 50) // another target starts afresh
	if g.elapsed != 0 || g.best != 50 {
		t.Errorf("a new target did not reset: elapsed %v best %v", g.elapsed, g.best)
	}
}
