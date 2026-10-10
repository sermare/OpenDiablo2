package d2combat

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func traceString(ef []Effect) string {
	s := ""
	for _, e := range ef {
		s += fmt.Sprint(e.Name, e.Args, ";")
	}

	return s
}

func goldenTrace(raw [][]json.RawMessage) string {
	s := ""

	for _, t := range raw {
		var name string

		_ = json.Unmarshal(t[0], &name)

		args := []int{}

		for _, a := range t[1:] {
			var v int

			_ = json.Unmarshal(a, &v)
			args = append(args, v)
		}

		s += fmt.Sprint(name, args, ";")
	}

	return s
}

// TestOracleReaction compares ReactionEffects with the effect trace of the real 0x57ae50.
func TestOracleReaction(t *testing.T) {
	var g struct {
		React [][]json.RawMessage `json:"react"`
	}

	loadJSON(t, "react_golden.json", &g)

	bad := 0

	for _, row := range g.React {
		var tot, frame, dtype, cls, mode, s36, s15, s41, s42, s44, since, fbr, stag, skp, skd, hpbar2, pre, hp160, post int
		var hasm []int
		var trace [][]json.RawMessage

		vals := []interface{}{&dtype, &cls, &mode, &s36, &s15, &s41, &s42, &s44, &since, &fbr, &stag, &skp, &skd, &hpbar2, &hasm, &pre, &hp160, &post, &trace, &frame, &tot}
		for i, v := range vals {
			if err := json.Unmarshal(row[i], v); err != nil {
				t.Fatal(err)
			}
		}

		in := ReactionIn{
			DefenderType: dtype, ClassID: cls, Mode: mode, Result: uint32(pre), Frame: int32(frame), LastBlock: int32(frame - since), FasterBlock: int32(fbr),
			HPBarStat: int32(hp160), HPBarCalc: uint8(hpbar2), TotalPos: tot > 0,
			State36: s36 == 1, State15: s15 == 1, State41: s41 == 1, State42: s42 == 1, State44: s44 == 1,
			SkillRef: skp == 1, SkillData: skd == 1, HasKnock: hasm[0] == 1, HasBlock: hasm[1] == 1, HasHit: hasm[2] == 1,
			Stagger: func() bool { return stag == 1 },
		}
		o := ReactionEffects(in)

		want := goldenTrace(trace)
		got := traceString(o.Effects)

		// the golden records the absolute frame for setstat; normalise both
		if got != want || int(o.Result) != post {
			bad++

			if bad <= 8 {
				t.Errorf("reaction dtype=%d cls=%#x mode=%d res=%#x s36=%d s15=%d s41/42/44=%d%d%d since=%d fbr=%d skp=%d skd=%d:\n got %s res %#x\n want %s res %#x", dtype, cls, mode, pre, s36, s15, s41, s42, s44, since, fbr, skp, skd, got, o.Result, want, post)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d reaction cases differ", bad, len(g.React))
	}
}

// TestOracleStagger compares StaggerGate with the real 0x57aa60 including generator state.
func TestOracleStagger(t *testing.T) {
	var g struct {
		Stag [][]int64 `json:"stag"`
	}

	loadJSON(t, "react_golden.json", &g)

	bad := 0

	for _, c := range g.Stag {
		dtype, fr, total, poison, hc, maxLife, hasHit := c[0], c[1], c[2], c[3], c[4], c[5], c[6]
		seed := &d2rand.Seed{Lo: uint32(c[7]), Hi: uint32(c[8])}
		got := StaggerGate(seed.Step, fr == 1, int32(poison), int32(total), int(hc), int32(maxLife), dtype == 1, hasHit == 1)
		want := c[9] == 1

		if got != want || int64(seed.Lo) != c[10] || int64(seed.Hi) != c[11] {
			bad++

			if bad <= 5 {
				t.Errorf("stagger %v: got %v seed %x:%x", c, got, seed.Lo, seed.Hi)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d stagger cases differ", bad, len(g.Stag))
	}
}
