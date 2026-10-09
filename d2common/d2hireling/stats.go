package d2hireling

// MaxLevel is the highest merc level reachable by gaining experience, 98 (V,
// Game.exe: experience.txt "MaxLvl" row of class 0 = 99 is kept in
// DAT_009648e8; MERC_AwardExperience 0x57c860 gives nothing at level >= 98
// and its level-up loop stops at 98; MERC_SetLevelStats 0x570690 sets the
// next-level experience to 0 from level 98 on).
const MaxLevel = 98

// MaxLoadLevel is the highest level a save can load at: the d2s loader
// (0x568730) scans levels while level+1 <= 99, so a merc with enough stored
// experience loads at 99 although it can never earn that level (V).
const MaxLoadLevel = 99

const (
	minHP        = 40 // V
	minStrDex    = 10 // V
	maxReviveGld = 50000
)

// Stats are the per-level numbers of a merc (MERC_SetLevelStats, V).
type Stats struct {
	Level, MaxHP       int
	Str, Dex           int
	Defense, AR        int
	DmgMin, DmgMax     int
	Resist             int // applied to fire, cold, lightning and poison
	Experience, NextXP int
}

// ExpThreshold is MERC_CalcExpThreshold: the experience needed to be level
// level, (L+1)*L*L*ExpPerLvl (V).
func ExpThreshold(level, expPerLvl int) int64 {
	l := int64(level)

	return (l + 1) * l * l * int64(expPerLvl)
}

// StatsFor computes the stats of a merc of the given Id at a level. d is
// level minus the base level of the row valid at that level. Division
// truncates toward zero as in C (Go does the same).
func (t *Table) StatsFor(id, level int) (Stats, *Record) {
	r := t.Find(id, level)
	if r == nil {
		return Stats{}, nil
	}

	return r.StatsAt(level), r
}

// StatsAt computes the stats for this row at a level.
func (r *Record) StatsAt(level int) Stats {
	d := level - r.Level

	s := Stats{Level: level}
	s.MaxHP = maxInt(minHP, r.HP+r.HPPerLvl*d)
	s.Str = maxInt(minStrDex, r.Str+(r.StrLvl*d)/8)
	s.Dex = maxInt(minStrDex, r.Dex+(r.DexLvl*d)/8)
	s.Defense = maxInt(0, r.Defense+r.DefLvl*d)
	s.AR = maxInt(0, r.AR+r.ARLvl*d)
	s.DmgMin = maxInt(0, r.DmgMin+r.DmgLvl*d/8)
	s.DmgMax = maxInt(1, r.DmgMax+r.DmgLvl*d/8)
	s.Resist = maxInt(0, r.Resist+r.ResistLvl*d/4)
	s.Experience = int(ExpThreshold(level, r.ExpPerLvl))

	if level < MaxLevel {
		s.NextXP = int(ExpThreshold(level+1, r.ExpPerLvl))
	}

	return s
}

// LevelFromExp derives the merc level from experience when a save loads (the
// d2s does not store it), as MERC_LoadFromD2sHeader 0x568730 does (V): start
// at level 1 and step to level+1 while level+1 <= MaxLoadLevel and the
// threshold of level+1, computed with the ExpPerLvl of the row valid at the
// CURRENT level, is <= exp. Minimum 1.
func (t *Table) LevelFromExp(id int, exp uint32) int {
	level := 1

	for level+1 <= MaxLoadLevel {
		r := t.Find(id, level)
		if r == nil || ExpThreshold(level+1, r.ExpPerLvl) > int64(exp) {
			break
		}

		level++
	}

	return level
}

// LevelAfterGain is the level-up loop of MERC_AwardExperience 0x57c860 (V):
// from level, the ExpPerLvl of the row valid at the STARTING level is used for
// every step, and the loop never goes past MaxLevel (98).
func (t *Table) LevelAfterGain(id, level int, exp uint32) int {
	r := t.Find(id, level)
	if r == nil {
		return level
	}

	for n := level + 1; n <= MaxLevel; n++ {
		if ExpThreshold(n, r.ExpPerLvl) > int64(exp) {
			break
		}

		level = n
	}

	return level
}

// ReviveCost is MERC_CalcReviveCost: min(50000, (lvl*lvl/2)*15) (V).
func ReviveCost(level int) int {
	c := (level * level / 2) * 15
	if c > maxReviveGld {
		return maxReviveGld
	}

	return c
}

// HireCost is the price of an offer at a level: max(Gold, (delta*15+100) *
// Gold/100) where delta is the level above the row's base level (V).
func HireCost(r *Record, level int) int {
	d := level - r.Level

	return maxInt(r.Gold, (d*15+100)*r.Gold/100)
}

// OwnerLevelBand is the level a merc offered to an owner gets: owner level
// minus 1..5 (roll in 0..4 gives -5..-1), minimum 2 (V).
func OfferLevel(ownerLevel, roll int) int {
	return maxInt(2, ownerLevel+roll-5)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// StartExp is the experience a freshly hired merc of this Id and level has:
// the threshold of its level with the ExpPerLvl of the row valid there, so
// that LevelFromExp gives the level back.
func (t *Table) StartExp(id, level int) uint32 {
	r := t.Find(id, level)
	if r == nil {
		return 0
	}

	return uint32(ExpThreshold(level, r.ExpPerLvl))
}
