package d2monsters

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Monster-cast summons: the skills.txt rows a monstats skill slot can name
// whose srvdofunc creates monsters (Nest, MinionSpawner, Overseer Whip,
// Impregnate, Hydra, DiabPrison). The plan is a pure function of the tables
// (PlanSummon); the Director lands it at the hit frame of the cast
// (landSummon), counts the live summons of the caster and refuses past the
// limit.
//
// Rules taken from the tables, with their status:
//   - class: the row's `summon` column; when empty (Nest, MinionSpawner,
//     EvilHutSpawner) the caster's monstats `spawn` column. UNVERIFIED for the
//     fallback (the exe helper 0x63fac0 MONSTER_ComputeSpawnOffsetFromSkill is
//     only named in the notes, not read; Ghidra was unreachable).
//   - cell: Nest and MinionSpawner place at the caster plus monstats
//     spawnx/spawny subtiles (UNVERIFIED: not rotated by facing); the others
//     at the aim point of the cast.
//   - Hydra: three at (-1,-1) (0,0) (1,-1) around the aim point, lifetime
//     Param1 frames (both from d2skill.doHydraFn, VERIFIED/U there); the
//     limit is petmax (99 in the 1.14b table).
//   - how many per cast, how often, and the stop count are the AI's:
//     FoulCrowNest aip1 gap / aip3 limit, MinionSpawner aip1..aip3 (ported in
//     d2common/d2monster, VERIFIED); a cast spawns len(Cells) units.
//   - Overseer Whip (131) also buffs its target with bloodlust in the exe; the
//     buff is NOT simulated, only the table's summon column. UNVERIFIED.
//   - DiabPrison: one boneprison1 at the aim point (UNVERIFIED count).

// SummonKind classifies a summoning srvdofunc.
type SummonKind int

// Summon kinds.
const (
	SummonNone SummonKind = iota
	SummonNest
	SummonSpawner
	SummonWhip
	SummonWorm
	SummonHydra
	SummonPrison
)

var summonKinds = map[int]SummonKind{91: SummonNest, 135: SummonSpawner, 131: SummonWhip,
	133: SummonWorm, 144: SummonHydra, 104: SummonPrison}

// SummonKindOf is the summon kind of a skills.txt srvdofunc.
func SummonKindOf(srvdofunc int) SummonKind { return summonKinds[srvdofunc] }

// SummonPlan is what one cast creates.
type SummonPlan struct {
	Kind  SummonKind
	Class string // monstats key
	Mode  string // spawn animation token ("NU"...), informational
	// Cells are subtile offsets of each unit from the anchor.
	Cells [][2]int
	// AtCaster anchors at the caster (Cells[0] is the monstats spawn offset);
	// false anchors at the aim point of the cast.
	AtCaster bool
	// MaxAlive is the most live summons of Class one caster may have (0 = no
	// limit from the table).
	MaxAlive int
	// Frames is the lifetime (0 = until killed).
	Frames int
	UMod   int
}

// PlanSummon reads the plan of a cast of rec by caster; ok is false when the
// row summons nothing or names no class.
func PlanSummon(rec *d2records.SkillRecord, caster *d2records.MonStatRecord) (SummonPlan, bool) {
	if rec == nil || caster == nil {
		return SummonPlan{}, false
	}

	k := SummonKindOf(rec.Srvdofunc)
	if k == SummonNone {
		return SummonPlan{}, false
	}

	p := SummonPlan{Kind: k, Class: rec.Summon, Mode: rec.Summode, UMod: rec.Sumumod, Cells: [][2]int{{0, 0}}}

	if p.Class == "" {
		p.Class = caster.SpawnKey
	}

	if p.Class == "" {
		return SummonPlan{}, false
	}

	switch k {
	case SummonNest, SummonSpawner:
		p.AtCaster = true
		p.Cells = [][2]int{{caster.SpawnOffsetX, caster.SpawnOffsetY}}

		if caster.SpawnAnimationKey != "" {
			p.Mode = caster.SpawnAnimationKey
		}
	case SummonHydra:
		p.Cells = [][2]int{{-1, -1}, {0, 0}, {1, -1}}
		p.Frames = rec.Param1
	}

	if rec.Pettype != "" && !strings.EqualFold(rec.Pettype, "none") && rec.Petmax != nil {
		p.MaxAlive = rec.Petmax.Eval(d2calc.ZeroEnv{})
	}

	return p, true
}

// summonRoom is how many of the plan's units a caster with alive live
// summons of the class may create now.
func (p SummonPlan) summonRoom(alive int) int {
	n := len(p.Cells)
	if p.MaxAlive > 0 && alive+n > p.MaxAlive {
		n = p.MaxAlive - alive
	}

	if n < 0 {
		n = 0
	}

	return n
}

// ---- Director ----

// liveSummons counts the living units the caster summoned of a class
// (case-insensitive monstats key; "" counts every class).
func (d *Director) liveSummons(casterID uint32, class string) int {
	n := 0

	for _, u := range d.units {
		if u.summoner == casterID && u.summoner != 0 && u.m.Alive() &&
			(class == "" || strings.EqualFold(u.m.Stat.Key, class)) {
			n++
		}
	}

	return n
}

// CountMinions implements d2monster.MinionCounter (MinionSpawner): its live
// summons of the class its `spawn` column names.
func (d *Director) CountMinions(b *d2monster.Brain) int {
	u := d.unitOf(b)
	if u == nil {
		return 0
	}

	return d.liveSummons(b.ID, u.m.Stat.SpawnKey)
}

// SpawnerCellsFree implements d2monster.SpawnSpace: the cell a nest lays its
// spawn on has room (UNVERIFIED: the exe tests the spawner cells around the
// tile; here the offset cell or a free one within 3 subtiles).
func (d *Director) SpawnerCellsFree(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u == nil || d.fp == nil {
		return true
	}

	_, ok := d2path.NearestFree(d.fp, d2path.MaskMonster,
		d2path.Point{X: b.X + u.m.Stat.SpawnOffsetX, Y: b.Y + u.m.Stat.SpawnOffsetY}, 3)

	return ok
}

// landSummon creates the units of a summoning cast at its hit frame. It
// returns false when the skill is not a summon.
func (d *Director) landSummon(u *unit) bool {
	if u.skill == nil || u.skill.rec == nil {
		return false
	}

	plan, ok := PlanSummon(u.skill.rec, u.m.Stat)
	if !ok {
		return false
	}

	stat := d.FindStat(plan.Class)
	if stat == nil {
		d.emit("summon", "SUMMON refused caster=%s skill=%s class=%s: unknown class", u.m.Label(), u.skill.name, plan.Class)

		return true
	}

	room := plan.summonRoom(d.liveSummons(u.b.ID, plan.Class))
	if room == 0 {
		d.emit("summon", "SUMMON refused caster=%s skill=%s class=%s: limit %d", u.m.Label(), u.skill.name, plan.Class,
			plan.MaxAlive)

		return true
	}

	ax, ay := u.aimX, u.aimY
	if plan.AtCaster {
		ax, ay = u.m.SubtilePos()
	}

	for i := 0; i < room; i++ {
		c := plan.Cells[i]

		p, found := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: ax + c[0], Y: ay + c[1]}, 6)
		if !found {
			continue
		}

		m, err := d.spawn(stat, p.X, p.Y, nil)
		if err != nil {
			continue
		}

		nu := d.byEntity[m.ID()]
		nu.summoner = u.b.ID

		if plan.Frames > 0 {
			nu.b.Scratch[0] = d.frame + plan.Frames // Hydra AI expiry frame (VERIFIED slot)
		}

		d.Counters.Summoned++
		d.emit("summon", "SUMMON caster=%s skill=%s class=%s id=%d pos=(%d,%d) frames=%d", u.m.Label(), u.skill.name,
			stat.Key, nu.b.ID, p.X, p.Y, plan.Frames)
	}

	return true
}
