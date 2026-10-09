// Package d2missile simulates server-side missiles: creation from a
// missiles.txt row, the per-frame standard move (MISSILE_SrvDoStandardMove,
// 0x5abd00), the collision test along the path and the hit/expire processing
// (MISSILE_ProcessHitOrExpire, 0x5aba10), see missiles-pathing.md and
// skills-combat.md (e) in the notes.
//
// The package is pure: the map, the units and the damage sink are behind the
// small World interface, and the missile data are plain structs a caller
// builds from missiles.txt.
//
// Modelled (verified in the notes unless marked): lifetime in frames
// (Range + LevRange*level), velocity ((VelLev*lvl)/8 + Vel) << 8 with the
// 75/100 step scale, Accel/MaxVel (unit of Accel UNVERIFIED), the Activate
// collision delay, wall tests with the CollideType masks, unit tests along the
// cells traversed in a frame, CollideFriend, LastCollide, NextHit/NextDelay,
// Pierce charges, the ToHit roll, CollideKill, AlwaysExplode and the
// HitSubMissile spawning of hit functions 2 and 4.
//
// Not modelled: homing/SpecialSetup, the per-missile pSrvDoFunc specials
// (only the standard move and SubMissile spawning of do func 2), area hit
// functions (1, 24), periodic effects (20), the 64 direction quantisation of
// the path (the direction is exact here), the Explosion missile is client
// only (it has no pSrvDoFunc) and is reported as an event.
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
	ResultFlags        int
	HitFlags           int
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
}
