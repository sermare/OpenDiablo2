package d2drop

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// w44: statistical checks of the drop function against the distribution
// derived from the exe rules (ITEMGEN_GetTreasureClassByLevel 6567f0,
// ITEMGEN_InitItemBaseStats 555b10, ITEMGEN_ApplyGoldFindBonus 556a10).

// TestW44UpgradeInvariantsRealTables checks, for every grouped class of the
// real TreasureClassEx and every level 1..110, the exe walk: the result is in
// the same group and its level is <= the monster level unless it is the start
// class itself.
func TestW44UpgradeInvariantsRealTables(t *testing.T) {
	rt := loadReal(t)
	grouped := 0

	for _, tc := range rt.tcs.list {
		if tc.Group == 0 {
			continue
		}

		grouped++

		for lvl := 1; lvl <= 110; lvl++ {
			got := rt.tcs.Upgrade(tc, lvl)
			if got.Group != tc.Group {
				t.Fatalf("%s lvl %d left its group", tc.Name, lvl)
			}

			if got != tc && got.Level > lvl {
				t.Fatalf("%s lvl %d -> %s level %d above the monster", tc.Name, lvl, got.Name, got.Level)
			}
		}
	}

	if grouped == 0 {
		t.Fatal("no grouped class in the real table")
	}

	t.Logf("%d grouped classes checked", grouped)
}

// TestW44UpgradedRollMatchesAnalytic: rolling a grouped base class with a
// monster level draws from the upgraded class, so the empirical mean drop
// count equals the analytic mean of Upgrade(start, level).
func TestW44UpgradedRollMatchesAnalytic(t *testing.T) {
	rt := loadReal(t)
	d := &Dropper{TCs: rt.tcs}
	rng := d2rand.New(44)
	compared := 0

	for _, tc := range rt.tcs.list {
		if tc.Group == 0 || tc.Picks <= 0 || compared >= 12 {
			continue
		}

		for _, lvl := range []int{5, 40, 85} {
			up := rt.tcs.Upgrade(tc, lvl)
			want := expectedDrops(rt.tcs, up.Name, 1, 0)

			if math.IsNaN(want) || want > 3 || want < 0.2 {
				continue
			}

			const n = 6000

			total := 0

			for i := 0; i < n; i++ {
				out, err := d.Roll(&Context{RNG: rng, ILvl: lvl, Players: 1, UpgradeLevel: lvl}, tc.Name)
				if err != nil {
					t.Fatal(err)
				}

				total += len(out)
			}

			got := float64(total) / n
			if math.Abs(got-want) > 0.06+0.08*want {
				t.Errorf("%s lvl %d (-> %s): mean %.3f, analytic %.3f", tc.Name, lvl, up.Name, got, want)
			}

			compared++
		}
	}

	if compared == 0 {
		t.Skip("no suitable grouped class")
	}
}

// TestW44GoldDistribution: GoldAmount is uniform on [ilvl, 6*ilvl-1]
// (rand(5*ilvl)+ilvl); with the multiplier and gold find the mean is
// scaled by mul/256 and (100+gf)/100.
func TestW44GoldDistribution(t *testing.T) {
	for _, ilvl := range []int{1, 10, 45, 99} {
		rng := d2rand.New(uint32(ilvl))
		n := 40000
		min, max := 1<<30, 0
		sum := 0.0
		gfSum := 0.0

		for i := 0; i < n; i++ {
			g := GoldAmount(rng, ilvl, 0)
			if g < min {
				min = g
			}

			if g > max {
				max = g
			}

			sum += float64(g)
			gfSum += float64(ApplyGoldFind(ScaleGoldMul(g, 512), 150, 0))
		}

		lo, hi := ilvl, 6*ilvl-1
		if ilvl == 1 {
			hi = 5
		}

		if min < lo || max > hi {
			t.Errorf("ilvl %d: range %d..%d outside %d..%d", ilvl, min, max, lo, hi)
		}

		want := float64(lo+hi) / 2
		if math.Abs(sum/float64(n)-want) > 0.02*want+0.1 {
			t.Errorf("ilvl %d: mean %.2f want %.2f", ilvl, sum/float64(n), want)
		}

		wantGF := want * 2 * 2.5
		if math.Abs(gfSum/float64(n)-wantGF) > 0.03*wantGF+1 {
			t.Errorf("ilvl %d: mul2 gf150 mean %.2f want %.2f", ilvl, gfSum/float64(n), wantGF)
		}
	}
}
