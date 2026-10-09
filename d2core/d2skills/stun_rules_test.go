package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func stunMonster(id, velocity int, boss bool, typeFlags uint16) *d2mapentity.Monster {
	m := &d2mapentity.Monster{Stat: &d2records.MonStatRecord{}, TypeFlags: typeFlags}
	m.Stat.ID, m.Stat.SpeedBase, m.Stat.IsSpecialBoss = id, velocity, boss

	return m
}

// hurt() hands the monster facts to d2state through applyMonsterStateRules; this runs every kind of monster
// through it and the real ApplyHit (the oracle agreement of the rules themselves is in d2state).
func TestMonsterStateRulesReachTheStates(t *testing.T) {
	const mercRogue = 0x10f

	cases := []struct {
		name       string
		m          *d2mapentity.Monster
		wantStun   bool
		wantFrames int // stun length when it lands
	}{
		{"plain monster is stunned", stunMonster(100, 100, false, 0), true, 100},
		{"boss is never stunned", stunMonster(101, 100, true, 0), false, 0},
		{"monster with velocity 0 is never stunned", stunMonster(102, 0, false, 0), false, 0},
		{"mercenary class stun is cut to 13 frames", stunMonster(mercRogue, 100, false, 0), true, d2state.SpecialStunFrames},
	}

	for _, c := range cases {
		e := newTestEngine()
		set := d2state.New()
		h := d2state.Hit{StunLen: 100}

		e.applyMonsterStateRules(&h, c.m, nil, set)

		if !h.MonsterRules {
			t.Fatalf("%s: MonsterRules not set", c.name)
		}

		set.ApplyHit(0, h)

		in := set.Get(0, d2state.Stun)
		if (in != nil) != c.wantStun {
			t.Errorf("%s: stunned=%v, want %v", c.name, in != nil, c.wantStun)

			continue
		}

		if in != nil && in.Until != c.wantFrames {
			t.Errorf("%s: stun until %d, want %d", c.name, in.Until, c.wantFrames)
		}
	}
}

func TestUniqueTypeMaskIsDataFlag8(t *testing.T) {
	e := newTestEngine()
	h := d2state.Hit{}

	e.applyMonsterStateRules(&h, stunMonster(100, 100, false, d2mapentity.MonTypeUnique), nil, d2state.New())

	if !h.DataFlag8 {
		t.Error("a monster with the 0x8 type mask must carry data flag 8")
	}

	h = d2state.Hit{}
	e.applyMonsterStateRules(&h, stunMonster(100, 100, false, d2mapentity.MonTypeSuperUnique), nil, d2state.New())

	if h.DataFlag8 {
		t.Error("the super unique mask (0x2) is not data flag 8")
	}
}

func TestUninterruptableStateBlocksFreeze(t *testing.T) {
	defs := d2state.Defs{
		"uninterruptable": {ID: StateUninterruptable, Name: "uninterruptable"},
		d2state.Freeze:    {ID: 1, Name: d2state.Freeze},
	}

	e := newTestEngine()
	set := d2state.New()
	set.SetDefs(defs)
	set.Apply(0, d2state.Instance{Name: "uninterruptable"})

	h := d2state.Hit{FreezeLen: 100, HasColdEffect: true, ColdEffect: -50}
	e.applyMonsterStateRules(&h, stunMonster(100, 100, false, 0), nil, set)

	if !h.Uninterruptable {
		t.Fatal("state 0x36 not seen")
	}

	set.ApplyHit(0, h)

	if set.Active(0, d2state.Freeze) || set.Active(0, d2state.Chill) {
		t.Error("an uninterruptable monster is neither frozen nor chilled")
	}

	// without the state a plain monster freezes
	e2 := newTestEngine()
	plain := d2state.New()
	h2 := d2state.Hit{FreezeLen: 100, HasColdEffect: true, ColdEffect: -50}
	e2.applyMonsterStateRules(&h2, stunMonster(100, 100, false, 0), nil, plain)
	plain.ApplyHit(0, h2)

	if !plain.Active(0, d2state.Freeze) {
		t.Error("a plain monster must freeze")
	}
}
