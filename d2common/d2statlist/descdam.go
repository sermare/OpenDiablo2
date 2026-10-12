package d2statlist

import "strconv"

// The per skill damage functions of the character screen's hand skill boxes (wave 47). SKILLDESC_DrawSkillDescription
// (0x4ea490) calls the entry of the function table at 0x72bf20 selected by skilldesc.txt descdam (index = descdam,
// 1..24), passing the unit, the skill, the skills.txt record, the skill level and the box anchors; the function computes
// the min and max and prints them through SKILLDESC_AppendMinMaxText (0x4e6870). Ported here (VERIFIED in Game.exe
// 1.14b by disassembly unless marked UNVERIFIED): descdam 5 (0x4e7ce0, the most common one) and descdam 7 (0x4e6fd0),
// with their helpers SKILLDESC_ApplyElementalDamageBonuses (0x4e5b00), SKILLDESC_ComputeAndApplyDamageBonuses
// (0x4e67a0) and SKILLDESC_ComputeScaledDamageBonuses (0x4e67f0). The other table entries are not ported: Go keeps
// the generic weapon box (SkillDamageBox) for them.

// Box colours beyond white / red / blue that the elemental part sets (the exe's font colour indexes).
const (
	BoxGreen  BoxColour = 2 // poison
	BoxYellow BoxColour = 9 // lightning
)

// ElemStat is one elemental damage stat triple of the unit: min and max damage and the mastery percent stat.
type ElemStat struct{ Min, Max, Pct int }

// ElemTable lists the stat ids the elemental helper 0x4e5b00 reads (table at 0x6dcf1c, VERIFIED): min, max and the
// percent stat per element (-1: none) and the colour the box takes when the element contributes (table 0x6dcf4c).
var ElemTable = [4]struct {
	MinID, MaxID, PctID int
	Colour              BoxColour
}{
	{0x30, 0x31, 0x149, BoxRed},    // fire
	{0x32, 0x33, 0x14a, BoxYellow}, // lightning
	{0x36, 0x37, 0x14b, BoxBlue},   // cold
	{0x34, 0x35, -1, BoxBlue},      // magic
}

// Poison stats read by 0x4e5b00: min / max per frame (0x39 / 0x3a), mastery percent 0x14c, the length stat 0x65 and,
// when that is not positive, stat 0x3b divided by max(stat 0x146, 1).
const (
	StatPoisonLenDefault                           = 0x3b
	StatPoisonPct, StatPoisonLen, StatPoisonLenDiv = 0x14c, 0x65, 0x146
)

// ElemStats are the elemental damage numbers of the whole stat list (items, skills, states), filled by Compute.
type ElemStats struct {
	Elem   [4]ElemStat // fire, lightning, cold, magic in the order of ElemTable
	Poison ElemStat
	// PoisonLen is the poison duration in frames after the 0x65 / 0x3b / 0x146 rule above.
	PoisonLen int
}

func elemStatsOf(list *List) ElemStats {
	var e ElemStats

	for i, row := range ElemTable {
		e.Elem[i] = ElemStat{Min: int(list.Get(row.MinID)), Max: int(list.Get(row.MaxID))}
		if row.PctID >= 0 {
			e.Elem[i].Pct = int(list.Get(row.PctID))
		}
	}

	e.Poison = ElemStat{Min: int(list.Get(StatPoisonMin)), Max: int(list.Get(StatPoisonMax)), Pct: int(list.Get(StatPoisonPct))}

	e.PoisonLen = int(list.Get(StatPoisonLen))
	if e.PoisonLen < 1 {
		div := int(list.Get(StatPoisonLenDiv))
		if div < 2 {
			div = 1
		}

		e.PoisonLen = int(list.Get(StatPoisonLenDefault)) / div
	}

	return e
}

// DamageAcc is the (min, max, colour) accumulator the exe's helpers add to.
type DamageAcc struct {
	Min, Max int
	Colour   BoxColour
}

// pctOf is v*p/100 with the exe's truncating division (the 0x100000 / 0x10000 branches of the exe only avoid
// overflow and give the same value in range).
func pctOf(v, p int) int { return v * p / 100 }

// ApplyElemental is SKILLDESC_ApplyElementalDamageBonuses (0x4e5b00): the unit's elemental damage stats are added to
// the accumulator (each element: min and max raised by their mastery percent, min lowered to max when it exceeds
// it), poison adds its per frame damage times its length >> 8 and turns the colour green, a positive max raises min
// to at least 1 and max to at least min + 1, a bit 8 skill adds the state based elemental damage through stateElem,
// and the dmgblue / dmgred state masks set the colour (red wins). UNVERIFIED: that the colour slot is the one the
// weapon part (0x4e5860) also writes (assumed from the shared pointer).
func (t *Totals) ApplyElemental(acc *DamageAcc, bit8 bool, stateElem func(*DamageAcc), dmgBlue, dmgRed bool) {
	for i, row := range ElemTable {
		e := t.Elem.Elem[i]
		lo, hi := e.Min, e.Max

		if row.PctID >= 0 && e.Pct != 0 {
			lo += pctOf(lo, e.Pct)
			hi += pctOf(hi, e.Pct)
		}

		if hi <= lo {
			lo = hi
		}

		acc.Min += lo
		acc.Max += hi

		if lo != 0 || hi != 0 {
			acc.Colour = row.Colour
		}
	}

	if p := t.Elem.Poison; p.Max != 0 {
		lo, hi := p.Min, p.Max
		if p.Pct != 0 {
			lo += pctOf(lo, p.Pct)
			hi += pctOf(hi, p.Pct)
		}

		acc.Colour = BoxGreen
		acc.Min += (t.Elem.PoisonLen * lo) >> 8
		acc.Max += (t.Elem.PoisonLen * hi) >> 8
	}

	if acc.Max > 0 {
		if acc.Min < 1 {
			acc.Min = 1
		}

		if acc.Max < acc.Min+1 {
			acc.Max = acc.Min + 1
		}
	}

	if bit8 && stateElem != nil {
		stateElem(acc)
	}

	if dmgBlue {
		acc.Colour = BoxBlue
	}

	if dmgRed {
		acc.Colour = BoxRed
	}
}

// DescDamInput is everything a descdam function needs besides the unit's stat totals.
type DescDamInput struct {
	// DescDam is skilldesc.txt descdam (index of the function table at 0x72bf20).
	DescDam int
	// SrcDam is skills.txt srcdam (+0x1a5).
	SrcDam int
	// EType is the skills.txt elemental type (+0x1dc): 1 fire, 2 lightning, 3 magic, 4 cold, 5 poison.
	EType int
	// Calc1 and Calc2 are the evaluated skilldesc.txt "ddam calc1" / "ddam calc2" (descdam 7: the percent that joins
	// the pool and the flat damage added after the percent).
	Calc1, Calc2 int
	// PhysMin and PhysMax are SKILLDESC_CalcSkillMinDamage / MaxDamage (0x648f90 / 0x6490d0, weapon flag 0) shifted
	// right by 8; ElemMin, ElemMax and ElemLen are SKILL_GetElementalMin/Max/Length (0x646100 / 0x646200 / 0x6462f0,
	// flag 1) in the exe's 24.8 form (the function shifts them itself).
	PhysMin, PhysMax          int
	ElemMin, ElemMax, ElemLen int
	// Bit8 is skills.txt flag word 4 bit 3: the column "finishing" (UNVERIFIED by value; the loader table at
	// DATATBL_LoadSkillsTable 0x615e53 stores the column at dword 4, bit 3, read from the immediates).
	Bit8       bool
	StateBonus int // SKILLDESC_GetStateBasedDamageBonus (0x4e4030), see StateBonusOf
	StateElem  func(*DamageAcc)
	DmgBlue    bool
	DmgRed     bool
	Mastery    int
}

// ElemColour is SKILLDESC_MapElementTypeToDamageType (0x4e3ff0): skills.txt etype to the starting font colour.
func ElemColour(etype int) BoxColour {
	switch etype {
	case 1:
		return BoxRed
	case 2:
		return BoxYellow
	case 3, 4:
		return BoxBlue
	case 5:
		return BoxGreen
	}

	return BoxWhite
}

// weaponElem is SKILLDESC_ComputeWeaponDamageRange at srcdam 0x80 followed by ApplyElemental (0x4e67a0 and the
// first half of 0x4e67f0).
func (t *Totals) weaponElem(in DescDamInput, poolBonus, extra int, initColour BoxColour) DamageAcc {
	w := t.SkillDamageBox(SkillBox{
		SrcDam: 128, Extra: extra, PoolBonus: poolBonus, Bit8: in.Bit8, StateBonus: in.StateBonus,
		DmgBlue: in.DmgBlue, DmgRed: in.DmgRed, Mastery: in.Mastery, InitColour: initColour,
	})
	acc := DamageAcc{Min: w.Min, Max: w.Max, Colour: w.Colour}
	t.ApplyElemental(&acc, in.Bit8, in.StateElem, in.DmgBlue, in.DmgRed)

	return acc
}

// DescDamBox runs descdam function 5 or 7 for the character screen. ok is false for the other indexes.
//
// descdam 5 (0x4e7ce0): colour = ElemColour(etype); when srcdam != 0 the weapon box at full range plus the unit's
// elemental damage is scaled by srcdam/128 (truncating, 0x4e67f0); then the skill's physical min / max (>>8) and its
// elemental min / max (>>8) are added, a poison skill (etype 5) multiplying the elemental part by its length.
//
// descdam 7 (0x4e6fd0): the weapon box at full range with ddam calc1 added to the percent pool and ddam calc2 added
// after it, plus the unit's elemental damage, scaled by srcdam/128 when srcdam != 128 (MulDiv, truncating), then the
// skill's physical and elemental parts (>>8).
func (t *Totals) DescDamBox(in DescDamInput) (DamageBox, bool) {
	var lo, hi int

	col := BoxWhite

	switch in.DescDam {
	case 5:
		col = ElemColour(in.EType)

		if in.SrcDam != 0 {
			acc := t.weaponElem(in, 0, 0, col)
			col = acc.Colour
			lo, hi = acc.Min*in.SrcDam/128, acc.Max*in.SrcDam/128
		}

		lo += in.PhysMin
		hi += in.PhysMax

		if in.EType == 5 {
			lo += (in.ElemMin * in.ElemLen) >> 8
			hi += (in.ElemMax * in.ElemLen) >> 8
		} else {
			lo += in.ElemMin >> 8
			hi += in.ElemMax >> 8
		}
	case 7:
		acc := t.weaponElem(in, in.Calc1, in.Calc2, BoxWhite)
		col = acc.Colour
		lo, hi = acc.Min, acc.Max

		if in.SrcDam != 128 {
			lo, hi = lo*in.SrcDam/128, hi*in.SrcDam/128
		}

		lo += in.PhysMin + in.ElemMin>>8
		hi += in.PhysMax + in.ElemMax>>8
	default:
		return DamageBox{}, false
	}

	return DamageBox{Min: lo, Max: hi, Colour: col}, true
}

// MinMaxText is SKILLDESC_AppendMinMaxText (0x4e6870): the box always prints a range; an upper end not above the
// lower one becomes lower + 1.
func MinMaxText(lo, hi int) string {
	if hi <= lo {
		hi = lo + 1
	}

	return strconv.Itoa(lo) + "-" + strconv.Itoa(hi)
}

// StateCharge is one charge-up state of the unit as SKILLDESC_GetStateBasedDamageBonus (0x4e4030) reads it: the
// builder skill that put it there and its level (stats 0x15e / 0x15f), and the value of the builder's aurastat1
// stat on the state's list (the charge count).
type StateCharge struct {
	PrgDam      int // builder's skills.txt prgdam; only 1 counts
	Param1      int // builder's skills.txt param1 and param2
	Param2      int
	Level       int
	Count       int
	HasAuraStat bool // aurastat1 >= 0
}

// StateBonusOf is SKILLDESC_GetStateBasedDamageBonus: the sum over charge states with a positive level and count of
// SKILLS_GetBasePlusPerLevelValue (param1 + (level-1)*param2) times the count, for prgdam 1 builders only.
func StateBonusOf(charges []StateCharge) int {
	sum := 0

	for _, c := range charges {
		if c.Level < 1 || c.Count < 1 || c.PrgDam != 1 || !c.HasAuraStat {
			continue
		}

		sum += (c.Param1 + (c.Level-1)*c.Param2) * c.Count
	}

	return sum
}
