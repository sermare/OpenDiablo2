package d2summon

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// ResistCap is the highest resistance a minion can reach through a skill
// (skills.txt writes min(...,85) into the calcs; the cap is applied here too
// so a stray formula cannot exceed it).
const ResistCap = 85

// Mods are the level-scaled numbers a summoning skill adds to the template.
// Aura holds the evaluated aurastat1..6 (these apply to the minion), Passive
// the evaluated passivestat1..5 of the skill (UNVERIFIED which of the two
// groups the exe attaches to the pet: both lists of the real skills.txt only
// use stats that make sense on a minion, so both are applied alike).
type Mods struct {
	// HPPct is the life bonus in percent (calc1 of golems, Raise Skeleton and
	// the vines; d2skill.SummonOrder.HPPct).
	HPPct   int
	Aura    []d2skill.StatMod
	Passive []d2skill.StatMod
	// LevelAC and LevelAR are the armor class and attack rating that
	// SKILL_ComputeSummonLevel (0x5c2850) adds as stats 0x1f / 0x13 from the
	// MonLvl row of the summon's level (VERIFIED); see Templates.LevelBonus.
	LevelAC, LevelAR int
}

// FromOrder takes the modifiers of a d2skill summon order plus the skill's
// evaluated passive stats.
func FromOrder(o *d2skill.SummonOrder, passive []d2skill.StatMod) Mods {
	return Mods{HPPct: o.HPPct, Aura: o.Stats, Passive: passive}
}

// Stats are the combat numbers of one minion.
type Stats struct {
	MaxHP     int // whole points
	Defense   int
	AR        int
	DmgMin    int
	DmgMax    int
	Res       [numRes]int
	Str, Dex  int
	WalkPct   int // velocitypercent added to Walk and Run
	Walk, Run int // after WalkPct
	Thorns    int // thorns_percent: damage reflected, percent
	FireMin   int // flat fire damage added to every hit (Fire Golem)
	FireMax   int
	// Absorb maps item_absorb*_percent stats to their value.
	Absorb map[string]int
	// Other collects the stats that have no dedicated field, by name.
	Other map[string]int
}

// Compute builds the stats of a minion of template t on difficulty diff with
// the skill modifiers m. r rolls the life between minHP and maxHP; with a nil
// r the maximum is taken. The order of the steps is UNVERIFIED (the exe was
// not traced for it):
//
//	life   = roll(minHP..maxHP) * (100 + HPPct) / 100 + maxhp/256
//	damage = A1 min/max * (100 + damagepercent) / 100
//	AC     = AC + armorclass, then * (100 + item_armor_percent) / 100
//	AR     = A1TH + tohit
//
// maxhp comes in 8.8 fixed point (skills.txt multiplies by 256).
func Compute(t *Template, diff Difficulty, m Mods, r *d2rand.Seed) Stats {
	if diff < Normal || diff > Hell {
		diff = Normal
	}

	b := t.Diff[diff]

	hp := b.MaxHP
	if r != nil && b.MaxHP > b.MinHP {
		hp = b.MinHP + int(r.Roll(int32(b.MaxHP-b.MinHP+1)))
	}

	s := Stats{Walk: t.Walk, Run: t.Run, Res: b.Res, Absorb: map[string]int{}, Other: map[string]int{}}
	s.Defense = b.AC + m.LevelAC
	s.AR = b.A1TH + m.LevelAR
	s.DmgMin, s.DmgMax = b.A1Min, b.A1Max

	var dmgPct, armorPct, flatHP int

	for _, st := range append(append([]d2skill.StatMod{}, m.Aura...), m.Passive...) {
		switch name := strings.ToLower(st.Stat); name {
		case "damagepercent":
			dmgPct += st.Value
		case "tohit":
			s.AR += st.Value
		case "armorclass":
			s.Defense += st.Value
		case "item_armor_percent", "skill_armor_percent":
			armorPct += st.Value
		case "maxhp":
			flatHP += st.Value
		case "strength":
			s.Str += st.Value
		case "dexterity":
			s.Dex += st.Value
		case "velocitypercent":
			s.WalkPct += st.Value
		case "thorns_percent":
			s.Thorns += st.Value
		case "firemindam":
			s.FireMin += st.Value
		case "firemaxdam":
			s.FireMax += st.Value
		case "fireresist":
			s.Res[ResFire] += st.Value
		case "lightresist":
			s.Res[ResLightning] += st.Value
		case "coldresist":
			s.Res[ResCold] += st.Value
		case "poisonresist":
			s.Res[ResPoison] += st.Value
		case "damageresist":
			s.Res[ResPhysical] += st.Value
		case "magicresist":
			s.Res[ResMagic] += st.Value
		default:
			if strings.HasPrefix(name, "item_absorb") {
				s.Absorb[name] += st.Value
			} else {
				s.Other[name] += st.Value
			}
		}
	}

	s.MaxHP = hp*(100+m.HPPct)/100 + flatHP/256
	if s.MaxHP < 1 {
		s.MaxHP = 1
	}

	s.DmgMin = s.DmgMin * (100 + dmgPct) / 100
	s.DmgMax = s.DmgMax * (100 + dmgPct) / 100

	if armorPct != 0 {
		s.Defense = s.Defense * (100 + armorPct) / 100
	}

	for i := range s.Res {
		if s.Res[i] > ResistCap {
			s.Res[i] = ResistCap
		}
	}

	if s.WalkPct != 0 {
		s.Walk = s.Walk * (100 + s.WalkPct) / 100
		s.Run = s.Run * (100 + s.WalkPct) / 100
	}

	return s
}
