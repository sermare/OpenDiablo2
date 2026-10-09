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
	Velocity int     // 8.8, current
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

	age      int
	lastHit  string
	nextHit  map[string]int
	dead     bool
	explodes bool
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

	ux, uy := 1.0, 0.0
	if dist > 0 {
		ux, uy = dx/dist, dy/dist
	}

	if p.Angle != 0 {
		c, sn := math.Cos(p.Angle), math.Sin(p.Angle)
		ux, uy = ux*c-uy*sn, ux*sn+uy*c
	}

	sp := p.Spec

	vel := p.Velocity
	if vel == 0 {
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
		nextHit: map[string]int{},
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

// unitMask says which unit kinds a collide type hits (table 0x739948, mask
// bits 0x80 players, 0x100 monsters).
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

// wallMask is the cell flag set that destroys the missile. Types without a
// unit predicate (0, 4, 6) use bits 0x1|0x4; the others the wall bit 0x4 of
// their mask, plus 0x1 for type 8 (verified from the mask table).
func wallMask(ct int) uint16 {
	switch ct {
	case 0, 4, 6, 8:
		return d2path.FlagWalk | d2path.FlagWall
	}

	return d2path.FlagWall
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

func (s *Sim) stepOne(m *Missile) {
	sp := m.Spec
	m.age++

	if sp.Accel != 0 && m.age%accelPeriod == 0 {
		m.Velocity += sp.Accel // unit of Accel UNVERIFIED (8.8 per 5 frames assumed)
		if m.Velocity < 0 {
			m.Velocity = 0
		}

		if sp.MaxVel > 0 && m.Velocity > sp.MaxVel<<8 {
			m.Velocity = sp.MaxVel << 8
		}
	}

	// displacement per frame in subtiles: step(8.8) * 16 / 65536 (verified scale)
	stepSub := float64(d2combat.MissileStep(m.Velocity)) / 4096.0

	ox, oy := m.X, m.Y
	arrived := false

	if m.ClampDist > 0 && m.Travel+stepSub >= m.ClampDist {
		stepSub = m.ClampDist - m.Travel
		arrived = true
	}

	m.X += m.DX * stepSub
	m.Y += m.DY * stepSub
	m.Travel += stepSub

	m.Life--
	if m.Life < 1 || arrived {
		s.expire(m)
		return
	}

	if m.Life > m.CollideFrom {
		return // still in the activation delay
	}

	for _, c := range cellsBetween(ox, oy, m.X, m.Y) {
		if s.testCell(m, c.x, c.y) {
			return
		}
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

// testCell runs the collision test of one subtile; true if the missile died.
func (s *Sim) testCell(m *Missile, cx, cy int) bool {
	ct := m.Spec.CollideType
	players, monsters := unitMask(ct)

	if players || monsters {
		for _, t := range s.World.Targets(cx, cy) {
			if (t.IsPlayer() && !players) || (!t.IsPlayer() && !monsters) {
				continue
			}

			s.process(m, t)

			if m.dead {
				return true
			}
		}
	}

	if s.World.Flags(cx, cy)&wallMask(ct) != 0 {
		m.X, m.Y = float64(cx)+0.5, float64(cy)+0.5
		m.dead = true
		s.emit(Event{Kind: EventWall, Missile: m})
		s.explode(m)

		return true
	}

	return false
}

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

	if sp.NextHit && m.nextHit[t.ID()] > frame {
		return
	}

	m.lastHit = t.ID()

	if sp.NextHit {
		m.nextHit[t.ID()] = frame + sp.NextDelay
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

	if hit {
		dmg := m.Damage.Roll(m.Owner.Roller)
		s.emit(Event{Kind: EventHit, Missile: m, Target: t, Damage: dmg, Chance: chance, Roll: roll})

		if m.OnHit != nil {
			m.OnHit(m, t)
		}

		s.spawnHitSubMissiles(m)
	} else {
		s.emit(Event{Kind: EventMiss, Missile: m, Target: t, Chance: chance, Roll: roll})
	}

	if sp.Pierce && m.Pierce > 0 {
		m.Pierce--
		s.emit(Event{Kind: EventPierce, Missile: m, Target: t})

		return
	}

	if sp.CollideKill {
		m.dead = true

		if hit || sp.AlwaysExplode {
			s.explode(m)
		}
	}
}

// spawnHitSubMissiles runs pSrvHitFunc 2 (HitSubMissile1) and 4 (all four).
func (s *Sim) spawnHitSubMissiles(m *Missile) {
	var names []string

	switch m.Spec.SrvHitFunc {
	case 2:
		names = m.Spec.HitSubMissile[:1]
	case 4:
		names = m.Spec.HitSubMissile[:]
	default:
		return
	}

	for _, n := range names {
		sub := s.lookup(n)
		if sub == nil {
			continue
		}

		_, _ = s.Create(CreateParams{Spec: sub, Owner: m.Owner, SkillID: m.SkillID, Level: m.Level,
			Damage: m.Damage, X: m.X, Y: m.Y, DestX: m.X, DestY: m.Y})
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

// expire ends a missile whose lifetime ran out (or that reached its clamped
// destination). It explodes only with AlwaysExplode.
func (s *Sim) expire(m *Missile) {
	m.dead = true
	s.emit(Event{Kind: EventExpire, Missile: m})

	if m.Spec.AlwaysExplode {
		s.explode(m)
	}
}

// String describes a missile for logs.
func (m *Missile) String() string {
	return fmt.Sprintf("%s#%d@(%.1f,%.1f) life=%d/%d", m.Spec.Name, m.ID, m.X, m.Y, m.Life, m.Total)
}
