package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
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
func (d *Director) ParkLevel() *ParkedLevel {
	out := &ParkedLevel{}

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
// were restored.
func (d *Director) RestoreLevel(p *ParkedLevel) int {
	if p == nil {
		return 0
	}

	for _, u := range p.units {
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

	return len(p.units)
}

// DiscardPlacements removes the hostile DS1 monster placements of a freshly
// built map that the director has not adopted yet. A restored level brings its
// own monsters, so the placements would be duplicates (and would bring killed
// monsters back). It returns how many were removed.
func (d *Director) DiscardPlacements() int {
	n := 0

	for id, e := range d.engine.Entities() {
		npc, ok := e.(*d2mapentity.NPC)
		if !ok || d.seenNPC[id] {
			continue
		}

		d.seenNPC[id] = true

		stat := d.statByID[npc.MonstatID()]
		if stat == nil || !IsHostile(stat) {
			continue
		}

		d.engine.RemoveEntity(npc)

		n++
	}

	return n
}
