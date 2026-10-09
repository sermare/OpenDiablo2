package d2combat

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type stateSeq struct {
	T     []json.RawMessage
	Seed  []uint32
	Steps [][]json.RawMessage
	Fin   map[string][]json.RawMessage
	Bits  [][]int
	Seed2 []uint32
	TSeed []uint32
	TSd2  []uint32
}

func (s *stateSeq) UnmarshalJSON(buf []byte) error {
	var m struct {
		T      []json.RawMessage            `json:"t"`
		Seed   []uint32                     `json:"seed"`
		Steps  [][]json.RawMessage          `json:"steps"`
		Fin    map[string][]json.RawMessage `json:"fin"`
		Bits   [][]int                      `json:"bits"`
		Seed2  []uint32                     `json:"seed2"`
		TSeed  []uint32                     `json:"tseed"`
		TSeed2 []uint32                     `json:"tseed2"`
	}

	if err := json.Unmarshal(buf, &m); err != nil {
		return err
	}

	s.T, s.Seed, s.Steps, s.Fin, s.Bits, s.Seed2, s.TSeed, s.TSd2 = m.T, m.Seed, m.Steps, m.Fin, m.Bits, m.Seed2, m.TSeed, m.TSeed2

	return nil
}

func rawInt(t *testing.T, r json.RawMessage) int {
	t.Helper()

	var v int
	if err := json.Unmarshal(r, &v); err != nil {
		t.Fatal(err)
	}

	return v
}

func fmtUnit(u *StateUnit) string {
	var keys []int
	for k := range u.Lists {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	s := ""

	for _, k := range keys {
		s += fmt.Sprintf("[%d end=%d %v]", k, u.Lists[k].End, u.Lists[k].Stats)
	}

	return s
}

// TestOracleTimedStates replays random sequences of stun/poison/burn/chill/
// freeze applications against the real appliers' end frames, stats, state
// bits and random-number consumption.
func TestOracleTimedStates(t *testing.T) {
	var g struct {
		Seqs []stateSeq `json:"seqs"`
	}

	loadJSON(t, "states_golden.json", &g)

	bad := 0

	for si, s := range g.Seqs {
		var cols []int
		_ = json.Unmarshal(s.T[2], &cols)

		ttype, diff := rawInt(t, s.T[0]), rawInt(t, s.T[1])
		st := StateTarget{
			Monster: ttype == 1, Record: ttype == 1, ColdEffect: cols[diff], StunFlag: rawInt(t, s.T[3]) != 0,
			FreezeDivisor: rawInt(t, s.T[4]), ChillDivisor: rawInt(t, s.T[5]), ChillImmune: rawInt(t, s.T[6]) == 1,
			DataFlag8: rawInt(t, s.T[7]) == 1, Boss: rawInt(t, s.T[8]) == 1, Special: rawInt(t, s.T[9]) == 1,
		}

		src := &d2rand.Seed{Lo: s.Seed[0], Hi: s.Seed[1]}
		tgt := &d2rand.Seed{Lo: s.TSeed[0], Hi: s.TSeed[1]}
		u := NewStateUnit()

		for _, step := range s.Steps {
			var kind string
			_ = json.Unmarshal(step[0], &kind)

			frame, ln, val := rawInt(t, step[1]), rawInt(t, step[2]), rawInt(t, step[3])

			switch kind {
			case "stun":
				u.ApplyStun(frame, ln, st, src)
			case "poison":
				u.ApplyDot(StatePoison, frame, val, ln)
			case "burn":
				u.ApplyDot(StateBurn, frame, val, ln)
			case "chill":
				u.ApplyChill(frame, ln, st, tgt)
			case "freeze":
				u.ApplyFreeze(frame, ln, st, tgt)
			}
		}

		ok := len(u.Lists) == len(s.Fin)

		for k, v := range s.Fin {
			var id int
			_, _ = fmt.Sscanf(k, "%d", &id)

			l := u.Lists[id]
			if l == nil {
				ok = false

				continue
			}

			var stats [][]int
			_ = json.Unmarshal(v[1], &stats)

			if l.End != rawInt(t, v[0]) || len(l.Stats) != len(stats) {
				ok = false
			}

			for _, p := range stats {
				if l.Stats[p[0]] != p[1] {
					ok = false
				}
			}
		}

		on := 0

		for _, b := range s.Bits {
			if u.Bits[b[0]] != (b[1] == 1) {
				ok = false
			}

			if b[1] == 1 {
				on++
			}
		}

		for _, v := range u.Bits {
			if v {
				on--
			}
		}

		if on != 0 || src.Lo != s.Seed2[0] || src.Hi != s.Seed2[1] || tgt.Lo != s.TSd2[0] || tgt.Hi != s.TSd2[1] {
			ok = false
		}

		if !ok {
			bad++

			if bad <= 6 {
				t.Errorf("seq %d target=%v steps=%v: got %s bits=%v seed=%x:%x/%x:%x", si, st, stepsString(s.Steps), fmtUnit(u), u.Bits,
					src.Lo, src.Hi, tgt.Lo, tgt.Hi)
				t.Errorf("   want fin=%v bits=%v seed=%v/%v", s.Fin, s.Bits, s.Seed2, s.TSd2)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d state sequences differ", bad, len(g.Seqs))
	}
}

func stepsString(steps [][]json.RawMessage) string {
	s := ""

	for _, st := range steps {
		for _, f := range st {
			s += string(f) + " "
		}

		s += "| "
	}

	return s
}
