package d2missile

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Spec holds the missiles.txt columns the simulation reads.
type Spec struct {
	ID   int
	Name string

	SrvDoFunc, SrvHitFunc, SrvDmgFunc int

	Vel, VelLev, MaxVel, Accel int
	Range, LevRange            int
	Activate                   int

	// CollideType: 0 none, 1 walls+players, 2 walls+monsters, 3 both (default),
	// 5 like 2, 6 walls only, 8 like 3 and also non walkable cells.
	CollideType   int
	CollideKill   bool
	CollideFriend bool
	LastCollide   bool
	Collision     bool // stamps the missile bit into the grid (not simulated)
	Pierce        bool
	Explosion     bool // client side explosion: no default damage
	AlwaysExplode bool
	ToHit         bool // rolls to hit against defense
	CanSlow       bool
	NextHit       bool
	NextDelay     int
	Size          int

	SubLoop            bool
	SubStart, SubStop  int
	ExplosionMissile   string
	SubMissile         [3]string
	HitSubMissile      [4]string
	SkillName          string // missiles.txt Skill: damage comes from that skill
	SrvCalc1, DmgCalc1 *d2calc.Program
	Param              [5]int // Param1..5
	SHitPar            [3]int // sHitPar1..3
	DParam             [2]int // dParam1..2
	HitClass           int
	// SrcDam is the missiles.txt SrcDamage column (byte +0x12d of the record):
	// -1 (0xff) turns the skill's SrcDam off for this missile, which also
	// turns off its critical strike roll (0x64cbde, verified).
	SrcDam      int
	ResultFlags int
	HitFlags    int
}

// Table resolves missiles by id or name.
type Table interface {
	ByID(id int) *Spec
	ByName(name string) *Spec
}

// Owner identifies who fired a missile and carries its roll inputs.
type Owner struct {
	ID           string
	IsPlayer     bool
	Level        int
	AttackRating int
	Roller       d2combat.Roller
	// Gone, when set, reports that the owner is dead or gone; SrvDoFunc 7
	// (Guided Arrow) destroys its missile then (0x5ac2c0, verified).
	Gone func() bool
}

// Target is a unit a missile can hit.
type Target interface {
	ID() string
	IsPlayer() bool
	Alive() bool
	Level() int
	// Defense returns total defense including the vs-missile bonus when
	// missile is true.
	Defense(missile bool) int
}

// World is the environment a Sim runs in.
type World interface {
	d2path.Grid
	// Targets returns the living units whose footprint covers the subtile.
	Targets(x, y int) []Target
	// IsEnemy reports whether the owner may damage the target.
	IsEnemy(o Owner, t Target) bool
	// Frame is the current game frame (25 Hz).
	Frame() int
}

// Finder is optionally implemented by a World to let hit function 10 (Guided
// Arrow, 0x5a8100 -> 0x5a8060) look for a new target where the arrow ran out.
type Finder interface {
	// EnemiesWithin lists the living enemies of the owner that the exe's scan
	// (0x569510 with filter 0x569100) offers: players and monsters, not in a
	// town, targetable, in line of sight of the owner, whose subtile position
	// is within radius subtiles (euclidean) of (x, y). The sim keeps the one
	// with the lowest Serial (unit id), verified 0x569a40.
	EnemiesWithin(o Owner, x, y float64, radius int) []Target
}

// EventKind classifies a simulation event.
type EventKind string

// Event kinds reported through Sim.OnEvent.
const (
	EventCreate  EventKind = "create"
	EventMove    EventKind = "move"
	EventHit     EventKind = "hit"     // a target was struck (Damage rolled)
	EventMiss    EventKind = "miss"    // the to-hit roll failed
	EventWall    EventKind = "wall"    // destroyed by a wall
	EventExpire  EventKind = "expire"  // lifetime ran out
	EventExplode EventKind = "explode" // client side explosion missile
	EventPierce  EventKind = "pierce"  // passed through a target
	// EventVanish: destroyed without running the hit function (the exe's
	// "return 2" paths: entering a wall bit that is not in the CollideType
	// block mask, owner gone for SrvDoFunc 7).
	EventVanish EventKind = "vanish"
	// EventArea: an area damage hit function (1, 14) fired at the missile;
	// Damage is rolled once and applies to every enemy within Radius subtiles
	// (squared distance <= Radius^2, verified 0x569510).
	EventArea EventKind = "area"
)

// Event is one thing that happened to a missile.
type Event struct {
	Kind    EventKind
	Missile *Missile
	Target  Target
	// Damage is the rolled damage (EventHit), before resists, 8.8 fixed point.
	Damage d2combat.Damage
	Chance int // to-hit chance (EventHit / EventMiss), 0 when no roll
	Roll   int
	Name   string // explosion missile name (EventExplode)
	Radius int    // EventArea: radius in subtiles
}
