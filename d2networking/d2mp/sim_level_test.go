package d2mp

import "testing"

func monstersOf(r *Replica) (out []*RUnit) {
	for _, u := range r.Units() {
		if u.Kind == KindMonster {
			out = append(out, u)
		}
	}

	return out
}

func heroLevel(r *Replica, id uint32) (uint16, bool) {
	for _, h := range r.Heroes() {
		if h.ID == id {
			return h.Level, true
		}
	}

	return 0, false
}

// Two heroes in two levels: each only receives and affects the units of its
// own level, the other hero is not in view, and the roster stays global.
func TestPerLevelWorldsTwoClients(t *testing.T) {
	h := newHarness(t, NewEngineRules(30, 30, 2), 20)
	h.join(1, "Alice", 10)
	h.join(2, "Bobby", 10)
	h.settle()

	a, b := h.rep[1], h.rep[2]

	// one level (the default): both see each other and the two dummies; no
	// roster-only events are sent at all
	for _, r := range []*Replica{a, b} {
		if len(monstersOf(r)) != 2 || len(r.Units()) != 4 {
			t.Fatalf("single level view: %d monsters, %d units", len(monstersOf(r)), len(r.Units()))
		}

		if r.Stats[EvHero] != 0 {
			t.Fatalf("EvHero sent while everybody shares a level: %d", r.Stats[EvHero])
		}
	}

	dummies := monstersOf(a)

	// invalid or no-op level changes
	for _, tc := range []struct {
		id    uint32
		level uint16
		want  bool
	}{{2, 0, false}, {2, 137, false}, {9, 2, false}, {2, 1, true}} {
		if got := h.sim.ChangeLevel(tc.id, tc.level); got != tc.want {
			t.Errorf("ChangeLevel(%d, %d) = %v, want %v", tc.id, tc.level, got, tc.want)
		}
	}

	h.settle()

	if b.Stats[EvLevel] != 1 { // only the join
		t.Fatalf("a no-op level change was announced: %d EvLevel", b.Stats[EvLevel])
	}

	// Bobby goes to level 2
	if !h.sim.ChangeLevel(2, 2) {
		t.Fatal("ChangeLevel refused")
	}

	h.settle()

	if b.Level != 2 || b.Unit(2) == nil {
		t.Fatalf("Bobby is in level %d", b.Level)
	}

	for _, u := range b.Units() {
		if u.Level != 2 {
			t.Errorf("Bobby received unit %d of level %d", u.ID, u.Level)
		}
	}

	if b.Unit(1) != nil || len(monstersOf(b)) == 0 {
		t.Errorf("Bobby sees Alice: %v, monsters: %d", b.Unit(1) != nil, len(monstersOf(b)))
	}

	if a.Unit(2) != nil || a.Level != 1 || len(monstersOf(a)) != 2 {
		t.Errorf("Alice sees Bobby: %v level %d monsters %d", a.Unit(2) != nil, a.Level, len(monstersOf(a)))
	}

	// the roster is global: both know the other and where it is
	if l, ok := heroLevel(a, 2); !ok || l != 2 {
		t.Errorf("Alice's roster: Bobby in level %d (known %v)", l, ok)
	}

	if l, ok := heroLevel(b, 1); !ok || l != 1 {
		t.Errorf("Bobby's roster: Alice in level %d (known %v)", l, ok)
	}

	// Bobby cannot affect the town's dummies from level 2, and what happens to
	// them is not sent to him
	d := dummies[0]
	bEv := b.Stats[EvHit] + b.Stats[EvDeath] + b.Stats[EvAttack]
	aAtk := a.Stats[EvAttack]

	h.sim.Interact(2, KindMonster, d.ID)
	h.advance(1500)

	if u := h.sim.Unit(d.ID); u.HP != u.MaxHP || u.Dead {
		t.Fatalf("Bobby hurt a unit of another level: hp %d/%d", u.HP, u.MaxHP)
	}

	if a.Stats[EvAttack] != aAtk {
		t.Error("Alice saw an attack from another level")
	}

	// Alice kills a dummy: Bobby learns nothing about it
	h.sim.Interact(1, KindMonster, d.ID)

	killed := false

	for i := 0; i < 60 && !killed; i++ { // the corpse is removed after a while
		h.advance(100)

		u := h.sim.Unit(d.ID)
		killed = u == nil || u.Dead
	}

	if !killed {
		t.Fatal("Alice could not kill the dummy of her own level")
	}

	h.advance(3000)

	if got := b.Stats[EvHit] + b.Stats[EvDeath] + b.Stats[EvAttack]; got != bEv {
		t.Errorf("Bobby received %d combat events of another level", got-bEv)
	}

	// Bobby returns: sees Alice and the dead dummy again, Alice sees him
	if !h.sim.ChangeLevel(2, 1) {
		t.Fatal("return refused")
	}

	h.settle()

	if b.Level != 1 || b.Unit(1) == nil || a.Unit(2) == nil {
		t.Fatal("heroes do not see each other after Bobby returned")
	}

	if b.Unit(d.ID) != nil || b.Unit(dummies[1].ID) == nil {
		t.Errorf("returned hero should see the surviving dummy only")
	}

	if h.sim.Digest(1) != a.Digest() || h.sim.Digest(1) != b.Digest() {
		t.Error("digests differ after the round trip")
	}
}

// A hero joining while another is elsewhere learns it for the roster, and a
// party formed across levels reaches the roster of both.
func TestPerLevelRosterJoinAndParty(t *testing.T) {
	h := newHarness(t, NewEngineRules(30, 30, 1), 20)
	h.join(1, "Alice", 10)
	h.sim.ChangeLevel(1, 2)
	h.settle()

	h.join(2, "Bobby", 10) // the town of act 1; Alice is in level 2
	h.settle()

	if l, ok := heroLevel(h.rep[2], 1); !ok || l != 2 {
		t.Fatalf("a joiner does not know Alice is in level 2: %d %v", l, ok)
	}

	if l, ok := heroLevel(h.rep[1], 2); !ok || l != 1 {
		t.Fatalf("Alice does not know about the joiner: %d %v", l, ok)
	}

	if h.rep[1].Unit(2) != nil || h.rep[2].Unit(1) != nil {
		t.Error("heroes of different levels are in view")
	}

	// party commands work across levels
	h.sim.Command(1, Command{Type: CmdPartyInvite, Target: 2})
	h.sim.Command(2, Command{Type: CmdPartyAccept})
	h.settle()

	partyOf := func(r *Replica, id uint32) uint16 {
		if u := r.Unit(id); u != nil {
			return u.Party
		}

		for _, hu := range r.Heroes() {
			if hu.ID == id {
				return hu.Party
			}
		}

		return 0
	}

	for _, r := range []*Replica{h.rep[1], h.rep[2]} {
		p1, p2 := partyOf(r, 1), partyOf(r, 2)
		if p1 == 0 || p1 != p2 {
			t.Errorf("viewer %d: party ids %d and %d", r.Self(), p1, p2)
		}
	}
}

func TestEvHeroCodec(t *testing.T) {
	in := []Event{{Type: EvHero, ID: 7, Unit: Unit{ID: 7, Kind: KindPlayer, Type: 2, Level: 40, Name: "Far", Party: 3}}}

	out, err := DecodeEvents(EncodeEvents(in))
	if err != nil || len(out) != 1 || out[0].Type != EvHero || out[0].Unit.Name != "Far" || out[0].Unit.Level != 40 ||
		out[0].Unit.Party != 3 || out[0].ID != 7 {
		t.Fatalf("round trip: %+v %v", out, err)
	}
}
