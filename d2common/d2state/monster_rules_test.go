package d2state

import "testing"

// Without MonsterRules the stun and freeze flags do nothing: every existing caller keeps its behaviour.
func TestMonsterRulesOffChangesNothing(t *testing.T) {
	s := New()
	s.ApplyHit(0, Hit{StunLen: 400, FreezeLen: 100, Boss: true, Immobile: true, Special: true, DataFlag8: true, Uninterruptable: true,
		StunRoll: func(int) int { return 0 }})

	if in := s.Get(0, Stun); in == nil || in.Until != MaxStunFrames {
		t.Errorf("stun must land and be capped at %d without MonsterRules: %+v", MaxStunFrames, in)
	}

	if !s.Active(0, Freeze) {
		t.Error("freeze must land without MonsterRules")
	}
}

// The data flag 8 roll comes from the attacker's generator and is consumed once, only for such monsters.
func TestStunRollOnlyForDataFlag8(t *testing.T) {
	calls := 0
	roll := func(n int) int { calls++; return 50 }

	New().ApplyHit(0, Hit{MonsterRules: true, StunLen: 10, StunRoll: roll})

	if calls != 0 {
		t.Errorf("a plain monster must not consume the roll, %d calls", calls)
	}

	s := New()
	s.ApplyHit(0, Hit{MonsterRules: true, DataFlag8: true, StunLen: 10, StunRoll: roll})

	if calls != 1 || s.Active(0, Stun) {
		t.Errorf("flag 8 with roll 50: %d call(s), stunned=%v, want one call and no stun", calls, s.Active(0, Stun))
	}
}

func TestActiveID(t *testing.T) {
	s := New()

	if s.ActiveID(0, 0x36) {
		t.Error("no table: false")
	}

	s.SetDefs(Defs{"uninterruptable": {ID: 0x36, Name: "uninterruptable"}})

	if s.ActiveID(0, 0x36) {
		t.Error("the state is not on the unit yet")
	}

	s.Apply(0, Instance{Name: "uninterruptable", Until: 10})

	if !s.ActiveID(5, 0x36) || s.ActiveID(10, 0x36) || s.ActiveID(5, 0x37) {
		t.Error("ActiveID must follow the instance's window and match only its own id")
	}
}
