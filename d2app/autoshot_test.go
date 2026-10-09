package d2app

import "testing"

func TestAutoShot(t *testing.T) {
	t.Run("off without the variable", func(t *testing.T) {
		t.Setenv("OD2_AUTOSHOT", "")

		if newAutoShot() != nil {
			t.Fatal("expected no autoshot")
		}
	})

	t.Run("requests a frame capture after the delay", func(t *testing.T) {
		t.Setenv("OD2_AUTOSHOT", "/tmp/od2-test-shot.png")
		t.Setenv("OD2_AUTOSHOT_SECONDS", "2")

		a := &App{autoShot: newAutoShot()}

		a.advanceAutoShot(1)

		if a.captureState != captureStateNone {
			t.Fatal("captured too early")
		}

		a.advanceAutoShot(1.5)

		if a.captureState != captureStateFrame || a.capturePath != "/tmp/od2-test-shot.png" {
			t.Fatalf("state=%v path=%q", a.captureState, a.capturePath)
		}
	})
}
