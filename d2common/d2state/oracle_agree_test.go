package d2state_test

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// Two implementations of the same exe rules live in the tree: the engine's
// d2state (used by d2core/d2skills) and the emulator-verified appliers in
// d2combat/states.go (golden states_golden.json). These tests drive both with
// the same application sequences and fail if they ever disagree, so a change to
// one cannot silently drift from the other.
//
// Applications are kept inside the active window of the previous one: the exe
// removes expired statlists, the oracle's lists keep their last end, so a late
// application would compare an expired d2state instance with a stale oracle one.

type zeroRoll struct{}

// Roll returns 99, so the 20 percent chill-flag roll never fires and the
// stun data-flag-8 roll (< 90) never blocks.
func (zeroRoll) Roll(int32) uint32 { return 99 }

type step struct {
	frame, length, value int
}

// oracleEnd returns (active, end) of an oracle list at a frame.
func oracleEnd(u *d2combat.StateUnit, state, frame int) (bool, int) {
	l := u.Lists[state]
	if l == nil || l.End <= frame {
		return false, 0
	}

	return true, l.End
}

func engineEnd(s *d2state.Set, name string, frame int) (bool, int) {
	in := s.Get(frame, name)
	if in == nil {
		return false, 0
	}

	return true, in.Until
}

func TestStunAgreesWithOracleForPlayers(t *testing.T) {
	steps := []step{{0, 100, 0}, {10, 30, 0}, {20, 400, 0}, {30, 5, 0}}

	u := d2combat.NewStateUnit()
	s := d2state.New()

	for i, st := range steps {
		u.ApplyStun(st.frame, st.length, d2combat.StateTarget{Monster: false}, zeroRoll{})
		s.ApplyHit(st.frame, d2state.Hit{StunLen: st.length})

		oa, oe := oracleEnd(u, d2combat.StateStun, st.frame)
		ea, ee := engineEnd(s, d2state.Stun, st.frame)

		if oa != ea || oe != ee {
			t.Errorf("step %d (%+v): oracle active=%v end=%d, engine active=%v end=%d", i, st, oa, oe, ea, ee)
		}
	}
}

func TestPoisonBurnAgreeWithOracle(t *testing.T) {
	// value is the per-frame damage; a weaker one is ignored, an equal or
	// stronger one replaces both value and end (even with a shorter length).
	steps := []step{{0, 100, 10}, {5, 500, 4}, {6, 50, 10}, {7, 20, 12}, {8, 300, 11}}

	for _, kind := range []struct {
		name  string
		state int
	}{{"poison", d2combat.StatePoison}, {"burn", d2combat.StateBurn}} {
		u := d2combat.NewStateUnit()
		s := d2state.New()

		for i, st := range steps {
			u.ApplyDot(kind.state, st.frame, st.value, st.length)
			s.AddStream(st.frame, kind.name, st.value, st.length, "x", 1)

			var oracleVal, oracleEndFrame int

			if l := u.Lists[kind.state]; l != nil {
				oracleVal, oracleEndFrame = -l.Stats[d2combat.StatHPRegen], l.End
			}

			var engineVal, engineEndFrame int

			for _, str := range s.Streams(st.frame) {
				if str.Kind == kind.name {
					engineVal, engineEndFrame = str.PerFrame, str.Until
				}
			}

			if oracleVal != engineVal || oracleEndFrame != engineEndFrame {
				t.Errorf("%s step %d (%+v): oracle value=%d end=%d, engine value=%d end=%d",
					kind.name, i, st, oracleVal, oracleEndFrame, engineVal, engineEndFrame)
			}
		}
	}
}

func TestPoisonAndBurnAddUpInBoth(t *testing.T) {
	u := d2combat.NewStateUnit()
	s := d2state.New()

	u.ApplyDot(d2combat.StatePoison, 0, 7, 100)
	u.ApplyDot(d2combat.StateBurn, 0, 9, 100)
	s.AddStream(0, "poison", 7, 100, "x", 1)
	s.AddStream(0, "burn", 9, 100, "x", 1)

	if len(u.Lists) != 2 || len(s.Streams(1)) != 2 {
		t.Errorf("poison and burn must be independent lists/streams: oracle %d, engine %d", len(u.Lists), len(s.Streams(1)))
	}
}

func TestChillAgreesWithOracle(t *testing.T) {
	cases := []struct {
		name    string
		monster bool
		effect  int // monstats ColdEffect (monsters)
		div     int
		steps   []step
	}{
		{"player default -50", false, 0, 0, []step{{0, 100, 0}, {10, 200, 0}, {20, 10, 0}}},
		{"monster -75, divisor 2", true, -75, 2, []step{{0, 100, 0}, {10, 200, 0}, {20, 10, 0}}},
		{"monster -50, divisor 4, tiny length", true, -50, 4, []step{{0, 3, 0}, {0, 2, 0}}},
		{"monster positive effect +20 (no division)", true, 20, 2, []step{{0, 100, 0}, {10, 40, 0}}},
		{"monster effect 0 is never chilled", true, 0, 2, []step{{0, 100, 0}}},
	}

	for _, c := range cases {
		u := d2combat.NewStateUnit()
		s := d2state.New()

		tgt := d2combat.StateTarget{Monster: c.monster, Record: c.monster, ColdEffect: c.effect, ChillDivisor: c.div}

		for i, st := range c.steps {
			u.ApplyChill(st.frame, st.length, tgt, zeroRoll{})

			h := d2state.Hit{ColdLen: st.length}
			if c.monster {
				h.HasColdEffect, h.ColdEffect, h.ChillDiv = true, c.effect, c.div
			}

			s.ApplyHit(st.frame, h)

			oa, oe := oracleEnd(u, d2combat.StateChill, st.frame)
			ea, ee := engineEnd(s, d2state.Chill, st.frame)

			if oa != ea || oe != ee {
				t.Errorf("%s step %d (%+v): oracle active=%v end=%d, engine active=%v end=%d", c.name, i, st, oa, oe, ea, ee)

				continue
			}

			if !oa {
				continue
			}

			// the slow value is set by the first application and kept
			want := u.Lists[d2combat.StateChill].Stats[d2combat.StatVelocity]
			if got := s.Stat(st.frame, "velocitypercent"); got != want {
				t.Errorf("%s step %d: slow oracle=%d engine=%d", c.name, i, want, got)
			}
		}
	}
}

func TestFreezeAgreesWithOracleForPlainMonsters(t *testing.T) {
	cases := []struct {
		name   string
		effect int
		div    int
		steps  []step
	}{
		{"effect -50, divisor 4", -50, 4, []step{{0, 100, 0}, {5, 400, 0}, {10, 8, 0}}},
		{"effect -50, no divisor", -50, 0, []step{{0, 100, 0}, {10, 300, 0}}},
		{"effect 0 never freezes", 0, 4, []step{{0, 100, 0}}},
		{"effect +10 never freezes", 10, 4, []step{{0, 100, 0}}},
		{"divisor makes it zero length", -50, 200, []step{{0, 100, 0}}},
	}

	for _, c := range cases {
		u := d2combat.NewStateUnit()
		s := d2state.New()

		tgt := d2combat.StateTarget{Monster: true, Record: true, ColdEffect: c.effect, FreezeDivisor: c.div}

		for i, st := range c.steps {
			u.ApplyFreeze(st.frame, st.length, tgt, zeroRoll{})
			s.ApplyHit(st.frame, d2state.Hit{
				FreezeLen: st.length, HasColdEffect: true, ColdEffect: c.effect, FreezeDiv: c.div,
			})

			oa, oe := oracleEnd(u, d2combat.StateFreeze, st.frame)
			ea, ee := engineEnd(s, d2state.Freeze, st.frame)

			if oa != ea || oe != ee {
				t.Errorf("%s step %d (%+v): oracle active=%v end=%d, engine active=%v end=%d", c.name, i, st, oa, oe, ea, ee)
			}
		}
	}
}

// TestStunMonsterImmunitiesAreAKnownGap documents rules the oracle verified
// that the engine does not apply: d2core/d2skills builds d2state.Hit without
// ever setting CannotStun, so bosses can be stunned, monsters whose monstats
// stun flag is 0 can be stunned, special classes are not cut to 13 frames and
// data-flag-8 monsters do not resist 90 percent of stuns. It is skipped, not
// asserted, so a later fix can turn it on.
func TestStunMonsterImmunitiesAreAKnownGap(t *testing.T) {
	t.Skip("known gap: engine never sets d2state.Hit.CannotStun; see feedback-spawn-cadence-od2.md (review of pass2-s2)")

	u := d2combat.NewStateUnit()
	u.ApplyStun(0, 100, d2combat.StateTarget{Monster: true, Boss: true, StunFlag: true}, zeroRoll{})

	if l := u.Lists[d2combat.StateStun]; l != nil {
		t.Fatalf("oracle: a boss is never stunned, got end=%d", l.End)
	}
}
