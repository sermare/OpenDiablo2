package d2equip

import "testing"

func TestSocketRefusal(t *testing.T) {
	ok := SocketAttempt{TargetIdentified: true, TargetSocketed: true, TargetHasInv: true, Sockets: 2, Filled: 1,
		FillerIsSocket: true, FillerIdentified: true}

	if r := SocketRefusal(ok); r != "" {
		t.Errorf("valid socketing refused: %s", r)
	}

	mut := func(f func(*SocketAttempt)) SocketAttempt {
		a := ok
		f(&a)

		return a
	}

	for name, a := range map[string]SocketAttempt{
		"target unidentified": mut(func(a *SocketAttempt) { a.TargetIdentified = false }),
		"not a filler":        mut(func(a *SocketAttempt) { a.FillerIsSocket = false }),
		"jewel unidentified":  mut(func(a *SocketAttempt) { a.FillerIdentified = false }),
		"no socket flag":      mut(func(a *SocketAttempt) { a.TargetSocketed = false }),
		"no inventory base":   mut(func(a *SocketAttempt) { a.TargetHasInv = false }),
		"full":                mut(func(a *SocketAttempt) { a.Filled = 2 }),
	} {
		if SocketRefusal(a) == "" {
			t.Errorf("%s: not refused", name)
		}
	}
}
