package d2state

import (
	"os"
	"path/filepath"
	"testing"
)

// realDefs reads the shipped States.txt from $D2_TABLES/states/patch_d2.
func realDefs(t *testing.T) Defs {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "states", "patch_d2", "States.txt"))
	if err != nil {
		t.Skip(err)
	}

	d, err := ParseDefs(data)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func withDefs(d Defs) *Set {
	s := New()
	s.SetDefs(d)

	return s
}

// TestRealStateRows pins the states.txt values the rules below are built on.
func TestRealStateRows(t *testing.T) {
	d := realDefs(t)

	ids := map[string]int{"none": 0, "freeze": 1, "poison": 2, "amplifydamage": 9, "cold": 11, "weaken": 19, "stunned": 21,
		"dimvision": 23, "taunt": 27, "sanctuary": 47, "ironmaiden": 55, "terror": 56, "attract": 57, "lifetap": 58,
		"confuse": 59, "decrepify": 60, "lowerresist": 61, "blood_mana": 114, "burning": 115, "skilldelay": 121}
	for n, id := range ids {
		if got, ok := d[n]; !ok || got.ID != id {
			t.Errorf("%s: id %d ok=%v, want %d", n, got.ID, ok, id)
		}
	}

	// the eleven monster curses carry the curse flag; Amplify Damage's stat is damageresist
	for _, n := range []string{"amplifydamage", "weaken", "dimvision", "taunt", "ironmaiden", "terror", "attract", "lifetap",
		"confuse", "decrepify", "lowerresist"} {
		if !d[n].Curse {
			t.Errorf("%s must be a curse", n)
		}
	}

	if d["amplifydamage"].Stat != "damageresist" || d["weaken"].Stat != "damagepercent" || d["cold"].Stat != "velocitypercent" {
		t.Error("state stat columns changed")
	}

	if d["stunned"].Overlay1 != "stun" || d["amplifydamage"].Overlay1 != "curseamplifydamage" {
		t.Error("overlay1 column changed")
	}

	// stun has no colour change; freeze, cold, holywindcold and blue are the blue ones
	if d["stunned"].ColorPri != 0 || d["freeze"].ColorPri != 100 || d["freeze"].ColorShift != 108 || !d["freeze"].Blue ||
		d["cold"].ColorShift != 108 || d["poison"].ColorShift != 104 || d["poison"].ColorPri != 95 {
		t.Errorf("colour columns: freeze=%+v cold=%+v poison=%+v", d["freeze"], d["cold"], d["poison"])
	}

	// groups: armors replace each other (1), burst of speed and fade (2), forms (3)
	groups := map[string]int{"frozenarmor": 1, "chillingarmor": 1, "shiverarmor": 1, "bonearmor": 1, "quickness": 2, "fade": 2,
		"wolf": 3, "bear": 3, "maul": 3, "feralrage": 3, "delerium": 3}
	for n, g := range groups {
		if d[n].Group != g {
			t.Errorf("%s group = %d, want %d", n, d[n].Group, g)
		}
	}

	// every colour shift indexes the 111 PL2 hue variations
	for n, df := range d {
		if df.ColorShift < 0 || df.ColorShift > 110 {
			t.Errorf("%s colorshift %d outside the PL2 HueVariations", n, df.ColorShift)
		}
	}

	if !d["freeze"].Shatter || !d["freeze"].MonStayDeath || d["poison"].MonStayDeath || !d["cloak_of_shadows"].RemHit {
		t.Error("freeze/shatter/remhit flags changed")
	}
}

func TestRealCurseReplacement(t *testing.T) {
	d := realDefs(t)

	cases := []struct {
		name     string
		first    string
		second   string
		wantBoth bool
	}{
		{"amplify then weaken: one curse", "amplifydamage", "weaken", false},
		{"lifetap then ironmaiden", "lifetap", "ironmaiden", false},
		{"terror (howl) then decrepify", "terror", "decrepify", false},
		{"curse and an aura coexist", "weaken", "might", true},
		{"curse and a chill coexist", "amplifydamage", "cold", true},
		{"two armors share group 1", "frozenarmor", "shiverarmor", false},
		{"armor and energy shield coexist", "frozenarmor", "energyshield", true},
		{"burst of speed and fade share group 2", "quickness", "fade", false},
		{"wolf form replaces maul", "maul", "wolf", false},
		{"shrine states are not curses", "shrine_armor", "amplifydamage", true},
		{"poison and stun coexist", "poison", "stunned", true},
	}

	for _, c := range cases {
		s := withDefs(d)
		s.Apply(0, Instance{Name: c.first, Until: 1000})
		s.Apply(10, Instance{Name: c.second, Until: 1000})

		if both := s.Active(11, c.first) && s.Active(11, c.second); both != c.wantBoth {
			t.Errorf("%s: both active = %v, want %v", c.name, both, c.wantBoth)
		}

		if !s.Active(11, c.second) {
			t.Errorf("%s: the new state must be active", c.name)
		}
	}
}

func TestRealDeathRules(t *testing.T) {
	d := realDefs(t)

	cases := []struct {
		kind string
		keep []string
		drop []string
	}{
		{"monster", []string{"freeze", "revive", "shatter"}, []string{"poison", "amplifydamage", "stunned", "cold", "terror"}},
		{"player", []string{"alignment", "sync_warped", "corpse_noselect"}, []string{"freeze", "frozenarmor", "poison", "amplifydamage"}},
		// verified 0x627890: a boss's statlists follow monstaydeath
		{"boss", []string{"dopplezon", "revive", "freeze"}, []string{"poison", "weaken"}},
	}

	for _, c := range cases {
		s := withDefs(d)

		for _, n := range append(append([]string{}, c.keep...), c.drop...) {
			s.Apply(0, Instance{Name: n, Until: 1000})
		}

		s.AddStream(0, "poison", 256, 100, "x", 1)
		s.Death(c.kind)

		for _, n := range c.keep {
			if !s.Active(1, n) {
				t.Errorf("%s: %s must survive death", c.kind, n)
			}
		}

		for _, n := range c.drop {
			if s.Active(1, n) {
				t.Errorf("%s: %s must end on death", c.kind, n)
			}
		}

		if len(s.Streams(1)) != 0 {
			t.Errorf("%s: DoT streams must end on death", c.kind)
		}
	}
}

func TestRealColorShift(t *testing.T) {
	d := realDefs(t)

	cases := []struct {
		name   string
		states []string
		want   int
		ok     bool
	}{
		{"none", nil, 0, false},
		{"stun alone has no colour", []string{"stunned"}, 0, false},
		{"freeze", []string{"freeze"}, 108, true},
		{"poison", []string{"poison"}, 104, true},
		{"cold beats poison (blue, priority 100 over 95)", []string{"poison", "cold"}, 108, true},
		{"poison beats red (95 over 70)", []string{"red", "poison"}, 104, true},
		{"revive beats red (85 over 70)", []string{"red", "revive"}, 73, true},
		{"poison beats revive (95 over 85)", []string{"revive", "poison"}, 104, true},
		{"freeze and cold tie: same shift", []string{"cold", "freeze"}, 108, true},
	}

	for _, c := range cases {
		s := withDefs(d)
		for _, n := range c.states {
			s.Apply(0, Instance{Name: n, Until: 100})
		}

		if got, ok := s.ColorShift(5); got != c.want || ok != c.ok {
			t.Errorf("%s: shift = %d,%v want %d,%v", c.name, got, ok, c.want, c.ok)
		}

		if _, ok := s.ColorShift(100); ok {
			t.Errorf("%s: colour must end with the states", c.name)
		}
	}
}

func TestRealRemHitAndShatter(t *testing.T) {
	d := realDefs(t)
	s := withDefs(d)
	s.Apply(0, Instance{Name: "cloak_of_shadows", Until: 500})
	s.Apply(0, Instance{Name: "frozenarmor", Until: 500})
	s.Apply(0, Instance{Name: "freeze", Until: 500})

	if !s.Shatters(1) {
		t.Error("a frozen unit shatters")
	}

	if got := s.Hit(1); len(got) != 1 || got[0] != "cloak_of_shadows" {
		t.Errorf("Hit removed %v, want only cloak_of_shadows", got)
	}

	if !s.Active(2, "frozenarmor") || !s.Active(2, "freeze") {
		t.Error("a hit must not end other states")
	}

	if s.Shatters(600) {
		t.Error("shatter ends with the freeze")
	}
}

// Synthetic checks that need no game data.
func TestParseDefsSynthetic(t *testing.T) {
	const tbl = "state\tid\tgroup\tcurse\tremhit\tcolorpri\tcolorshift\tblue\n" +
		"none\t\t\t\t\t\t\t\n" +
		"a\t\t4\t1\t\t10\t5\t\n" +
		"b\t\t4\t\t1\t20\t6\t1\n"

	d, err := ParseDefs([]byte(tbl))
	if err != nil {
		t.Fatal(err)
	}

	if d["b"].ID != 2 || d["a"].Group != 4 || !d["a"].Curse || !d["b"].RemHit || !d["b"].Blue || d["b"].ColorShift != 6 {
		t.Errorf("defs = %+v", d)
	}

	if _, err := ParseDefs(nil); err == nil {
		t.Error("an empty table must be an error, not a panic")
	}

	s := withDefs(d)
	s.Apply(0, Instance{Name: "a", Until: 50})
	s.Apply(1, Instance{Name: "b", Until: 50})

	if s.Active(2, "a") {
		t.Error("same group: b must replace a")
	}

	// without defs the set behaves as before
	plain := New()
	plain.Apply(0, Instance{Name: "a", Until: 50})
	plain.Apply(1, Instance{Name: "b", Until: 50})

	if !plain.Active(2, "a") || !plain.Active(2, "b") {
		t.Error("a set without defs has no exclusion")
	}
}

func TestLengthRules(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"no resist", LengthAfterResist(100, 0), 100},
		{"50 resist halves", LengthAfterResist(100, 50), 50},
		{"75 resist", LengthAfterResist(100, 75), 25},
		{"immune removes it", LengthAfterResist(100, 100), 0},
		{"over 100", LengthAfterResist(100, 120), 0},
		{"negative resist lengthens (U)", LengthAfterResist(100, -50), 150},
		{"zero length", LengthAfterResist(0, 0), 0},
		{"curse resist 0 keeps length", CurseLength(300, 0), 300},
		{"curse resist 40", CurseLength(300, 40), 180},
		{"curse resist 100 rejects", CurseLength(300, 100), 0},
		{"stun below cap", StunLength(30), 30},
		{"stun cap", StunLength(999), MaxStunFrames},
		{"stun cap is 250", MaxStunFrames, 250},
	}

	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestChillUsesColdEffect(t *testing.T) {
	cases := []struct {
		name      string
		eff       int
		has       bool
		wantChill bool
		wantSpeed int
		wantAtk   int
	}{
		{"unknown effect: default 50", 0, false, true, -50, -50},
		{"normal -50", -50, true, true, -50, -50},
		{"hell -33", -33, true, true, -33, -33},
		{"none: skipped", 0, true, false, 0, 0},
	}

	for _, c := range cases {
		s := New()
		s.ApplyHit(0, Hit{ColdLen: 100, FreezeLen: 40, ColdEffect: c.eff, HasColdEffect: c.has})

		if got := s.Active(1, Chill); got != c.wantChill {
			t.Errorf("%s: chill active = %v", c.name, got)
		}

		if got := s.Active(1, Freeze); got != c.wantChill {
			t.Errorf("%s: freeze active = %v", c.name, got)
		}

		if got := s.SpeedPct(1); got != c.wantSpeed {
			t.Errorf("%s: speed = %d, want %d", c.name, got, c.wantSpeed)
		}

		if got := s.AttackSpeedPct(1); got != c.wantAtk {
			t.Errorf("%s: attack speed = %d, want %d (chill must not triple)", c.name, got, c.wantAtk)
		}

		if got := s.OtherAnimPct(1); got != c.wantAtk {
			t.Errorf("%s: other anim = %d, want %d", c.name, got, c.wantAtk)
		}
	}
}

func TestStunCapAndRefresh(t *testing.T) {
	s := New()
	s.ApplyHit(0, Hit{StunLen: 400})

	if !s.Active(249, Stun) || s.Active(250, Stun) {
		t.Error("stun must last at most 250 frames")
	}

	// verified 0x578830: a second stun replaces the first, even a shorter one
	s.ApplyHit(100, Hit{StunLen: 20})

	if s.Active(120, Stun) {
		t.Error("current behaviour: the newest stun replaces the old one (U)")
	}
}

// TestTickOrder pins the per-frame order of Set.Tick: the DoT of the final
// frame of a stream is paid, a state with Until == frame is gone on that frame
// and listed as expired, and the 8.8 remainder carries across frames.
func TestTickOrder(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "stunned", Until: 3})
	s.AddStream(0, "poison", 128, 3, "x", 1) // half a hp per frame for 3 frames

	var poison []int

	var expired [][]string

	for f := 0; f < 5; f++ {
		r := s.Tick(f)
		poison = append(poison, r.Poison)
		expired = append(expired, r.Expired)
	}

	want := []int{0, 1, 0, 0, 0}
	for i := range want {
		if poison[i] != want[i] {
			t.Fatalf("poison per frame = %v, want %v", poison, want)
		}
	}

	// 3 frames of 128/256 = 384/256: one whole hp now, 128/256 is carried and lost
	if len(expired[3]) != 1 || expired[3][0] != "stunned" {
		t.Errorf("expired = %v, want stunned on frame 3", expired)
	}
}

// TestApplyKeepsPrevious: Apply returns the instance it replaced only if it
// was still active.
func TestApplyKeepsPrevious(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "x", Until: 10})

	if prev := s.Apply(5, Instance{Name: "x", Until: 20}); prev == nil || prev.Until != 10 {
		t.Errorf("prev = %+v", prev)
	}

	if prev := s.Apply(30, Instance{Name: "x", Until: 40}); prev != nil {
		t.Errorf("expired state returned as previous: %+v", prev)
	}
}
