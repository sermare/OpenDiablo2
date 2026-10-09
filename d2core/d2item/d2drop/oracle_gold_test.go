package d2drop

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type goldGolden struct {
	Amount []struct {
		IL int       `json:"il"`
		F  int       `json:"f"`
		S  uint32    `json:"s"`
		A  int       `json:"a"`
		E  [2]uint32 `json:"e"`
	} `json:"amount"`
	GF []struct {
		G  int  `json:"g"`
		KF int  `json:"kf"`
		OF int  `json:"of"`
		HO int  `json:"ho"`
		HK int  `json:"hk"`
		HI int  `json:"hi"`
		R  *int `json:"r"`
	} `json:"gf"`
	MF []struct {
		K  int `json:"k"`
		KV int `json:"kv"`
		OV int `json:"ov"`
		HO int `json:"ho"`
		HU int `json:"hu"`
		R  int `json:"r"`
	} `json:"mf"`
}

func TestOracleGold(t *testing.T) {
	var g goldGolden

	readGolden(t, "gold.json", &g)

	bad := 0

	for _, c := range g.Amount {
		rng := &d2rand.Seed{Lo: c.S, Hi: 0x29a}
		if got := GoldAmount(rng, c.IL, c.F); got != c.A || rng.Lo != c.E[0] || rng.Hi != c.E[1] {
			bad++

			if bad <= 5 {
				t.Errorf("amount ilvl %d forced %d seed %d: got %d end (%d,%d), real %d end %v",
					c.IL, c.F, c.S, got, rng.Lo, rng.Hi, c.A, c.E)
			}
		}
	}

	for _, c := range g.GF {
		if c.HK == 0 || c.HI == 0 {
			if c.R != nil {
				t.Errorf("gold find without killer/item changed the gold: %+v", c)
			}

			continue
		}

		owner := 0
		if c.HO != 0 {
			owner = c.OF
		}

		if c.R == nil {
			t.Errorf("gold find case did not set the gold: %+v", c)
			continue
		}

		if got := ApplyGoldFind(c.G, c.KF, owner); got != *c.R {
			bad++

			if bad <= 8 {
				t.Errorf("gold find %+v owner %d: got %d", c, owner, got)
			}
		}
	}

	for _, c := range g.MF {
		own, owner := c.KV, 0
		if c.HO != 0 {
			owner = c.OV
		}

		want := c.R
		if c.HU == 0 {
			own, owner = 0, 0
		}

		if got := MagicFindOf(c.K, own, owner); got != want {
			bad++

			if bad <= 8 {
				t.Errorf("magic find %+v: got %d", c, got)
			}
		}
	}

	t.Logf("%d amounts, %d gold finds, %d magic finds, %d mismatches", len(g.Amount), len(g.GF), len(g.MF), bad)
}

// TestOracleGoldMultiplier compares the gold scaling of the roller: for the
// gold drops of an entry with mul=N the stat is set to amount*mul>>8.
func TestOracleGoldMultiplier(t *testing.T) {
	var g struct {
		Cases []struct {
			TC string    `json:"tc"`
			IL int       `json:"il"`
			S  uint32    `json:"s"`
			A  int       `json:"a"`
			SC []int     `json:"sc"`
			E  [2]uint32 `json:"e"`
		} `json:"cases"`
	}

	readGolden(t, "goldmul.json", &g)

	rt := loadReal(t)
	d := &Dropper{TCs: rt.tcs, Items: rt.items, Ratios: rt}
	bad, withMul := 0, 0

	for _, c := range g.Cases {
		rng := &d2rand.Seed{Lo: c.S, Hi: 0x29a}

		drops, err := d.Roll(&Context{RNG: rng, ILvl: c.IL, Players: 1}, c.TC)
		if err != nil {
			t.Fatal(err)
		}

		var got []int

		for _, dr := range drops {
			if dr.Code == GoldCode && dr.Mul != 0 {
				got = append(got, ScaleGoldMul(c.A, dr.Mul))
			}
		}

		if len(got) > 0 {
			withMul++
		}

		if fmt.Sprint(got) != fmt.Sprint(c.SC) {
			bad++

			if bad <= 5 {
				t.Errorf("%s ilvl %d seed %d amount %d: got %v, real %v", c.TC, c.IL, c.S, c.A, got, c.SC)
			}
		}
	}

	t.Logf("%d cases (%d with scaled gold), %d mismatches", len(g.Cases), withMul, bad)
}
