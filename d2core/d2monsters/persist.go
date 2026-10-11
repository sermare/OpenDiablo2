package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Level persistence. The original keeps a level's units while the game lives:
// monsters the hero killed stay dead and the ones he left behind stay where
// they were. The map engine rebuilds the map from the seed on every level
// change (ResetMap drops every entity), so the director parks the living
// hostile monsters of the level the hero leaves and puts the same ones back
// when the level is built again, instead of populating it anew. The game
// screen drops its director with the map, so the monsters move to the director
// of the rebuilt map (RestoreLevel renumbers them).

// ParkedLevel is the set of living hostile monsters of one level.
type ParkedLevel struct {
	units []*unit
	at    int // game frame of the departure (clock: Game.gameFrame)
}

// InactiveLifeFrames is how long a monster may stay out of its room and still
// come back with the life it had (exe MONAI_RestoreInactiveMonster 0x5401d0,
// 0x1d4c = 7500 frames, 5 minutes at 25 Hz). After that it returns at full
// life.
const InactiveLifeFrames = 7500

// RestoredLife is the life a monster has when its room wakes again: the life
// it left with when under InactiveLifeFrames frames passed, else the maximum.
func RestoredLife(hp, maxHP, elapsed int) int {
	if elapsed < InactiveLifeFrames && hp > 0 && hp <= maxHP {
		return hp
	}

	return maxHP
}

// Len is the number of parked monsters.
func (p *ParkedLevel) Len() int {
	if p == nil {
		return 0
	}

	return len(p.units)
}

// ParkLevel detaches the living hostile monsters from the director (their
// entities stay valid but are no longer simulated) and returns them. Corpses,
// mercenaries and summoned allies are not parked: corpses are gone after a
// while anyway and the hero's followers travel with him.
func (d *Director) ParkLevel(now int) *ParkedLevel {
	out := &ParkedLevel{at: now}

	for _, u := range d.sortedUnits() {
		if u.friendly() || u.b.Allied || !u.m.Alive() {
			continue
		}

		d.engine.RemoveEntity(u.m)
		d.forget(u)

		out.units = append(out.units, u)
	}

	return out
}

// RestoreLevel puts parked monsters back into the (rebuilt) map and the
// director. They come back calm (no target, standing) where they were; their
// hit points, level and group links are those they had. It returns how many
// were restored. now is the game frame on the clock ParkLevel used; a level
// left for InactiveLifeFrames or more brings its monsters back at full life.
func (d *Director) RestoreLevel(p *ParkedLevel, now int) int {
	return d.RestoreLevelRules(p, now, 0, false)
}

// Level 108 (the Chaos Sanctuary) and the monster class that survives the
// restore once the level's quest condition holds (exe 0x5401d0: level 0x6c
// with the game predicate 0x5b2e40 true restores only class 0xf3 = Diablo).
const (
	chaosSanctuaryLevel  = 108
	restoreOnlyMonsterID = 0xf3
)

// RestoresMonster reports whether a parked monster of the given monstats class
// comes back on a level. questSet is the exe's predicate for level 108 (the
// Terror's End condition; which exact quest byte it reads is UNVERIFIED).
// The exe tests it only for level 108; elsewhere every monster returns.
func RestoresMonster(level int, questSet bool, class int) bool {
	if level == chaosSanctuaryLevel && questSet {
		return class == restoreOnlyMonsterID
	}

	return true
}

// RestoreLevelRules is RestoreLevel with the level-108 rule: level is the
// level id and questSet the quest predicate (see RestoresMonster). Monsters
// the rule refuses are dropped.
func (d *Director) RestoreLevelRules(p *ParkedLevel, now, level int, questSet bool) int {
	if p == nil {
		return 0
	}

	elapsed := now - p.at
	restored := 0

	for _, u := range p.units {
		if !RestoresMonster(level, questSet, u.m.MonstatID()) {
			continue
		}

		restored++

		u.m.Vitals.HP = RestoredLife(u.m.Vitals.HP, u.m.Vitals.MaxHP, elapsed)

		if u.m.Mode() == d2monster.ModeWalk || u.m.Mode() == d2monster.ModeRun {
			u.m.StopMoving()
		}

		if mode := u.m.Mode(); mode != d2monster.ModeNeutral && mode != d2monster.ModeWalk && mode != d2monster.ModeRun {
			u.m.SetMode(d2monster.ModeNeutral)
		}

		// the brain id keys the director's tables and the footprints, and the
		// block test belongs to this director's footprints: a director is
		// created per map, so the ids of the old one may be taken here
		d.nextID++
		u.b.ID = d.nextID

		b := u.b
		u.m.Blocker = func(x, y int) bool { return d.fp.BlockedFor(b.ID, x, y) }

		u.mv, u.hadTarget, u.attackTarget, u.blocked = nil, false, 0, 0
		u.b.HasTarget, u.b.TargetID = false, 0
		u.b.WakeNow(d.frame)

		x, y := u.m.SubtilePos()
		d.fp.Move(u.b.ID, x, y, d2path.FlagMonster)

		d.units[u.b.ID] = u
		d.byEntity[u.m.ID()] = u
		d.engine.AddEntity(u.m)
	}

	return restored
}

// DiscardPlacements removes the hostile DS1 monster placements of a freshly
// built map that the director has not adopted yet. A restored level brings its
// own monsters, so the placements would be duplicates (and would bring killed
// monsters back). It returns how many were removed.
func (d *Director) DiscardPlacements() int {
	n := 0

	for _, pl := range d.freshPlacements() {
		npc := pl.npc

		stat := d.statByID[npc.MonstatID()]
		if stat == nil || !IsHostile(stat) {
			continue
		}

		d.engine.RemoveEntity(npc)

		n++
	}

	return n
}
