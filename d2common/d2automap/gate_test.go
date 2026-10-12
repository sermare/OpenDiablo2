package d2automap

import "testing"

func TestRevealGate(t *testing.T) {
	var g RevealGate
	if !g.Ready() {
		t.Fatal("unarmed gate must be ready")
	}

	g.Arm()

	for i := 0; i < UpdateDelayFrames; i++ {
		if g.Ready() {
			t.Fatalf("frame %d must be skipped", i)
		}
	}

	if !g.Ready() || !g.Ready() {
		t.Fatal("gate must open after the delay")
	}
}
