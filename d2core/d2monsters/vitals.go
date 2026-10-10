package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monstats"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// profileFromRecord converts a monstats row into the AI profile for a
// difficulty.
func profileFromRecord(r *d2records.MonStatRecord, diff d2monster.Difficulty) *d2monster.Profile {
	pick := func(n, nm, h int) int { return [3]int{n, nm, h}[diff] }

	p := &d2monster.Profile{
		Class:  r.ID,
		ID:     r.Key,
		AI:     r.AiKey,
		AIDel:  pick(r.AiDelayNormal, r.AiDelayNightmare, r.AiDelayHell),
		AIDist: pick(r.AiDistanceNormal, r.AiDistanceNightmare, r.AiDistanceHell),
		Threat: r.ThreatLevel,
		Melee:  r.IsMelee,
		Walk:   r.SpeedBase,
		Run:    r.SpeedRun,
	}

	p.AIP[1] = pick(r.AiParameterNormal1, r.AiParameterNightmare1, r.AiParameterHell1)
	p.AIP[2] = pick(r.AiParameterNormal2, r.AiParameterNightmare2, r.AiParameterHell2)
	p.AIP[3] = pick(r.AiParameterNormal3, r.AiParameterNightmare3, r.AiParameterHell3)
	p.AIP[4] = pick(r.AiParameterNormal4, r.AiParameterNightmare4, r.AiParameterHell4)
	p.AIP[5] = pick(r.AiParameterNormal5, r.AiParameterNightmare5, r.AiParameterHell5)
	p.AIP[6] = pick(r.AiParameterNormal6, r.AiParameterNightmare6, r.AiParameterHell6)
	p.AIP[7] = pick(r.AiParameterNormal7, r.AiParameterNightmare7, r.AiParameterHell7)
	p.AIP[8] = pick(r.AiParameterNormal8, r.AiParameterNightmare8, r.AiParameterHell8)

	names := [d2monster.NumSkills]string{r.SkillId1, r.SkillId2, r.SkillId3, r.SkillId4,
		r.SkillId5, r.SkillId6, r.SkillId7, r.SkillId8}
	modes := [d2monster.NumSkills]string{r.SkillAnimation1, r.SkillAnimation2, r.SkillAnimation3,
		r.SkillAnimation4, r.SkillAnimation5, r.SkillAnimation6, r.SkillAnimation7, r.SkillAnimation8}
	levels := [d2monster.NumSkills]int{r.SkillLevel1, r.SkillLevel2, r.SkillLevel3, r.SkillLevel4,
		r.SkillLevel5, r.SkillLevel6, r.SkillLevel7, r.SkillLevel8}

	for i := range names {
		p.Skills[i].Name = names[i]
		p.Skills[i].Level = levels[i]
		p.Skills[i].Mode, _ = d2monster.ParseMode(modes[i])
	}

	return p
}

// computeVitals derives level, hit points, defense, attack ratings, damage
// and experience. For classes without noRatio the numbers are the monlvl.txt
// row of the monster level times the monstats ratio columns in percent
// (VERIFIED, MONTBL_GetLevelScaledStats); noRatio classes use the monstats
// values as they are. Which monlvl column set the game uses (L-* here, the
// plain ones with Options.Classic) is UNVERIFIED.
//
// The monster level follows d2monster.ResolveLevel (VERIFIED 0x00571c4f): the
// monstats Level of the difficulty, except in an expansion game (Options.
// Expansion) in Nightmare/Hell for classes with neither noRatio nor boss, which
// take the levels.txt MonLvl of the area set with SetAreaLevel. Hit points and
// experience get the player-count bonus (Director.PlayerCount: the live count or Options.Players, raised to the OD2_PLAYERS override; classes
// with Align != 0 are exempt) and the hit points are capped at 0x7fffff
// (VERIFIED 0x00571af0 / 0x00571760).
func (d *Director) computeVitals(r *d2records.MonStatRecord, b *d2monster.Brain) d2mapentity.MonsterVitals {
	diff := d.opt.Difficulty
	pick := func(n, nm, h int) int { return [3]int{n, nm, h}[diff] }

	level := d2monster.ResolveLevel(d.classInfo(r), diff, [3]int{r.LevelNormal, r.LevelNightmare, r.LevelHell},
		d.areaLevel, d.opt.Expansion)
	if d.forceLevel > 0 { // summoned minions take their owner's level
		level = d.forceLevel
	}

	v := d2mapentity.MonsterVitals{Level: level, Difficulty: diff}

	var lv lvlNums

	if rec := d.asset.Records.Monster.Levels[level]; rec != nil && !r.IgnoreMonLevelTxt {
		lv = monlvlRow(rec, d.opt.Classic)[diff]
	} else {
		lv.hp, lv.ac, lv.th, lv.dm, lv.xp = 100, 100, 100, 100, 100 // raw values: ratio of 100%
	}

	scale := d2monstats.MulDiv100

	hpMin := scale(lv.hp, pick(r.MinHPNormal, r.MinHPNightmare, r.MinHPHell))
	hpMax := scale(lv.hp, pick(r.MaxHPNormal, r.MaxHPNightmare, r.MaxHPHell))

	if hpMax < hpMin {
		hpMax = hpMin
	}

	hpPct, xpPct, _ := d2monstats.PlayerBonus(d.PlayerCount(), int(r.Alignment))

	v.MaxHP = hpMin + b.Roll(hpMax-hpMin+1)
	v.MaxHP += d2monstats.MulDiv(v.MaxHP, hpPct, 100) // the bonus comes before the cap
	if v.MaxHP > d2monstats.MaxHP {
		v.MaxHP = d2monstats.MaxHP
	}

	if v.MaxHP < 1 {
		v.MaxHP = 1
	}

	v.HP = v.MaxHP
	v.Defense = scale(lv.ac, pick(r.ArmorClassNormal, r.ArmorClassNightmare, r.ArmorClassHell))
	v.Experience = scale(lv.xp, pick(r.ExperienceNormal, r.ExperienceNightmare, r.ExperienceHell))
	v.Experience += d2monstats.MulDiv(v.Experience, xpPct, 100)
	v.TreasureClass = [3]string{r.TreasureClassNormal, r.TreasureClassNightmare, r.TreasureClassHell}[diff]

	v.A1 = MonsterAttackFrom(scale(lv.th, pick(r.AttackRatingA1Normal, r.AttackRatingA1Nightmare, r.AttackRatingA1Hell)),
		scale(lv.dm, pick(r.DamageMinA1Normal, r.DamageMinA1Nightmare, r.DamageMinA1Hell)),
		scale(lv.dm, pick(r.DamageMaxA1Normal, r.DamageMaxA1Nightmare, r.DamageMaxA1Hell)))
	v.A2 = MonsterAttackFrom(scale(lv.th, pick(r.AttackRatingA2Normal, r.AttackRatingA2Nightmare, r.AttackRatingA2Hell)),
		scale(lv.dm, pick(r.DamageMinA2Normal, r.DamageMinA2Nightmare, r.DamageMinA2Hell)),
		scale(lv.dm, pick(r.DamageMaxA2Normal, r.DamageMaxA2Nightmare, r.DamageMaxA2Hell)))

	v.S1 = MonsterAttackFrom(scale(lv.th, pick(r.AttackRatingS1Normal, r.AttackRatingS1Nightmare, r.AttackRatingS1Hell)),
		scale(lv.dm, pick(r.DamageMinS1Normal, r.DamageMinS1Nightmare, r.DamageMinS1Hell)),
		scale(lv.dm, pick(r.DamageMaxS1Normal, r.DamageMaxS1Nightmare, r.DamageMaxS1Hell)))

	// Casters (Skeleton Mage) have no physical A1 damage: their A1 is the
	// first elemental damage column, scaled like physical damage. Poison and
	// cold lengths are not modelled (the hit is instant). UNVERIFIED reading
	// of El1*.
	if v.A1.Max == 0 && r.ElementAttackMode1 == "A1" {
		v.A1 = MonsterAttackFrom(scale(lv.th, 100),
			scale(lv.dm, pick(r.ElementDamageMin1Normal, r.ElementDamageMin1Nightmare, r.ElementDamageMin1Hell)),
			scale(lv.dm, pick(r.ElementDamageMax1Normal, r.ElementDamageMax1Nightmare, r.ElementDamageMax1Hell)))
	}

	if v.A1.ToHit == 0 {
		v.A1.ToHit = scale(lv.th, 100)
	}

	if v.S1.ToHit == 0 {
		v.S1.ToHit = v.A1.ToHit
	}

	if v.A2.ToHit == 0 {
		v.A2.ToHit = v.A1.ToHit
	}

	return v
}

// MonsterAttackFrom builds an attack, keeping Max >= Min.
func MonsterAttackFrom(toHit, min, max int) d2mapentity.MonsterAttack {
	if max < min {
		max = min
	}

	return d2mapentity.MonsterAttack{ToHit: toHit, Min: min, Max: max}
}

type lvlNums struct{ hp, ac, th, dm, xp int }

// monlvlRow flattens a monlvl.txt row to the numbers vitals use, by
// difficulty, from the LoD (L-*) or the plain columns.
func monlvlRow(rec *d2records.MonsterLevelRecord, classic bool) [3]lvlNums {
	if classic {
		b := rec.BattleNet

		return [3]lvlNums{
			{b.Normal.Hitpoints, b.Normal.DefenseRating, b.Normal.AttackRating, b.Normal.Damage, b.Normal.Experience},
			{b.Nightmare.Hitpoints, b.Nightmare.DefenseRating, b.Nightmare.AttackRating, b.Nightmare.Damage,
				b.Nightmare.Experience},
			{b.Hell.Hitpoints, b.Hell.DefenseRating, b.Hell.AttackRating, b.Hell.Damage, b.Hell.Experience},
		}
	}

	l := rec.Ladder

	return [3]lvlNums{
		{l.Normal.Hitpoints, l.Normal.DefenseRating, l.Normal.AttackRating, l.Normal.Damage, l.Normal.Experience},
		{l.Nightmare.Hitpoints, l.Nightmare.DefenseRating, l.Nightmare.AttackRating, l.Nightmare.Damage,
			l.Nightmare.Experience},
		{l.Hell.Hitpoints, l.Hell.DefenseRating, l.Hell.AttackRating, l.Hell.Damage, l.Hell.Experience},
	}
}

// MonsterResists are the resistances of a class in a difficulty, in the order
// physical, magic, fire, lightning, cold, poison (the monstats ResDm, ResMa,
// ResFi, ResLi, ResCo, ResPo columns with their (N) / (H) variants). They are
// used as they are: the difficulty penalty only applies to players.
func MonsterResists(r *d2records.MonStatRecord, diff d2monster.Difficulty) [6]int {
	pick := func(n, nm, h int) int { return [3]int{n, nm, h}[diff] }

	return [6]int{
		pick(r.ResistancePhysicalNormal, r.ResistancePhysicalNightmare, r.ResistancePhysicalHell),
		pick(r.ResistanceMagicNormal, r.ResistanceMagicNightmare, r.ResistanceMagicHell),
		pick(r.ResistanceFireNormal, r.ResistanceFireNightmare, r.ResistanceFireHell),
		pick(r.ResistanceLightningNormal, r.ResistanceLightningNightmare, r.ResistanceLightningHell),
		pick(r.ResistanceColdNormal, r.ResistanceColdNightmare, r.ResistanceColdHell),
		pick(r.ResistancePoisonNormal, r.ResistancePoisonNightmare, r.ResistancePoisonHell),
	}
}

// LifeRangeAt is the minimum and maximum life (whole points) of a monster class
// at a level in the current difficulty, without the player-count bonus: the
// first two outputs of MONTBL_GetLevelScaledStats (0x6551e0, VERIFIED), which
// Revive rolls between at the corpse's level (SRVDO_058 0x5c35a0).
func (d *Director) LifeRangeAt(r *d2records.MonStatRecord, level int) (lo, hi int) {
	diff := d.opt.Difficulty
	pick := func(n, nm, h int) int { return [3]int{n, nm, h}[diff] }

	var lv lvlNums

	if rec := d.asset.Records.Monster.Levels[level]; rec != nil && !r.IgnoreMonLevelTxt {
		lv = monlvlRow(rec, d.opt.Classic)[diff]
	} else {
		lv.hp = 100 // raw values: ratio of 100%
	}

	lo = d2monstats.MulDiv100(lv.hp, pick(r.MinHPNormal, r.MinHPNightmare, r.MinHPHell))
	hi = d2monstats.MulDiv100(lv.hp, pick(r.MaxHPNormal, r.MaxHPNightmare, r.MaxHPHell))

	if hi < lo {
		hi = lo
	}

	return lo, hi
}
