package d2combat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The golden file holds numbers only, produced by running the real Game.exe
// 1.14b functions (GetDefense 0x6225a0, GetPlayerAttackRating 0x622710,
// RollToHit 0x57b9c0, GetBlockChance 0x6228d0, GetEffectiveResist 0x579b10) in
// an x86 emulator with fake units (stat reads served from tables).
type combatGolden struct {
	Defense     [][]int `json:"defense"`     // ac dex pct1 pct2 b6 -> def
	ClassBase   []int   `json:"classbase"`   // class -> AR at dex 7
	BlockFactor []int   `json:"blockfactor"` // class -> block factor
	ToHit       [][]int `json:"tohit"`       // atype cls ar adex arpct bonus ac ddex vs miss alvl dlvl dpct -> chance
	Block       [][]int `json:"block"`       // cls tb dex lvl inc -> chance
	Resist      [][]int `json:"resist"`      // stat maxstat pierce res max pv ign expn diff penv -> res
}

func loadCombatGolden(t *testing.T) combatGolden {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join("testdata", "combat_golden.json"))
	if err != nil {
		t.Fatal(err)
	}

	var g combatGolden
	if err = json.Unmarshal(buf, &g); err != nil {
		t.Fatal(err)
	}

	return g
}

func TestOracleDefense(t *testing.T) {
	g := loadCombatGolden(t)

	for _, c := range g.Defense {
		if got := DefenseWithFinalPct(c[0], c[1], c[2]+c[3], c[4]); got != c[5] {
			t.Errorf("defense ac=%d dex=%d pct=%d+%d b6=%d: got %d want %d", c[0], c[1], c[2], c[3], c[4], got, c[5])
		}
	}
}

func TestOracleToHit(t *testing.T) {
	g := loadCombatGolden(t)
	bad := 0

	for _, c := range g.ToHit {
		atype, cls, ar, adex, arpct, bonus, ac, ddex, vs, _, alvl, dlvl, dpct, want := c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7], c[8], c[9], c[10], c[11], c[12], c[13]

		in := ToHitInput{
			Defense:       Defense(ac, ddex, dpct) + vs,
			AttackerLevel: alvl, DefenderLevel: dlvl,
		}

		if atype == 0 {
			in.AttackRating = PlayerAttackRating(ar, adex, g.ClassBase[cls]-5*(7-7)) // classbase measured at dex 7
			in.AttackRatingPct = arpct + bonus
		} else {
			in.AttackRating = MonsterAttackRating(ar, bonus, adex)
			in.AttackRatingPct = arpct
		}

		if got := ToHitChance(in); got != want {
			bad++

			if bad <= 8 {
				t.Errorf("tohit %v: got %d want %d", c, got, want)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d to-hit cases differ", bad, len(g.ToHit))
	}
}

func TestOracleBlock(t *testing.T) {
	g := loadCombatGolden(t)

	for _, c := range g.Block {
		cls, tb, dex, lvl, inc, want := c[0], c[1], c[2], c[3], c[4], c[5]

		got := PlayerBlockChance(PlayerBlockInput{
			HasShield: true, ToBlock: tb, ClassBlockFactor: g.BlockFactor[cls],
			Dex: dex, Level: lvl, IncludeDex: inc == 1,
		})
		if got != want {
			t.Errorf("block %v: got %d want %d", c, got, want)
		}
	}
}

func TestOracleResist(t *testing.T) {
	g := loadCombatGolden(t)
	bad := 0

	for _, c := range g.Resist {
		stat, maxStat, pierce, res, mx, pv, ign, expn, diff, penv, want := c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7], c[8], c[9], c[10]

		in := ResistInput{
			Resist: res, Pierce: pv, HasPierce: pierce != -1,
			MaxResistBonus: mx, HasMaxResist: maxStat != -1,
			IsPhysical:          stat == 36,
			NoDifficultyPenalty: stat == 36 || stat == 37,
			Ignore:              ign == 1,
		}

		if expn == 1 {
			in.DifficultyPenalty = penv
		} else {
			in.DifficultyPenalty = ClassicResistPenalty(diff)
		}

		if got := EffectiveResist(in); got != want {
			bad++

			if bad <= 8 {
				t.Errorf("resist %v: got %d want %d", c, got, want)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d resist cases differ", bad, len(g.Resist))
	}
}
