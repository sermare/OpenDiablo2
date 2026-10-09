package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// This file answers the optional d2monster interfaces of the post-target
// checks (d2common/d2monster/postcheck.go) from the map collision data.
//
// Live versus gated (see the Options fields):
//   - wounded MonTeleport: gated by Brain.CanTeleport (AiGeneral flag 0x20),
//     which no class sets; Options.CanTeleport is the host switch (nil = off).
//   - level-threat re-target: gated by Options.LevelThreat (levels.txt byte
//     +0x2f, column UNVERIFIED); nil = threat 0 = off.
//   - idle wander: needs Brain.Aggressive (set by nothing yet) or own-tile
//     flag 0x40, which the static map never carries (missile bit); inert.
//   - SandRaider overlay: forwarded to Options.OnOverlay, a visual hook.

const (
	// teleportAttempts is the retry count of 0x54ba70 (VERIFIED: 20).
	teleportAttempts = 20
	// reachMask is the probe mask of 0x5db3a0 (VERIFIED immediate 0x805).
	reachMask uint16 = 0x805
	// tileFlagMask is the own-tile mask of 0x5dd6b0 / 0x5dd7f0 (VERIFIED 0x40).
	tileFlagMask uint16 = d2path.FlagMissile
	// threatRetargetDist is the edge-distance limit of callback 0x5dba10.
	threatRetargetDist = 0x30
)

// rect is an inclusive subtile rectangle.
type rect struct{ left, top, right, bottom int }

// pickTeleportDest is 0x54ba70: up to 20 random subtiles in r drawn from the
// region seed, accepted when the footprint (radius size-1) is free of the
// monster block mask. UNVERIFIED: the exe's placement test internals.
func pickTeleportDest(g d2path.Grid, rng *d2rand.Seed, r rect, size int) (d2monster.Point, bool) {
	w, h := r.right-r.left+1, r.bottom-r.top+1
	if w <= 0 || h <= 0 {
		return d2monster.Point{}, false
	}

	rad := 0
	if size > 1 {
		rad = size - 1
	}

	for i := 0; i < teleportAttempts; i++ {
		x := r.left + int(rng.Roll(int32(w)))
		y := r.top + int(rng.Roll(int32(h)))

		if footprintFree(g, x, y, rad) {
			return d2monster.Point{X: x, Y: y}, true
		}
	}

	return d2monster.Point{}, false
}

func footprintFree(g d2path.Grid, x, y, rad int) bool {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if d2path.Blocked(g, x+dx, y+dy, d2path.MaskMonster|d2path.MaskUnits) {
				return false
			}
		}
	}

	return true
}

// targetReachable is 0x5db3a0: three probes (mask 0x805) at the target's tile
// and at two offsets of 2 and 4 subtiles toward the monster. UNVERIFIED: the
// offsets and the polarity; true (reachable) when any probe is clear.
func targetReachable(g d2path.Grid, from d2monster.Point, t d2monster.Target) bool {
	sx, sy := sign(from.X-t.X), sign(from.Y-t.Y)

	for _, k := range []int{0, 2, 4} {
		if !d2path.Blocked(g, t.X+sx*k, t.Y+sy*k, reachMask) {
			return true
		}
	}

	return false
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}

// ---- d2monster.TeleportPlanner / SkillCaster / Healer ----

// regionRect is the rectangle teleports may land in: the whole map (the exe
// uses the DRLG region of the unit, which this engine does not model).
func (d *Director) regionRect() rect {
	s := d.engine.Size()

	return rect{1, 1, s.Width*subtilesPerTile - 2, s.Height*subtilesPerTile - 2}
}

// TeleportDest implements d2monster.TeleportPlanner.
func (d *Director) TeleportDest(b *d2monster.Brain) (d2monster.Point, bool) {
	if d.opt.IsTownLevel != nil && d.opt.IsTownLevel(b.LevelID) {
		return d2monster.Point{}, false
	}

	return pickTeleportDest(d.fp.grid, d.regionRNG, d.regionRect(), b.Size)
}

// CastSkillAt implements d2monster.SkillCaster for MonTeleport: the unit
// jumps to the point. Other skills are not simulated and report false.
func (d *Director) CastSkillAt(b *d2monster.Brain, skill int, _ d2monster.Mode, p d2monster.Point) bool {
	u := d.unitOf(b)
	if u == nil || skill != teleportSkill {
		return false
	}

	u.m.TeleportTo(p.X, p.Y)
	b.X, b.Y = p.X, p.Y
	d.footprint(u)
	u.mv = nil
	b.WakeNow(d.frame)

	return true
}

// teleportSkill is MonTeleport (VERIFIED immediate 0xb8).
const teleportSkill = 0xb8

// HealByLevel implements d2monster.Healer: life += unit level, capped at max.
// (The exe adds the level << 8 in its 8.8 fixed point life stat.)
func (d *Director) HealByLevel(b *d2monster.Brain) {
	u := d.unitOf(b)
	if u == nil {
		return
	}

	v := &u.m.Vitals
	v.HP += v.Level

	if v.HP > v.MaxHP {
		v.HP = v.MaxHP
	}
}

// ---- level threat, reachability, retarget, tile flag, overlay ----

// LevelThreat implements d2monster.LevelThreatSource; 0 (off) unless the host
// supplies Options.LevelThreat.
func (d *Director) LevelThreat(b *d2monster.Brain) int {
	if d.opt.LevelThreat == nil {
		return 0
	}

	return d.opt.LevelThreat(b.LevelID)
}

// Reachable implements d2monster.TargetReachability.
func (d *Director) Reachable(b *d2monster.Brain, t d2monster.Target) bool {
	return targetReachable(d.fp.grid, d2monster.Point{X: b.X, Y: b.Y}, t)
}

// ThreatTarget implements d2monster.ThreatRetargeter: the nearest living hero
// other than the current target within 48 whose position is reachable.
func (d *Director) ThreatTarget(b *d2monster.Brain, cur d2monster.Target) (d2monster.Target, int, bool) {
	var (
		best  d2monster.Target
		bestD int
		found bool
	)

	for _, id := range d.targetIDs() {
		p := d.targets[id]
		if p == nil || id == cur.ID || !d.targetable(p) {
			continue
		}

		x, y := playerSubtile(p)
		t := d2monster.Target{ID: id, X: x, Y: y, Size: 1, IsPlayer: true}
		dist := d2monster.EdgeDistance(b.X-x, b.Y-y, b.Size)

		if dist > threatRetargetDist || !d.Reachable(b, t) {
			continue
		}

		if !found || dist < bestD {
			best, bestD, found = t, dist, true
		}
	}

	return best, bestD, found
}

// OwnTileFlagged implements d2monster.TileFlagger.
func (d *Director) OwnTileFlagged(b *d2monster.Brain) bool {
	return d.fp.Flags(b.X, b.Y)&tileFlagMask != 0
}

// ShowOverlay implements d2monster.OverlayShower (purely visual).
func (d *Director) ShowOverlay(b *d2monster.Brain, overlay int) {
	if d.opt.OnOverlay != nil {
		d.opt.OnOverlay(b.ID, overlay)
	}
}
