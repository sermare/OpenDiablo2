package d2autoscript

import "testing"

// A kill step may run 200 s: the runner must not go on to the next step before it ends (a check after
// kill:all,200 ran in the middle of the fight when the limit was 120 s).
func TestBusyTimeoutCoversLongKills(t *testing.T) {
	if BusyTimeout < 200+DefaultLootSeconds {
		t.Fatalf("BusyTimeout %.0f is shorter than a long kill step", BusyTimeout)
	}
}
