package d2mp

import "testing"

func TestEngineRulesLevel(t *testing.T) {
	r := NewEngineRules(126.5, 117.5, 3)

	for _, tc := range []struct {
		name  string
		level uint16
		mons  int
	}{{"town", 1, 3}, {"field", 2, 10}} {
		l := r.Level(99, tc.level)

		if !l.Walkable(l.SpawnX, l.SpawnY) {
			t.Fatalf("%s: spawn %.1f,%.1f not walkable", tc.name, l.SpawnX, l.SpawnY)
		}

		if len(l.Monsters) != tc.mons {
			t.Fatalf("%s: %d monsters, want %d", tc.name, len(l.Monsters), tc.mons)
		}

		for _, m := range l.Monsters {
			if !l.Walkable(m.X, m.Y) {
				t.Fatalf("%s: monster at %.1f,%.1f not walkable", tc.name, m.X, m.Y)
			}
		}

		again := r.Level(99, tc.level)
		if len(again.Monsters) != len(l.Monsters) || again.Monsters[0] != l.Monsters[0] {
			t.Fatalf("%s: level not deterministic", tc.name)
		}
	}
}

// Heroes of an engine game stand where the engine puts them, and the dummies can be killed.
func TestEngineRulesDummyKill(t *testing.T) {
	h := newHarness(t, NewEngineRules(126.5, 117.5, 3), 30)
	h.join(1, "A", 20)
	h.join(2, "B", 20)
	h.settle()

	for _, id := range []uint32{1, 2} {
		x, y, ok := h.rep[id].Pos(1)
		if !ok || x != Snap(126.5) || y != Snap(117.5) {
			t.Fatalf("hero 1 seen by %d at %v,%v ok=%v", id, x, y, ok)
		}
	}

	var dummies []*Unit

	for _, u := range h.sim.UnitsIn(1) {
		if u.Kind == KindMonster {
			dummies = append(dummies, u)
		}
	}

	if len(dummies) != 3 {
		t.Fatalf("%d dummies in town", len(dummies))
	}

	h.sim.Interact(2, KindMonster, dummies[0].ID)
	h.advance(4000)

	if h.sim.Stats.Kills != 1 {
		t.Fatalf("kills %d, the dummy did not die", h.sim.Stats.Kills)
	}

	h.settle()

	if u := h.rep[1].Unit(dummies[0].ID); u != nil && !u.Dead {
		t.Fatal("hero 1 did not see the kill")
	}

	if h.sim.Unit(1).Dead || h.sim.Unit(2).Dead {
		t.Fatal("a hero died to passive dummies")
	}

	if d := h.sim.Digest(1); d != h.rep[1].Digest() || d != h.rep[2].Digest() {
		t.Fatal("digests differ")
	}
}
