package d2monstats

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// Difficulty indexes: normal, nightmare, hell (game+0x6d in the exe).
const (
	Normal = iota
	Nightmare
	Hell
	numDiff
)

// Attack identifies a monster attack slot.
type Attack int

// Attack slots.
const (
	A1 Attack = iota
	A2
	S1
	numAttacks
)

// LevelRow is one monlvl.txt row, indexed [expansion][difficulty].
type LevelRow struct {
	AC, TH, HP, DM, XP [2][numDiff]int
}

// MonLvl is the monlvl.txt table indexed by monster level.
type MonLvl []LevelRow

// AttackStats is a to-hit value with a damage range.
type AttackStats struct{ TH, Min, Max int }

// Class holds the scaling columns of one monstats row.
type Class struct {
	ID            string
	Index         int
	NoRatio, Boss bool
	// PrimeEvil is the primeevil column; see LevelExclusion.
	PrimeEvil bool
	// Align is the monstats Align column (+0x4c): 0 enemy, 1 friendly, 2 neutral.
	// Nonzero skips player-count scaling (0x00571760); 1 skips the classic
	// adjustment (0x0063ff30). Column identity inferred from the data.
	Align        int
	Level        [numDiff]int
	MinHP, MaxHP [numDiff]int
	AC, Exp      [numDiff]int
	Attacks      [numAttacks][numDiff]AttackStats
}

// Stats is the result of Scale.
type Stats struct {
	Level        int
	HP           int // rolled between HPMin and HPMax
	HPMin, HPMax int
	AC, TH       int // TH is the A1 attack rating
	XP           int
	HP256        int // exe stat 6/7 value, HP<<8
	Players      int // player count recorded at spawn (exe stat 0x64), min 1
	Attacks      [numAttacks]AttackStats
}

var diffSuffix = [numDiff]string{"", "(N)", "(H)"}

var atk = [numAttacks]string{"A1", "A2", "S1"}

// LoadMonLvl parses monlvl.txt.
func LoadMonLvl(buf []byte) (MonLvl, error) {
	d := d2txt.LoadDataDictionary(buf)

	var t MonLvl

	for d.Next() {
		var r LevelRow

		for exp := 0; exp < 2; exp++ {
			pre := ""
			if exp == 1 {
				pre = "L-"
			}

			for di := 0; di < numDiff; di++ {
				k := func(c string) int { return d.Number(pre + c + diffSuffix[di]) }
				r.AC[exp][di], r.TH[exp][di], r.HP[exp][di] = k("AC"), k("TH"), k("HP")
				r.DM[exp][di], r.XP[exp][di] = k("DM"), k("XP")
			}
		}

		if d.Number("Level") != len(t) {
			return nil, fmt.Errorf("monlvl: row %d has Level %d", len(t), d.Number("Level"))
		}

		t = append(t, r)
	}

	if d.Err != nil {
		return nil, d.Err
	}

	if len(t) == 0 {
		return nil, errors.New("monlvl: no rows")
	}

	return t, nil
}

// LoadClasses parses the scaling columns of monstats.txt, keyed by Id.
func LoadClasses(buf []byte) (map[string]*Class, error) {
	d := d2txt.LoadDataDictionary(buf)
	out := map[string]*Class{}

	for d.Next() {
		c := &Class{
			ID: d.String("Id"), Index: d.Number("hcIdx"),
			NoRatio: d.Number("noRatio") != 0, Boss: d.Number("boss") != 0,
			PrimeEvil: d.Number("primeevil") != 0, Align: d.Number("Align"),
		}

		for di := 0; di < numDiff; di++ {
			s := diffSuffix[di]
			c.Level[di] = d.Number("Level" + s)
			c.MinHP[di], c.MaxHP[di] = d.Number("MinHP"+s), d.Number("MaxHP"+s)

			if di == Normal { // the normal columns are spelled minHP / maxHP
				c.MinHP[di], c.MaxHP[di] = d.Number("minHP"), d.Number("maxHP")
			}

			c.AC[di], c.Exp[di] = d.Number("AC"+s), d.Number("Exp"+s)

			for a := Attack(0); a < numAttacks; a++ {
				c.Attacks[a][di] = AttackStats{
					TH:  d.Number(atk[a] + "TH" + s),
					Min: d.Number(atk[a] + "MinD" + s),
					Max: d.Number(atk[a] + "MaxD" + s),
				}
			}
		}

		out[c.ID] = c
	}

	if d.Err != nil {
		return nil, d.Err
	}

	return out, nil
}

// MulDiv is the exe's multiply-divide (0x0047f2c0), VERIFIED to TRUNCATE
// toward zero (IMUL then IDIV, no half-way bias) for the value ranges monster
// scaling uses; it is not the Win32 rounding MulDiv.
func MulDiv(a, b, c int) int {
	return a * b / c
}

// maxHP is the cap on the pre-shift hit points (0x7fffff, stored <<8).
const maxHP = 0x7fffff

// ResolveLevel is the monster level rule (VERIFIED at 0x00571af0): the
// monstats Level of the difficulty is the default; only in Nightmare and Hell,
// and only when an area is known, non-noRatio non-boss classes take the area
// MonLvl instead. Normal difficulty always uses the monstats Level.
func (c *Class) ResolveLevel(diff, areaLevel int) int {
	if diff <= Normal || c.NoRatio || c.excluded() || areaLevel <= 0 {
		return c.Level[diff]
	}

	return areaLevel
}

// Scale computes the stats of a monster of class c. roll(n) must return a
// uniform integer in [0,n); it picks the hit points between the class' min
// and max. expansion selects the Lord of Destruction monlvl columns.
func (t MonLvl) Scale(c *Class, diff, areaLevel int, expansion bool, roll func(n int) int) Stats {
	return t.ScaleOpts(c, diff, areaLevel, expansion, roll, Options{})
}

// ScaleOpts is Scale plus the player-count bonus and the optional classic-mode
// adjustment (see Options).
func (t MonLvl) ScaleOpts(c *Class, diff, areaLevel int, expansion bool, roll func(n int) int, o Options) Stats {
	diff = clamp(diff, 0, Hell)
	lvl := c.ResolveLevel(diff, areaLevel)
	s := Stats{Level: lvl}
	row := t[clamp(lvl, 0, len(t)-1)]

	e := 0
	if expansion {
		e = 1
	}

	scale := func(base, pct int) int {
		if c.NoRatio {
			return pct
		}

		return MulDiv(base, pct, 100)
	}

	lo, hi := c.MinHP[diff], c.MaxHP[diff]
	if hi < lo {
		hi = lo
	}

	s.HPMin, s.HPMax = scale(row.HP[e][diff], lo), scale(row.HP[e][diff], hi)
	s.HP = s.HPMin

	if s.HPMax > s.HPMin && roll != nil {
		s.HP += roll(s.HPMax - s.HPMin + 1)
	}

	hpPct, xpPct, players := PlayerBonus(o.Players, c.Align)
	s.Players = players

	// VERIFIED 0x00571af0: the bonus is added to (min+roll) BEFORE the cap.
	s.HP += MulDiv(s.HP, hpPct, 100)
	if s.HP > maxHP {
		s.HP = maxHP
	}

	s.HP256 = s.HP << 8

	s.AC = scale(row.AC[e][diff], c.AC[diff])
	s.XP = scale(row.XP[e][diff], c.Exp[diff])
	s.XP += MulDiv(s.XP, xpPct, 100)

	for a := Attack(0); a < numAttacks; a++ {
		in := c.Attacks[a][diff]
		s.Attacks[a] = AttackStats{
			TH:  scale(row.TH[e][diff], in.TH),
			Min: scale(row.DM[e][diff], in.Min),
			Max: scale(row.DM[e][diff], in.Max),
		}
	}

	s.TH = s.Attacks[A1].TH

	if o.Classic && diff > Normal && c.Align != 1 {
		s.applyClassic(c, diff)
	}

	return s
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
