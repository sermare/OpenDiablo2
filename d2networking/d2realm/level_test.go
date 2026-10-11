package d2realm

import (
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
)

func monsterIDs(r *d2mp.Replica) (out []uint32) {
	for _, u := range r.Units() {
		if u.Kind == d2mp.KindMonster {
			out = append(out, u.ID)
		}
	}

	return out
}

// Two real clients on one realm: the engine's LevelChange moves the hero's
// simulation unit, so the hero in another level neither receives nor affects
// the town's units, is not in view, yet stays in the roster.
func TestLevelChangeSplitsTheWorld(t *testing.T) {
	tb := newTable(t, d2mp.NewEngineRules(30, 30, 2), 2)
	a, b := tb.players[0], tb.players[1]

	for _, p := range tb.players {
		if !p.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Level == 1 && len(r.Units()) >= 4 }) {
			t.Fatal("a player never saw the town")
		}
	}

	var dummies []uint32

	a.View(func(r *d2mp.Replica) { dummies = monsterIDs(r) })

	if len(dummies) != 2 {
		t.Fatalf("town dummies: %v", dummies)
	}

	// B walks through a door of its own map: LevelChange (act 1, Blood Moor = 2)
	if err := b.Send(LevelChange{Act: 0, Level: 2}); err != nil {
		t.Fatal(err)
	}

	if !b.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Level == 2 && len(monsterIDs(r)) > 0 }) {
		t.Fatal("B never arrived in level 2")
	}

	b.View(func(r *d2mp.Replica) {
		for _, u := range r.Units() {
			if u.Level != 2 {
				t.Errorf("B received unit %d of level %d", u.ID, u.Level)
			}
		}

		if r.Unit(a.Joined.UnitID) != nil {
			t.Error("B sees A across levels")
		}
	})

	if !a.WaitFor(wait, func(r *d2mp.Replica) bool { return r.Unit(b.Joined.UnitID) == nil }) {
		t.Fatal("A still sees B")
	}

	// the roster is global
	a.View(func(r *d2mp.Replica) {
		found := false

		for _, h := range r.Heroes() {
			found = found || h.ID == b.Joined.UnitID && h.Level == 2
		}

		if !found {
			t.Errorf("A's roster lacks B in level 2: %+v", r.Heroes())
		}
	})

	// the lobby notice still goes to A
	if got := a.Drain(func(e interface{}) bool { _, ok := e.(PlayerLevel); return ok }); len(got) != 1 {
		t.Errorf("A got %d PlayerLevel notices, want 1", len(got))
	}

	// B attacks a town dummy by id: nothing happens to it
	if err := b.Interact(d2mp.KindMonster, dummies[0]); err != nil {
		t.Fatal(err)
	}

	time.Sleep(600 * time.Millisecond)

	tb.srv.mu.Lock()
	u := tb.srv.games[tb.game].sim.Unit(dummies[0])
	hurt := u.HP != u.MaxHP
	tb.srv.mu.Unlock()

	if hurt {
		t.Error("B hurt a town dummy from level 2")
	}

	// a refused level (not a level id) leaves B where it is
	if err := b.Send(LevelChange{Act: 0, Level: 500}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	tb.srv.mu.Lock()
	lvl := tb.srv.games[tb.game].sim.Level(b.Joined.UnitID)
	tb.srv.mu.Unlock()

	if lvl != 2 {
		t.Errorf("B is in level %d after a bad LevelChange", lvl)
	}

	// B comes back: both see each other again and the worlds agree
	if err := b.Send(LevelChange{Act: 0, Level: 1}); err != nil {
		t.Fatal(err)
	}

	if !tb.converged(5*time.Second, 1) {
		t.Fatal("worlds did not converge after B returned to the town")
	}
}
