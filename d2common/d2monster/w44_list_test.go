package d2monster

import "testing"

func TestW44List(t *testing.T) {
	seen := map[string]bool{}

	for _, l := range [][]string{FaithfulA(), FaithfulB(), FaithfulBatch3(), FaithfulBatch4(), FaithfulBatch5(),
		FaithfulBatch7(), FaithfulBatch8(), FaithfulSpawners(), FaithfulW20(), FaithfulW38(), FaithfulW40()} {
		for _, n := range l {
			seen[n] = true
		}
	}

	for _, n := range FaithfulW44() {
		if seen[n] {
			t.Errorf("%s already listed", n)
		}

		if _, ok := AITargetMode(n); !ok {
			t.Errorf("%s is not an exe AI name", n)
		}

		if _, ok := Lookup(n); !ok {
			t.Errorf("%s is not registered", n)
		}
	}
}

// 0x5e09c0: a command naming a dead unit is popped and the minion falls back to its own target (host answers
// not-found for dead units).
func TestW44MinionDeadCommandPops(t *testing.T) {
	b := brainAt(profile("Minion"))
	b.PushCommand(Command{Type: CmdAlert, Target: 77, Count: 1 << 30})

	w := newFake(20, false)
	runOne(t, w, b)

	if b.PeekCommand() != nil {
		t.Fatal("command with an unresolvable unit must be popped")
	}
}
