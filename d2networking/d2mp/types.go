package d2mp

import (
	"math"
)

// SubPerTile is the number of sub-tile steps per tile on the wire (the d2gs
// packets use sub-tiles, verified in d2gs.MoveToLocation).
const SubPerTile = 5

// Kind of a unit; the values follow the d2gs unit type numbers (0 player,
// 1 monster, 2 object, 3 missile, 4 item).
type Kind uint8

// Unit kinds.
const (
	KindPlayer  Kind = 0
	KindMonster Kind = 1
	KindObject  Kind = 2
	KindMissile Kind = 3
	KindItem    Kind = 4
)

func (k Kind) String() string {
	switch k {
	case KindPlayer:
		return "player"
	case KindMonster:
		return "monster"
	case KindObject:
		return "object"
	case KindMissile:
		return "missile"
	case KindItem:
		return "item"
	}

	return "unit"
}

// Object types of KindObject units (OUR numbering for the headless rules).
const (
	ObjChest    uint16 = 1
	ObjWaypoint uint16 = 2
	ObjPortal   uint16 = 3
)

// Object states.
const (
	ObjClosed uint8 = 0
	ObjOpen   uint8 = 1
)

// Seg is one straight movement: from (X0,Y0) at server time T0 to (X1,Y1) at
// Speed tiles per second. Positions are sub-tile aligned (they are what the
// wire carries).
type Seg struct {
	X0, Y0, X1, Y1 float64
	Speed          float64 // tiles per second; 0 = standing at X0,Y0
	T0             uint32  // server ms
}

// Len returns the length of the segment in tiles.
func (s Seg) Len() float64 { return math.Hypot(s.X1-s.X0, s.Y1-s.Y0) }

// Duration returns the time in ms the segment takes.
func (s Seg) Duration() float64 {
	if s.Speed <= 0 {
		return 0
	}

	return s.Len() / s.Speed * 1000
}

// PosAt returns the position on the segment at server time t (ms).
func (s Seg) PosAt(t float64) (x, y float64) {
	d := s.Duration()
	el := t - float64(s.T0)

	if d <= 0 || el <= 0 {
		return s.X0, s.Y0
	}

	if el >= d {
		return s.X1, s.Y1
	}

	f := el / d

	return s.X0 + (s.X1-s.X0)*f, s.Y0 + (s.Y1-s.Y0)*f
}

// Moving reports whether the segment is still being walked at t.
func (s Seg) Moving(t float64) bool { return t < float64(s.T0)+s.Duration() }

// Snap rounds a tile coordinate to the wire resolution.
func Snap(v float64) float64 { return math.Round(v*SubPerTile) / SubPerTile }

// Sub converts tiles to wire sub-tiles.
func Sub(v float64) uint16 {
	s := math.Round(v * SubPerTile)
	if s < 0 {
		s = 0
	}

	if s > math.MaxUint16 {
		s = math.MaxUint16
	}

	return uint16(s)
}

// FromSub converts wire sub-tiles to tiles.
func FromSub(s uint16) float64 { return float64(s) / SubPerTile }

// Unit is the state both the server and the replicas keep about one unit.
type Unit struct {
	ID    uint32
	Kind  Kind
	Type  uint16 // monster id, object type, missile/skill id
	Level uint16 // level id the unit is in
	Name  string // hero or monster name; item code
	Owner uint32 // missile: caster
	HP    int32
	MaxHP int32
	Dead  bool
	State uint8  // objects: ObjClosed/ObjOpen
	Dest  uint16 // portals: destination level
	Party uint16 // players: party id (0 none)
	XP    uint32 // players: experience gained in this game
	Gold  uint32 // players
	Mon   uint16 // monsters: level (for experience)

	Segs []Seg // newest last; the server keeps one, replicas keep a few
}

func (u *Unit) seg(t float64) Seg {
	for i := len(u.Segs) - 1; i >= 0; i-- {
		if float64(u.Segs[i].T0) <= t {
			return u.Segs[i]
		}
	}

	if len(u.Segs) > 0 {
		return u.Segs[0]
	}

	return Seg{}
}

// PosAt returns the unit's position in tiles at server time t.
func (u *Unit) PosAt(t float64) (x, y float64) { return u.seg(t).PosAt(t) }

// Moving reports whether the unit is walking at t.
func (u *Unit) Moving(t float64) bool { return len(u.Segs) > 0 && u.seg(t).Moving(t) }

// Final returns where the unit ends up (end of the newest segment).
func (u *Unit) Final() (x, y float64) {
	if len(u.Segs) == 0 {
		return 0, 0
	}

	s := u.Segs[len(u.Segs)-1]

	return s.X1, s.Y1
}

// Walk and run speeds in tiles per second. UNVERIFIED (placeholders; an engine
// host passes its own through Rules).
const (
	WalkSpeed = 6.0
	RunSpeed  = 9.0
)
