package d2skill

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
)

const maxCalcDepth = 8

// MissileFields lets miss('name'.field) reach missile data; optional.
type MissileFields interface {
	MissileField(name, field string, level int) int
}

// Env evaluates calc strings for one skill at one level for one caster
// (SKILL_GetCalcFieldValue, 0x6477d0). It implements d2calc.Env.
type Env struct {
	S        *Skill
	Level    int
	Unit     Unit
	Registry *Registry
	Missiles MissileFields
	depth    *int
}

// NewEnv creates an evaluation context.
func NewEnv(sk *Skill, level int, u Unit, reg *Registry) *Env {
	d := 0

	return &Env{S: sk, Level: level, Unit: u, Registry: reg, depth: &d}
}

func (e *Env) sub(sk *Skill, level int) *Env {
	return &Env{S: sk, Level: level, Unit: e.Unit, Registry: e.Registry, Missiles: e.Missiles, depth: e.depth}
}

func (e *Env) eval(p *d2calc.Program) int {
	if p.Empty() || *e.depth >= maxCalcDepth {
		return 0
	}

	*e.depth++

	defer func() { *e.depth-- }()

	return p.Eval(e)
}

// Eval evaluates a program in this context.
func (e *Env) Eval(p *d2calc.Program) int { return e.eval(p) }

// Field implements d2calc.Env. The codes are the rows of skillcalc.txt;
// descriptor-missile (m1en...) and mastery (math, madm, macr) fields are not
// modelled and read 0 (mastery is added as 0 to the enma/exma family).
func (e *Env) Field(code string) int {
	s, lvl := e.S, e.Level

	switch code {
	case "ln12":
		return d2calc.LN(s.Params[1], s.Params[2], lvl)
	case "ln34":
		return d2calc.LN(s.Params[3], s.Params[4], lvl)
	case "ln56":
		return d2calc.LN(s.Params[5], s.Params[6], lvl)
	case "ln78":
		return d2calc.LN(s.Params[7], s.Params[8], lvl)
	case "dm12":
		return d2calc.DM(s.Params[1], s.Params[2], lvl)
	case "dm34":
		return d2calc.DM(s.Params[3], s.Params[4], lvl)
	case "dm56":
		return d2calc.DM(s.Params[5], s.Params[6], lvl)
	case "dm78":
		return d2calc.DM78(s.Params[7], s.Params[8], lvl)
	case "par1", "par2", "par3", "par4", "par5", "par6", "par7", "par8":
		return s.Params[code[3]-'0']
	case "lvl":
		return lvl
	case "edmn":
		return int(s.ElemMin(e, lvl) >> 8)
	case "edmx":
		return int(s.ElemMax(e, lvl) >> 8)
	case "enma":
		return e.withMastery(int(s.ElemMin(e, lvl))) >> 8
	case "exma":
		return e.withMastery(int(s.ElemMax(e, lvl))) >> 8
	case "edln", "edma": // the mastery flag of 0x6462f0 is ignored by the game
		return s.ElemLen(e, lvl)
	case "edns":
		return int(s.ElemMin(e, lvl))
	case "edxs":
		return int(s.ElemMax(e, lvl))
	case "enms":
		return e.withMastery(int(s.ElemMin(e, lvl)))
	case "exms":
		return e.withMastery(int(s.ElemMax(e, lvl)))
	case "math":
		return e.masteryCalc(0)
	case "madm":
		return e.masteryCalc(1)
	case "macr":
		return e.masteryCalc(2)
	case "toht":
		return s.ToHitBonus(e, lvl)
	case "mana", "mps", "usmc":
		return e.rawManaField(code)
	case "ulvl":
		return e.Unit.Level()
	case "blvl":
		return e.Unit.BaseSkillLevel(s.ID)
	case "clc1", "clc2", "clc3", "clc4":
		return e.eval(s.Calc[code[3]-'0'])
	case "len":
		return e.eval(s.AuraLenCalc)
	case "rng":
		return e.eval(s.AuraRangeCalc)
	case "ast1", "ast2", "ast3", "ast4", "ast5", "ast6":
		return e.eval(s.AuraStatCalc[code[3]-'0'])
	case "pst1", "pst2", "pst3", "pst4", "pst5":
		return e.eval(s.PassiveCalc[code[3]-'0'])
	case "pets":
		return e.eval(s.PetMax)
	case "skpt":
		return e.eval(s.Skpoints)
	}

	return 0
}

// withMastery adds the caster's elemental mastery to a damage value
// (SKILL_ElementalMasteryBonus, 0x646040): v*stat/100 with the stat of the
// skill's element type (fire 0x149, ltng 0x14a, cold and frze 0x14b, pois
// 0x14c; none for the other types). Oracle verified.
func (e *Env) withMastery(v int) int {
	if e.Level < 1 || e.Unit == nil {
		return v
	}

	var stat string

	switch e.S.EType {
	case "fire":
		stat = "passive_fire_mastery"
	case "ltng":
		stat = "passive_ltng_mastery"
	case "cold", "frze":
		stat = "passive_cold_mastery"
	case "pois":
		stat = "passive_pois_mastery"
	default:
		return v
	}

	if m := e.Unit.Stat(stat); m != 0 {
		v += mulDiv(v, m, 100)
	}

	return v
}

// masteryCalc is the math, madm and macr fields (SKILL_GetMasteryBonus,
// 0x6491d0): kind 0 to-hit, 1 damage, 2 critical. The skill's passive stats are
// searched for the melee mastery stat (0x156..0x158) or its thrown variant
// (0x159..0x15b); the matching passivecalc is evaluated. 0 for level < 1 or
// when the skill has no such stat.
func (e *Env) masteryCalc(kind int) int {
	if e.Level < 1 {
		return 0
	}

	names := [3][2]string{
		{"passive_mastery_melee_th", "passive_mastery_throw_th"},
		{"passive_mastery_melee_dmg", "passive_mastery_throw_dmg"},
		{"passive_mastery_melee_crit", "passive_mastery_throw_crit"},
	}[kind]

	for i := 1; i <= 5; i++ {
		if n := e.S.PassiveStat[i]; n == names[0] || n == names[1] {
			return e.eval(e.S.PassiveCalc[i])
		}
	}

	return 0
}

// rawManaField is the mana, mps and usmc fields (0x6477d0 cases 21, 22, 42):
// the cost WITHOUT the minmana floor and without the free-skill rule, in the
// raw (mana + lvlmana*(lvl-1)) << manashift form, 0 for level < 1.
//
//	usmc = raw
//	mana = raw >> 8                    (arithmetic shift)
//	mps  = ((mana + lvlmana*(lvl-1)) * 25 / 2 << shift) >> 8
func (e *Env) rawManaField(code string) int {
	s, lvl := e.S, e.Level
	if lvl < 1 {
		return 0
	}

	base := int32(s.LvlMana*(lvl-1) + s.Mana)
	shift := uint(s.ManaShift) & 0x1f

	switch code {
	case "usmc":
		return int(base << shift)
	case "mana":
		return int(base<<shift) >> 8
	}

	return int((base*25)/2<<shift) >> 8
}

// Skill implements d2calc.Env: a field of another skill at that skill's level
// for the caster; 0 when the caster does not have it. blvl is the base level.
func (e *Env) Skill(name, field string) int {
	if e.Registry == nil {
		return 0
	}

	other := e.Registry.ByName(name)
	if other == nil {
		return 0
	}

	if field == "blvl" {
		return e.Unit.BaseSkillLevel(other.ID)
	}

	lvl := e.Unit.SkillLevel(other.ID)
	if lvl < 1 {
		return 0
	}

	return e.sub(other, lvl).Field(field)
}

// Miss implements d2calc.Env.
func (e *Env) Miss(name, field string) int {
	if e.Missiles == nil {
		return 0
	}

	return e.Missiles.MissileField(name, field, e.Level)
}

// Stat implements d2calc.Env: the caster's stat. The modes base, mod and accr
// are not distinguished (U: the game uses the base value, the percent variant
// or the accumulated total).
func (e *Env) Stat(name, _ string) int { return e.Unit.Stat(name) }

// Sklvl implements d2calc.Env; the argument meaning is UNVERIFIED and unused by
// the shipped tables.
func (e *Env) Sklvl(int, int, int) int { return 0 }

// Rand implements d2calc.Env: uniform in [a, b] (inclusive, oracle verified
// against the game's rand function 0x644b40, which rolls b-a+1 values) with
// the caster's generator.
func (e *Env) Rand(a, b int) int {
	if r := e.Unit.Roller(); r != nil {
		return a + int(r.Roll(int32(b-a+1)))
	}

	return a
}

// ManaCost is the skill's mana cost in 8.8 at a level (d2combat.ManaCost).
func (s *Skill) ManaCost(level int) int {
	return d2combat.ManaCost(int16(s.Mana), int16(s.LvlMana), int16(s.MinMana), int16(s.ManaShift), level)
}

// ToHitBonus is SKILL_GetToHitBonus (0x645da0): ToHitCalc if set, else
// ToHit + (lvl-1)*LevToHit.
func (s *Skill) ToHitBonus(e *Env, level int) int {
	if level < 1 { // 0x645da0 returns 0 before looking at either column
		return 0
	}

	if !s.ToHitCalc.Empty() {
		return e.eval(s.ToHitCalc)
	}

	if level < 1 {
		return s.ToHit
	}

	return s.ToHit + (level-1)*s.LevToHit
}
