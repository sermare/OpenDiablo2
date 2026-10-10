package d2mp

import (
	"math"
	"sort"
)

// DefaultInterpMs is how far behind the estimated server time remote units are
// drawn: enough to always have the next movement segment in hand when the
// previous one ends, so remote units glide instead of stopping and jumping.
const DefaultInterpMs = 100

const (
	clockWindow = 32
	blendMs     = 200.0
	keepSegs    = 8
	snapDist    = 3.0 // a correction of the local hero larger than this snaps
)

// RUnit is a unit as a replica sees it.
type RUnit struct {
	Unit
	// LastAttack is the estimated server time of the unit's latest attack or
	// cast event (animation hook); 0 = never.
	LastAttack uint32

	corrX, corrY, corrAt float64
}

// ReplicaConfig configures a Replica.
type ReplicaConfig struct {
	Self  uint32 // the local hero's unit id
	Seed  uint32 // game seed (for the level layout)
	Rules Rules  // nil = DefaultRules
	// Clock returns the local time in ms (monotonic). Required.
	Clock func() float64
	// InterpMs is the interpolation delay of remote units (0 = DefaultInterpMs).
	InterpMs float64
	// Speeds used to predict the local hero's walk (0 = package defaults).
	WalkSpeed, RunSpeed float64
}

// Replica is one client's copy of the world around its hero. It applies the
// server's events and answers where every unit is drawn.
type Replica struct {
	cfg   ReplicaConfig
	rules Rules

	Level uint16
	Def   *LevelDef // layout of the current level, built from the game seed

	units    map[uint32]*RUnit
	serverMs uint32
	offsets  []float64 // local - server samples
	predict  []Seg     // walks sent but not yet confirmed, oldest first

	// State of the local hero that is not a unit property.
	Inv   []string
	Gold  uint32
	XP    uint32
	Trade *TradeView // open trade window, nil when none
	// LastTrade is the final view of the latest finished or cancelled trade.
	LastTrade *TradeView
	Msgs      []string

	// Handler, if set, sees every applied event (sounds, floating numbers...).
	Handler func(Event)
	// Stats counts applied events by type.
	Stats map[EvType]int
}

// NewReplica returns an empty replica.
func NewReplica(cfg ReplicaConfig) *Replica {
	if cfg.Rules == nil {
		cfg.Rules = DefaultRules{}
	}

	if cfg.InterpMs == 0 {
		cfg.InterpMs = DefaultInterpMs
	}

	if cfg.WalkSpeed == 0 {
		cfg.WalkSpeed = WalkSpeed
	}

	if cfg.RunSpeed == 0 {
		cfg.RunSpeed = RunSpeed
	}

	return &Replica{cfg: cfg, rules: cfg.Rules, units: map[uint32]*RUnit{}, Stats: map[EvType]int{}}
}

// Self returns the local hero's unit id.
func (r *Replica) Self() uint32 { return r.cfg.Self }

// Now returns the estimated current server time in ms. The estimate uses the
// smallest observed (local - server) offset, that is the fastest packets, so
// jitter makes it lag by at most the jitter, never run ahead.
func (r *Replica) Now() float64 {
	if len(r.offsets) == 0 {
		return float64(r.serverMs)
	}

	m := r.offsets[0]
	for _, o := range r.offsets {
		m = math.Min(m, o)
	}

	return r.cfg.Clock() - m
}

func (r *Replica) renderTime(id uint32) float64 {
	if id == r.cfg.Self {
		return r.Now()
	}

	return r.Now() - r.cfg.InterpMs
}

// Unit returns a unit (nil when unknown).
func (r *Replica) Unit(id uint32) *RUnit { return r.units[id] }

// Units returns the units of the current level ordered by id.
func (r *Replica) Units() []*RUnit {
	out := make([]*RUnit, 0, len(r.units))
	for _, u := range r.units {
		out = append(out, u)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// Pos returns where a unit is drawn now.
func (r *Replica) Pos(id uint32) (x, y float64, ok bool) {
	u := r.units[id]
	if u == nil {
		return 0, 0, false
	}

	t := r.renderTime(id)
	x, y = u.PosAt(t)

	if u.corrAt > 0 {
		if f := 1 - (r.Now()-u.corrAt)/blendMs; f > 0 {
			x, y = x+u.corrX*f, y+u.corrY*f
		}
	}

	return x, y, true
}

// Digest hashes the logical state of the current level (see DigestUnits).
func (r *Replica) Digest() uint64 {
	us := make([]*Unit, 0, len(r.units))
	for _, u := range r.units {
		us = append(us, &u.Unit)
	}

	return DigestUnits(us)
}

// PredictWalk starts the local hero walking at once, before the server has
// confirmed. The server's own segment replaces it only if it disagrees.
func (r *Replica) PredictWalk(run bool, x, y float64) {
	u := r.units[r.cfg.Self]
	if u == nil || u.Dead {
		return
	}

	now := r.Now()
	cx, cy := u.PosAt(now)
	x0, y0 := Snap(cx), Snap(cy)
	speed := r.cfg.WalkSpeed

	if run {
		speed = r.cfg.RunSpeed
	}

	ex, ey := x0, y0
	if r.Def != nil {
		ex, ey = March(r.Def, x0, y0, x, y)
	}

	seg := Seg{X0: x0, Y0: y0, X1: ex, Y1: ey, Speed: math.Round(speed*100) / 100, T0: uint32(now)}
	if ex == x0 && ey == y0 {
		seg.Speed = 0
	}

	u.Segs = append(u.Segs, seg)
	r.trim(u, now)
	r.predict = append(r.predict, seg)
}

func (r *Replica) trim(u *RUnit, t float64) {
	if len(u.Segs) <= keepSegs {
		return
	}

	// keep the segment in force at t and everything after
	cut := 0

	for i, s := range u.Segs {
		if float64(s.T0) <= t {
			cut = i
		}
	}

	if cut > 0 {
		u.Segs = u.Segs[cut:]
	}
}

// Apply applies one event from the server.
func (r *Replica) Apply(e Event) {
	r.Stats[e.Type]++

	switch e.Type {
	case EvTick:
		r.serverMs = uint32(e.A)
		r.offsets = append(r.offsets, r.cfg.Clock()-float64(e.A))

		if len(r.offsets) > clockWindow {
			r.offsets = r.offsets[1:]
		}
	case EvLevel:
		if e.ID == r.cfg.Self {
			r.Level = e.Level
			r.Def = r.rules.Level(LevelSeed(r.cfg.Seed, e.Level), e.Level)
			r.units = map[uint32]*RUnit{}
			r.predict = nil
		}
	case EvSpawn:
		u := &RUnit{Unit: e.Unit}
		u.Segs = append([]Seg(nil), e.Unit.Segs...)

		if old := r.units[e.ID]; old != nil {
			u.LastAttack = old.LastAttack
		}

		r.units[e.ID] = u
	case EvSeg:
		r.applySeg(e)
	case EvAttack:
		if u := r.units[e.ID]; u != nil {
			u.LastAttack = uint32(r.Now())
		}
	case EvHit:
		if u := r.units[e.ID]; u != nil {
			u.HP = e.B
		}
	case EvDeath:
		if u := r.units[e.ID]; u != nil {
			u.Dead, u.HP = true, 0
		}
	case EvRemove:
		if e.ID != r.cfg.Self {
			delete(r.units, e.ID)
		}
	case EvObject:
		if u := r.units[e.ID]; u != nil {
			u.State = uint8(e.A)
		}
	case EvVitals:
		if u := r.units[e.ID]; u != nil {
			u.HP, u.MaxHP, u.Dead = e.A, e.B, false
		}
	case EvXP:
		if e.ID == r.cfg.Self {
			r.XP = uint32(e.B)
		}
	case EvParty:
		if u := r.units[e.ID]; u != nil {
			u.Party = uint16(e.A)
		}
	case EvTrade:
		tv := e.Trade

		if tv.State == TradeDone || tv.State == TradeCancelled {
			r.LastTrade, r.Trade = &tv, nil
		} else {
			r.Trade = &tv
		}
	case EvInv:
		r.Inv, r.Gold = e.Items, uint32(e.A)
	case EvMsg:
		r.Msgs = append(r.Msgs, e.Text)
	}

	if r.Handler != nil {
		r.Handler(e)
	}
}

func (r *Replica) applySeg(e Event) {
	u := r.units[e.ID]
	if u == nil {
		return
	}

	if e.ID != r.cfg.Self {
		u.Segs = append(u.Segs, e.Seg)
		r.trim(u, r.renderTime(e.ID))

		return
	}

	// the local hero: keep a matching prediction, adopt anything else smoothly
	for i, p := range r.predict {
		if dist(p.X1, p.Y1, e.Seg.X1, e.Seg.Y1) <= 0.25 && math.Abs(p.Speed-e.Seg.Speed) < 0.01 {
			r.predict = r.predict[i+1:]

			return
		}
	}

	r.predict = nil

	now := r.Now()
	if float64(e.Seg.T0) > now+1 { // the next leg of a path: queue it
		u.Segs = append(u.Segs, e.Seg)

		return
	}

	bx, by := u.PosAt(now)
	u.Segs = []Seg{e.Seg}
	nx, ny := u.PosAt(now)

	if d := dist(bx, by, nx, ny); d > 0 && d <= snapDist {
		u.corrX, u.corrY, u.corrAt = bx-nx, by-ny, now
	} else {
		u.corrAt = 0
	}
}
