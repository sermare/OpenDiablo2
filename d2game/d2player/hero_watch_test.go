package d2player

import "testing"

func TestHeroWatchMoved(t *testing.T) {
	var w heroWatch
	w.arm(heroWatchMoved, 10, 10)

	for _, p := range [][2]float64{{10, 10}, {10.5, 10.5}, {10.6, 10.6}} {
		if fired, _ := w.update(p[0], p[1], 0.04); fired {
			t.Fatalf("fired too early at %v", p)
		}
	}

	if fired, d := w.update(11, 11, 0.04); !fired || d < 1.4 {
		t.Fatalf("want fired with dist 1.41, got %v %v", fired, d)
	}

	if fired, _ := w.update(20, 20, 0.04); fired {
		t.Fatal("a fired watch must disarm")
	}
}

func TestHeroWatchStill(t *testing.T) {
	var w heroWatch
	w.arm(heroWatchStill, 0, 0)

	// moving resets the timer, however long it lasts
	for i := 1; i <= 100; i++ {
		if fired, _ := w.update(float64(i)*0.1, 0, 0.04); fired {
			t.Fatalf("fired while walking at step %d", i)
		}
	}

	steps := 0

	for ; steps < 100; steps++ {
		if fired, _ := w.update(10, 0, 0.04); fired {
			break
		}
	}

	// 0.5 s at 0.04 s per update is 13 standing updates
	if steps < 11 || steps > 14 {
		t.Fatalf("fired after %d standing updates, want about 12", steps)
	}
}

func TestHeroWatchIdle(t *testing.T) {
	var w heroWatch
	if fired, _ := w.update(1, 1, 1); fired {
		t.Fatal("an unarmed watch fired")
	}
}
