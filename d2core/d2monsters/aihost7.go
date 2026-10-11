package d2monsters

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Host side of the batch-7 AI interfaces (d2common/d2monster/ai_faithful_5.go,
// ai_fb_1.go, ai_faithful_7.go): the target vitals the Succubus and Fetish read,
// the skills.txt aurastate of a monster skill slot, and the collision line the
// Diablo and UberIzual handlers ask for.

var (
	_ d2monster.FB1Unit         = (*Director)(nil)
	_ d2monster.AuraStateSource = (*Director)(nil)
	_ d2monster.LineChecker     = (*Director)(nil)
)

// vitalsOf returns hit points, maximum hit points and maximum mana of the unit
// a target id names: a hero, a mercenary, a converted monster or a monster.
// Monsters carry no mana (maxMana 0).
func (d *Director) vitalsOf(id uint32) (hp, maxHP, maxMana int, ok bool) {
	switch {
	case id >= unitTargetBase:
		if u := d.units[id-unitTargetBase]; u != nil {
			return u.m.Vitals.HP, u.m.Vitals.MaxHP, 0, true
		}
	case id >= mercTargetBase:
		if u := d.units[id-mercTargetBase]; u != nil {
			return u.m.Vitals.HP, u.m.Vitals.MaxHP, 0, true
		}
	default:
		if p := d.playerFor(id); p != nil && p.Stats != nil {
			return p.Stats.Health, p.Stats.MaxHealth, p.Stats.MaxMana, true
		}
	}

	return 0, 0, 0, false
}

// LifePercent implements d2monster.FB1Unit (STATS_GetLifePercent of the
// target): 100 when the unit is unknown.
func (d *Director) LifePercent(t d2monster.Target) int {
	hp, maxHP, _, ok := d.vitalsOf(t.ID)
	if !ok || maxHP <= 0 {
		return 100
	}

	if hp < 0 {
		hp = 0
	}

	return hp * 100 / maxHP
}

// MaxHP implements d2monster.FB1Unit.
func (d *Director) MaxHP(t d2monster.Target) int {
	_, maxHP, _, _ := d.vitalsOf(t.ID)

	return maxHP
}

// MaxMana implements d2monster.FB1Unit.
func (d *Director) MaxMana(t d2monster.Target) int {
	_, _, maxMana, _ := d.vitalsOf(t.ID)

	return maxMana
}

// HasStatListFlag implements d2monster.FB1Unit (STATS_FindStatListByFlags). The
// engine keeps no flagged stat lists on units, so no list is ever found; the
// Succubus gate on flag 0x20 therefore never blocks (UNVERIFIED meaning).
func (d *Director) HasStatListFlag(d2monster.Target, uint32) bool { return false }

// SlotAuraState implements d2monster.AuraStateSource: the states.txt id of the
// skills.txt aurastate of the skill in the monstats slot, -1 when the slot is
// empty, unknown or its skill has no aurastate.
func (d *Director) SlotAuraState(b *d2monster.Brain, slot int) int {
	if b == nil || b.Profile == nil || slot < 0 || slot >= d2monster.NumSkills ||
		!b.Profile.Skills[slot].Used() || d.asset == nil || d.asset.Records == nil {
		return -1
	}

	rec := d.asset.Records.GetSkillByName(b.Profile.Skills[slot].Name)
	if rec == nil || strings.TrimSpace(rec.Aurastate) == "" {
		return -1
	}

	if st, ok := d.asset.Records.States[strings.ToLower(strings.TrimSpace(rec.Aurastate))]; ok && st != nil {
		return st.ID
	}

	if st, ok := d.asset.Records.States[rec.Aurastate]; ok && st != nil {
		return st.ID
	}

	return -1
}

// LineBlocked implements d2monster.LineChecker: COLLISION_CanTraceLineBetweenUnits
// with the exe masks 4 and 6 (both are the wall-type blockers of the static
// map; the finer difference between the two masks is not decoded). True when a
// wall lies between the monster and the target.
func (d *Director) LineBlocked(b *d2monster.Brain, t d2monster.Target, _ int) bool {
	clear, _ := d2path.TraceLine(d.grid, d2path.FlagWall, d2path.Point{X: b.X, Y: b.Y}, d2path.Point{X: t.X, Y: t.Y})

	return !clear
}
