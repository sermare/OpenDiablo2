package d2monsters

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Group placement parameters (engine choices, UNVERIFIED: the notes do not
// record how the original places a group inside a room).
const (
	packSpacing   = 2  // subtiles between group members
	packMinRadius = 6  // search radius around the group centre
	packPerMember = 2  // extra radius per member
	packMaxRadius = 24 // never search farther than this
	roomFreeRing  = 8  // ring searched for a free centre cell inside a room
)

// Room is a rectangle of subtiles that can be populated.
type Room struct{ X0, Y0, W, H int }

// PackResult is a spawned natural group.
type PackResult struct {
	Plan     d2monster.Pack
	Leader   *d2mapentity.Monster
	Monsters []*d2mapentity.Monster
}

// Class implements d2monster.ClassSource from the loaded monstats.
func (d *Director) Class(id int) (d2monster.ClassInfo, bool) {
	st := d.statByID[id]
	if st == nil {
		return d2monster.ClassInfo{}, false
	}

	return d.classInfo(st), true
}

// classInfo converts a monstats row to the data the group planner reads.
func (d *Director) classInfo(st *d2records.MonStatRecord) d2monster.ClassInfo {
	c := d2monster.ClassInfo{
		Class: st.ID, Key: st.Key,
		MinGrp: st.MinionGroupMin, MaxGrp: st.MinionGroupMax,
		PartyMin: st.MinionPartyMin, PartyMax: st.MinionPartyMax,
		Rarity: st.Rarity, IsSpawn: st.IsLevelSpawnable, Ranged: st.IsRanged,
		NoRatio: st.IgnoreMonLevelTxt, Boss: st.IsSpecialBoss,
		SetBoss: st.IsLeader, BossXfer: st.TransferLeadership,
		Minion1: -1, Minion2: -1,
	}

	if m := d.minionStat(st.MinionId1); m != nil {
		c.Minion1 = m.ID
	}

	if m := d.minionStat(st.MinionId2); m != nil {
		c.Minion2 = m.ID
	}

	return c
}

func (d *Director) minionStat(key string) *d2records.MonStatRecord {
	if strings.TrimSpace(key) == "" {
		return nil
	}

	return d.asset.Records.Monster.Stats[key]
}

// SetAreaLevel sets the levels.txt monster level new spawns use for classes
// that are not noRatio/boss (0 = unknown: the monstats level is used).
func (d *Director) SetAreaLevel(level int) { d.areaLevel = level }

// AreaLevelOf is the MonLvl of a levels.txt area for the director's
// difficulty (the Ex columns for the expansion), or 0 when unknown.
func (d *Director) AreaLevelOf(levelID int) int {
	det := d.asset.Records.GetLevelDetails(levelID)
	if det == nil {
		return 0
	}

	pick := [3][2]int{
		{det.MonsterLevelNormal, det.MonsterLevelNormalEx},
		{det.MonsterLevelNightmare, det.MonsterLevelNightmareEx},
		{det.MonsterLevelHell, det.MonsterLevelHellEx},
	}[d.opt.Difficulty]

	if d.opt.Expansion {
		return pick[1]
	}

	return pick[0]
}

// SpawnGroup plans and spawns a natural group of a class around a centre:
// MinGrp..MaxGrp monsters, the first leads the others.
func (d *Director) SpawnGroup(stat *d2records.MonStatRecord, center d2path.Point) (*PackResult, error) {
	return d.SpawnPack(d2monster.PlanGroup(d.packRNG, d.classInfo(stat)), center)
}

// SpawnSuperUnique spawns a superuniques.txt boss with its followers
// (MinGrp..MaxGrp from the row). The unique's modifiers (monumod) and its
// hit point bonus are not applied.
func (d *Director) SpawnSuperUnique(key string, center d2path.Point) (*PackResult, error) {
	rec := d.asset.Records.Monster.Unique.Super[key]
	if rec == nil {
		return nil, fmt.Errorf("unknown super unique %q", key)
	}

	stat := d.FindStat(rec.Class)
	if stat == nil {
		return nil, fmt.Errorf("super unique %q: unknown class %q", key, rec.Class)
	}

	return d.SpawnPack(d2monster.PlanSuperUnique(d.packRNG, key, d.classInfo(stat), rec.MinGrp, rec.MaxGrp), center)
}

// memberTypeFlags is the +0x16 type mask of a pack member (see the
// MonType* constants): super unique leader 2 and unique leader 8, champions
// 4 each, followers of a super unique or unique leader 0x10 (minion). The
// followers of an ordinary group stay unflagged.
func memberTypeFlags(plan d2monster.Pack, leader bool) uint16 {
	switch {
	case plan.SuperUnique != "":
		if leader {
			return d2mapentity.MonTypeSuperUnique
		}

		return d2mapentity.MonTypeMinion
	case plan.Kind == d2monster.PackChampion:
		return d2mapentity.MonTypeChampion
	case plan.Kind == d2monster.PackUnique:
		if leader {
			return d2mapentity.MonTypeUnique
		}

		return d2mapentity.MonTypeMinion
	}

	return 0
}

// SpawnChampionGroup spawns a champion pack of a class (all members champions).
// PopulateRoom does not roll for champion / unique packs yet (the exe's
// chances are not recorded in the notes).
func (d *Director) SpawnChampionGroup(stat *d2records.MonStatRecord, center d2path.Point) (*PackResult, error) {
	return d.SpawnPack(d2monster.PlanChampion(d.packRNG, d.classInfo(stat)), center)
}

// SpawnUniqueGroup spawns a unique (rare) pack: a unique leader and minions.
func (d *Director) SpawnUniqueGroup(stat *d2records.MonStatRecord, center d2path.Point) (*PackResult, error) {
	return d.SpawnPack(d2monster.PlanUnique(d.packRNG, d.classInfo(stat)), center)
}

// SpawnPack creates the units of a plan on free cells around centre, links
// the followers to the leader (MONAI_AddMinionToLeader) and logs the
// composition. Members that find no free cell are dropped.
func (d *Director) SpawnPack(plan d2monster.Pack, center d2path.Point) (*PackResult, error) {
	if len(plan.Members) == 0 {
		return nil, fmt.Errorf("empty pack")
	}

	radius := packMinRadius + packPerMember*len(plan.Members)
	if radius > packMaxRadius {
		radius = packMaxRadius
	}

	cells := d2path.PlaceCluster(d.fp, d2path.MaskMonster|d2path.MaskUnits, center, len(plan.Members), packSpacing, radius)
	res := &PackResult{Plan: plan}

	for i, c := range cells {
		stat := d.statByID[plan.Members[i].Class]
		if stat == nil {
			continue
		}

		m, err := d.Spawn(stat, c.X, c.Y)
		if err != nil {
			d.Infof("pack member %s not spawned: %v", stat.Key, err)

			continue
		}

		m.TypeFlags |= memberTypeFlags(plan, res.Leader == nil)

		if res.Leader == nil {
			res.Leader = m
			d.recordRank(m, plan)
		} else {
			res.Leader.Brain.AddMinion(m.Brain)
			m.LeaderID = res.Leader.Brain.ID
			m.Modifiers = append([]int(nil), res.Leader.Modifiers...) // MONSTER_CopyLeaderUModsToMinion

			if plan.Kind == d2monster.PackChampion { // a champion pack shares the leader's modifiers
				m.TypeFlags |= d2mapentity.MonTypeModsRolled
			}
		}

		res.Monsters = append(res.Monsters, m)
	}

	if res.Leader == nil {
		return nil, fmt.Errorf("no room for a pack at (%d,%d)", center.X, center.Y)
	}

	d.Counters.Packs++
	d.emit("pack", "MONSTER pack leader=%s id=%d size=%d planned=%d followers=%d unique=%q area_level=%d composition=[%s]",
		res.Leader.Label(), res.Leader.Brain.ID, len(res.Monsters), len(plan.Members), len(res.Leader.Brain.Minions),
		plan.SuperUnique, d.areaLevel, d.composition(plan, len(res.Monsters)))

	return res, nil
}

// composition summarises a plan as "key:count" (followers drawn from the
// minion columns are marked with a +).
func (d *Director) composition(plan d2monster.Pack, spawned int) string {
	counts := map[string]int{}

	for i, m := range plan.Members {
		if i >= spawned {
			break
		}

		name := fmt.Sprint(m.Class)
		if st := d.statByID[m.Class]; st != nil {
			name = st.Key
		}

		if m.Minion {
			name += "+"
		}

		counts[name]++
	}

	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}

	sort.Strings(names)

	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = fmt.Sprintf("%s:%d", n, counts[n])
	}

	return strings.Join(parts, " ")
}

// levelMonsterNames lists the mon1..mon10 columns of a levels.txt row for a
// difficulty.
func levelMonsterNames(det *d2records.LevelDetailRecord, diff d2monster.Difficulty) []string {
	switch diff {
	case d2monster.Nightmare:
		return []string{det.MonsterID1Nightmare, det.MonsterID2Nightmare, det.MonsterID3Nightmare,
			det.MonsterID4Nightmare, det.MonsterID5Nightmare, det.MonsterID6Nightmare, det.MonsterID7Nightmare,
			det.MonsterID8Nightmare, det.MonsterID9Nightmare, det.MonsterID10Nightmare}
	case d2monster.Hell:
		return []string{det.MonsterID1Hell, det.MonsterID2Hell, det.MonsterID3Hell, det.MonsterID4Hell,
			det.MonsterID5Hell, det.MonsterID6Hell, det.MonsterID7Hell, det.MonsterID8Hell, det.MonsterID9Hell,
			det.MonsterID10Hell}
	}

	return []string{det.MonsterID1Normal, det.MonsterID2Normal, det.MonsterID3Normal, det.MonsterID4Normal,
		det.MonsterID5Normal, det.MonsterID6Normal, det.MonsterID7Normal, det.MonsterID8Normal,
		det.MonsterID9Normal, det.MonsterID10Normal}
}

// PopulateRoom fills a room of a levels.txt area with natural groups: the
// level's monster types are drawn once (at most 13, without replacement), the
// number of groups follows the area's density, and each group picks a type by
// rarity and a free centre cell inside the room. Champion and unique packs
// (umon, MonUMin/Max) are not generated (the notes do not cover them).
func (d *Director) PopulateRoom(room Room, levelID int) ([]*PackResult, error) {
	det := d.asset.Records.GetLevelDetails(levelID)
	if det == nil {
		return nil, fmt.Errorf("unknown level %d", levelID)
	}

	d.SetAreaLevel(d.AreaLevelOf(levelID))

	var list []d2monster.ClassInfo

	for _, name := range levelMonsterNames(det, d.opt.Difficulty) {
		if st := d.minionStat(name); st != nil {
			list = append(list, d.classInfo(st))
		}
	}

	types := d2monster.PickLevelTypes(d.packRNG, list, det.NumMonsterTypes, det.MonsterPreferRanged)
	density := [3]int{det.MonsterDensityNormal, det.MonsterDensityNightmare, det.MonsterDensityHell}[d.opt.Difficulty]
	groups := d2monster.GroupsForRoomFrac(room.W*room.H/(subtilesPerTile*subtilesPerTile), density,
		d2monster.AvgGroupSize(types), func(n int) int { return int(d.packRNG.Roll(int32(n))) })

	var out []*PackResult

	for i := 0; i < groups; i++ {
		ci, ok := d2monster.PickByRarity(d.packRNG, types)
		if !ok {
			break
		}

		stat := d.statByID[ci.Class]
		centre := d2path.Point{X: room.X0 + int(d.packRNG.Roll(int32(room.W))), Y: room.Y0 + int(d.packRNG.Roll(int32(room.H)))}

		free, ok := d2path.NearestFree(d.fp, d2path.MaskMonster|d2path.MaskUnits, centre, roomFreeRing)
		if !ok {
			continue
		}

		if res, err := d.SpawnGroup(stat, free); err == nil {
			out = append(out, res)
		}
	}

	return out, nil
}

// leaderDied updates the group bookkeeping when a unit dies: a dead follower
// leaves its leader's list; a dead leader releases its followers
// (MONAI_ReleaseAllMinions), except when the class has BossXfer, in which
// case the first living follower takes over the others (UNVERIFIED reading
// of the flag).
func (d *Director) leaderDied(u *unit) {
	b := u.b

	if b.Leader != nil {
		b.Leader.RemoveMinion(b)

		return
	}

	if len(b.Minions) == 0 {
		return
	}

	followers := append([]*d2monster.Brain(nil), b.Minions...)
	live := make([]bool, len(followers))

	for i, f := range followers {
		fu := d.units[f.ID]
		live[i] = fu != nil && fu.m.Alive()

		b.RemoveMinion(f)
	}

	info, _ := d.Class(b.Class)
	succ := d2monster.LeaderSuccessor(info, live)

	if succ < 0 {
		return
	}

	nl := followers[succ]

	for i, f := range followers {
		if i != succ && live[i] {
			nl.AddMinion(f)
		}
	}

	d.emit("leader", "MONSTER leader id=%d died, id=%d takes over %d followers", b.ID, nl.ID, len(nl.Minions))
}

// recordRank stores what makes the leader of a pack special on its unit: the
// super unique key and hcIdx with the modifiers of its row (Mod1..3), or the
// modifiers rolled for a champion / unique (type bit 0x1 = modifiers rolled).
// The counts per rank and difficulty are UNVERIFIED: champion 2, unique
// 1 + difficulty + 1 (Mod1..3 of a super unique are used as they are).
func (d *Director) recordRank(m *d2mapentity.Monster, plan d2monster.Pack) {
	switch {
	case plan.SuperUnique != "":
		m.SuperUnique = plan.SuperUnique

		if rec := d.asset.Records.Monster.Unique.Super[plan.SuperUnique]; rec != nil {
			m.SuperUniqueIdx, _ = strconv.Atoi(rec.HcIdx)

			if rec.Name != "" { // the boss shows (and reports in kill events) its own name, "Ismail Vilehand"
				m.SetLabel(rec.Name)
			}

			for _, id := range rec.Mod {
				if id > 0 {
					m.Modifiers = append(m.Modifiers, id)
				}
			}
		}

		m.TypeFlags |= d2mapentity.MonTypeModsRolled
	case m.TypeFlags&(d2mapentity.MonTypeChampion|d2mapentity.MonTypeUnique) != 0:
		n, champion := 2+int(d.opt.Difficulty), m.TypeFlags&d2mapentity.MonTypeChampion != 0
		if champion {
			n = 2
		}

		m.Modifiers = d2monster.PickUniqueMods(d.packRNG, d.modCandidates(champion), n)
		m.TypeFlags |= d2mapentity.MonTypeModsRolled
	}
}

// modCandidates lists the monumod rows a champion or unique may roll with the
// cpick / upick weight of the game's difficulty (enabled rows; the champion-only
// flag and the expansion-only rows follow the columns). Rows 1 and 2 (the name
// seed and the hit point bonus) are always added by the game and never picked.
func (d *Director) modCandidates(champion bool) []d2monster.ModCandidate {
	var out []d2monster.ModCandidate

	for _, r := range d.asset.Records.Monster.Unique.Mods {
		if !r.Enabled || r.ID <= 2 || (r.ExpansionOnly && !d.opt.Expansion) || (r.Champion && !champion) {
			continue
		}

		pf := [3]*d2records.PickFreq{r.PickFrequencies.Normal, r.PickFrequencies.Nightmare, r.PickFrequencies.Hell}[d.opt.Difficulty]
		w := pf.Unique

		if champion {
			w = pf.Champion
		}

		out = append(out, d2monster.ModCandidate{ID: r.ID, Weight: w})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID }) // map order must not change the roll

	return out
}
