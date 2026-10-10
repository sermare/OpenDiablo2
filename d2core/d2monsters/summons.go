package d2monsters

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Monster-cast summons: the skills.txt rows a monstats skill slot can name
// whose srvdofunc creates monsters (Nest, MinionSpawner, Hydra, DiabPrison). The plan is a pure function of the tables
// (PlanSummon); the Director lands it at the hit frame of the cast
// (landSummon), counts the live summons of the caster and refuses past the
// limit.
//
// Rules, with their status (exe read of 1.14b, notes in d2-re-notes/exe-verify-unverified.md):
//   - class: the row's `summon` column; when empty (Nest, MinionSpawner,
//     EvilHutSpawner) the caster's monstats `spawn` column. Nest (0x5ca0b0) and
//     MinionSpawner (0x5d0ab0) take the class from the skill's stored target
//     data, which the AI queued from that column: consistent, not traced.
//   - cell: Nest and MinionSpawner place at the cell stored in the cast (the
//     caster plus spawnx/spawny; UNVERIFIED: not rotated by facing); Hydra and
//     DiabPrison at the aim point of the cast.
//   - Hydra (0x5c8ab0, VERIFIED): three units per cast at a table of three
//     offsets around the aim point, classes consecutive from the skill's pick,
//     lifetime Param1 frames; petmax is evaluated and handed on, there is no
//     count check in the handler (the pet registry holds the limit).
//   - DiabPrison (0x5cb900 -> 0x5b1100, VERIFIED): four units per cast around
//     the TARGET unit, boneprison1..4 (class 340 plus the entry index), at the
//     subtile offsets (1,1) (1,-1) (-1,-1) (-1,1).
//   - Overseer Whip (131, 0x5d0580) and Impregnate (133, 0x5d0850) create NO
//     unit (VERIFIED): the whip morphs the target into the next class of its
//     chain and applies a state, Impregnate puts a timed state on the target.
//     They are not summons here (the older stand-in laid suicideminion1 and
//     painworm1 at the aim point).
//   - how many per cast, how often, and the stop count are the AI's:
//     FoulCrowNest aip1 gap / aip3 limit, MinionSpawner aip1..aip3 (ported in
//     d2common/d2monster, VERIFIED); a cast spawns len(Cells) units.

// SummonKind classifies a summoning srvdofunc.
type SummonKind int

// Summon kinds.
const (
	SummonNone SummonKind = iota
	SummonNest
	SummonSpawner
	SummonHydra
	SummonPrison
)

var summonKinds = map[int]SummonKind{91: SummonNest, 135: SummonSpawner, 144: SummonHydra, 104: SummonPrison}

// SummonKindOf is the summon kind of a skills.txt srvdofunc.
func SummonKindOf(srvdofunc int) SummonKind { return summonKinds[srvdofunc] }

// SummonPlan is what one cast creates.
type SummonPlan struct {
	Kind  SummonKind
	Class string // monstats key
	// Classes names the class of each Cells entry when they differ (DiabPrison:
	// boneprison1..4); nil means every cell is Class.
	Classes []string
	Mode    string // spawn animation token ("NU"...), informational
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
	case SummonPrison:
		// the 0x73a668 template: four entries, the class advancing one per entry
		p.Cells = [][2]int{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}}
		p.Classes = numberedClasses(p.Class, len(p.Cells))
	}

	if rec.Pettype != "" && !strings.EqualFold(rec.Pettype, "none") && rec.Petmax != nil {
		p.MaxAlive = rec.Petmax.Eval(d2calc.ZeroEnv{})
	}

	if c := HostSummonCap(k, caster.AiKey); p.MaxAlive <= 0 || p.MaxAlive > c {
		p.MaxAlive = c
	}

	return p, true
}

// selfLimitedSummoners are the AI ports whose think function bounds the
// casts itself (cast counters against aip limits, and for the VileMother and
// MinionSpawner a live-brood count the Director answers): FoulCrowNest aip3,
// MosquitoNest aip1, Sarcophagus aip3, MinionSpawner aip1/aip2, VileMother
// aip1/aip2 (all ported VERIFIED in d2common/d2monster). Lower-case names.
var selfLimitedSummoners = map[string]bool{"foulcrownest": true, "mosquitonest": true, "sarcophagus": true,
	"minionspawner": true, "vilemother": true,
	"sandmaggotqueen": true} // SandMaggotQueen: aip1 brood limit (VERIFIED port)

// Host safety net (a bound and not a rule from the exe): the most live
// summons of one caster. What the exe says: no summoning handler counts live
// units. Casting is bounded by the AI: EvilHole (0x5fa590) lays aip1 units in
// total (10 in the 1.14b tables, one per aip2 frames, a total and not a live
// count), MinionSpawner (0x5e1ab0) casts while its cast counter is below aip1
// (100) and its live minions are below aip2 (25), the Hydra handler makes
// three per cast and DiabPrison four per cast with no check at all. So the
// numbers below stay UNVERIFIED host choices (the DiabPrison 4 equals one
// cast, the Nest 10 equals EvilHole's aip1 on every difficulty). A
// self-limited AI never gets near them (largest aip live limit in the 1.14b
// tables is 27); any other AI that casts a summoning skill (the EvilHole and
// HighPriest stand-in thinks cast every few ticks) is held to a few units, so
// a bug or a missing host interface can never make a caster lay units without
// end (the City of the Damned Stygian Hags did).
const (
	selfLimitedSummonCap = 40
	nestSummonCap        = 10
	hydraSummonCap       = 9 // three casts of three
	prisonSummonCap      = 4 // one cast of four
)

// HostSummonCap is the most live units of one caster's summoning skill the
// Director allows, from the skill kind and the caster's monstats AI.
func HostSummonCap(k SummonKind, ai string) int {
	if selfLimitedSummoners[strings.ToLower(ai)] {
		return selfLimitedSummonCap
	}

	switch k {
	case SummonHydra:
		return hydraSummonCap
	case SummonPrison:
		return prisonSummonCap
	default:
		return nestSummonCap
	}
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

// FBXSpawnCellsFree implements d2monster.FBXSpawnCells (Sarcophagus): the same
// room test as the nests.
func (d *Director) FBXSpawnCellsFree(b *d2monster.Brain) bool { return d.SpawnerCellsFree(b) }

// SpawnCellsFree implements d2monster.FB1Spawn (Nihlathak's summon).
func (d *Director) SpawnCellsFree(b *d2monster.Brain) bool { return d.SpawnerCellsFree(b) }

// summonBudget is how many units a cast of plan by the caster may create now:
// the plan size, trimmed so the caster's live summons of the class stay within
// plan.MaxAlive (the table's petmax or the host cap, see HostSummonCap).
func (d *Director) summonBudget(casterID uint32, plan SummonPlan) int {
	if plan.Classes != nil {
		return plan.summonRoom(d.liveSummons(casterID, "")) // the classes differ: count them all
	}

	return plan.summonRoom(d.liveSummons(casterID, plan.Class))
}

// numberedClasses is the class chain base, base+1 ... for n entries, from the
// trailing number of a monstats key ("boneprison1" -> boneprison1..4). The exe
// advances along the table's class links; in the 1.14b table the prison group
// is the numbered keys.
func numberedClasses(base string, n int) []string {
	i := len(base)
	for i > 0 && base[i-1] >= '0' && base[i-1] <= '9' {
		i--
	}

	if i == len(base) {
		return nil
	}

	num, _ := strconv.Atoi(base[i:])
	out := make([]string, n)

	for k := range out {
		out[k] = base[:i] + strconv.Itoa(num+k)
	}

	return out
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

	room := d.summonBudget(u.b.ID, plan)
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

		stat := stat
		if plan.Classes != nil {
			if st := d.FindStat(plan.Classes[i]); st != nil {
				stat = st
			}
		}

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

// SummonStats counts the living units made by monster-cast summons: the total
// and the largest number any one caster has alive (test/diagnostic only).
func (d *Director) SummonStats() (live, maxPerCaster int) {
	per := map[uint32]int{}

	for _, u := range d.units {
		if u.summoner != 0 && u.m.Alive() {
			live++
			per[u.summoner]++

			if per[u.summoner] > maxPerCaster {
				maxPerCaster = per[u.summoner]
			}
		}
	}

	return live, maxPerCaster
}

// ---- VileMother brood (Flesh Spawner, Stygian Hag, Grotesque) ----

// FBXBroodClass implements d2monster.FBXBrood: the class id of the young the
// mother lays (her monstats `spawn` column); -1 when she has none.
func (d *Director) FBXBroodClass(b *d2monster.Brain) int {
	u := d.unitOf(b)
	if u == nil || u.m.Stat.SpawnKey == "" {
		return -1
	}

	if st := d.FindStat(u.m.Stat.SpawnKey); st != nil {
		return st.ID
	}

	return -1
}

// FBXCellFreeAt implements d2monster.FBXBrood: a monster can stand on the
// subtile p.
func (d *Director) FBXCellFreeAt(_ *d2monster.Brain, p d2monster.Point) bool {
	return d.fp == nil || !d2path.Blocked(d.fp, p.X, p.Y, d2path.MaskMonster)
}

// FBXScan implements d2monster.FBXScanner for the one scan the Director
// answers: the VileMother's brood count (aip2, "young alive"), the living
// units of a class within the squared radius of the mother. Without it the
// count was always 0, so every mother laid her whole brood limit (aip1: 21 on
// Hell) one after another and 150 Stygian Dogs killed the level 94 hero in
// the City of the Damned (scenario 9h-act45-playthrough). Other scans keep
// finding nothing.
func (d *Director) FBXScan(b *d2monster.Brain, q d2monster.FBXScanQuery) d2monster.FBXScanResult {
	if q.Kind != d2monster.FBXScanLinkedClass || q.Class < 0 {
		return d2monster.FBXScanResult{}
	}

	n := 0

	for _, u := range d.units {
		if u.b == b || !u.m.Alive() || u.m.Stat.ID != q.Class {
			continue
		}

		dx, dy := u.b.X-b.X, u.b.Y-b.Y
		if dx*dx+dy*dy <= q.Radius2 {
			n++
		}
	}

	return d2monster.FBXScanResult{Count: n, Found: n > 0}
}
