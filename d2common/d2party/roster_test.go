package d2party

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func game() *Roster {
	r := New()
	r.Add(Member{ID: "a", Name: "Ann", Level: 20, Area: 1})
	r.Add(Member{ID: "b", Name: "Bob", Level: 10, Area: 1})
	r.Add(Member{ID: "c", Name: "Cy", Level: 5, Area: 1})
	r.Add(Member{ID: "d", Name: "Di", Level: 30, Area: 2})

	return r
}

func TestInviteAcceptLeave(t *testing.T) {
	r := game()

	if _, err := r.Accept("b"); err != ErrNoInvite {
		t.Fatalf("accept without invite: %v", err)
	}

	if err := r.Invite("a", "a"); err != ErrSelf {
		t.Fatalf("self invite: %v", err)
	}

	if err := r.Invite("a", "zz"); err != ErrUnknownPlayer {
		t.Fatalf("unknown invite: %v", err)
	}

	if err := r.Invite("a", "b"); err != nil || r.InvitedBy("b") != "a" {
		t.Fatalf("invite: %v %q", err, r.InvitedBy("b"))
	}

	p, err := r.Accept("b")
	if err != nil || p == 0 || !r.SameParty("a", "b") || r.InvitedBy("b") != "" {
		t.Fatalf("accept: %d %v", p, err)
	}

	if err := r.Invite("a", "b"); err != ErrAlreadyParty {
		t.Fatalf("re-invite: %v", err)
	}

	// c joins through b; a, b, c share one party
	_ = r.Invite("b", "c")

	if _, err := r.Accept("c"); err != nil || len(r.PartyMembers("a")) != 3 {
		t.Fatalf("third member: %v", err)
	}

	// leaving a party of three keeps the other two; leaving again dissolves it
	if !r.Leave("c") || r.SameParty("a", "c") || !r.SameParty("a", "b") {
		t.Fatal("c leaving")
	}

	if !r.Leave("b") || r.PartyID("a") != 0 || r.PartyID("b") != 0 {
		t.Fatal("a party of one must dissolve")
	}

	if r.Leave("a") {
		t.Fatal("a was not in a party any more")
	}

	if err := r.Decline("a"); err != ErrNoInvite {
		t.Fatalf("decline: %v", err)
	}
}

func TestPartyFull(t *testing.T) {
	r := New()
	for i := 0; i < d2enum.MaxPlayersInGame+1; i++ {
		r.Add(Member{ID: string(rune('a' + i)), Level: 10})
	}

	for i := 1; i < d2enum.MaxPlayersInGame; i++ {
		id := string(rune('a' + i))
		if err := r.Invite("a", id); err != nil {
			t.Fatal(err)
		}

		if _, err := r.Accept(id); err != nil {
			t.Fatal(err)
		}
	}

	last := string(rune('a' + d2enum.MaxPlayersInGame))
	if err := r.Invite("a", last); err != ErrPartyFull {
		t.Fatalf("a full party must refuse: %v", err)
	}
}

func TestHostility(t *testing.T) {
	r := game()

	tests := []struct {
		from, to string
		want     error
	}{
		{"a", "b", nil},
		{"a", "c", ErrLevelTooLow}, // c is level 5
		{"c", "a", ErrLevelTooLow},
		{"a", "a", ErrSelf},
		{"a", "x", ErrUnknownPlayer},
	}

	for _, tc := range tests {
		if got := r.SetHostile(tc.from, tc.to, true); got != tc.want {
			t.Errorf("SetHostile(%s,%s) = %v want %v", tc.from, tc.to, got, tc.want)
		}
	}

	// directional: a declared, b did not
	if !r.CanAttack("a", "b") || r.CanAttack("b", "a") {
		t.Fatal("attack permission follows the declaring side")
	}

	if r.Relation("a", "b") != d2enum.PlayerRelationEnemy || r.Relation("b", "a") != d2enum.PlayerRelationEnemy {
		t.Fatal("enemy shows on both sides")
	}

	if r.Relation("a", "d") != d2enum.PlayerRelationNeutral {
		t.Fatal("neutral by default")
	}

	// hostile players cannot party
	if err := r.Invite("b", "a"); err != ErrHostile {
		t.Fatalf("invite across hostility: %v", err)
	}

	_ = r.SetHostile("a", "b", false)

	if r.CanAttack("a", "b") {
		t.Fatal("peace must stop attacks")
	}
}

func TestNoFriendlyFire(t *testing.T) {
	r := game()

	if err := r.Invite("a", "b"); err != nil {
		t.Fatal(err)
	}

	if _, err := r.Accept("b"); err != nil {
		t.Fatal(err)
	}

	if r.CanAttack("a", "b") || r.Relation("a", "b") != d2enum.PlayerRelationFriend {
		t.Fatal("party members are friends and cannot attack each other")
	}

	// declaring hostility against a party mate is allowed and leaves the party
	// (VERIFIED 0x5a3870)
	d, err := r.DeclareHostile("a", "b")
	if err != nil || !d.Changed || !d.LeftParty || !d.PortalsToClose() {
		t.Fatalf("hostility inside a party: %+v %v", d, err)
	}

	if r.SameParty("a", "b") || !r.CanAttack("a", "b") {
		t.Fatal("after declaring, the declarer is out of the party and may fight")
	}
}

func TestRemoveClearsRelations(t *testing.T) {
	r := game()
	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")
	_ = r.Invite("a", "c")
	_ = r.SetHostile("d", "a", true)
	r.Remove("a")

	if r.Has("a") || r.PartyID("b") != 0 || r.InvitedBy("c") != "" || r.Hostile("d", "a") {
		t.Fatal("removing a player must clear its party, invitations and hostility")
	}
}

func TestShareXP(t *testing.T) {
	r := game()
	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")
	_ = r.Invite("a", "d")
	_, _ = r.Accept("d") // d is in another area: takes no part
	_ = r.Invite("a", "c")
	_, _ = r.Accept("c")

	tests := []struct {
		name   string
		killer string
		xp     int
		want   map[string]int
	}{
		{"party, by level (20:10:5)", "a", 350, map[string]int{"a": 200, "b": 100, "c": 50}},
		{"remainder goes to the killer", "b", 100, map[string]int{"a": 57, "b": 29, "c": 14}},
		{"another area kills alone", "d", 90, map[string]int{"d": 90}},
		{"zero", "a", 0, map[string]int{"a": 0}},
	}

	for _, tc := range tests {
		got := r.ShareXP(tc.killer, tc.xp)
		sum := 0

		for _, s := range got {
			sum += s.XP

			if w, ok := tc.want[s.ID]; !ok || w != s.XP {
				t.Errorf("%s: %s got %d want %v", tc.name, s.ID, s.XP, tc.want)
			}
		}

		if len(got) != len(tc.want) || sum != tc.xp {
			t.Errorf("%s: %v sums to %d, want %d over %d members", tc.name, got, sum, tc.xp, len(tc.want))
		}
	}

	solo := New()
	solo.Add(Member{ID: "z", Level: 3})

	if s := solo.ShareXP("z", 77); len(s) != 1 || s[0].XP != 77 {
		t.Fatalf("solo: %v", s)
	}

	if solo.ShareXP("nobody", 5) != nil {
		t.Fatal("unknown killer")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	r := game()
	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")
	_ = r.Invite("a", "c")
	_ = r.SetHostile("d", "a", true)

	c := New()
	c.Restore(r.Snapshot())

	if c.Summary() != r.Summary() || c.InvitedBy("c") != "a" || !c.SameParty("a", "b") || !c.Hostile("d", "a") {
		t.Fatalf("replica differs:\n%s\n%s", c.Summary(), r.Summary())
	}

	// ids of new parties do not collide with restored ones
	_ = c.Invite("c", "d")
	_, _ = c.Accept("d")

	if c.PartyID("c") == c.PartyID("a") {
		t.Fatal("party ids collide")
	}
}

func TestShareXPRules(t *testing.T) {
	r := game()
	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")
	_ = r.Invite("a", "c")
	_, _ = r.Accept("c")
	_ = r.Invite("a", "d")
	_, _ = r.Accept("d") // d is in another area: takes no part

	sum := func(s []XPShare) (n int) {
		for _, x := range s {
			n += x.XP
		}

		return n
	}

	for _, xp := range []int{0, 1, 7, 100, 12345, 1 << 30} {
		s := r.ShareXP("a", xp)
		if sum(s) != xp {
			t.Errorf("xp %d: shares add up to %d", xp, sum(s))
		}

		if xp == 0 {
			continue
		}

		if len(s) != 3 {
			t.Fatalf("xp %d: %d shares, want 3 (same area only)", xp, len(s))
		}
	}

	// proportional to level 20:10:5 (UNVERIFIED weights), remainder to the killer
	s := r.ShareXP("a", 3500)
	got := map[string]int{}
	for _, x := range s {
		got[x.ID] = x.XP
	}

	if got["a"] != 2000 || got["b"] != 1000 || got["c"] != 500 {
		t.Errorf("shares %v", got)
	}

	// a killer without a party keeps everything; unknown killer: nothing
	solo := New()
	solo.Add(Member{ID: "z", Level: 50, Area: 1})

	if s := solo.ShareXP("z", 99); len(s) != 1 || s[0].XP != 99 {
		t.Errorf("solo %v", s)
	}

	if s := solo.ShareXP("nobody", 99); s != nil {
		t.Errorf("unknown killer %v", s)
	}
}

// TestShareKillXP: the party split of the exe with per-member level scaling,
// and a solo player exactly as the solo rule (d2herostats.KillXP).
func TestShareKillXP(t *testing.T) {
	r := game() // a=20, b=10, c=5 (same area), d elsewhere
	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")
	_ = r.Invite("a", "c")
	_, _ = r.Accept("c")

	got := map[string]int{}
	for _, s := range r.ShareKillXP("a", 3500, 10, 99) {
		got[s.ID] = s.XP
	}

	// total = 3500 + 2*3500*89>>8 = 5932, split 20:10:5, then each share scaled
	pool := SplitKillXP(3500, []int{20, 10, 5})
	want := map[string]int{
		"a": d2herostats.KillXP(pool[0], 10, 20, 99, 0),
		"b": d2herostats.KillXP(pool[1], 10, 10, 99, 0),
		"c": d2herostats.KillXP(pool[2], 10, 5, 99, 0),
	}

	if len(got) != 3 || got["a"] != want["a"] || got["b"] != want["b"] || got["c"] != want["c"] {
		t.Errorf("got %v want %v pool %v", got, want, pool)
	}

	if pool[0]+pool[1]+pool[2] < 5900 || pool[0]+pool[1]+pool[2] > 5932 {
		t.Errorf("party bonus pool %v", pool)
	}

	// a solo player (alone in another area): exactly the solo rule
	lvl := 1

	for _, in := range r.Snapshot().Players {
		if in.ID == "d" {
			lvl = in.Level
		}
	}

	s := r.ShareKillXP("d", 1000, 40, 99)
	if len(s) != 1 || s[0].ID != "d" || s[0].XP != d2herostats.KillXP(1000, 40, lvl, 99, 0) {
		t.Errorf("solo: %v", s)
	}

	if r.ShareKillXP("nobody", 5, 5, 99) != nil {
		t.Error("unknown killer")
	}
}

// A party never exceeds the 8 recipients of the exe (the roster refuses a ninth member).
func TestShareKillXPCapsRecipients(t *testing.T) {
	r := New()
	id := func(i int) string { return string(rune('a' + i)) }

	for i := 0; i < 10; i++ {
		r.Add(Member{ID: id(i), Name: id(i), Level: 30, Area: 1})
	}

	for i := 1; i < 10; i++ {
		if err := r.Invite("a", id(i)); err == nil {
			_, _ = r.Accept(id(i))
		}
	}

	s := r.ShareKillXP("a", 1000, 30, 99)
	if len(s) != MaxRecipients || MaxRecipients != d2enum.MaxPlayersInGame {
		t.Fatalf("%d recipients", len(s))
	}
}
