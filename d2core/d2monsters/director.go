// Package d2monsters runs hostile monsters inside the engine: it builds
// d2monster brains from monstats, feeds them the map and the players through
// the d2monster.World interface, resolves attacks with d2combat, applies
// hero damage, and rolls loot with d2drop on death.
//
// The AI is the pure d2common/d2monster package; this package is the glue and
// therefore the place where engine-level simplifications are listed:
//
//   - Units occupy one collision cell each (monster flag 0x100, player 0x80,
//     corpse 0x8000) and block each other through d2path.MaskUnits. Which mask
//     the original uses for unit blocking is not recorded (UNVERIFIED).
//   - Monster attacks resolve at the animation's halfway frame. Melee connects
//     within the reach of the mode (7 subtiles); ranged attacks (a missile
//     column for the mode) launch a Shot through the Launcher hook: the
//     built-in launcher flies a straight bolt that is stopped by walls and by
//     the first hero cell it enters. missiles.txt behaviour beyond velocity is
//     not simulated; feat/skill-pipeline can replace the launcher.
//   - Not ported (notes: not read / unverified): forced AI states (flee, fear,
//     confuse, charm: the state table at 0x73a548), the Summoner, Vulture and
//     the other bosses, monster-vs-monster targeting, aidel throttling after
//     hit recovery, champion/unique modifiers (monumod).
//   - Hero defense is dexterity/4 only (equipment defense is not read).
//   - Monster level is the monstats Level column (not the area's MonLvl).
//   - Monster stats use monlvl.txt L-* columns scaled by the monstats ratios.
package d2monsters

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	logPrefix = "Monsters"

	frameSeconds   = 1.0 / 25.0 // the game runs at 25 Hz
	maxCatchUp     = 0.25       // seconds of simulation per rendered frame, at most
	corpseSeconds  = 12.0       // corpses are removed after this long (engine choice, UNVERIFIED: the original's timing is not in the notes)
	blockedRetry   = 8          // refused steps in a row before a blocked monster re-thinks
	adoptInterval  = 0.5        // seconds between scans for DS1 monster placements
	replanDistance = 5          // the target moved this far since planning: re-plan (VERIFIED, notes)
	meleeInRange   = 7          // edge distance at which melee attacks connect (reach, VERIFIED value 7)
	rangedInRange  = 20         // ranged attack distance (engine choice, UNVERIFIED)
	heroReach      = 7          // hero melee reach, subtiles (engine choice)
)

// Options configure a Director.
type Options struct {
	// Seed is the game seed per-monster seeds are derived from.
	Seed uint32
	// Difficulty selects the monstats columns.
	Difficulty d2monster.Difficulty
	// LogStats logs a DIFFTEST line with the level, life, defense, attack and
	// resistances of every spawned monster.
	LogStats bool
	// Classic selects the plain monlvl.txt columns (HP, AC...) instead of the
	// LoD columns (L-HP...) that the expansion uses (UNVERIFIED which the
	// original takes; the L- columns are the default).
	Classic bool
	// Expansion selects the MonLvl*Ex columns of levels.txt for the area
	// monster level.
	Expansion bool
	// IgnoreTown lets monsters target heroes standing in town (for tests; the
	// original never aggroes onto players in town).
	IgnoreTown bool
	// OnSound, if set, receives the monsters' MonSounds.txt sounds (attack,
	// weapon, skill, hit, death, taunt, neutral, footstep) with their position.
	OnSound func(SoundEvent)
	// OnHeroHit, if set, is called when a monster's attack hit a hero (after the
	// damage was applied): the hero's armor may lose durability.
	OnHeroHit func(p *d2mapentity.Player)
	// OnHeroStrike, if set, is called when a hero's swing hit a monster: the weapon
	// may lose durability.
	OnHeroStrike func(p *d2mapentity.Player)
}

// Counters tally what happened, for autotest summaries.
type Counters struct {
	Spawned, Aggro, Attacks, AttackHits, HeroSwings, HeroHits, Deaths, Drops, HeroDeaths int

	// mercenaries
	MercSpawns, MercAttacks, MercHits, MercSkills, MercDeaths, MercRevives, MercTeleports, MercLevelUps int
	// Shots is projectiles launched, ShotHits those that reached a hero.
	Shots, ShotHits int
	// Packs is natural groups spawned; BlockedSteps counts steps refused
	// because another unit stood in the way; HitRecoveries counts monsters
	// interrupted by damage; MaxStack is the most live monsters ever seen in
	// one subtile (1 when nobody stacks).
	Packs, BlockedSteps, HitRecoveries, MaxStack int
	// Minions summoned; MinionAttacks swings and MinionHits that connected.
	Minions, MinionAttacks, MinionHits int
	// UnitFights counts monster-against-monster attacks (converted, confused or
	// attracted monsters).
	UnitFights int
}

// unit is a monster plus its engine-side state.
type unit struct {
	m  *d2mapentity.Monster
	b  *d2monster.Brain
	mv *moveIntent

	merc         *mercUnit // non-nil for a hired mercenary
	hadTarget    bool
	nextIdle     int // frame of the next idle vocal, 0 = not scheduled
	nextStep     int // frame of the next footstep, 0 = not walking
	attackTarget uint32
	aimX, aimY   int // ground point of the last attack request (Target ID 0)
	blocked      int // consecutive refused steps
	removeAt     float64

	// ally marks a summoned minion (see minions.go); stunUntil and fleeUntil
	// are frames until which a stun/freeze or fear holds the monster.
	ally                *allyState
	stunUntil           int
	fleeUntil           int
	slowPct             int
	lastFlee, lastThink int
	lastLabel           string // the AI state last traced (forced.go)
}

type moveIntent struct {
	target     *d2monster.Target
	reach      int
	run        bool
	plannedAtX int
	plannedAtY int
}

// Director owns every hostile monster of a map.
type Director struct {
	*d2util.Logger

	asset   *d2asset.AssetManager
	engine  *d2mapengine.MapEngine
	players func() []*d2mapentity.Player
	opt     Options

	frame    int
	acc      float64
	clock    float64
	adoptAcc float64
	nextID   uint32

	units    map[uint32]*unit // by brain id
	byEntity map[string]*unit
	seenNPC  map[string]bool
	statByID map[int]*d2records.MonStatRecord
	targets  map[uint32]*d2mapentity.Player
	grid     mapGrid // static map flags (line of sight)
	fp       *footprints
	fpPlayer map[uint32]bool
	launcher Launcher
	hero     *d2rand.Seed
	hire     *d2hireling.Table
	mercs    map[*d2mapentity.Player]*unit
	killer   *unit // the merc whose hit is being resolved (kill credit)
	snd      *rand.Rand
	packRNG  *d2rand.Seed

	// ExpBonusPct, when set, returns the percent of extra experience per kill
	// (the experience shrine).
	ExpBonusPct func() int

	// PartyXP, when set, is offered the experience of every kill by a hero
	// (the amount after the shrine bonus). It returns true when the amount is
	// handled elsewhere (a network party: the server splits it among the
	// members and each gets its share back as a packet); false leaves the whole
	// amount to the killer.
	PartyXP func(src *d2mapentity.Player, xp int, monster string) bool

	pvp map[string]*d2rand.Seed // hero id -> its roller for swings at other heroes

	areaLevel int // levels.txt MonLvl of the current area (0 = unknown)
	// forceLevel, when set, replaces the resolved monster level of a spawn.
	forceLevel int

	// HeroDefense, if set, is called when a monster attack hits a hero. It
	// returns the damage that gets through (0 when avoided or absorbed) and
	// a note for the log (skills: dodge, avoid, Energy Shield, Bone Armor,
	// Thorns...). melee is false for projectiles.
	HeroDefense func(p *d2mapentity.Player, attacker *d2mapentity.Monster, melee bool, dmg int) (int, string)

	// Counters are updated as events happen.
	Counters Counters
	// OnEvent, if set, receives every log line the director emits as a
	// structured event (kind is spawn, aggro, attack, hit, death, drop...).
	OnEvent func(kind, line string)
	// OnKill, if set, is called when a monster dies (the quest system listens).
	OnKill func(KillEvent)
}

// KillEvent describes a monster death for OnKill.
type KillEvent struct {
	Monster *d2mapentity.Monster
	Class   int    // monstats id
	Label   string // the monster's display name (a super unique's own name)
	ByHero  bool   // the hero (or a hero's pet) dealt the killing blow
}

// NewDirector creates a director for a map engine. players returns the
// heroes monsters may target.
func NewDirector(asset *d2asset.AssetManager, engine *d2mapengine.MapEngine,
	players func() []*d2mapentity.Player, l d2util.LogLevel, opt Options) *Director {
	d := &Director{
		Logger:   d2util.NewLogger(),
		asset:    asset,
		engine:   engine,
		players:  players,
		opt:      opt,
		units:    map[uint32]*unit{},
		byEntity: map[string]*unit{},
		seenNPC:  map[string]bool{},
		statByID: map[int]*d2records.MonStatRecord{},
		targets:  map[uint32]*d2mapentity.Player{},
		mercs:    map[*d2mapentity.Player]*unit{},
		grid:     mapGrid{engine},
		snd:      newSoundRand(opt.Seed),
		fpPlayer: map[uint32]bool{},
		packRNG:  d2rand.New(opt.Seed ^ 0x5041434b),
	}

	d.fp = newFootprints(d.grid)
	d.launcher = newBoltLauncher(d.grid)

	d.Logger.SetLevel(l)
	d.Logger.SetPrefix(logPrefix)

	for _, st := range asset.Records.Monster.Stats {
		d.statByID[st.ID] = st
	}

	return d
}

// emit logs a line and forwards it to OnEvent.
func (d *Director) emit(kind, format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	d.Info(line)

	if d.OnEvent != nil {
		d.OnEvent(kind, line)
	}
}

// Monsters returns the live (not yet removed) hostile monsters, corpses
// included; mercenaries and summoned minions are not part of it (see Merc, Minions).
func (d *Director) Monsters() []*d2mapentity.Monster {
	out := make([]*d2mapentity.Monster, 0, len(d.units))
	for _, u := range d.units {
		if !u.friendly() && !u.b.Allied { // converted monsters are the hero's friends
			out = append(out, u.m)
		}
	}

	return out
}

// FindStat resolves a monstats key ("skeleton1") or hcIdx ("0") or name
// ("Skeleton", case-insensitive match of the key).
func (d *Director) FindStat(ref string) *d2records.MonStatRecord {
	ref = strings.TrimSpace(ref)

	if st := d.asset.Records.Monster.Stats[ref]; st != nil {
		return st
	}

	if id, err := strconv.Atoi(ref); err == nil {
		return d.statByID[id]
	}

	for key, st := range d.asset.Records.Monster.Stats {
		if strings.EqualFold(key, ref) {
			return st
		}
	}

	return nil
}

// Spawn creates a monster of the given monstats record at a subtile position.
func (d *Director) Spawn(stat *d2records.MonStatRecord, subX, subY int) (*d2mapentity.Monster, error) {
	return d.spawn(stat, subX, subY, nil)
}

func (d *Director) spawn(stat *d2records.MonStatRecord, subX, subY int, ally *allyState) (*d2mapentity.Monster, error) {
	prof := profileFromRecord(stat, d.opt.Difficulty)
	d.nextID++

	b := d2monster.NewBrain(d.nextID, stat.ID, d.opt.Difficulty, prof, d.opt.Seed)
	b.X, b.Y = subX, subY

	m, err := d.engine.NewMonster(subX, subY, stat, 0, b)
	if err != nil {
		return nil, err
	}

	if m.StatEx.SizeX/2 > 1 {
		b.Size = m.StatEx.SizeX / 2
	}

	m.Vitals = d.computeVitals(stat, b)
	b.Wake = d.frame // think on the next frame

	m.Blocker = func(x, y int) bool { return d.fp.BlockedFor(b.ID, x, y) }
	d.fp.Move(b.ID, subX, subY, d2path.FlagMonster)

	u := &unit{m: m, b: b, ally: ally}
	d.units[b.ID] = u
	d.byEntity[m.ID()] = u
	d.engine.AddEntity(m)

	if ally != nil {
		return m, nil
	}

	d.Counters.Spawned++
	d.emit("spawn", "MONSTER spawn name=%s id=%s class=%d ai=%s level=%d hp=%d defense=%d pos=(%d,%d)%s",
		m.Label(), stat.Key, stat.ID, prof.AI, m.Vitals.Level, m.Vitals.HP, m.Vitals.Defense, subX, subY,
		implementedNote(b))

	if d.opt.LogStats {
		// compared with the real tables by the difficulty scenario (verify.d/9a)
		v := m.Vitals
		d.emit("diff", "DIFFTEST monster id=%s difficulty=%d level=%d hp=%d defense=%d xp=%d tc=%q "+
			"a1=%d/%d-%d a2=%d/%d-%d res=%v", stat.Key, int(d.opt.Difficulty), v.Level, v.MaxHP, v.Defense,
			v.Experience, v.TreasureClass, v.A1.ToHit, v.A1.Min, v.A1.Max, v.A2.ToHit, v.A2.Min, v.A2.Max,
			MonsterResists(stat, d.opt.Difficulty))
	}

	return m, nil
}

func implementedNote(b *d2monster.Brain) string {
	if b.Def != nil && !b.Def.Implemented {
		return " (AI not ported: stands still)"
	}

	return ""
}

// Group links members to a leader: the leader's Fallen rally and other group
// commands reach them (MONAI_AddMinionToLeader). Natural pack generation from
// monstats MinGrp/MaxGrp and minion1/2 is not implemented; callers form groups.
func (d *Director) Group(leader *d2mapentity.Monster, members ...*d2mapentity.Monster) {
	for _, m := range members {
		leader.Brain.AddMinion(m.Brain)
	}
}

// SpawnNear places a monster on a free subtile around a position.
func (d *Director) SpawnNear(stat *d2records.MonStatRecord, subX, subY, ring int) (*d2mapentity.Monster, error) {
	p, ok := d2path.NearestFree(d.grid, d2path.MaskMonster, d2path.Point{X: subX, Y: subY}, ring+8)
	if !ok {
		return nil, fmt.Errorf("no free cell near (%d,%d)", subX, subY)
	}

	return d.Spawn(stat, p.X, p.Y)
}

// Advance runs the simulation for elapsed seconds (called once per rendered
// frame, after MapEngine.Advance).
func (d *Director) Advance(elapsed float64) {
	d.clock += elapsed

	d.adoptAcc += elapsed
	if d.adoptAcc >= adoptInterval {
		d.adoptAcc = 0
		d.adoptPlacements()
	}

	d.acc += elapsed
	if d.acc > maxCatchUp {
		d.acc = maxCatchUp
	}

	for d.acc >= frameSeconds {
		d.acc -= frameSeconds
		d.step()
	}
}

// step is one 25 Hz game frame.
func (d *Director) step() {
	d.frame++

	d.indexPlayers()
	d.launcher.Step()

	for _, u := range d.sortedUnits() {
		if !d.engineHas(u) {
			d.forget(u)
			continue
		}

		d.sync(u)
		d.footprint(u)
		d.handleEvents(u)
		d.ambientSounds(u)

		if u.merc != nil {
			d.stepMerc(u)
		}

		if u.ally != nil {
			d.allyStep(u)

			continue
		}

		if u.m.Alive() {
			if d.held(u) {
				continue
			}

			d.followIntent(u)

			if d2monster.Tick(d, u.b) {
				d.noteAggro(u)
			}

			d.traceState(u)
		} else if u.merc == nil && u.m.CorpseAge() > corpseSeconds { // merc corpses stay for a revive
			d.engine.RemoveEntity(u.m)
			d.forget(u)
		}
	}
}

func (d *Director) engineHas(u *unit) bool {
	_, ok := d.engine.Entities()[u.m.ID()]

	return ok
}

func (d *Director) forget(u *unit) {
	d.fp.Remove(u.b.ID)
	delete(d.units, u.b.ID)
	delete(d.byEntity, u.m.ID())
}

// sortedUnits gives a stable iteration order (determinism of the RNG use).
func (d *Director) sortedUnits() []*unit {
	out := make([]*unit, 0, len(d.units))
	for id := uint32(1); id <= d.nextID; id++ {
		if u, ok := d.units[id]; ok {
			out = append(out, u)
		}
	}

	return out
}

// sync copies entity state into the brain.
func (d *Director) sync(u *unit) {
	u.b.X, u.b.Y = u.m.SubtilePos()
	u.b.Mode = u.m.Mode()

	if mx := u.m.Vitals.MaxHP; mx > 0 {
		u.b.HPPercent = u.m.Vitals.HP * 100 / mx
	}
}

func (d *Director) noteAggro(u *unit) {
	if u.b.HasTarget && !u.hadTarget {
		d.Counters.Aggro++

		name := "hero"
		if p := d.targets[u.b.TargetID]; p != nil {
			name = p.Name()
		} else if u.b.TargetID >= mercTargetBase {
			name = "unit"
		}

		d.emit("aggro", "MONSTER aggro name=%s id=%d target=%s", u.m.Label(), u.b.ID, name)
		d.playPlans(u, tauntPlans(d.soundRecord(u)))
	}

	u.hadTarget = u.b.HasTarget
}

// adoptPlacements converts hostile monster placements that the map stamps
// created as NPCs (DS1 monster objects) into AI-driven monsters.
func (d *Director) adoptPlacements() {
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

		pos := npc.GetPosition()
		x, y := int(pos.X()), int(pos.Y())

		d.engine.RemoveEntity(npc)

		if _, err := d.Spawn(stat, x, y); err != nil {
			d.Infof("could not adopt DS1 monster %s: %v", stat.Key, err)
			d.engine.AddEntity(npc)
		} else {
			d.Infof("adopted DS1 placement %s at (%d,%d)", stat.Key, x, y)
		}
	}
}

// IsHostile says whether a monstats row is an enemy the director should run.
func IsHostile(st *d2records.MonStatRecord) bool {
	return st.Enabled && !st.IsNpc && !st.IsInteractable && st.Alignment == 0 &&
		st.AiKey != "" && !strings.EqualFold(st.AiKey, "Idle") && !strings.EqualFold(st.AiKey, "None")
}

// moveTo is the shared path request used by the Actor.
func (d *Director) moveTo(u *unit, dest d2monster.Point, target *d2monster.Target, reach int, run bool) bool {
	sx, sy := u.m.SubtilePos()
	from := d2path.Point{X: sx, Y: sy}
	to := d2path.Point{X: dest.X, Y: dest.Y}

	if target != nil && d2monster.EdgeDistance(sx-target.X, sy-target.Y, u.b.Size) <= reach {
		return false
	}

	mask := d2path.MaskMonster
	if u.m.Stat.CanOpenDoors {
		mask = d2path.MaskMonsterOpensDoors
	}

	// units block (UNVERIFIED mask), except the mover's own cell and the cell
	// it walks to (the target stands there)
	route, ok := d2path.FindPath(d.fp.ignoring(from, to), mask|d2path.MaskUnits, from, to)
	if !ok || len(route.Nodes) == 0 {
		d.Debugf("no path for %s from %v to %v", u.m.Label(), from, to)
		return false
	}

	path := make([]d2vector.Position, len(route.Nodes))
	for i, n := range route.Nodes {
		path[i] = d2vector.NewPosition(float64(n.X), float64(n.Y))
	}

	if !u.m.MoveAlong(path, run) {
		return false
	}

	u.mv = &moveIntent{reach: reach, run: run, plannedAtX: dest.X, plannedAtY: dest.Y}
	if target != nil {
		t := *target
		u.mv.target = &t
	}

	return true
}

// followIntent stops a chasing monster once it is within reach of its target
// and re-plans when the target moved.
func (d *Director) followIntent(u *unit) {
	if u.mv == nil {
		return
	}

	if !u.m.Moving() || (u.m.Mode() != d2monster.ModeWalk && u.m.Mode() != d2monster.ModeRun) {
		u.mv = nil

		return
	}

	if u.mv.target == nil {
		return
	}

	tx, ty, ok := d.targetPos(u, u.mv.target.ID)
	if !ok {
		return
	}

	sx, sy := u.m.SubtilePos()

	if d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size) <= u.mv.reach {
		u.m.StopMoving()
		u.mv = nil

		return
	}

	if abs(tx-u.mv.plannedAtX) >= replanDistance || abs(ty-u.mv.plannedAtY) >= replanDistance {
		t := *u.mv.target
		t.X, t.Y = tx, ty

		if !d.moveTo(u, d2monster.Point{X: tx, Y: ty}, &t, u.mv.reach, u.mv.run) {
			u.m.StopMoving()
			u.mv = nil
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
