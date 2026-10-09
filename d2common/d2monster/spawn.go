package d2monster

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monstats"
)

// This file plans natural monster groups from monstats / levels / superuniques
// data. It is pure: it only draws from an RNG and returns a plan; the engine
// places the members and creates the units.
//
// What the notes record (monster-ai.md "Spawn / activation / scaling",
// monster-ai-2.md section 7):
//
//	VERIFIED   per game and level the allowed monster types are drawn without
//	           replacement from the level's mon list, at most 13 types, only
//	           classes with the isSpawn flag (MONREGION_PickLevelMonsterTypes);
//	           the first draw prefers a flag the notes call byte +0xf bit 0x10
//	           (monstats flags bit 28 = rangedtype) when the level asks for it.
//	VERIFIED   group followers are linked with MONAI_AddMinionToLeader; there is
//	           no generic "follow the leader" in the think functions, cohesion
//	           comes from placement and from group commands.
//	VERIFIED   a super unique group is its class plus superuniques MinGrp..MaxGrp
//	           followers (The Countess: MinGrp = MaxGrp = 6). The notes say the
//	           followers come from the class' minion1/minion2 (corruptrogue1/4),
//	           but the real patch_d2 monstats.txt has those columns EMPTY for
//	           corruptrogue3; followers then copy the unique's class (UNVERIFIED
//	           fallback). Real data also shows that only SetBoss classes carry
//	           minions (fallen1: SetBoss BossXfer minion1=fallen1 Party 2..3).
//	VERIFIED   monster level: see ResolveLevel (expansion, Nightmare/Hell, neither
//	           noRatio nor boss).
//
// What the notes explicitly do NOT cover (open question, "NOT covered in this
// pass"): room activation, the preset population pass, champion / unique
// generation and the SetBoss / BossXfer semantics. Everything below that
// touches those is marked UNVERIFIED and is a plain reading of the monstats
// column names.

// MaxLevelTypes is the number of distinct monster types a level may use
// (VERIFIED, min(13, nmon)).
const MaxLevelTypes = 13

// ClassInfo is the spawn-related data of one monstats class.
type ClassInfo struct {
	Class int
	Key   string
	// MinGrp/MaxGrp is the number of monsters of the class in one natural
	// group. PartyMin/PartyMax is the number of followers of the group leader
	// drawn from Minion1/Minion2 (class ids, -1 when unset: class 0 is a real class).
	MinGrp, MaxGrp     int
	Minion1, Minion2   int
	PartyMin, PartyMax int
	Rarity             int
	IsSpawn            bool // monstats isSpawn
	Ranged             bool // monstats rangedtype
	NoRatio            bool // monstats noRatio
	Boss               bool // monstats boss
	SetBoss            bool // UNVERIFIED: the class leads the followers it spawns
	BossXfer           bool // UNVERIFIED: leadership passes on when the leader dies
}

// ClassSource resolves a class id.
type ClassSource interface {
	Class(id int) (ClassInfo, bool)
}

// Member is one planned unit of a group.
type Member struct {
	Class  int
	Minion bool // drawn from minion1/minion2 (a follower of the leader)
	Unique bool // the super unique itself
}

// Pack is a planned group: Members[0] is the leader, the others follow it.
type Pack struct {
	Members []Member
	// SuperUnique is the superuniques key when the leader is one.
	SuperUnique string
}

// roll draws in [0,n) (n<=0 gives 0 without consuming the RNG, like
// RAND_RollSeedBounded).
func roll(r *d2rand.Seed, n int) int {
	if n <= 0 {
		return 0
	}

	return int(r.Roll(int32(n)))
}

func between(r *d2rand.Seed, lo, hi int) int {
	if hi < lo {
		hi = lo
	}

	return lo + roll(r, hi-lo+1)
}

// PlanGroup plans a natural group of a class: MinGrp..MaxGrp monsters of the
// class (at least one); the leader additionally brings PartyMin..PartyMax
// followers drawn from minion1/minion2 (UNVERIFIED: that this applies to
// ordinary groups and not only to uniques).
func PlanGroup(r *d2rand.Seed, info ClassInfo) Pack {
	n := between(r, info.MinGrp, info.MaxGrp)
	if n < 1 {
		n = 1
	}

	p := Pack{}
	for i := 0; i < n; i++ {
		p.Members = append(p.Members, Member{Class: info.Class})
	}

	if cls := minionClasses(info); len(cls) > 0 {
		for i, party := 0, between(r, info.PartyMin, info.PartyMax); i < party; i++ {
			p.Members = append(p.Members, Member{Class: cls[roll(r, len(cls))], Minion: true})
		}
	}

	return p
}

// minionClasses lists the set minion classes.
func minionClasses(info ClassInfo) []int {
	var out []int

	for _, m := range []int{info.Minion1, info.Minion2} {
		if m >= 0 && (len(out) == 0 || out[0] != m) {
			out = append(out, m)
		}
	}

	return out
}

// PlanSuperUnique plans a super unique: the unique itself plus
// grpMin..grpMax (superuniques MinGrp/MaxGrp) followers drawn from the
// class' minion1/minion2 (VERIFIED for the Countess). If the class has no
// minions the followers are of its own class (UNVERIFIED).
func PlanSuperUnique(r *d2rand.Seed, key string, info ClassInfo, grpMin, grpMax int) Pack {
	p := Pack{SuperUnique: key, Members: []Member{{Class: info.Class, Unique: true}}}

	cls := minionClasses(info)
	if len(cls) == 0 {
		cls = []int{info.Class}
	}

	for i, n := 0, between(r, grpMin, grpMax); i < n; i++ {
		p.Members = append(p.Members, Member{Class: cls[roll(r, len(cls))], Minion: true})
	}

	return p
}

// PickLevelTypes is MONREGION_PickLevelMonsterTypes: from the level's monster
// list it draws up to min(MaxLevelTypes, nmon) distinct spawnable classes
// without replacement. When preferRanged (levels.txt rangedspawn, UNVERIFIED
// as the meaning of the exe's flag test) is set and a ranged class exists,
// the first draw is among the ranged classes only.
func PickLevelTypes(r *d2rand.Seed, list []ClassInfo, nmon int, preferRanged bool) []ClassInfo {
	pool := make([]ClassInfo, 0, len(list))

	for _, c := range list {
		if c.IsSpawn {
			pool = append(pool, c)
		}
	}

	limit := nmon
	if limit > MaxLevelTypes || limit <= 0 {
		limit = MaxLevelTypes
	}

	var out []ClassInfo

	for len(out) < limit && len(pool) > 0 {
		cand := indices(pool, func(c ClassInfo) bool { return len(out) == 0 && preferRanged && c.Ranged })
		if len(cand) == 0 {
			cand = indices(pool, func(ClassInfo) bool { return true })
		}

		k := cand[roll(r, len(cand))]
		out = append(out, pool[k])
		pool = append(pool[:k], pool[k+1:]...)
	}

	return out
}

func indices(pool []ClassInfo, keep func(ClassInfo) bool) []int {
	var out []int

	for i, c := range pool {
		if keep(c) {
			out = append(out, i)
		}
	}

	return out
}

// PickByRarity draws one class from the level's types weighted by monstats
// Rarity (a rarity of 0 counts as 1 so a class is never unreachable).
func PickByRarity(r *d2rand.Seed, types []ClassInfo) (ClassInfo, bool) {
	if len(types) == 0 {
		return ClassInfo{}, false
	}

	total := 0
	for _, t := range types {
		total += rarityOf(t)
	}

	n := roll(r, total)
	for _, t := range types {
		if n < rarityOf(t) {
			return t, true
		}

		n -= rarityOf(t)
	}

	return types[len(types)-1], true
}

func rarityOf(c ClassInfo) int {
	if c.Rarity < 1 {
		return 1
	}

	return c.Rarity
}

// GroupsForRoom estimates how many natural groups a room of the given size
// (in tiles) holds. The exe's population pass is not in the notes
// (UNVERIFIED): the monster count is tiles*density/100000 (density = levels
// MonDen, 0 for towns) with avgGroup monsters per group, rounded to nearest,
// at least 0.
func GroupsForRoom(tiles, density, avgGroup int) int {
	if tiles <= 0 || density <= 0 {
		return 0
	}

	if avgGroup < 1 {
		avgGroup = 1
	}

	monsters := tiles * density / 100000

	return (monsters + avgGroup/2) / avgGroup
}

// AvgGroupSize is the mean MinGrp..MaxGrp size over the types, weighted by
// rarity, floored at 1.
func AvgGroupSize(types []ClassInfo) int {
	sum, w := 0, 0

	for _, t := range types {
		sum += (t.MinGrp + t.MaxGrp) * rarityOf(t) // twice the mean
		w += rarityOf(t)
	}

	if w == 0 || sum/(2*w) < 1 {
		return 1
	}

	return sum / (2 * w)
}

// ResolveLevel is the monster level rule (VERIFIED 0x00571c4f, see
// d2monstats.Class.ResolveLevel, to which it delegates): the monstats Level
// of the difficulty (statLevels, indexed normal/nightmare/hell) is the
// default; the area's levels.txt MonLvl replaces it only in an expansion game,
// in Nightmare or Hell, for classes with neither noRatio nor boss, when the
// area level is known (> 0).
func ResolveLevel(info ClassInfo, diff Difficulty, statLevels [3]int, areaLevel int, expansion bool) int {
	c := d2monstats.Class{NoRatio: info.NoRatio, Boss: info.Boss, Level: statLevels}

	return c.ResolveLevel(clampDiff(int(diff)), areaLevel, expansion)
}

func clampDiff(d int) int {
	if d < 0 {
		return 0
	}

	if d > 2 {
		return 2
	}

	return d
}

// LeaderSuccessor picks the new leader of a group when the leader died and
// the class has BossXfer set (UNVERIFIED reading of the flag): the first
// living follower. It returns the index into live, or -1.
func LeaderSuccessor(info ClassInfo, live []bool) int {
	if !info.BossXfer {
		return -1
	}

	for i, ok := range live {
		if ok {
			return i
		}
	}

	return -1
}

// SortTypesByClass orders types by class id (tests and deterministic logs).
func SortTypesByClass(t []ClassInfo) {
	sort.Slice(t, func(i, j int) bool { return t[i].Class < t[j].Class })
}
