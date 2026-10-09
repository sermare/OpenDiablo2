package d2monster

// Post-target checks left out of the first port (MONAI_PostTargetChecks
// 0x5aefc0): the wounded MonTeleport 0x5aedc0, the level-threat re-target, and
// the idle wander of 0x5dd6b0 / 0x5dd7f0. Addresses are Game.exe 1.14b.

const (
	// skillMonTeleport is the skill id queued by 0x5aedc0 (VERIFIED immediate
	// 0xb8, mode 4 = A1 animation).
	skillMonTeleport = 0xb8
	// stateNoHeal is the unit state that blocks the emergency heal (VERIFIED
	// immediate 0x34; which state it is, is not recorded).
	stateNoHeal = 0x34

	teleportRollPct   = 40 // VERIFIED: unit seed % 100 < 0x28
	teleportActPct    = 15 // VERIFIED: bounded roll(100) < 0xf
	teleportHealPct   = 25 // VERIFIED: bounded roll(100) < 0x19
	teleportWoundedHP = 30 // VERIFIED: life% < 0x1e
	teleportNearDist  = 10 // VERIFIED: tick distance < 10 for non-melee units
	idleWanderRadius  = 5  // VERIFIED push 5 in 0x5dd6b0 / 0x5dd7f0
	threatWanderRange = 4  // VERIFIED push 4 in 0x5aefc0
)

// teleportClasses are the classes whose melee flag the exe forces on
// (VERIFIED immediates 10, 0x159, 0x22d at 0x5aee68..): they teleport only
// when wounded, never merely because a player is near.
var teleportClasses = map[int]bool{10: true, 0x159: true, 0x22d: true}

// TeleportPlanner is an optional World extension: the destination search of
// 0x54ba70 as the exe uses it in 0x5aedc0. VERIFIED shape: up to 20 attempts,
// each a uniformly random subtile inside the rectangle of the unit's region
// (NOT near the target), drawn from the REGION's own seed (+0x6c), accepted
// when a size-aware placement test passes; the point must then lie in a room
// (0x45ef50) that is not a town level (0x61a850). UNVERIFIED: the placement
// test internals. Because the region seed is used, the monster's own RNG is
// not consumed by the destination search.
type TeleportPlanner interface {
	TeleportDest(b *Brain) (Point, bool)
}

// SkillCaster is an optional Actor extension: queue a skill by id at a ground
// point (0x5dd8e0 with mode, skill id, no target, x, y).
type SkillCaster interface {
	CastSkillAt(b *Brain, skill int, mode Mode, p Point) bool
}

// Healer is an optional Actor extension: raise the unit's life by its level
// in whole points (VERIFIED: stat 0xc, the unit level, << 8 added to stat 6,
// capped so life does not exceed the maximum from 0x625f70).
type Healer interface {
	HealByLevel(b *Brain)
}

// LevelThreatSource is an optional Senses extension: the byte at +0x2f of the
// levels.txt record of the unit's room (0x619ef0 -> 0x6xxxxx levels lookup),
// 0 when the unit has no room. UNVERIFIED: which levels.txt column that is.
type LevelThreatSource interface {
	LevelThreat(b *Brain) int
}

// TargetReachability is an optional Senses extension standing for 0x5db3a0:
// three collision probes (mask 0x805) around the target (its tile and two
// offsets of 2..4 subtiles chosen by the edge distance). The polarity is
// inferred from the callback 0x5dba10, which accepts a candidate only when it
// is non-zero: UNVERIFIED, true here means "reachable".
type TargetReachability interface {
	Reachable(b *Brain, t Target) bool
}

// ThreatRetargeter is an optional Senses extension: the scan 0x5dbe00 with
// mode 0xb (VERIFIED: table 0x6e4890 entry 11 = scan the rooms around the
// players with callback 0x5dba10). The callback takes the nearest candidate
// other than the current target that is a living player or monster (type 0/1),
// has unit flag bit 2 (+0xc4), passes the 0x552270 relation test, is within 48
// (edge distance <= 0x30), passes 0x5db690 > 1 (UNVERIFIED) and is reachable.
type ThreatRetargeter interface {
	ThreatTarget(b *Brain, current Target) (Target, int, bool)
}

// TileFlagger is an optional Senses extension: the collision test at the
// unit's own tile with mask 0x40 (0x64ebc0). UNVERIFIED: what the bit means.
type TileFlagger interface {
	OwnTileFlagged(b *Brain) bool
}

// postTargetChecks is 0x5aefc0 after the target is known. It reports true when
// the tick ends. VERIFIED order: wake-up, wounded teleport, threat re-target.
func postTargetChecks(c *Ctx) bool {
	if c.Target == nil {
		return false
	}

	if postTargetWake(c) || woundedTeleport(c) {
		return true
	}

	return threatRetarget(c)
}

// woundedTeleport is MONAI_TryWoundedMonTeleport 0x5aedc0; true ends the tick.
// Order of RNG use (VERIFIED): (1) one unit-seed step, taken % 100, must be
// below 40, (2) a second roll(100) below 15, (3) only when life < 30%, a
// third roll(100) below 25 for the heal. Roll (1) is taken only when the unit
// is alive and flag 0x20 is set; (2) only when the HP/distance gate passes.
func woundedTeleport(c *Ctx) bool {
	b := c.B

	if !b.Mode.IsAlive() || !b.CanTeleport {
		return false
	}

	if b.Roll(100) >= teleportRollPct {
		return false
	}

	melee := b.Profile.Melee || teleportClasses[b.Class]
	wounded := b.HPPercent < teleportWoundedHP

	if !wounded && (melee || c.Dist >= teleportNearDist) {
		return false
	}

	if b.Roll(100) >= teleportActPct {
		return false
	}

	tp, ok := c.W.(TeleportPlanner)
	if !ok {
		return false
	}

	dest, ok := tp.TeleportDest(b)
	if !ok {
		return false
	}

	if wounded && b.Roll(100) < teleportHealPct && !c.W.HasState(b, stateNoHeal) {
		if h, ok := c.W.(Healer); ok {
			h.HealByLevel(b)
		}
	}

	if sc, ok := c.W.(SkillCaster); ok && sc.CastSkillAt(b, skillMonTeleport, ModeAttack1, dest) {
		c.busy()
	} else {
		// UNVERIFIED: the exe ignores the result and returns 1; without a
		// caster the port idles so the tick does not spin.
		c.Sleep(10)
	}

	return true
}

// threatRetarget is the tail of 0x5aefc0: a melee unit that can walk, in a
// level whose threat byte T is non-zero and below the tick distance, whose
// target is not reachable, takes the nearest other candidate (the target and
// distance in the tick params change, the in-range flag does not - VERIFIED,
// the exe only writes +8 and +0x14), or wanders 4 and ends the tick.
func threatRetarget(c *Ctx) bool {
	b := c.B

	if !b.Profile.Melee || b.Profile.NoWalk {
		return false
	}

	ls, ok := c.W.(LevelThreatSource)
	if !ok {
		return false
	}

	th := ls.LevelThreat(b)
	if th <= 0 || th >= c.Dist {
		return false
	}

	if r, ok := c.W.(TargetReachability); !ok || r.Reachable(b, *c.Target) {
		return false
	}

	if rt, ok := c.W.(ThreatRetargeter); ok {
		if t, d, found := rt.ThreatTarget(b, *c.Target); found {
			c.Target, c.Dist = &t, d

			return false
		}
	}

	if !c.Wander(threatWanderRange) {
		c.Sleep(10)
	}

	return true
}

// idleWander is the no-target wander of 0x5dd6b0 (aggressiveToo: the unit
// shuffles when it is aggressive or its own tile is flagged) and 0x5dd7f0
// (modes 4/5: only the tile test). Both need the walk mode (VERIFIED), then
// Wander(5) and the tick ends. It reports whether a wander (or its fallback
// sleep) was issued.
func idleWander(c *Ctx, aggressiveToo bool) bool {
	b := c.B
	if b.Profile.NoWalk {
		return false
	}

	flagged := false
	if tf, ok := c.W.(TileFlagger); ok {
		flagged = tf.OwnTileFlagged(b)
	}

	if !(aggressiveToo && b.Aggressive) && !flagged {
		return false
	}

	if !c.Wander(idleWanderRadius) {
		c.Sleep(10)
	}

	return true
}
