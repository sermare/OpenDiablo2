package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Sand Maggot Queen brood. MONAI_Think_SandMaggotQueen (0x5f8e50, ported
// VERIFIED in d2monster as thinkSandMaggotQueenB) plays the spawn animation
// and then asks the world to lay one young unit near the queen
// (MONSTER_FindSpawnPositionForMinion 0x63fff0); the class is the "summon
// class" kept in the AI data (+0x3c, MONAI_GetSummonClassFromAiData 0x58d4f0).
// The Director did not answer that question, so the queens laid nothing.
//
// The table answers it (verified in the 1.14b monstats): the five queens'
// `spawn` column names the adult young (maggotqueen1 lays sandmaggot1 ... 5,
// with spawnx 8, spawny 0, spawnmode S1), and those SandMaggots lay the eggs
// themselves. UNVERIFIED: that AI data +0x3c is simply that column (Ghidra
// had no program loaded in this pass), and the exact cell search.

// queenMayLay is the host bound (HostSummonCap): the queen is a self-limited
// AI, its own aip1 brood count bounds it, and the cap only backs it up.
func queenMayLay(live int, ai string) bool {
	return live < HostSummonCap(SummonNest, ai)
}

// FBXSpawnMinion implements d2monster.FBXMinionSpawner: lays one young (her
// `spawn` class) on the nearest free cell within 6 subtiles of the spawn
// offset from the queen. Returns true when one was
// created (the queen counts it against aip1).
func (d *Director) FBXSpawnMinion(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u == nil || d.fp == nil || d.asset == nil {
		return false
	}

	key := u.m.Stat.SpawnKey
	if key == "" {
		return false
	}

	stat := d.FindStat(key)
	if stat == nil {
		d.emit("summon", "QUEEN %s: unknown spawn class %q", u.m.Label(), key)

		return false
	}

	if !queenMayLay(d.liveSummons(b.ID, stat.Key), u.m.Stat.AiKey) {
		return false
	}

	p, ok := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: b.X + u.m.Stat.SpawnOffsetX, Y: b.Y + u.m.Stat.SpawnOffsetY}, 6)
	if !ok {
		return false
	}

	m, err := d.spawn(stat, p.X, p.Y, nil)
	if err != nil {
		return false
	}

	nu := d.byEntity[m.ID()]
	nu.summoner = b.ID
	d.Counters.Summoned++
	d.emit("summon", "QUEEN %s laid %s id=%d pos=(%d,%d)", u.m.Label(), stat.Key, nu.b.ID, p.X, p.Y)

	return true
}
