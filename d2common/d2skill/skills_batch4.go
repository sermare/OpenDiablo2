package d2skill

// Batch 4 of the exe re-reads (docs: d2-re-notes/skills-batch4.md). Addresses
// are Game.exe 1.14b; VERIFIED marks a rule read in the decompilation.

// NextAfter is the target pick of SKILL_ScanRadiusForTargets (0x569a80, its
// per-unit callback at 0x569a40, VERIFIED): among the ids that are in range
// (sorted ascending) it returns the index of the smallest id greater than
// last, and when none is greater it wraps to the smallest id of all. An empty
// list gives -1. Thunder Storm stores the id of the unit it struck last and
// passes it here the next time, so the strikes walk through every unit in
// range.
func NextAfter(ids []string, last string) int {
	if len(ids) == 0 {
		return -1
	}

	for i, id := range ids {
		if id > last {
			return i
		}
	}

	return 0
}

// StaticFieldDamage is the per-unit callback of SRVDO_StaticField (the code at
// 0x5c7750, called through SKILL_ForEachUnitInRadius; VERIFIED). hp and maxHP
// are whole life points. A unit whose life is at or below floorPct percent of
// its maximum is skipped (nothing happens, the callback returns 0); floorPct
// is the DifficultyLevels StaticFieldMin of the game's difficulty, 0 turning
// the test off. Otherwise the damage is calc1 percent of the CURRENT life, cut
// so that at least 1 life remains, and not lower than calc2 (that comparison
// is made on the 8.8 value, so minRaw is calc2 as the calc gives it). The
// floor is only a gate: one blow may take a unit below it.
func StaticFieldDamage(hp, maxHP, pct, minRaw, floorPct int) int {
	if hp < 1 {
		return 0
	}

	if floorPct != 0 && hp <= maxHP*floorPct/100 {
		return 0
	}

	dmg := hp * pct / 100
	if hp-dmg < 1 {
		dmg = hp - 1
	}

	if dmg<<8 < minRaw {
		dmg = minRaw >> 8
	}

	return dmg
}

// BonePrisonOffsets are the twelve cells SRVDO_062 (0x5c3c50, VERIFIED) places
// a prison piece on, relative to the target point: the x offsets come from the
// table at 0x6e45e0 and the y offsets from 0x6e4610, one pair per index (a
// ring about four subtiles from the centre, not the eight cells of radius 2
// the first port used). A piece whose cell cannot be spawned on is skipped;
// the first piece that does spawn becomes the leader the others are linked to.
var BonePrisonOffsets = [12][2]int{
	{-1, -4}, {1, -4}, {3, -3}, {4, -1}, {4, 1}, {3, 3},
	{-1, 4}, {1, 4}, {-3, 3}, {-4, -1}, {-4, 1}, {-3, -3},
}

// PoisonExplosionCells are the cells, relative to the corpse, where
// SRVDO_063 (0x5c3dc0, VERIFIED) puts its srvmissilea (poisonexplosioncloud)
// through MISSILE_SpawnSubMissileFanFromTable (0x5a6d70): the helper walks the
// two 16-entry tables at 0x6e39c0 (x) and 0x6e3980 (y), a square ring of
// radius 2 around the corpse, with the stride 2 the skill passes, which keeps
// every second entry. The cloud missile does not move (velocity 0) and lives
// 60 frames, so the skill leaves eight stationary poison clouds instead of a
// single instant area hit. The corpse gets state 0x76 (spent).
var PoisonExplosionCells = [8][2]int{
	{0, 2}, {2, 2}, {2, 0}, {2, -2}, {0, -2}, {-2, -2}, {-2, 0}, {-2, 2},
}
