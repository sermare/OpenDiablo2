package d2mp

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
)

// LevelSeed derives the seed of one level from the game seed, so every party
// of a game generates the same level (OUR extension; d2realm.LevelSeed is the
// same function).
func LevelSeed(gameSeed uint32, level uint16) uint32 {
	h := gameSeed ^ (uint32(level)+1)*0x9E3779B1
	h ^= h >> 16
	h *= 0x85EBCA6B
	h ^= h >> 13
	h *= 0xC2B2AE35
	h ^= h >> 16

	return h
}

// FirstSimID is the first unit id the simulation allocates for monsters,
// objects, missiles and items. Player ids come from the caller and must be
// below it.
const FirstSimID = 0x01000000

// Config configures a Sim.
type Config struct {
	Seed  uint32 // game seed
	Rules Rules  // nil = DefaultRules
	// TickMs is the simulation step (default 40 = 25 steps per second, the
	// game's frame rate).
	TickMs uint32
	// TradeLock is how long accepting is refused after an offer changed.
	TradeLock uint32
	// WalkSpeed and RunSpeed in tiles per second (0 = package defaults).
	WalkSpeed, RunSpeed float64
	// MaxLevel is the experience cap used by the party split (default 99).
	MaxLevel int
}

// JoinInfo describes a hero entering the game.
type JoinInfo struct {
	ID    uint32
	Name  string
	Class uint8
	Level uint8
	// Act (1..5) picks the town the hero starts in; StartLevel overrides it.
	Act        int
	StartLevel uint16
	Gold       uint32
	Items      []string // item codes in the inventory
	HP         int32    // 0 = from Rules.Hero
}

type goalKind uint8

const (
	goalNone goalKind = iota
	goalAttack
	goalInteract
	goalPickup
)

type invItem struct {
	id   uint32
	code string
}

type pstate struct {
	u                  *Unit
	class, clvl        uint8
	left, right        uint16
	goal               goalKind
	target             uint32
	nextAtk, nextCast  uint32
	nextPath           uint32
	inv                []invItem
	wps                map[uint16]bool
	meleeMin, meleeMax int
	tpPortals          [2]uint32
	moves              levelMoves
	protectUntil       uint32 // server ms until which state 0x6c holds
}

type mstate struct {
	u        *Unit
	def      MonsterDef
	target   uint32
	nextAtk  uint32
	nextPath uint32
}

type misstate struct {
	u     *Unit
	def   SkillDef
	owner uint32
	last  uint32
}

type timer struct {
	at uint32
	fn func()
}

// Sim is the authoritative game simulation. It is not safe for concurrent use;
// the caller (the realm) serialises access.
type Sim struct {
	cfg   Config
	rules Rules
	rng   *rand.Rand
	now   uint32

	units  map[uint32]*Unit
	pl     map[uint32]*pstate
	mons   map[uint32]*mstate
	mis    map[uint32]*misstate
	levels map[uint16]*LevelDef
	nextID uint32
	roster *d2party.Roster
	queues map[uint32][]Event
	paths  map[uint32]*pathState
	timers []timer
	trades map[uint32]*trade

	// Stats are counters for tests and logs.
	Stats struct{ Kills, PlayerDeaths, Casts, Hits, Drops int }
}

// NewSim returns a simulation at server time 0 with no players.
func NewSim(cfg Config) *Sim {
	if cfg.Rules == nil {
		cfg.Rules = DefaultRules{}
	}

	if cfg.TickMs == 0 {
		cfg.TickMs = 40
	}

	if cfg.TradeLock == 0 {
		cfg.TradeLock = 2000
	}

	if cfg.WalkSpeed == 0 {
		cfg.WalkSpeed = WalkSpeed
	}

	if cfg.RunSpeed == 0 {
		cfg.RunSpeed = RunSpeed
	}

	if cfg.MaxLevel == 0 {
		cfg.MaxLevel = 99
	}

	return &Sim{
		cfg: cfg, rules: cfg.Rules, rng: rand.New(rand.NewSource(int64(cfg.Seed) ^ 0x5eed)),
		units: map[uint32]*Unit{}, pl: map[uint32]*pstate{}, mons: map[uint32]*mstate{},
		mis: map[uint32]*misstate{}, levels: map[uint16]*LevelDef{}, nextID: FirstSimID,
		roster: d2party.New(), queues: map[uint32][]Event{}, paths: map[uint32]*pathState{}, trades: map[uint32]*trade{},
	}
}

// Now returns the server time in ms.
func (s *Sim) Now() uint32 { return s.now }

func sid(id uint32) string { return fmt.Sprint(id) }

// ---- event routing ----

func (s *Sim) send(to uint32, e Event) {
	if _, ok := s.pl[to]; ok {
		s.queues[to] = append(s.queues[to], e)
	}
}

func (s *Sim) sortedPlayers() []*pstate {
	out := make([]*pstate, 0, len(s.pl))
	for _, p := range s.pl {
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].u.ID < out[j].u.ID })

	return out
}

func (s *Sim) toLevel(level uint16, e Event, except uint32) {
	for _, p := range s.sortedPlayers() {
		if p.u.Level == level && p.u.ID != except {
			s.queues[p.u.ID] = append(s.queues[p.u.ID], e)
		}
	}
}

func (s *Sim) toAll(e Event) {
	for _, p := range s.sortedPlayers() {
		s.queues[p.u.ID] = append(s.queues[p.u.ID], e)
	}
}

// Take returns and clears the events queued for a player, led by a tick event
// carrying the server clock. It returns nil when nothing is queued.
func (s *Sim) Take(id uint32) []Event {
	q := s.queues[id]
	if len(q) == 0 {
		return nil
	}

	delete(s.queues, id)

	return append([]Event{{Type: EvTick, A: int32(s.now)}}, q...)
}

// ---- levels ----

func (s *Sim) levelDef(level uint16) *LevelDef {
	if d, ok := s.levels[level]; ok {
		return d
	}

	d := s.rules.Level(LevelSeed(s.cfg.Seed, level), level)
	s.levels[level] = d

	for _, m := range d.Monsters {
		def := s.rules.Monster(m.Type)
		u := &Unit{ID: s.newID(), Kind: KindMonster, Type: m.Type, Level: level, Name: def.Name,
			HP: def.HP, MaxHP: def.HP, Mon: uint16(def.Level)}
		x, y := Snap(m.X), Snap(m.Y)
		u.Segs = []Seg{{X0: x, Y0: y, X1: x, Y1: y}}
		s.units[u.ID] = u
		s.mons[u.ID] = &mstate{u: u, def: def}
	}

	for _, o := range d.Objects {
		u := &Unit{ID: s.newID(), Kind: KindObject, Type: o.Type, Level: level, Dest: o.Dest}
		x, y := Snap(o.X), Snap(o.Y)
		u.Segs = []Seg{{X0: x, Y0: y, X1: x, Y1: y}}
		s.units[u.ID] = u
	}

	return d
}

func (s *Sim) newID() uint32 {
	id := s.nextID
	s.nextID++

	return id
}

// unitsIn returns the units in a level (not inventory items), by id.
func (s *Sim) unitsIn(level uint16) []*Unit {
	var out []*Unit

	for _, u := range s.units {
		if u.Level == level {
			out = append(out, u)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

func (s *Sim) hasPlayers(level uint16) bool {
	for _, p := range s.pl {
		if p.u.Level == level {
			return true
		}
	}

	return false
}

func (s *Sim) spawnEvent(u *Unit) Event {
	c := *u
	if len(u.Segs) > 0 {
		c.Segs = []Seg{u.Segs[len(u.Segs)-1]}
	} else {
		c.Segs = []Seg{{}}
	}

	return Event{Type: EvSpawn, ID: u.ID, Unit: c}
}

func (s *Sim) pos(u *Unit) (float64, float64) { return u.PosAt(float64(s.now)) }

func dist(ax, ay, bx, by float64) float64 { return math.Hypot(ax-bx, ay-by) }

// moveUnit starts a straight walk towards (x, y), stopping at the first
// blocked cell, and announces it. speed 0 stands still.
func (s *Sim) moveUnit(u *Unit, x, y, speed float64) {
	lv := s.levelDef(u.Level)
	cx, cy := s.pos(u)
	x0, y0 := Snap(cx), Snap(cy)
	ex, ey := s.march(lv, x0, y0, x, y)
	seg := Seg{X0: x0, Y0: y0, X1: ex, Y1: ey, Speed: math.Round(speed*100) / 100, T0: s.now}

	if ex == x0 && ey == y0 {
		seg.Speed = 0
	}

	u.Segs = []Seg{seg}
	s.toLevel(u.Level, Event{Type: EvSeg, ID: u.ID, Seg: seg}, 0)
}

// march walks from (x0,y0) towards (x,y) in small steps and returns the last
// walkable wire-aligned point.
func (s *Sim) march(lv *LevelDef, x0, y0, x, y float64) (float64, float64) {
	return March(lv, x0, y0, x, y)
}

// March walks from (x0,y0) towards (x,y) in small steps and returns the last
// walkable wire-aligned point. The server and the predicting client use it, so
// both agree where a walk ends.
func March(lv *LevelDef, x0, y0, x, y float64) (float64, float64) {
	d := dist(x0, y0, x, y)
	if d < 0.01 {
		return x0, y0
	}

	steps := int(math.Ceil(d / 0.2))
	ex, ey := x0, y0

	for i := 1; i <= steps; i++ {
		f := float64(i) / float64(steps)
		px, py := Snap(x0+(x-x0)*f), Snap(y0+(y-y0)*f)

		if !lv.Walkable(px, py) {
			break
		}

		ex, ey = px, py
	}

	return ex, ey
}

func (s *Sim) stopAt(u *Unit) {
	delete(s.paths, u.ID)
	cx, cy := s.pos(u)
	s.moveUnit(u, cx, cy, 0)
}

// ---- joining and leaving ----

// Join enters a hero into the game, in the town of its act.
func (s *Sim) Join(in JoinInfo) {
	if _, ok := s.pl[in.ID]; ok {
		return
	}

	level := in.StartLevel
	if level == 0 {
		act := in.Act
		if act < 1 || act > 5 {
			act = 1
		}

		level = uint16(d2level.ActStartLevel(act))
	}

	hp, mn, mx := s.rules.Hero(in.Class, in.Level)
	if in.HP > 0 {
		hp = in.HP
	}

	u := &Unit{ID: in.ID, Kind: KindPlayer, Type: uint16(in.Class), Name: in.Name, HP: hp, MaxHP: hp, Gold: in.Gold}
	p := &pstate{u: u, class: in.Class, clvl: in.Level, meleeMin: mn, meleeMax: mx, wps: map[uint16]bool{level: true}}

	s.pl[in.ID] = p
	s.units[in.ID] = u
	s.roster.Add(d2party.Member{ID: sid(in.ID), Name: in.Name, Class: d2enum.Hero(in.Class), Level: int(in.Level), Area: int(level)})

	for _, code := range in.Items {
		it := &Unit{ID: s.newID(), Kind: KindItem, Name: code, Owner: in.ID}
		s.units[it.ID] = it
		p.inv = append(p.inv, invItem{it.ID, code})
	}

	s.enterLevel(p, level, 0, 0, true)
	s.sendInv(p)
}

// Leave removes a hero from the game.
func (s *Sim) Leave(id uint32) {
	p, ok := s.pl[id]
	if !ok {
		return
	}

	s.cancelTrade(id, "partner left")
	s.toLevel(p.u.Level, Event{Type: EvRemove, ID: id}, id)
	s.removePortals(p)

	for _, it := range p.inv {
		delete(s.units, it.id)
	}

	s.roster.Remove(sid(id))
	s.syncParties()
	delete(s.pl, id)
	delete(s.units, id)
	delete(s.queues, id)

	for _, m := range s.mons {
		if m.target == id {
			m.target = 0
		}
	}
}

// enterLevel puts a hero into a level (at the spawn point when x,y are 0) and
// tells everybody involved.
func (s *Sim) enterLevel(p *pstate, level uint16, x, y float64, first bool) {
	u := p.u
	lv := s.levelDef(level)

	if !first {
		s.toLevel(u.Level, Event{Type: EvRemove, ID: u.ID}, u.ID)
	}

	if x == 0 && y == 0 {
		x, y = lv.SpawnX, lv.SpawnY
	}

	x, y = Snap(x), Snap(y)
	if !lv.Walkable(x, y) {
		x, y = lv.SpawnX, lv.SpawnY
	}

	u.Level = level
	u.Segs = []Seg{{X0: x, Y0: y, X1: x, Y1: y, T0: s.now}}
	p.goal, p.target = goalNone, 0
	s.roster.SetArea(sid(u.ID), int(level))

	s.send(u.ID, Event{Type: EvLevel, ID: u.ID, Level: level})
	s.send(u.ID, s.spawnEvent(u))

	for _, o := range s.unitsIn(level) {
		if o.ID != u.ID {
			s.send(u.ID, s.spawnEvent(o))
		}
	}

	s.toLevel(level, s.spawnEvent(u), u.ID)
	s.announceHero(p, level, first)
}

// announceHero keeps the roster global while the world is per level: viewers
// outside the hero's level learn who it is and where (EvHero; viewers inside
// get the EvSpawn), and a joining hero learns the heroes of the other levels.
// With every hero in one level nothing is sent.
func (s *Sim) announceHero(p *pstate, level uint16, first bool) {
	u := p.u
	info := s.heroEvent(u)

	for _, o := range s.sortedPlayers() {
		if o.u.ID == u.ID {
			continue
		}

		if o.u.Level != level {
			s.send(o.u.ID, info)
		}

		if first && o.u.Level != level {
			s.send(u.ID, s.heroEvent(o.u))
		}
	}
}

func (s *Sim) heroEvent(u *Unit) Event {
	c := *u
	c.Segs = nil

	return Event{Type: EvHero, ID: u.ID, Unit: c}
}

// maxLevelID is the highest level id of the game's levels.txt (136, verified in d2level).
const maxLevelID = 136

// ChangeLevel hands a hero over to another level (the client walked through a
// stair or door of its own map): it leaves the units of the old level, appears
// at the level's spawn and receives the units of the new one. Returns false
// for an unknown hero or level id. Moving to the level the hero is in is a
// no-op.
func (s *Sim) ChangeLevel(id uint32, level uint16) bool {
	p, ok := s.pl[id]
	if !ok || level == 0 || level > maxLevelID {
		return false
	}

	if p.u.Level == level {
		return true
	}

	s.cancelTrade(id, "left the level")
	delete(s.paths, id)
	s.enterLevel(p, level, 0, 0, false)
	s.protectAfterMove(p)

	return true
}

// ---- digest ----

// DigestUnits hashes the logical state of units (identity, type, life, state,
// final position), independent of in-flight movement and of missiles.
func DigestUnits(us []*Unit) uint64 {
	cp := make([]*Unit, 0, len(us))

	for _, u := range us {
		if u.Kind != KindMissile {
			cp = append(cp, u)
		}
	}

	sort.Slice(cp, func(i, j int) bool { return cp[i].ID < cp[j].ID })

	h := fnv.New64a()

	for _, u := range cp {
		x, y := u.Final()
		fmt.Fprintf(h, "%d|%d|%d|%d|%s|%d|%d|%d|%t|%d|%d|%d|%d,%d;", u.ID, u.Kind, u.Type, u.Level, u.Name, u.Owner,
			u.HP, u.MaxHP, u.Dead, u.State, u.Dest, u.Party, Sub(x), Sub(y))
	}

	return h.Sum64()
}

// Digest hashes the units of one level.
func (s *Sim) Digest(level uint16) uint64 { return DigestUnits(s.unitsIn(level)) }

// Unit returns a unit (for tests and the realm).
func (s *Sim) Unit(id uint32) *Unit { return s.units[id] }

// UnitsIn returns the units of a level.
func (s *Sim) UnitsIn(level uint16) []*Unit { return s.unitsIn(level) }

// Level returns the level of a player (0 when unknown).
func (s *Sim) Level(id uint32) uint16 {
	if p, ok := s.pl[id]; ok {
		return p.u.Level
	}

	return 0
}

// Inventory returns the item references ("id:code") and gold of a player.
func (s *Sim) Inventory(id uint32) ([]string, uint32) {
	p, ok := s.pl[id]
	if !ok {
		return nil, 0
	}

	return invRefs(p.inv), p.u.Gold
}

func invRefs(inv []invItem) []string {
	out := make([]string, len(inv))
	for i, it := range inv {
		out[i] = fmt.Sprintf("%d:%s", it.id, it.code)
	}

	return out
}

func (s *Sim) sendInv(p *pstate) {
	s.send(p.u.ID, Event{Type: EvInv, ID: p.u.ID, A: int32(p.u.Gold), Items: invRefs(p.inv)})
}

func (s *Sim) msg(to uint32, format string, args ...interface{}) {
	s.send(to, Event{Type: EvMsg, ID: to, Text: fmt.Sprintf(format, args...)})
}
