package d2animspeed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// animGolden holds numbers only, produced by running the real Game.exe 1.14b
// PATH_UpdateUnitVelocityAndAnimRate (0x624150) in an x86 emulator with fake
// player units (stats served from tables, AnimData speed in a fake record).
// Each case is [utype, class, mode, speed, ias, fhr, fcr, fbr, frw, s43, s44,
// s45, arg2, arg3, [monflags, mon+0x32, mon+0x36, mon+0x38], rate(+0x4c),
// rate(+0x3c), [[pathPtr, velocity]...]]. arg2 and arg3 never change a result.
type animCase struct {
	typ, class, mode, speed int
	ias, fhr, fcr, fbr, frw int
	s43, s44, s45           int
	arg2, arg3              int
	mon                     [4]int
	rate, rate3c            int
	vel                     [][2]int
}

func loadAnimGolden(t *testing.T) []animCase {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join("testdata", "anim_golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	var g struct {
		Cases [][]interface{} `json:"cases"`
	}

	if err = json.Unmarshal(buf, &g); err != nil {
		t.Fatal(err)
	}

	out := make([]animCase, 0, len(g.Cases))

	for _, c := range g.Cases {
		n := func(i int) int { return int(c[i].(float64)) }
		ac := animCase{
			typ: n(0), class: n(1), mode: n(2), speed: n(3),
			ias: n(4), fhr: n(5), fcr: n(6), fbr: n(7), frw: n(8),
			s43: n(9), s44: n(10), s45: n(11), arg2: n(12), arg3: n(13),
			rate: n(15), rate3c: n(16),
		}

		for i, v := range c[14].([]interface{}) {
			ac.mon[i] = int(v.(float64))
		}

		for _, v := range c[17].([]interface{}) {
			p := v.([]interface{})
			ac.vel = append(ac.vel, [2]int{int(p[0].(float64)), int(p[1].(float64))})
		}

		out = append(out, ac)
	}

	return out
}

// TestOracleUnitRate compares UnitRate with the real function for every
// recorded case (players and monsters, all modes 0..19).
func TestOracleUnitRate(t *testing.T) {
	cases := loadAnimGolden(t)
	if len(cases) < 3000 {
		t.Fatalf("only %d golden cases", len(cases))
	}

	bad := 0

	for i, c := range cases {
		if c.typ == 1 && c.mode > 15 { // monsters have 16 modes (0..15)
			continue
		}

		in := UnitInputs{
			Speed: c.speed, IAS: c.ias, FHR: c.fhr, FCR: c.fcr, FBR: c.fbr, FRW: c.frw,
			Stat43: c.s43, Stat44: c.s44, Stat45: c.s45,
			MonFlagBit0: c.mon[0]&1 != 0, MonVelocity: c.mon[1], MonWalkRate: c.mon[2], MonRunRate: c.mon[3],
		}

		rate, vel, hasVel := UnitRate(UnitKind(c.typ), c.mode, in)
		if rate != c.rate || hasVel != (len(c.vel) > 0) || (hasVel && vel != c.vel[0][1]) {
			bad++

			if bad < 10 {
				t.Errorf("case %d type %d mode %d: got rate %d vel %d/%v, oracle rate %d vel %v (%+v)",
					i, c.typ, c.mode, rate, vel, hasVel, c.rate, c.vel, c)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d cases differ", bad, len(cases))
	}
}

// The oracle also pins the unit-visible mirror of the rate (+0x3c) which is
// written only by the cast, attack and non-keep generic paths.
func TestOracleRate3cMirror(t *testing.T) {
	for _, c := range loadAnimGolden(t) {
		if c.typ == 1 && c.mode > 15 {
			continue
		}

		a := ModeAction(UnitKind(c.typ), c.mode)
		want := 0

		if a == RuleCast || a == RuleAttack {
			want = c.rate
		}

		if c.rate3c != want {
			t.Fatalf("type %d mode %d: +0x3c = %d, want %d (rate %d)", c.typ, c.mode, c.rate3c, want, c.rate)
		}
	}
}
