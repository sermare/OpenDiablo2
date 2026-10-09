package d2combat

import (
	"encoding/json"
	"testing"
)

type misCase struct {
	Rec                                                                  []int
	Mid, Lvl, Fl, Velo, Rngo, Sub, P40, P44, Slow, SlowPct, Dx, Dy, Dist int
	HasTgt, OK                                                           interface{}
	Cap                                                                  map[string][][]int
}

func (m *misCase) UnmarshalJSON(buf []byte) error {
	return json.Unmarshal(buf, &[]interface{}{&m.Rec, &m.Mid, &m.Lvl, &m.Fl, &m.Velo, &m.Rngo, &m.Sub, &m.P40, &m.P44, &m.Slow, &m.SlowPct,
		&m.Dx, &m.Dy, &m.Dist, &m.HasTgt, &m.OK, &m.Cap})
}

// TestOracleMissileCreate checks MissileCreateTiming against the real
// MISSILE_CreateServerMissile (0x59d5d0) run in the emulator for random
// missiles.txt records and create structs (numbers only).
func TestOracleMissileCreate(t *testing.T) {
	var g struct {
		Cases []misCase `json:"cases"`
	}

	loadJSON(t, "missile_create_golden.json", &g)

	bad, n := 0, 0

	for _, c := range g.Cases {
		ok, _ := c.OK.(float64)
		if ok != 1 {
			continue
		}

		n++

		r := c.Rec
		in := MissileCreateIn{
			Flags: uint32(c.Fl), Level: c.Lvl, VelOverride: c.Velo, RangeOverride: c.Rngo, SubLoops: c.Sub, Param40: c.P40, Param44: c.P44,
			Vel: uint8(r[0]), VelLev: uint8(r[1]), Range: int16(r[2]), LevRange: int16(r[3]), MaxVel: uint8(r[4]), Accel: int16(r[5]),
			SubLoop: r[6] != 0, SubStart: uint8(r[7]), SubStop: uint8(r[8]), Byte136: uint8(r[9]),
			CanSlow: r[10] == 1, OwnerSlowed: c.Slow == 1, SlowPct: c.SlowPct, LobDist: c.Dist,
		}
		got := MissileCreateTiming(in)

		want := MissileCreateOut{}
		if p := c.Cap["path50"]; len(p) > 1 {
			want.Velocity = p[1][1]
		}

		want.Life = c.Cap["life1"][0][1]
		want.Life3 = c.Cap["life3"][0][1]
		want.Accel = c.Cap["accel"][0][1]
		want.MaxVel = c.Cap["maxvel"][0][1]

		if l := c.Cap["life2"]; len(l) > 1 {
			want.LobLife = l[1][1]
		}

		if got.Velocity != want.Velocity || got.Life != want.Life || got.Life3 != want.Life3 || got.Accel != want.Accel ||
			got.MaxVel != want.MaxVel || got.LobLife != want.LobLife {
			bad++

			if bad <= 8 {
				t.Errorf("create %v fl=%#x lvl=%d: got %+v want %+v", c.Rec, c.Fl, c.Lvl, got, want)
			}
		}
	}

	if n < 1000 {
		t.Fatalf("only %d created-missile cases", n)
	}

	if bad > 0 {
		t.Errorf("%d of %d create cases differ", bad, n)
	}
}
