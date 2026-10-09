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
		return d2calc.DM(s.Params[7], s.Params[8], lvl)
	case "par1", "par2", "par3", "par4", "par5", "par6", "par7", "par8":
		return s.Params[code[3]-'0']
	case "lvl":
		return lvl
	case "edmn", "enma":
		return int(s.ElemMin(e, lvl) >> 8)
	case "edmx", "exma":
		return int(s.ElemMax(e, lvl) >> 8)
	case "edln", "edma":
		return s.ElemLen(e, lvl)
	case "edns", "enms":
		return int(s.ElemMin(e, lvl))
	case "edxs", "exms":
		return int(s.ElemMax(e, lvl))
	case "toht":
		return s.ToHitBonus(e, lvl)
	case "mana":
		return int(s.calcMana(lvl, false)) >> 8
	case "mps":
		return int(s.calcMana(lvl, true)) >> 8 // U: units of mana per second at 25 fps
	case "usmc":
		return int(s.calcMana(lvl, false))
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

// Rand implements d2calc.Env: uniform in [a, b) with the caster's generator.
func (e *Env) Rand(a, b int) int {
	if r := e.Unit.Roller(); r != nil {
		return a + int(r.Roll(int32(b-a)))
	}

	return a
}

// ManaCost is the skill's mana cost in 8.8 at a level (d2combat.ManaCost).
func (s *Skill) ManaCost(level int) int {
	return d2combat.ManaCost(int16(s.Mana), int16(s.LvlMana), int16(s.MinMana), int16(s.ManaShift), level)
}

// calcMana is the mana arithmetic of the skillcalc fields mana, mps and usmc
// (SKILL_GetCalcFieldValue, 0x6477d0, verified): 0 below level 1, otherwise
// (lvlmana*(lvl-1)+mana), times 25/2 first for mps, shifted left by manashift.
// Unlike SKILL_PayManaCost it applies neither minmana nor the free-skill
// shortcut, and a negative result stays negative. 32 bit as in the game.
func (s *Skill) calcMana(level int, perSecond bool) int32 {
	if level < 1 {
		return 0
	}

	v := int32(s.LvlMana)*int32(level-1) + int32(s.Mana)
	if perSecond {
		v = v * 25 / 2
	}

	return v << (uint(s.ManaShift) & 0x1f)
}

// ToHitBonus is SKILL_GetToHitBonus (0x645da0): ToHitCalc if set, else
// ToHit + (lvl-1)*LevToHit.
func (s *Skill) ToHitBonus(e *Env, level int) int {
	if !s.ToHitCalc.Empty() {
		return e.eval(s.ToHitCalc)
	}

	if level < 1 {
		return s.ToHit
	}

	return s.ToHit + (level-1)*s.LevToHit
}
