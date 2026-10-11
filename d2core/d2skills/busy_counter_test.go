package d2skills

import "testing"

// A busy refusal is counted apart from Refused, so the scripted cast test (which repeats Wasted casts up to a
// limit) neither spends retries on it nor reports it as a refused cast.
func TestBusyIsNotWasted(t *testing.T) {
	c := Counters{Busy: 9}
	if c.Wasted() != 0 {
		t.Errorf("busy casts are wasted: %d", c.Wasted())
	}

	c.Refused, c.EmptyArea = 2, 1
	if c.Wasted() != 3 {
		t.Errorf("Wasted = %d, want 3", c.Wasted())
	}
}
