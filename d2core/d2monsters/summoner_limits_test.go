package d2monsters

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// auditWorld is the host of the summoner audit: a World whose Cast makes the
// same decision landSummon does (PlanSummon, then the Director's own live
// count and summonBudget) and whose optional host interfaces are the
// Director's real ones (CountMinions, FBXScan, FBXBroodClass, spawn cells), on
// a real Director that holds the units. Nothing else of the engine is needed.
type auditWorld struct {
	d      *Director
	frame  int
	dist   int
	near   bool
	rec    map[int]*d2records.SkillRecord // monstats slot -> skills.txt row
	stats  map[string]*d2records.MonStatRecord
	casts  map[uint32]int
	nextID uint32
}

func (w *auditWorld) Frame() int { return w.frame }
func (w *auditWorld) target() d2monster.Target {
	return d2monster.Target{ID: 1, X: 100 + w.dist, Y: 100, Size: 1, IsPlayer: true}
}

func (w *auditWorld) Nearest(*d2monster.Brain) (d2monster.Target, int, bool) {
	return w.target(), w.dist, true
}

func (w *auditWorld) AttackTarget(*d2monster.Brain) (d2monster.Target, int, bool) {
	return w.target(), w.dist, true
}
func (w *auditWorld) InRange(*d2monster.Brain, d2monster.Target, int) bool { return w.near }
func (w *auditWorld) DyingNear(*d2monster.Brain, int) bool                 { return false }
func (w *auditWorld) HasState(*d2monster.Brain, int) bool                  { return false }
func (w *auditWorld) SetSpeed(*d2monster.Brain, int)                       {}
func (w *auditWorld) Attack(*d2monster.Brain, d2monster.Mode, d2monster.Target) bool {
	return true
}

func (w *auditWorld) MoveTo(*d2monster.Brain, d2monster.Point, *d2monster.Target, int, bool) bool {
	return true
}

func (w *auditWorld) KillSelf(b *d2monster.Brain) { b.Mode = d2monster.ModeDead }

// the Director's real host answers
func (w *auditWorld) CountMinions(b *d2monster.Brain) int { return w.d.CountMinions(b) }
func (w *auditWorld) SpawnerCellsFree(b *d2monster.Brain) bool {
	return w.d.SpawnerCellsFree(b)
}
func (w *auditWorld) FBXSpawnCellsFree(b *d2monster.Brain) bool { return w.d.FBXSpawnCellsFree(b) }
func (w *auditWorld) SpawnCellsFree(b *d2monster.Brain) bool    { return w.d.SpawnCellsFree(b) }
func (w *auditWorld) FBXBroodClass(b *d2monster.Brain) int      { return w.d.FBXBroodClass(b) }
func (w *auditWorld) FBXCellFreeAt(b *d2monster.Brain, p d2monster.Point) bool {
	return w.d.FBXCellFreeAt(b, p)
}

func (w *auditWorld) FBXScan(b *d2monster.Brain, q d2monster.FBXScanQuery) d2monster.FBXScanResult {
	return w.d.FBXScan(b, q)
}

// Cast is landSummon without the map: the same plan and budget, units added to
// the Director.
func (w *auditWorld) Cast(b *d2monster.Brain, slot int, _ d2monster.Target) bool {
	rec := w.rec[slot]
	if rec == nil {
		return true
	}

	u := w.d.unitOf(b)

	plan, ok := PlanSummon(rec, u.m.Stat)
	if !ok {
		return true
	}

	w.casts[b.ID]++

	room := w.d.summonBudget(b.ID, plan)
	for i := 0; i < room; i++ {
		w.nextID++
		young := &d2monster.Brain{ID: w.nextID, X: b.X + plan.Cells[i][0], Y: b.Y + plan.Cells[i][1], Size: 1}
		m := &d2mapentity.Monster{}
		m.Stat = w.stats[plan.Class]
		w.d.units[w.nextID] = &unit{m: m, b: young, summoner: b.ID}
	}

	return true
}

// killOldest removes the caster's oldest live summon (a hero kill).
func (w *auditWorld) killOldest(casterID uint32) {
	var oldest uint32

	for id, u := range w.d.units {
		if u.summoner == casterID && (oldest == 0 || id < oldest) {
			oldest = id
		}
	}

	if oldest != 0 {
		delete(w.d.units, oldest)
	}
}

func TestSummonerAIsStayWithinTheirLimits(t *testing.T) {
	nest := &d2records.SkillRecord{Srvdofunc: 91}
	spawner := &d2records.SkillRecord{Srvdofunc: 135}
	whip := &d2records.SkillRecord{Srvdofunc: 131, Summon: "suicideminion1"}
	worm := &d2records.SkillRecord{Srvdofunc: 133, Summon: "painworm1"}
	hydra := &d2records.SkillRecord{Srvdofunc: 144, Summon: "hydra1", Pettype: "hydra", Param1: 250}
	prison := &d2records.SkillRecord{Srvdofunc: 104, Summon: "boneprison1", Pettype: "none"}

	all := func(r *d2records.SkillRecord) map[int]*d2records.SkillRecord {
		m := map[int]*d2records.SkillRecord{}
		for i := 0; i < d2monster.NumSkills; i++ {
			m[i] = r
		}

		return m
	}

	type row struct {
		name     string
		ai       string // monstats AI of the caster
		aip      []int  // aip1..
		skills   map[int]*d2records.SkillRecord
		spawn    string // caster monstats spawn column
		class    string // class the cast creates (Nest/Spawner: spawn)
		maxLive  int    // the AI's own live limit, or the host cap for AIs without one
		maxCasts int    // the AI's own cast limit (-1: none, the host cap alone bounds it)
		mustCast bool   // the row must actually summon (guards against a dead test)
	}

	rows := []row{
		{"FoulCrowNest aip3 6", "FoulCrowNest", []int{10, 0, 6}, map[int]*d2records.SkillRecord{0: nest}, "foulcrow1", "foulcrow1", 6, 6, true},
		{"MosquitoNest aip1 4", "MosquitoNest", []int{4, 30, 10}, map[int]*d2records.SkillRecord{0: nest}, "mosquito1", "mosquito1", 5, 5, true},
		{"MinionSpawner aip1 9 aip2 4", "MinionSpawner", []int{9, 4, 20, 30}, map[int]*d2records.SkillRecord{0: spawner}, "minion1", "minion1", 4, 9, true},
		{"Sarcophagus aip3 5", "Sarcophagus", []int{10, 0, 5}, map[int]*d2records.SkillRecord{0: nest}, "mummy1", "mummy1", 6, 6, true},
		{"VileMother aip1 8 aip2 3", "VileMother", []int{8, 3, 100, 100, 0, 0, 0, 8}, map[int]*d2records.SkillRecord{0: nest},
			"vilechild2", "vilechild2", 3, 8, true},
		// EvilHole lays its minions itself (SpawnHoleMinion, counter aip1) and casts nothing
		{"EvilHole (own counter aip1, no cast)", "EvilHole", []int{10, 50}, map[int]*d2records.SkillRecord{0: nest}, "fallen1", "fallen1",
			nestSummonCap, -1, false},
		// no AI limit of their own: the host cap holds them
		{"GenericSpawner", "GenericSpawner", []int{80, 0, 15}, map[int]*d2records.SkillRecord{0: nest}, "imp1", "imp1", nestSummonCap, -1, false},
		{"HighPriest Hydra (Skill1 every 100 frames)", "HighPriest", []int{75, 0, 0, 100, 0, 0, 0, 30}, map[int]*d2records.SkillRecord{0: hydra},
			"", "hydra1", hydraSummonCap, -1, true},
		{"Nihlathak Overseer Whip in slot 5", "Nihlathak", []int{30, 20, 80, 100, 5}, map[int]*d2records.SkillRecord{1: whip, 4: whip},
			"", "suicideminion1", whipSummonCap, -1, false},
		{"Overseer Whip", "Overseer", []int{250, 50, 50, 10, 4, 0, 0, 0}, map[int]*d2records.SkillRecord{0: whip, 1: whip, 2: whip},
			"", "suicideminion1", whipSummonCap, -1, false},
		{"PutridDefiler Impregnate", "PutridDefiler", []int{15, 5}, map[int]*d2records.SkillRecord{0: worm}, "", "painworm1",
			wormSummonCap, -1, false},
		{"Diablo with a DiabPrison in every slot", "Diablo", []int{0, 0, 0}, all(prison), "", "boneprison1", prisonSummonCap, -1, false},
		{"BloodRaven with a Nest in every slot", "BloodRaven", []int{0, 0, 0}, all(nest), "zombie2", "zombie2", nestSummonCap, -1, false},
		{"Council Member in every slot", "HighPriest", []int{75, 0, 0, 100, 0, 0, 0, 30}, all(hydra), "", "hydra1", hydraSummonCap, -1, true},
	}

	const (
		frames       = 6000
		actionFrames = 12 // a cast animation at 25 frames a second is about half a second
	)

	for _, r := range rows {
		for _, near := range []bool{false, true} {
			for _, killEvery := range []int{0, 40} {
				t.Run(fmt.Sprintf("%s/near=%v/kill=%d", r.name, near, killEvery), func(t *testing.T) {
					def, ok := d2monster.Lookup(r.ai)
					if !ok {
						t.Skipf("AI %s is not registered", r.ai)
					}

					rm, err := d2records.NewRecordManager(d2util.LogLevelNone)
					if err != nil {
						t.Fatal(err)
					}

					rm.Monster.Stats = d2records.MonStats{}
					stats := map[string]*d2records.MonStatRecord{}
					mk := func(key string, id int) *d2records.MonStatRecord {
						st := &d2records.MonStatRecord{Key: key, ID: id}
						rm.Monster.Stats[key] = st
						stats[key] = st

						return st
					}

					young := mk(r.class, 500)
					casterStat := mk("caster1", 501)
					casterStat.AiKey, casterStat.SpawnKey = r.ai, r.spawn

					d := &Director{asset: &d2asset.AssetManager{Records: rm}, units: map[uint32]*unit{},
						byEntity: map[string]*unit{}, statByID: map[int]*d2records.MonStatRecord{500: young, 501: casterStat}}

					prof := &d2monster.Profile{AI: r.ai, ID: r.ai}
					copy(prof.AIP[1:], r.aip)

					for slot := range r.skills {
						prof.Skills[slot] = d2monster.SkillSlot{Name: "sk", Mode: d2monster.ModeSkill1}
					}

					b := d2monster.NewBrain(7, 501, d2monster.Hell, prof, 0xC0FFEE)
					b.X, b.Y, b.Def, b.Aggressive = 100, 100, def, true

					m := &d2mapentity.Monster{}
					m.Stat = casterStat
					d.units[b.ID] = &unit{m: m, b: b}

					w := &auditWorld{d: d, dist: 10, near: near, rec: r.skills, stats: stats, casts: map[uint32]int{}, nextID: 1000}
					if near {
						w.dist = 2
					}

					maxLive := 0

					for f := 0; f < frames; f++ {
						w.frame = f

						if killEvery > 0 && f%killEvery == 0 {
							w.killOldest(b.ID)
						}

						d2monster.Tick(w, b)

						// a cast or attack leaves the AI waiting for the engine to wake it when the
						// animation ends: the Director does that after the action's frames
						if b.Mode.IsAlive() && b.Wake > frames+1000 {
							b.WakeNow(f + actionFrames)
						}

						if live := d.liveSummons(b.ID, r.class); live > maxLive {
							maxLive = live
						}
					}

					if maxLive > r.maxLive {
						t.Errorf("%d live summons at once, the limit is %d", maxLive, r.maxLive)
					}

					if r.maxCasts >= 0 && w.casts[b.ID] > r.maxCasts {
						t.Errorf("%d casts, the AI's own limit is %d", w.casts[b.ID], r.maxCasts)
					}

					if r.mustCast && !near && w.casts[b.ID] == 0 {
						t.Error("never summoned: the row tests nothing")
					}
				})
			}
		}
	}
}

// TestDirectorBroodCounters checks the Director's real answers to the
// VileMother: the class she lays, the living young within radius 25 (dead,
// foreign classes and far ones do not count), and free cells.
func TestDirectorBroodCounters(t *testing.T) {
	rm, err := d2records.NewRecordManager(d2util.LogLevelNone)
	if err != nil {
		t.Fatal(err)
	}

	rm.Monster.Stats = d2records.MonStats{}
	child := &d2records.MonStatRecord{Key: "vilechild2", ID: 7}
	other := &d2records.MonStatRecord{Key: "zombie1", ID: 8}
	mother := &d2records.MonStatRecord{Key: "vilemother2", ID: 9, SpawnKey: "vilechild2", AiKey: "VileMother"}
	rm.Monster.Stats["vilechild2"], rm.Monster.Stats["zombie1"], rm.Monster.Stats["vilemother2"] = child, other, mother

	d := &Director{asset: &d2asset.AssetManager{Records: rm}, units: map[uint32]*unit{}, byEntity: map[string]*unit{}}

	add := func(id uint32, st *d2records.MonStatRecord, x, y int) *unit {
		m := &d2mapentity.Monster{}
		m.Stat = st
		u := &unit{m: m, b: &d2monster.Brain{ID: id, X: x, Y: y, Size: 1}}
		d.units[id] = u

		return u
	}

	mu := add(1, mother, 100, 100)
	add(2, child, 105, 100) // near
	add(3, child, 100, 124) // 24 away: counts
	add(4, child, 100, 126) // 26 away: does not
	add(5, other, 101, 100) // other class
	add(6, child, 99, 100)  // near
	mu.summoner = 0

	if c := d.FBXBroodClass(mu.b); c != 7 {
		t.Errorf("brood class %d, want 7", c)
	}

	if c := d.FBXBroodClass(add(10, other, 0, 0).b); c != -1 {
		t.Errorf("a monster without a spawn column lays class %d, want -1", c)
	}

	got := d.FBXScan(mu.b, d2monster.FBXScanQuery{Kind: d2monster.FBXScanLinkedClass, Class: 7, Radius2: 25 * 25})
	if got.Count != 3 {
		t.Errorf("young alive = %d, want 3", got.Count)
	}

	if r := d.FBXScan(mu.b, d2monster.FBXScanQuery{Kind: d2monster.FBXScanOverseer, Radius2: 50}); r.Found || r.Count != 0 {
		t.Errorf("unanswered scan kinds must find nothing: %+v", r)
	}

	if !d.FBXCellFreeAt(mu.b, d2monster.Point{X: 1, Y: 1}) || !d.FBXSpawnCellsFree(mu.b) || !d.SpawnCellsFree(mu.b) {
		t.Error("without a collision grid every cell is free")
	}
}
