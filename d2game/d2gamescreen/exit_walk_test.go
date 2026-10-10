package d2gamescreen

import "testing"

// Playtest bug (Act 3): in the Great Marsh the scripted walk to Flayer Jungle left through the border to
// Spider Forest, which the marsh also touches.
func TestEdgeWantedOnlyTheBorderOfTheWalk(t *testing.T) {
	for _, c := range []struct {
		want, to int
		ok       bool
	}{{0, 76, true}, {78, 78, true}, {78, 76, false}, {76, 78, false}} {
		if got := edgeWanted(c.want, c.to); got != c.ok {
			t.Errorf("edgeWanted(%d, %d) = %v, want %v", c.want, c.to, got, c.ok)
		}
	}
}

// Playtest bug: the walk from the east end of Dry Hills to the border of Rocky Waste took longer than the
// 60 s time-out although the hero walked all the time; the clock runs only while he does not get closer.
func TestExitWalkProgressRestartsTheClock(t *testing.T) {
	e := &exitWalk{candidates: [][2]float64{{0, 0}, {1, 1}}}

	e.progress(80)
	e.elapsed = 50

	e.progress(79.8) // less than walkProgressStep closer: no progress
	if e.elapsed != 50 {
		t.Errorf("elapsed = %v after a tiny step, want 50", e.elapsed)
	}

	e.progress(70) // really closer
	if e.elapsed != 0 || e.best != 70 {
		t.Errorf("elapsed = %v best = %v after real progress", e.elapsed, e.best)
	}

	e.elapsed = 30
	e.next = 1 // the next candidate starts again
	e.progress(75)

	if e.elapsed != 0 || e.best != 75 || e.bestCand != 1 {
		t.Errorf("elapsed = %v best = %v cand = %d after switching candidate", e.elapsed, e.best, e.bestCand)
	}
}
