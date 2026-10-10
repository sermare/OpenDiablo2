package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Host side of the EvilHole / HighPriest / GenericSpawner / InvisoSpawner
// ports (d2common/d2monster/ai_faithful_spawners.go, rules in
// d2-re-notes/ai-spawners.md). The AIs bound themselves with aip counters;
// the units they lay are still recorded as the spawner's summons so
// liveSummons and HostSummonCap see them.
//
// Class ids are the exe's hardcoded ids (0x63fff0, 0x5e4fb0, 0x5def70); where
// the table lacks the id the `spawn` column of the spawner is used.

const (
	holeMinionClass     = 0x13  // Fallen, EvilHole BaseId 0x141
	demonHoleMinion     = 0x2c8 // for caster class 0x2c7
	demonHoleClass      = 0x2c7
	invisoClass         = 0x60 // before the level tier scaling of 0x63fcb0 (not ported)
	genericDefaultClass = 0x1c5
	genericFallback     = 0x1f0
	invisoSearchRadius  = 8
)

func (d *Director) spawnClassOf(u *unit, id int) *d2records.MonStatRecord {
	if st := d.statByID[id]; st != nil {
		return st
	}

	if u != nil && u.m.Stat.SpawnKey != "" {
		return d.FindStat(u.m.Stat.SpawnKey)
	}

	return nil
}

func (d *Director) layUnit(u *unit, st *d2records.MonStatRecord, x, y, ring int) bool {
	if st == nil {
		return false
	}

	p, ok := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: x, Y: y}, ring)
	if !ok {
		return false
	}

	m, err := d.spawn(st, p.X, p.Y, nil)
	if err != nil {
		return false
	}

	if nu := d.byEntity[m.ID()]; nu != nil {
		nu.summoner = u.b.ID
	}

	d.Counters.Summoned++

	return true
}

// SpawnHoleMinion implements d2monster.SpawnerHost for EvilHole.
func (d *Director) SpawnHoleMinion(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u == nil {
		return false
	}

	id := holeMinionClass
	if b.Class == demonHoleClass {
		id = demonHoleMinion
	}

	return d.layUnit(u, d.spawnClassOf(u, id), b.X, b.Y, 6)
}

// SpawnRoomUnit implements d2monster.SpawnerHost for InvisoSpawner. The exe
// draws the point from the room's own generator (20 tries) inside the room;
// here the spawner's Aux generator picks a point within invisoSearchRadius
// subtiles (UNVERIFIED simplification) and the class is the unscaled 0x60.
func (d *Director) SpawnRoomUnit(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u == nil {
		return false
	}

	st := d.spawnClassOf(u, invisoClass)

	for try := 0; try < 20; try++ {
		x := b.X + int(b.Aux.Roll(2*invisoSearchRadius+1)) - invisoSearchRadius
		y := b.Y + int(b.Aux.Roll(2*invisoSearchRadius+1)) - invisoSearchRadius

		if d.fp != nil && d2path.Blocked(d.fp, x, y, d2path.MaskMonster) {
			continue
		}

		return d.layUnit(u, st, x, y, 0)
	}

	return false
}

// PickGenericSpawnClass implements d2monster.SpawnerHost for GenericSpawner:
// the default class of 0x5e4fb0 (the level's monster list is not consulted);
// falls back to the spawner's own `spawn` column.
func (d *Director) PickGenericSpawnClass(b *d2monster.Brain) (int, bool) {
	u := d.unitOf(b)

	for _, id := range []int{genericDefaultClass, genericFallback} {
		if st := d.statByID[id]; st != nil {
			return st.ID, true
		}
	}

	if u != nil && u.m.Stat.SpawnKey != "" {
		if st := d.FindStat(u.m.Stat.SpawnKey); st != nil {
			return st.ID, true
		}
	}

	return -1, false
}

// woundedAlly answers FBXScanWoundedAlly (0x5df240): the living monster of
// the caster's side within Radius2 with the lowest life percent below
// LifeBelow.
func (d *Director) woundedAlly(b *d2monster.Brain, q d2monster.FBXScanQuery) d2monster.FBXScanResult {
	best := q.LifeBelow
	var res d2monster.FBXScanResult

	for _, u := range d.units {
		if u.b == b || !u.m.Alive() || u.b.Allied != b.Allied {
			continue
		}

		dx, dy := u.b.X-b.X, u.b.Y-b.Y
		if dx*dx+dy*dy > q.Radius2 {
			continue
		}

		if pct := u.b.HPPercent; pct < best {
			best = pct
			res = d2monster.FBXScanResult{Found: true, T: d2monster.Target{ID: u.b.ID, X: u.b.X, Y: u.b.Y,
				Size: u.b.Size}}
		}
	}

	return res
}
