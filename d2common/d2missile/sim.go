package d2missile

import (
	"errors"
	"fmt"
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

const (
	// maxCreateDistance: the exe refuses to create a missile whose target is
	// 100 or more subtiles away in x or y (verified).
	maxCreateDistance = 100
	// accelPeriod: path acceleration is applied every 5th frame (verified).
	accelPeriod = 5
	// cellSample is the sampling step along a frame's movement used to find
	// the subtiles traversed.
	cellSample = 0.25
)

// Missile is one live server missile.
type Missile struct {
	ID      uint32
	Spec    *Spec
	Owner   Owner
	SkillID int
	Level   int
	Damage  DamageDesc

	X, Y     float64 // subtile position
	DX, DY   float64 // unit direction
	Velocity int     // 8.8, current, before the exe's 75% path scale (display/API)
	Life     int     // remaining frames
	Total    int     // lifetime at creation
	// CollideFrom: unit collisions are only tested once Life <= CollideFrom
	// (the Activate delay, verified).
	CollideFrom int
	Pierce      int // remaining pierce charges (stat 0x148 in the game)

	// Travel/ClampDist implement the lob clamp: the missile ends at the target.
	Travel, ClampDist float64

	// OnHit is called for every target the missile strikes (also when it
	// deals no damage), e.g. to apply Howl's fear state.
	OnHit func(m *Missile, t Target)
	// Tag is free for the caller.
	Tag interface{}

	// Home is the unit a homing missile steers toward. For SrvDoFunc 7
	// (Guided Arrow) the exe's rule is modelled (re-aim every Param1 frames
	// while 4..24 subtiles away, verified 0x5ac2c0); other missiles with Home
	// (Bone Spirit) turn instantly every frame (UNVERIFIED, their do function
	// was not read). HitEvery, when
	// set, replaces the table's NextHit/NextDelay with a per-target interval
	// (fire walls and other lingering missiles). ScalePct scales every rolled
	// damage component (damage "per second" listings).
	Home     Target
	HitEvery int
	ScalePct int

	// HomeMode is the exe's missile data field +0x28 as used by Guided Arrow
	// (SRVDO_010 0x5da2f0, SrvDoFunc 7, hit func 10; verified): 1 = homing on
	// Home, 2 = aimed at the ground, 4 = already re-targeted once (5 / 6 are
	// the re-targeted homing / ground legs).
	HomeMode int
	// AreaRadius is the radius in subtiles of the area hit functions when the
	// table's sHitPar1 is empty (the exe evaluates a skill calc then).
	AreaRadius int
	// HitSubRange overrides the lifetime of the sub missiles of hit func 14
	// (the exe passes the skill's linear value 0x150/0x154, flag 0x8000).
	HitSubRange int

	legX, legY float64 // dest - source at creation: the length of a ground leg
	parked     bool    // arrived at its aim point but still alive (hit func 10)

	// pathVel is the exe's path velocity: the creation velocity * 75/100
	// (verified, 0x59d5d0). Accel is added to it every 5th frame (verified,
	// PATH_ApplyAccelAndScaleStep 0x651750) and it is clamped to MaxVel<<8,
	// which is NOT scaled by 75% (verified: CreateServerMissile passes the raw
	// MaxVel byte << 8 to 0x649a90). accel is zeroed once the cap is hit.
	pathVel, accel int
	age            int
	lastHit        string
	nextHit        map[string]int
	dead           bool
	explodes       bool
}

// Dead reports whether the missile has been destroyed.
func (m *Missile) Dead() bool { return m.dead }

// CreateParams describe a missile to create (the notes' create struct).
type CreateParams struct {
	Spec    *Spec
	Owner   Owner
	SkillID int
	Level   int
	Damage  DamageDesc

	X, Y         float64 // start, subtile
	DestX, DestY float64 // aim point, subtile

	// Velocity, if non-zero, overrides the table velocity (8.8, flag 4).
	Velocity int
	// Range, if > 0, overrides the lifetime in frames (flag 0x8000).
	Range int
	// Angle rotates the direction (radians), used for fan-outs.
	Angle float64
	// ClampToDest stops the missile at the aim point (lob, flag 0x400/0x420).
	ClampToDest bool
	// Pierce is the number of targets the missile may pass through when its
	// Pierce flag is set.
	Pierce int
	// Slowed/SlowPct model state 0x57 on the owner for CanSlow missiles.
	Slowed  bool
	SlowPct int

	OnHit func(m *Missile, t Target)
	Tag   interface{}

	// Stationary makes the missile stay where it is created (traps, walls).
	Stationary bool
	// Home, HitEvery and ScalePct: see Missile.
	Home     Target
	HitEvery int
	ScalePct int

	// PierceChance is the owner's skill_pierce + item_pierce percent (stats
	// 0xa6 and 0x9c). For a missile with the Pierce flag the exe rolls up to 4
	// pierce charges from it at creation (0x59d4e0, verified). Ignored when
	// Pierce is given.
	PierceChance int
	// AreaRadius, HitSubRange: see Missile.
	AreaRadius, HitSubRange int
	// HomeMode overrides the Guided Arrow mode (default: 1 when Home is set
	// else 2, for SrvDoFunc 7 / hit func 10 missiles).
	HomeMode int
}

// Positioned is implemented by targets that know where they are; homing
// missiles need it.
type Positioned interface {
	SubPos() (x, y float64)
}

// Sim owns the live missiles of a world.
type Sim struct {
	World   World
	Table   Table
	OnEvent func(Event)

	missiles []*Missile
	pending  []*Missile
	stepping bool
	nextID   uint32
	// targetNext is the NextHit state 0x56: it lives on the target, so it is
	// shared by every NextHit missile (verified 0x5ab5d0 / 0x5aba10).
	targetNext map[string]int
}

// NewSim creates a simulation.
func NewSim(w World, t Table) *Sim { return &Sim{World: w, Table: t} }

// Missiles returns the live missiles.
func (s *Sim) Missiles() []*Missile { return s.missiles }

func (s *Sim) emit(e Event) {
	if s.OnEvent != nil {
		s.OnEvent(e)
	}
}

// ErrTooFar is returned when the aim point is out of the 100 subtile range.
var ErrTooFar = errors.New("d2missile: target out of range")

// Create makes a missile (MISSILE_CreateServerMissile, 0x59d5d0).
func (s *Sim) Create(p CreateParams) (*Missile, error) {
	if p.Spec == nil {
		return nil, errors.New("d2missile: nil spec")
	}

	dx, dy := p.DestX-p.X, p.DestY-p.Y
	if math.Abs(dx) >= maxCreateDistance || math.Abs(dy) >= maxCreateDistance {
		return nil, ErrTooFar
	}

	dist := math.Hypot(dx, dy)

	// An aim point equal to the start becomes start+(1,1) (verified,
	// 0x59d7aa), i.e. a diagonal direction.
	ux, uy := math.Sqrt2/2, math.Sqrt2/2
	if dist > 0 {
		ux, uy = dx/dist, dy/dist
	}

	if p.Angle != 0 {
		c, sn := math.Cos(p.Angle), math.Sin(p.Angle)
		ux, uy = ux*c-uy*sn, ux*sn+uy*c
	}

	sp := p.Spec

	vel := p.Velocity
	if p.Stationary {
		vel = 0
	} else if vel == 0 {
		vel = d2combat.MissileVelocity(uint8(sp.Vel), uint8(sp.VelLev), p.Level, sp.CanSlow && p.Slowed, p.SlowPct)
	}

	life := p.Range
	if life <= 0 {
		life = d2combat.MissileRange(int16(sp.Range), int16(sp.LevRange), p.Level, sp.SubLoop,
			uint8(sp.SubStart), uint8(sp.SubStop), 0, false)
	}

	s.nextID++

	m := &Missile{
		ID: s.nextID, Spec: sp, Owner: p.Owner, SkillID: p.SkillID, Level: p.Level, Damage: p.Damage,
		X: p.X, Y: p.Y, DX: ux, DY: uy, Velocity: vel, Life: life, Total: life,
		CollideFrom: life - sp.Activate, Pierce: p.Pierce, OnHit: p.OnHit, Tag: p.Tag,
		Home: p.Home, HitEvery: p.HitEvery, ScalePct: p.ScalePct, nextHit: map[string]int{},
	}

	m.pathVel = d2combat.MissileStep(vel)
	m.accel = sp.Accel
	m.AreaRadius, m.HitSubRange = p.AreaRadius, p.HitSubRange
	m.legX, m.legY = dx, dy

	if m.Pierce == 0 && sp.Pierce && p.PierceChance > 0 {
		m.Pierce = PierceCharges(p.PierceChance, p.Owner.Roller)
	}

	if sp.SrvDoFunc == 7 || sp.SrvHitFunc == 10 {
		m.HomeMode = p.HomeMode
		if m.HomeMode == 0 {
			m.HomeMode = 2
			if p.Home != nil {
				m.HomeMode = 1
			}
		}
	}

	if p.ClampToDest {
		m.ClampDist = dist
	}

	if s.stepping {
		s.pending = append(s.pending, m)
	} else {
		s.missiles = append(s.missiles, m)
	}

	s.emit(Event{Kind: EventCreate, Missile: m})

	return m, nil
}

// unitMask says which unit kinds a collide type hits (table 0x739948 and its
// unit predicates 0x5a6210 / 0x5a61d0 / 0x5a6270, verified): type 1 players
// (the exe also accepts monsters that carry state 0x69 with stat 0xac == 2,
// which the sim does not model), 2 and 5 monsters, 3 and 8 players and
// monsters. Type 7 (predicate 0x5a62b0) hits other missiles that have a flag
// bit set in the global byte at 0x6cf260; missile versus missile is not
// modelled. 0, 4 and 6 hit no units.
func unitMask(ct int) (players, monsters bool) {
	switch ct {
	case 1:
		return true, false
	case 2, 5:
		return false, true
	case 3, 8:
		return true, true
	}

	return false, false
}

// wallBlock is the terrain part of the block mask of a collide type (0x73994c
// + 8*type: 1 0x84, 2 0x104, 3 0x184, 5 0x104, 6 0x4, 7 0x40, 8 0x185,
// verified bytes). A missile cannot enter such a cell: the step stops at the
// last free cell and the missile ends through ProcessHitOrExpire(0, 1), i.e.
// the hit function runs (UNVERIFIED: the cell test 0x6513e0 that applies the
// mask was not read).
func wallBlock(ct int) uint16 {
	switch ct {
	case 1, 2, 3, 5, 6:
		return d2path.FlagWall
	case 8:
		return d2path.FlagWalk | d2path.FlagWall
	}

	return 0
}

// RemoveOwned ends every live missile fired by an owner without events (no
// hit, no explosion) and returns them. UNVERIFIED whether the original ends
// the missiles of a dead owner; nothing calls this for hero death.
func (s *Sim) RemoveOwned(ownerID string) []*Missile {
	return s.removeIf(func(m *Missile) bool { return m.Owner.ID == ownerID })
}

// Clear ends every live missile without events and returns them: the area
// they flew in is gone (missiles are units of one level).
func (s *Sim) Clear() []*Missile {
	return s.removeIf(func(*Missile) bool { return true })
}

func (s *Sim) removeIf(match func(*Missile) bool) []*Missile {
	var out []*Missile

	filter := func(list []*Missile) []*Missile {
		live := list[:0]

		for _, m := range list {
			switch {
			case m.dead:
			case match(m):
				m.dead = true
				out = append(out, m)
			default:
				live = append(live, m)
			}
		}

		return live
	}

	s.missiles = filter(s.missiles)
	s.pending = filter(s.pending)

	return out
}

// Step advances every missile by one 25 Hz frame.
func (s *Sim) Step() {
	s.stepping = true

	for _, m := range s.missiles {
		if !m.dead {
			s.stepOne(m)
		}
	}

	s.stepping = false

	live := s.missiles[:0]

	for _, m := range s.missiles {
		if !m.dead {
			live = append(live, m)
		}
	}

	s.missiles = append(live, s.pending...)
	s.pending = s.pending[:0]
}

func (s *Sim) vanish(m *Missile) {
	m.dead = true
	s.emit(Event{Kind: EventVanish, Missile: m})
}

func (s *Sim) stepOne(m *Missile) {
	sp := m.Spec
	m.age++

	if m.accel != 0 && m.age%accelPeriod == 0 {
		m.pathVel += m.accel // verified: added to the 75%-scaled path velocity
		if m.pathVel > sp.MaxVel<<8 {
			m.pathVel = sp.MaxVel << 8
			m.accel = 0
		} else if m.pathVel < 0 {
			m.pathVel = 0
		}

		m.Velocity = m.pathVel * 100 / 75
	}

	switch {
	case sp.SrvDoFunc == 7:
		if !s.guidedTurn(m) {
			return
		}
	case m.Home != nil && m.Home.Alive():
		// legacy instant steering for callers that set Home on other missiles
		// (Bone Spirit); the exe's turn rule is only known for SrvDoFunc 7.
		if pt, ok := m.Home.(Positioned); ok {
			hx, hy := pt.SubPos()
			aim(m, hx, hy)
		}
	}

	// displacement per frame in subtiles: step(8.8) * 16 / 65536 (verified scale)
	stepSub := float64(m.pathVel) / 4096.0
	if m.parked {
		stepSub = 0
	}

	ox, oy := m.X, m.Y
	arrived := false

	if m.ClampDist > 0 && m.Travel+stepSub >= m.ClampDist {
		stepSub = m.ClampDist - m.Travel
		arrived = true
	}

	m.X += m.DX * stepSub
	m.Y += m.DY * stepSub
	m.Travel += stepSub

	if arrived {
		// the path is finished: the exe returns from the standard move right
		// after ProcessHitOrExpire(0, 1), before the life is decremented.
		m.ClampDist = 0

		if s.finish(m, EventExpire) {
			m.parked = true
		}

		return
	}

	m.Life--
	if m.Life < 1 {
		s.finish(m, EventExpire)
		return
	}

	if m.parked {
		return
	}

	units := m.Life <= m.CollideFrom // the Activate delay only disables unit hits

	if m.pathVel == 0 { // stationary: the exe re-reads the cell it stands on
		cx, cy := int(math.Floor(m.X)), int(math.Floor(m.Y))

		if sp.CollideType != 0 && s.World.Flags(cx, cy)&(d2path.FlagWalk|d2path.FlagWall) != 0 {
			s.vanish(m) // cached cell flags & 5 -> return 2, no hit function (verified)
			return
		}

		if units {
			s.testUnits(m, cx, cy)
		}

		return
	}

	prev := cell{int(math.Floor(ox)), int(math.Floor(oy))}

	for _, c := range cellsBetween(ox, oy, m.X, m.Y) {
		if s.testCell(m, c, prev, units) {
			return
		}

		prev = c
	}
}

func aim(m *Missile, x, y float64) {
	if d := math.Hypot(x-m.X, y-m.Y); d > 0.01 {
		m.DX, m.DY = (x-m.X)/d, (y-m.Y)/d
	}
}

type cell struct{ x, y int }

// cellsBetween lists the subtiles entered when moving from (ox,oy) to (nx,ny),
// excluding the starting cell.
func cellsBetween(ox, oy, nx, ny float64) []cell {
	var out []cell

	last := cell{int(math.Floor(ox)), int(math.Floor(oy))}
	dist := math.Hypot(nx-ox, ny-oy)
	n := int(math.Ceil(dist / cellSample))

	if n < 1 {
		n = 1
	}

	for i := 1; i <= n; i++ {
		f := float64(i) / float64(n)
		c := cell{int(math.Floor(ox + (nx-ox)*f)), int(math.Floor(oy + (ny-oy)*f))}

		if c != last {
			out = append(out, c)
			last = c
		}
	}

	return out
}

// testCell runs the collision test of one entered subtile; true if the
// missile died (or was parked by a hit function). Collide type 0 skips
// everything (0x5abd00 returns 1 first).
func (s *Sim) testCell(m *Missile, c, prev cell, units bool) bool {
	ct := m.Spec.CollideType
	if ct == 0 {
		return false
	}

	flags := s.World.Flags(c.x, c.y)

	if flags&wallBlock(ct) != 0 {
		// blocked step: the missile stays in the last free cell
		m.X, m.Y = float64(prev.x)+0.5, float64(prev.y)+0.5

		if s.finish(m, EventWall) {
			m.parked = true
			return true
		}

		return true
	}

	if flags&(d2path.FlagWalk|d2path.FlagWall) != 0 {
		// a wall bit that is not in the block mask lets the missile in, but
		// the cached cell flags & 5 test then ends it without a hit function
		s.vanish(m)
		return true
	}

	if units {
		return s.testUnits(m, c.x, c.y)
	}

	return false
}

func (s *Sim) testUnits(m *Missile, cx, cy int) bool {
	players, monsters := unitMask(m.Spec.CollideType)
	if !players && !monsters {
		return false
	}

	for _, t := range s.World.Targets(cx, cy) {
		if (t.IsPlayer() && !players) || (!t.IsPlayer() && !monsters) {
			continue
		}

		s.process(m, t)

		if m.dead {
			return true
		}
	}

	return false
}

// Result bits of ProcessHitOrExpire's A value / of a hit function's return
// (verified 0x5aba10): the hit function's return value REPLACES A.
const (
	resKill   = 1 // destroy (if B&1)
	resDamage = 2 // apply the damage function to the target
	resKeep   = 4 // the missile survives, nothing else happens
)

// process is MISSILE_ProcessHitOrExpire for a target.
func (s *Sim) process(m *Missile, t Target) {
	sp := m.Spec
	frame := s.World.Frame()

	if !t.Alive() {
		return
	}

	if !sp.CollideFriend && !s.World.IsEnemy(m.Owner, t) {
		return
	}

	if sp.LastCollide && m.lastHit == t.ID() {
		return
	}

	// NextHit: state 0x56 sits on the target for NextDelay frames and blocks
	// every NextHit missile, not just this one (verified 0x5ab5d0). HitEvery is
	// a per-missile interval of the sim's own.
	switch {
	case m.HitEvery > 0:
		if m.nextHit[t.ID()] > frame {
			return
		}

		m.nextHit[t.ID()] = frame + m.HitEvery
	case sp.NextHit:
		if s.targetNext[t.ID()] > frame {
			return
		}

		if s.targetNext == nil {
			s.targetNext = map[string]int{}
		}

		s.targetNext[t.ID()] = frame + sp.NextDelay
	}

	m.lastHit = t.ID()

	// B: bit 1 may destroy, bit 2 runs the hit function. A pierce charge
	// (stat 0x148) turns B into 2 (0x5ab550, verified).
	b := 3
	pierced := false

	if sp.Pierce && m.Pierce > 0 {
		m.Pierce--
		b = 2
		pierced = true
	}

	a := resDamage
	if sp.Explosion {
		a = 0 // verified: the Explosion flag clears the damage bit
	}

	if sp.CollideKill {
		a |= resKill
	}

	hit, chance, roll := true, 0, 0

	if sp.ToHit {
		in := d2combat.ToHitInput{
			AttackRating:  m.Owner.AttackRating,
			Defense:       t.Defense(true),
			AttackerLevel: m.Owner.Level,
			DefenderLevel: t.Level(),
		}
		// the skill's to-hit is a percent bonus for players, a flat bonus for
		// monsters (COMBAT_RollToHit; the percent reading is UNVERIFIED)
		if m.Owner.IsPlayer {
			in.AttackRatingPct = m.Damage.ToHit
		} else {
			in.AttackRating += m.Damage.ToHit
		}

		hit, chance, roll = d2combat.RollToHit(m.Owner.Roller, in)
	}

	if !hit {
		s.emit(Event{Kind: EventMiss, Missile: m, Target: t, Chance: chance, Roll: roll})

		// verified (0x5aba10): a missed to-hit roll destroys the missile
		// (return 2), ignoring CollideKill and pierce; only AlwaysExplode runs
		// the hit func, and a result with bit 4 keeps the missile.
		if sp.AlwaysExplode {
			if ret, ok := s.hitFunc(m, t); ok && ret&resKeep != 0 {
				return
			}

			s.explode(m)
		}

		m.dead = true

		return
	}

	if pierced {
		s.emit(Event{Kind: EventPierce, Missile: m, Target: t})
	}

	if b&2 != 0 {
		if ret, ok := s.hitFunc(m, t); ok {
			a = ret

			if a&resKeep != 0 {
				return // passes through without damage
			}
		}
	}

	if a&resDamage != 0 {
		dmg := m.Damage.Roll(m.Owner.Roller)

		if m.ScalePct > 0 && m.ScalePct != 100 {
			dmg.Physical = dmg.Physical * int32(m.ScalePct) / 100
			dmg.Fire = dmg.Fire * int32(m.ScalePct) / 100
			dmg.Lightning = dmg.Lightning * int32(m.ScalePct) / 100
			dmg.Magic = dmg.Magic * int32(m.ScalePct) / 100
			dmg.Cold = dmg.Cold * int32(m.ScalePct) / 100
		}

		s.emit(Event{Kind: EventHit, Missile: m, Target: t, Damage: dmg, Chance: chance, Roll: roll})
	}

	if m.OnHit != nil {
		m.OnHit(m, t)
	}

	if a&resKill != 0 && b&1 != 0 {
		m.dead = true
		s.explode(m)
	}
}

func (s *Sim) lookup(name string) *Spec {
	if name == "" || s.Table == nil {
		return nil
	}

	return s.Table.ByName(name)
}

func (s *Sim) explode(m *Missile) {
	if m.Spec.ExplosionMissile != "" {
		s.emit(Event{Kind: EventExplode, Missile: m, Name: m.Spec.ExplosionMissile})
	}
}

// finish is ProcessHitOrExpire(0, expire=1): the hit function runs with a null
// target (also on a wall or at the end of the path; AlwaysExplode is not
// needed for that, verified 0x5aba10) and the missile is destroyed unless the
// hit function returned bit 4. It reports whether the missile survived.
func (s *Sim) finish(m *Missile, kind EventKind) bool {
	if ret, ok := s.hitFunc(m, nil); ok && ret&resKeep != 0 {
		return true
	}

	m.dead = true
	s.emit(Event{Kind: kind, Missile: m})

	if kind == EventWall || m.Spec.AlwaysExplode {
		s.explode(m)
	}

	return false
}

// PierceCharges rolls the pierce charges a missile with the Pierce flag
// starts with (0x59d4e0, verified): up to 4 times a percent roll below
// chance (owner's skill_pierce + item_pierce) adds a charge; the first failed
// roll stops it.
func PierceCharges(chance int, r d2combat.Roller) int {
	n := 0

	for r != nil && n < 4 && int(r.Roll(100)) < chance {
		n++
	}

	return n
}

// String describes a missile for logs.
func (m *Missile) String() string {
	return fmt.Sprintf("%s#%d@(%.1f,%.1f) life=%d/%d", m.Spec.Name, m.ID, m.X, m.Y, m.Life, m.Total)
}
