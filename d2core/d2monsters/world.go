package d2monsters

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

const subtilesPerTile = 5

// mapGrid adapts the map engine's DT1 subtile flags to a d2path.Grid.
// Unit footprints are not stamped, tiles without any floor/wall count as
// outside the grid (blocked, like the exe's 0x27 out-of-room read).
type mapGrid struct{ e *d2mapengine.MapEngine }

func (g mapGrid) Flags(x, y int) uint16 {
	size := g.e.Size()
	if x < 0 || y < 0 || x/subtilesPerTile >= size.Width || y/subtilesPerTile >= size.Height ||
		!g.e.TileExists(x/subtilesPerTile, y/subtilesPerTile) {
		return d2path.OutOfGrid
	}

	return flagsOf(g.e.SubTileAt(x, y))
}

func flagsOf(s *d2dt1.SubTileFlags) uint16 {
	var f uint16

	if s.BlockWalk {
		f |= d2path.FlagWalk
	}

	if s.BlockPlayerWalk {
		f |= d2path.FlagPlayerOnly
	}

	if s.BlockLOS {
		f |= d2path.FlagWall
	}

	return f
}

// Grid returns the collision view of the map (DT1 flags mapped to the exe's
// cell bits; tiles without floor read as the out-of-grid value 0x27), for
// other simulations such as missiles.
func (d *Director) Grid() d2path.Grid { return d.grid }

// playerSubtile is the hero's subtile position.
func playerSubtile(p *d2mapentity.Player) (x, y int) {
	return int(p.Position.X()), int(p.Position.Y())
}

// indexPlayers refreshes the target id table (stable: sorted by player id).
func (d *Director) indexPlayers() {
	list := d.players()
	sort.Slice(list, func(i, j int) bool { return list[i].ID() < list[j].ID() })

	for k := range d.targets {
		delete(d.targets, k)
	}

	for i, p := range list {
		d.targets[uint32(i+1)] = p
	}
}

func (d *Director) playerFor(id uint32) *d2mapentity.Player { return d.targets[id] }

func (d *Director) unitOf(b *d2monster.Brain) *unit { return d.units[b.ID] }

// ---- d2monster.Senses ----

// Frame implements d2monster.Senses.
func (d *Director) Frame() int { return d.frame }

func (d *Director) targetable(p *d2mapentity.Player) bool {
	if p.Stats == nil || p.Stats.Health <= 0 {
		return false
	}

	return d.opt.IgnoreTown || !p.IsInTown()
}

// Nearest implements d2monster.Senses: the nearest living hero.
func (d *Director) Nearest(b *d2monster.Brain) (d2monster.Target, int, bool) {
	var (
		best     d2monster.Target
		bestDist int
		found    bool
	)

	for id := uint32(1); id <= uint32(len(d.targets)); id++ {
		p := d.targets[id]
		if p == nil || !d.targetable(p) {
			continue
		}

		x, y := playerSubtile(p)
		dist := d2monster.EdgeDistance(b.X-x, b.Y-y, b.Size)

		if !found || dist < bestDist {
			best, bestDist, found = d2monster.Target{ID: id, X: x, Y: y, Size: 1, IsPlayer: true}, dist, true
		}
	}

	return best, bestDist, found
}

// AttackTarget implements d2monster.Senses: the nearest hero inside the
// monster's aggro radius (the exe's filter is UNVERIFIED).
func (d *Director) AttackTarget(b *d2monster.Brain) (d2monster.Target, int, bool) {
	t, dist, ok := d.Nearest(b)

	return t, dist, ok && dist <= b.Profile.Aggro()
}

// InRange implements d2monster.Senses. Melee monsters connect within the
// melee reach (7); ranged monsters within 20 with a clear line of sight.
func (d *Director) InRange(b *d2monster.Brain, t d2monster.Target, dist int) bool {
	u := d.unitOf(b)

	if u == nil || !u.m.Stat.IsRanged {
		return dist <= meleeInRange
	}

	if dist > rangedInRange {
		return false
	}

	clear, _ := d2path.TraceLine(d.grid, d2path.FlagWall, d2path.Point{X: b.X, Y: b.Y}, d2path.Point{X: t.X, Y: t.Y})

	return clear
}

// DyingNear implements d2monster.Senses.
func (d *Director) DyingNear(b *d2monster.Brain, radius int) bool {
	for _, u := range d.units {
		if u.b == b || u.m.Mode() != d2monster.ModeDying {
			continue
		}

		x, y := u.m.SubtilePos()
		if d2monster.Distance(b.X-x, b.Y-y) < radius {
			return true
		}
	}

	return false
}

// HasState implements d2monster.Senses: no unit states are modelled yet.
func (d *Director) HasState(*d2monster.Brain, int) bool { return false }

// ---- d2monster.Actor ----

// Attack implements d2monster.Actor.
func (d *Director) Attack(b *d2monster.Brain, mode d2monster.Mode, t d2monster.Target) bool {
	u := d.unitOf(b)
	if u == nil {
		return false
	}

	u.m.StopMoving()
	u.mv = nil
	u.m.Face(float64(t.X), float64(t.Y))

	if !u.m.SetMode(mode) {
		return false
	}

	u.attackTarget = t.ID

	if mode != d2monster.ModeAttack1 && mode != d2monster.ModeAttack2 {
		d.emit("skill", "MONSTER skill name=%s id=%d mode=%s minions=%d", u.m.Label(), b.ID, mode, len(b.Minions))
	}

	return true
}

// Cast implements d2monster.Actor: a skill is played in its monstats mode;
// damage comes from the same mode's attack profile (skills.txt effects such as
// Resurrect, Nova or Fire Wall are not simulated).
func (d *Director) Cast(b *d2monster.Brain, slot int, t d2monster.Target) bool {
	mode := b.Profile.Skills[slot].Mode
	if mode == 0 {
		mode = d2monster.ModeAttack1
	}

	return d.Attack(b, mode, t)
}

// MoveTo implements d2monster.Actor.
func (d *Director) MoveTo(b *d2monster.Brain, dest d2monster.Point, target *d2monster.Target, reach int, run bool) bool {
	u := d.unitOf(b)
	if u == nil {
		return false
	}

	return d.moveTo(u, dest, target, reach, run)
}

// SetSpeed implements d2monster.Actor; the argument's semantics are
// UNVERIFIED (monster-ai-2.md) so velocity stays at the monstats value.
func (d *Director) SetSpeed(*d2monster.Brain, int) {}

// Shout implements d2monster.Shouter.
func (d *Director) Shout(b *d2monster.Brain) {
	if u := d.unitOf(b); u != nil {
		d.Debugf("%s shouts", u.m.Label())
	}
}
