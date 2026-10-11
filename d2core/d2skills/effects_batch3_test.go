package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func formDefs() d2state.Defs {
	// States.txt group 3: wolf, bear, delerium, maul, feralrage (patch_d2)
	return d2state.Defs{
		"wolf": {Name: "wolf", Group: 3}, "bear": {Name: "bear", Group: 3}, "delerium": {Name: "delerium", Group: 3},
		"maul": {Name: "maul", Group: 3}, "feralrage": {Name: "feralrage", Group: 3}, "frenzy": {Name: "frenzy"},
	}
}

// Wearwolf from the human shape applies the form; the same cast again, or
// Wearbear from the wolf, only takes the shape back (0x5c4e80 + 0x56a480).
func TestToggleForm(t *testing.T) {
	cases := []struct {
		name        string
		have        []string
		cast        string
		wantApplied bool
		wantActive  []string
	}{
		{"human to wolf", nil, "wolf", true, nil},
		{"wolf again", []string{"wolf"}, "wolf", false, nil},
		{"wolf to bear", []string{"wolf"}, "bear", false, nil},
		{"wolf with feral rage to wolf", []string{"wolf", "feralrage"}, "wolf", false, nil},
		{"unrelated state stays", []string{"frenzy"}, "wolf", true, []string{"frenzy"}},
	}

	for _, c := range cases {
		s := d2state.New()
		s.SetDefs(formDefs())

		for _, n := range c.have {
			s.Apply(0, d2state.Instance{Name: n, Until: 1000})
		}

		if got := toggleForm(s, 5, c.cast); got != c.wantApplied {
			t.Errorf("%s: applied = %v, want %v", c.name, got, c.wantApplied)
		}

		for _, n := range c.wantActive {
			if !s.Active(6, n) {
				t.Errorf("%s: %s must stay", c.name, n)
			}
		}

		for _, n := range []string{"wolf", "bear", "feralrage"} {
			if s.Active(6, n) {
				t.Errorf("%s: %s still active after the switch", c.name, n)
			}
		}
	}
}

// Feral Rage and Maul coexist with the form: with no group clear the wolf stays.
func TestFeralRageKeepsTheForm(t *testing.T) {
	s := d2state.New()
	s.SetDefs(formDefs())
	s.Apply(0, d2state.Instance{Name: "wolf", Until: 1000})
	// NoGroup: the engine skips ClearGroup and applies straight away
	s.Apply(1, d2state.Instance{Name: "feralrage", Until: 600, Count: 1})

	if !s.Active(2, "wolf") || !s.Active(2, "feralrage") {
		t.Error("feral rage must not end the wolf form")
	}

	// the old behaviour (a group clear before the apply) ended the form
	s.ClearGroup(2, "feralrage")

	if s.Active(3, "wolf") {
		t.Error("sanity: a group clear ends the wolf")
	}
}

func TestInAuraRange(t *testing.T) {
	cases := []struct {
		dx, dy, r int
		want      bool
	}{{0, 0, 30, true}, {30, 0, 30, true}, {31, 0, 30, false}, {21, 21, 30, true}, {22, 22, 30, false}, {1, 0, 0, false}}

	for _, c := range cases {
		if got := inAuraRange(100+c.dx, 100+c.dy, 100, 100, c.r); got != c.want {
			t.Errorf("inAuraRange(d=%d,%d r=%d) = %v", c.dx, c.dy, c.r, got)
		}
	}
}

func TestCanConvertMonster(t *testing.T) {
	cases := []struct {
		name  string
		flags uint16
		want  bool
	}{
		{"plain", 0, true},
		{"mods rolled", d2mapentity.MonTypeModsRolled, true},
		{"champion", d2mapentity.MonTypeChampion, true},
		{"minion", d2mapentity.MonTypeMinion, true},
		{"unique", d2mapentity.MonTypeUnique, false},
		{"super unique", d2mapentity.MonTypeSuperUnique, false},
		{"unique pack leader", d2mapentity.MonTypeUnique | d2mapentity.MonTypeModsRolled, false},
	}

	for _, c := range cases {
		if got := canConvertMonster(c.flags); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
