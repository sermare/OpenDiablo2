package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// A "busy" refusal (CastAt during a running action) is retried by the scripted cast test, not counted as a
// failed cast, until the scenario has waited castBusyWaitMax seconds on it.
func TestBusyRetry(t *testing.T) {
	var wait float64

	if busyRetry(d2skill.ReasonLOS, &wait) || wait != 0 {
		t.Fatalf("a line-of-sight refusal is not a busy one (wait=%v)", wait)
	}

	n := 0
	for busyRetry(d2skill.ReasonBusy, &wait) {
		n++
		if n > 1000 {
			t.Fatal("busy retries never give up")
		}
	}

	max, step := float64(castBusyWaitMax), float64(castTestInterval)
	if want := int(max / step); n < want-1 || n > want+1 {
		t.Errorf("gave up after %d retries, want about %d", n, want)
	}

	// any other refusal clears the wait
	busyRetry(d2skill.ReasonMana, &wait)

	if wait != 0 {
		t.Errorf("wait not reset: %v", wait)
	}

	if !busyRetry(d2skill.ReasonBusy, &wait) {
		t.Error("a fresh busy refusal must be retried")
	}
}

// Busy does not count against a skill in the fight plan either.
func TestBusyIsNotRefusalForTheFightPlan(t *testing.T) {
	if refusalCounts(d2skill.ReasonBusy) {
		t.Error("busy counts against a skill")
	}

	if !refusalCounts(d2skill.ReasonLOS) {
		t.Error("a line-of-sight refusal must count")
	}
}
