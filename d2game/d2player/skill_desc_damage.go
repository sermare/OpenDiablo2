package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// descDamInputFor fills the game data inputs of the character screen's descdam functions (d2statlist.DescDamBox,
// wave 47) for one hand skill: skills.txt srcdam / etype / the finishing flag, the skill's physical and elemental
// numbers at its level (SKILLDESC_CalcSkillMin/MaxDamage with the weapon flag off, SKILL_GetElementalMin/Max/Length),
// the skilldesc.txt "ddam calc1/2" and the state based damage bonus of the charge-up states the hero holds. The skill
// level is the effective level (points plus item bonuses, d2hero.EffectiveSkillLevel). ok is false when the skill's descdam is not one of the ported functions or its data is missing.
func descDamInputFor(asset *d2asset.AssetManager, hs *d2hero.HeroSkill, level int, stats *d2hero.HeroStatsState,
	skills map[int]*d2hero.HeroSkill) (d2statlist.DescDamInput, bool) {
	var in d2statlist.DescDamInput

	if asset == nil || asset.Records == nil || hs == nil || hs.SkillRecord == nil || hs.SkillDescriptionRecord == nil {
		return in, false
	}

	desc := hs.SkillDescriptionRecord.Desc
	if desc.DescDam != 5 && desc.DescDam != 7 {
		return in, false
	}

	reg := asset.Records.SkillTable()

	sk := reg.ByID(hs.ID)
	if sk == nil {
		return in, false
	}

	if level < 1 {
		level = 1
	}

	pts := make(map[int]int, len(skills))
	for id, s := range skills {
		if s != nil {
			pts[id] = s.SkillPoints
		}
	}

	heroLevel := 0
	if stats != nil {
		heroLevel = stats.Level
	}

	env := d2skill.NewEnv(sk, level, tooltipUnit{level: heroLevel, points: pts}, reg)
	eval := func(src string) int {
		if src == "" {
			return 0
		}

		return env.Eval(d2calc.Compile(src, d2calc.KindSkill))
	}

	var pct int
	if stats != nil {
		pct = elemMasteryPct(stats.Totals, elemType(sk.EType))
	}

	in = d2statlist.DescDamInput{
		DescDam: desc.DescDam,
		SrcDam:  hs.SkillRecord.SrcDam,
		EType:   elemType(sk.EType),
		Calc1:   eval(desc.DdamCalc1),
		Calc2:   eval(desc.DdamCalc2),
		PhysMin: int(sk.PhysMin(env, level, 0, false) >> 8),
		PhysMax: int(sk.PhysMax(env, level, 0, false) >> 8),
		ElemMin: withElemMastery(int(sk.ElemMin(env, level)), pct),
		ElemMax: withElemMastery(int(sk.ElemMax(env, level)), pct),
		ElemLen: sk.ElemLen(env, level),
		Bit8:    hs.SkillRecord.Finishing,
	}

	if in.Bit8 && stats != nil && stats.SkillStats != nil {
		var charges []d2statlist.StateCharge

		for _, c := range d2skill.ChargeStatesOf(reg, stats.SkillStats(), func(id int) int { return pts[id] }) {
			charges = append(charges, d2statlist.StateCharge{PrgDam: c.PrgDam, Param1: c.Param1, Param2: c.Param2,
				Level: c.Level, Count: c.Count, HasAuraStat: true})
		}

		in.StateBonus = d2statlist.StateBonusOf(charges)
	}

	return in, true
}

// skillToHitBonus is SKILL_GetToHitBonus (0x645da0) at the effective level, ToHitCalc included.
func skillToHitBonus(asset *d2asset.AssetManager, hs *d2hero.HeroSkill, level int, stats *d2hero.HeroStatsState,
	skills map[int]*d2hero.HeroSkill) int {
	if asset == nil || asset.Records == nil || hs == nil || level < 1 {
		return 0
	}

	reg := asset.Records.SkillTable()

	sk := reg.ByID(hs.ID)
	if sk == nil {
		return 0
	}

	pts := make(map[int]int, len(skills))
	for id, s := range skills {
		if s != nil {
			pts[id] = s.SkillPoints
		}
	}

	heroLevel := 0
	if stats != nil {
		heroLevel = stats.Level
	}

	return sk.ToHitBonus(d2skill.NewEnv(sk, level, tooltipUnit{level: heroLevel, points: pts}, reg), level)
}

// elemMasteryPct is the stat SKILL_GetElementMasteryBonusPct (0x646040) reads for a skill's element: 0x149 fire,
// 0x14a lightning, 0x14b cold (etype 4 and 0xc), 0x14c poison, none for magic. SKILL_GetElementalMin/MaxDamage with
// the flag set (0x646100 / 0x646200, VERIFIED) add value * pct / 100 to their result.
func elemMasteryPct(t *d2statlist.Totals, etype int) int {
	if t == nil {
		return 0
	}

	switch etype {
	case 1:
		return t.Elem.Elem[0].Pct
	case 2:
		return t.Elem.Elem[1].Pct
	case 4:
		return t.Elem.Elem[2].Pct
	case 5:
		return t.Elem.Poison.Pct
	}

	return 0
}

// withElemMastery is v + v*pct/100 (MATH_MulDiv, truncating).
func withElemMastery(v, pct int) int { return v + v*pct/100 }
