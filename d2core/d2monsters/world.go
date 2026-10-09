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

	d.playerFootprints()
}

// playerFootprints keeps the heroes' footprint (flag 0x80) current.
func (d *Director) playerFootprints() {
	seen := map[uint32]bool{}

	for id, p := range d.targets {
		key := playerKeyBase + id
		seen[key] = true

		x, y := playerSubtile(p)
		d.fp.Move(key, x, y, d2path.FlagPlayer)
	}

	for key := range d.fpPlayer {
		if !seen[key] {
			d.fp.Remove(key)
		}
	}

	d.fpPlayer = seen
}

// footprint refreshes a monster's footprint: a living monster occupies its
// subtile (0x100); stacking is measured for the autotest summary.
func (d *Director) footprint(u *unit) {
	if !u.m.Alive() {
		return
	}

	x, y := u.m.SubtilePos()
	d.fp.Move(u.b.ID, x, y, d2path.FlagMonster)

	if n := len(d.fp.cells[d2path.Point{X: x, Y: y}]); n > d.Counters.MaxStack {
		d.Counters.MaxStack = n
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

// Nearest implements d2monster.Senses: the nearest living hero. A converted
// monster (Conversion) hunts the nearest hostile monster instead; hostile
// monsters also hunt living mercenaries and converted monsters (the exe scans
// the owner's party list for the monster's act, MONAI_FindNearestPlayerTarget).
func (d *Director) Nearest(b *d2monster.Brain) (d2monster.Target, int, bool) {
	var (
		best     d2monster.Target
		bestDist int
		found    bool
	)

	if u := d.unitOf(b); u != nil && u.friendly() {
		return d.nearestEnemy(b)
	}

	if b.Allied {
		return d.nearestEnemy(b)
	}

	// hostile monsters also hunt living mercenaries and converted monsters
	for _, mu := range d.sortedUnits() {
		if !mu.m.Alive() || mu.b == b {
			continue
		}

		var id uint32

		switch {
		case mu.merc != nil && (d.opt.IgnoreTown || !mu.merc.owner.IsInTown()):
			id = mercTargetBase + mu.b.ID
		case mu.merc == nil && mu.b.Allied:
			id = unitTargetBase + mu.b.ID
		default:
			continue
		}

		x, y := mu.m.SubtilePos()
		dist := d2monster.EdgeDistance(b.X-x, b.Y-y, b.Size)

		if !found || dist < bestDist {
			best, bestDist, found = d2monster.Target{ID: id, X: x, Y: y, Size: 1}, dist, true
		}
	}

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
	if u := d.unitOf(b); u != nil && u.friendly() {
		return d.nearestEnemy(b)
	}

	t, dist, ok := d.Nearest(b)

	return t, dist, ok && dist <= b.Profile.Aggro()
}

// InRange implements d2monster.Senses. The flag is judged for the monster's
// default attack (A1): melee attacks connect within the melee reach (7),
// attacks with a missile within the ranged distance and a clear line of
// sight. Other modes are checked again when they strike (see monsterStrike).
func (d *Director) InRange(b *d2monster.Brain, t d2monster.Target, dist int) bool {
	u := d.unitOf(b)

	if u == nil || !d.isRanged(u) {
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

// HasState implements d2monster.Senses. Unit states are not modelled in
// general; the forced conditions answer for the unit state their skill applies
// (terror 56, dim vision 23, taunt 27, confuse 59, attract 57, conversion 53).
func (d *Director) HasState(b *d2monster.Brain, state int) bool {
	return b.Forced != d2monster.ForcedNone && b.Forced.UnitStateID() == state
}

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

	if u.ally != nil { // the pet AI names its victim by unit id
		u.ally.strikeAt = d.units[t.ID]
	}
	d.playPlans(u, attackPlans(d.soundRecord(u), mode, d.snd.Intn))
	u.aimX, u.aimY = t.X, t.Y

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

	if handled, ok := d.fireTrap(d.unitOf(b), t); handled {
		return ok
	}

	if t.ID >= corpseTargetBase {
		if cu := d.units[t.ID-corpseTargetBase]; cu == nil || cu.raising || cu.m.Alive() {
			return false
		}

		d.units[t.ID-corpseTargetBase].raising = true // reserved until the cast lands or is lost
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

// targetPos resolves a target id to a subtile position for the unit u: a
// monster chases players or mercs, a merc follows its owner or chases a
// monster.
func (d *Director) targetPos(u *unit, id uint32) (x, y int, ok bool) {
	switch {
	case id >= unitTargetBase:
		if t := d.units[id-unitTargetBase]; t != nil && t.m.Alive() {
			x, y = t.m.SubtilePos()

			return x, y, true
		}
	case id >= mercTargetBase:
		if t := d.units[id-mercTargetBase]; t != nil && t.merc != nil {
			x, y = t.m.SubtilePos()

			return x, y, true
		}
	case u.merc != nil && id != d.ownerTargetID(u.merc):
		if t := d.units[id]; t != nil {
			x, y = t.m.SubtilePos()

			return x, y, true
		}
	default:
		if p := d.playerFor(id); p != nil {
			x, y = playerSubtile(p)

			return x, y, true
		}
	}

	return 0, 0, false
}

// targetIDs returns the target table ids in ascending order, so that loops
// over the heroes do not depend on map iteration order.
func (d *Director) targetIDs() []uint32 {
	ids := make([]uint32, 0, len(d.targets))
	for id := range d.targets {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	return ids
}
