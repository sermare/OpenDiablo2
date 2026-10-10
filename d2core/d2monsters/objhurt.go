package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// World objects hurt units without a monster or a skill behind the blow: the exploding barrel
// (0x5820c0 -> 0x5de7f0), the trapped exploding chest (0x57fb90) and the Storm Shrine (0x580cb0). The rules
// (who is hit, how much) live in d2common/d2object; these methods only apply a finished number to a hero
// or a monster the way a monster's blow does.

// HurtHero lowers the hero's life by dmg whole points. kind is "phys" (the flat damage reduction and the
// physical resist percent apply, VERIFIED order of 0x579c90) or an element (fire, cold, ltng, pois, mag: the
// shown resistance applies). It records the hit for item durability, logs an "object" event and counts a
// death. It returns the damage that was applied.
func (d *Director) HurtHero(p *d2mapentity.Player, dmg int, kind, source string) int {
	if p == nil || p.Stats == nil || dmg <= 0 || p.Stats.Health <= 0 {
		return 0
	}

	if t := p.Stats.Totals; t != nil && kind == "phys" {
		dmg = heroPhysicalDamage(dmg, t.DamageReduction, t.PhysResist)
	} else if kind != "phys" {
		dmg = applyElemResist(dmg, kind, p.Stats.Totals)
	}

	if dmg <= 0 {
		d.emit("object", "OBJECT hurt hero source=%s kind=%s dmg=0 hero_hp=%d/%d", source, kind, p.Stats.Health, p.Stats.MaxHealth)

		return 0
	}

	p.Stats.Health -= dmg
	if p.Stats.Health < 0 {
		p.Stats.Health = 0
	}

	if d.opt.OnHeroHit != nil {
		d.opt.OnHeroHit(p)
	}

	d.emit("object", "OBJECT hurt hero source=%s kind=%s dmg=%d hero_hp=%d/%d", source, kind, dmg, p.Stats.Health, p.Stats.MaxHealth)

	if p.Stats.Health == 0 {
		d.Counters.HeroDeaths++
		d.emit("herodeath", "HERO died name=%s killer=%s", p.Name(), source)
	}

	return dmg
}

// HurtMonster lowers a monster's life by dmg whole points for a blow from a world object. src is the hero
// credited with the kill (the object's activator), nil for none. Monster physical resistance is not applied
// (UNVERIFIED: the exe runs COMBAT_ApplyResistsToDamageStruct on the target). It returns the damage applied.
func (d *Director) HurtMonster(m *d2mapentity.Monster, dmg int, src *d2mapentity.Player, source string) int {
	if m == nil || dmg <= 0 || !m.Alive() {
		return 0
	}

	d.emit("object", "OBJECT hurt monster source=%s name=%s dmg=%d", source, m.Label(), dmg)
	d.Damage(m, dmg, src)

	return dmg
}

// LevelMonster draws one monster class of a levels.txt area the way the area's natural groups do (the area's
// types, drawn once and weighted by rarity), for the container spawn handlers 8/9 and the barrel monster
// (MONREGION_PickLevelMonsterClassBase 0x5801d0, 0x5847b0). It returns nil when the area has no types.
func (d *Director) LevelMonster(levelID int) *d2records.MonStatRecord {
	det := d.asset.Records.GetLevelDetails(levelID)
	if det == nil {
		return nil
	}

	d.SetAreaLevel(d.AreaLevelOf(levelID))

	types, ok := d.levelTypes[levelID]
	if !ok {
		var list []d2monster.ClassInfo

		for _, name := range levelMonsterNames(det, d.opt.Difficulty) {
			if st := d.minionStat(name); st != nil {
				list = append(list, d.classInfo(st))
			}
		}

		types = d2monster.PickLevelTypes(d.packRNG, list, det.NumMonsterTypes, det.MonsterPreferRanged)

		if d.levelTypes == nil {
			d.levelTypes = map[int][]d2monster.ClassInfo{}
		}

		d.levelTypes[levelID] = types
	}

	ci, ok := d2monster.PickByRarity(d.packRNG, types)
	if !ok {
		return nil
	}

	return d.statByID[ci.Class]
}

// UniqueUpgradeCandidate is MONSTER_IsUniqueUpgradeCandidate (0x580660) as far as the engine knows the
// monster: a living hostile ordinary monster that is neither a boss nor already special. UNVERIFIED: the
// exe also needs the monster in mode 1 or 2 and rejects two flags (stat record 0x80, data 0x1f) that are not
// identified.
func (d *Director) UniqueUpgradeCandidate(m *d2mapentity.Monster) bool {
	if m == nil || !m.Alive() || m.Stat == nil || m.Stat.IsSpecialBoss {
		return false
	}

	if m.TypeFlags&(d2mapentity.MonTypeSuperUnique|d2mapentity.MonTypeChampion|d2mapentity.MonTypeUnique|
		d2mapentity.MonTypeMinion|d2mapentity.MonTypeModsRolled) != 0 {
		return false
	}

	return d.byEntity[m.ID()] != nil
}

// MakeUnique turns a living monster into a rare/unique with rolled modifiers (MONSTER_InitAsUniqueWithRolledMods
// 0x5a2410: type bit 1 set, modifiers rolled, flag 0x800 on the unit). The modifier count and the pick
// weights are the ones PopulateRoom uses for unique leaders. UNVERIFIED: the unique's hit point and damage
// scaling are not re-applied to a monster that already exists.
func (d *Director) MakeUnique(m *d2mapentity.Monster) bool {
	if !d.UniqueUpgradeCandidate(m) {
		return false
	}

	m.TypeFlags |= d2mapentity.MonTypeUnique
	d.recordRank(m, d2monster.Pack{})
	d.emit("object", "OBJECT monster %s id=%d became unique mods=%v", m.Label(), d.UnitID(m), m.Modifiers)

	return true
}
