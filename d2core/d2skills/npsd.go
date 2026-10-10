package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Engine side of the Necromancer / Paladin / Sorceress / Druid skills that
// need more than damage: Conversion, the AI changing curses and life/mana
// leech (Hunger).

// forcedByStat maps the synthetic curse stats of d2skill.doCurseFn to the
// forced monster AI state (VERIFIED mapping in d2monster/forced.go).
var forcedByStat = map[string]d2monster.ForcedKind{
	d2state.StatBlind:    d2monster.ForcedBlind,
	d2state.StatConfused: d2monster.ForcedConfuse,
	d2state.StatAttract:  d2monster.ForcedAttract,
	d2state.StatTaunted:  d2monster.ForcedTaunt,
}

// forceFromMods puts the forced AI state of a curse on the monster: Dim Vision
// (state 10), Taunt (12), Confuse and Attract (target list 9).
func (e *Engine) forceFromMods(m *d2mapentity.Monster, inst d2state.Instance) {
	frames := inst.Until - e.frame
	if frames <= 0 {
		return
	}

	for _, md := range inst.Mods {
		kind, ok := forcedByStat[md.Stat]
		if !ok || md.Value == 0 {
			continue
		}

		if ai, err := e.monsters.ForceMonster(m, kind, frames, 0); err == nil {
			e.emit("state", "STATE forced unit=%s kind=%s frames=%d ai=%s", m.Label(), kind, frames, ai)
		}
	}
}

// ConvertScaled is the life and level of a converted monster that is above the
// caster's level. VERIFIED (0x5ce8c0): when the target's level (stat 0xc) is
// higher than the caster's, the monster's level becomes the caster's and its
// life and maximum life are scaled by caster/target (MulDiv), at least 1;
// SRVDO_ConversionRevertState (0x5ce710) puts the level back and scales the
// life by the current/max ratio onto the original maximum.
func ConvertScaled(level, hp, maxHP, casterLevel int) (newLevel, newHP, newMax int) {
	if level == 0 || casterLevel >= level {
		return level, hp, maxHP
	}

	scale := func(v int) int {
		v = v * casterLevel / level
		if v < 1 {
			return 1
		}

		return v
	}

	return casterLevel, scale(hp), scale(maxHP)
}

// RevertConverted restores a scaled monster: level and maximum life return to
// the originals and the life keeps its current/max ratio (0x5ce710), at least
// 1.
func RevertConverted(hp, curMax, origMax int) int {
	if curMax <= 0 {
		return 1
	}

	v := hp * origMax / curMax
	if v < 1 {
		v = 1
	}

	return v
}

// convert is the "convert" effect (Conversion, SRVDO_079): the monster becomes
// the hero's ally for the effect's length (forced charm AI), scaled down to
// the caster's level when above it, and reverts afterwards.
func (e *Engine) convert(p *d2mapentity.Player, sk *d2skill.Skill, ef *d2skill.Effect) {
	mt, _ := ef.Target.(*monsterTarget)
	if mt == nil || !mt.m.Alive() {
		return
	}

	m := mt.m
	v := &m.Vitals
	oldLevel, oldMax := v.Level, v.MaxHP

	lvl, hp, mx := ConvertScaled(v.Level, v.HP, v.MaxHP, ef.CasterLevel)
	scaled := lvl != oldLevel
	v.Level, v.HP, v.MaxHP = lvl, hp, mx

	ai, err := e.monsters.ForceMonster(m, d2monster.ForcedCharm, ef.Frames, 0)
	if err != nil {
		e.emit("state", "STATE convert skill=%q target=%s refused=%v", sk.Name, m.Label(), err)

		return
	}

	e.setOf(m.ID()).Apply(e.frame, d2state.Instance{Name: ef.State, Until: e.frame + ef.Frames, Source: p.ID(),
		SkillID: sk.ID, Level: ef.Level})
	e.emit("state", "STATE convert skill=%q target=%s frames=%d ai=%s level=%d->%d scaled=%v hp=%d/%d", sk.Name, m.Label(),
		ef.Frames, ai, oldLevel, lvl, scaled, v.HP, v.MaxHP)

	e.after(ef.Frames, func() {
		if scaled && m.Alive() {
			v.HP = RevertConverted(v.HP, v.MaxHP, oldMax)
			v.Level, v.MaxHP = oldLevel, oldMax
		}

		e.emit("state", "STATE convert_end target=%s alive=%v level=%d hp=%d/%d", m.Label(), m.Alive(), v.Level, v.HP, v.MaxHP)
	})
}

// leech gives the hero life and mana back, a percent of the damage a hit dealt
// (Hunger: the damage struct's life steal / mana steal fields).
func (e *Engine) leech(p *d2mapentity.Player, d *d2combat.Damage, dealt int) {
	if p == nil || dealt <= 0 {
		return
	}

	if d.LifeLeech > 0 {
		heal := dealt * int(d.LifeLeech) / 100
		p.Stats.Health = minInt(p.Stats.Health+heal, p.Stats.MaxHealth)
		e.emit("damage", "HEAL leech hero=%s heal=%d hero_hp=%d/%d", p.Name(), heal, p.Stats.Health, p.Stats.MaxHealth)
	}

	if d.ManaLeech > 0 {
		gain := dealt * int(d.ManaLeech) / 100
		p.Stats.Mana = minInt(p.Stats.Mana+gain, p.Stats.MaxMana)
		e.emit("damage", "HEAL mana_leech hero=%s mana=%d mana=%d/%d", p.Name(), gain, p.Stats.Mana, p.Stats.MaxMana)
	}
}

// CanConvert implements d2skill.Convertible: a monster that is not already
// converted (alignment 0 in the exe, VERIFIED). U: act bosses and other
// special bosses are refused here (the exe's SKILL_IsValidMonsterSkillTarget
// goes through a monstats type flag that was not mapped).
func (t *monsterTarget) CanConvert() bool {
	if t.m.Stat == nil || t.m.Stat.IsSpecialBoss {
		return false
	}

	return !t.e.setOf(t.m.ID()).Active(t.e.frame, "conversion")
}

// reviveCap limits a revived monster to the caster's level: VERIFIED
// (SRVDO_058_Revive 0x5c35a0) a corpse above the caster's level comes back at
// the caster's level with life scaled by caster/corpse (at least 1).
func reviveCap(m *d2mapentity.Monster, corpseLevel, casterLevel int) {
	if corpseLevel <= 0 || casterLevel <= 0 || corpseLevel <= casterLevel {
		return
	}

	lvl, hp, mx := ConvertScaled(corpseLevel, m.Vitals.HP, m.Vitals.MaxHP, casterLevel)
	m.Vitals.Level, m.Vitals.HP, m.Vitals.MaxHP = lvl, hp, mx
}
