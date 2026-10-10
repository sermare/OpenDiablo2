package d2monster

// Shared helpers and optional host interfaces of the faithful N-Z ports
// (ai_fb_2.go, ai_fb_3.go, ai_fb_npc.go). Every interface is OPTIONAL: a
// World that does not implement one gets the documented fallback, so the AIs
// run (with less engine detail) in any host and in the unit tests.

// FBXScanKind names one of the unit scans the exe performs with
// MONAI_ForEachUnitByModeCallback (0x5dbe00 family) and a per-AI callback.
// The callbacks of some scans are bare code labels that cannot be decompiled
// read-only, so those filters are UNVERIFIED and described by their use.
type FBXScanKind int

// Scan kinds.
const (
	// FBXScanSiegeTarget is the siege weapons' enemy scan (callback 0x5e05a0,
	// radius squared in the query; the nearest enemy wins; UNVERIFIED filter).
	FBXScanSiegeTarget FBXScanKind = iota + 1
	// FBXScanDefilerCorpse is the PutridDefiler's scan (callback 0x5eeb80,
	// mode 0 / radius 25: a living unit it can bite; UNVERIFIED filter).
	FBXScanDefilerCorpse
	// FBXScanKnightAlly is the OblivionKnight's scan (callback 0x5f9fb0, no
	// radius limit: the ally it heals/buffs; UNVERIFIED filter).
	FBXScanKnightAlly
	// FBXScanOverseer is the Overseer/Nihlathak scan (callback 0x5e1570, radius
	// 0x50, results: one ally and one enemy; UNVERIFIED filter).
	FBXScanOverseer
	// FBXScanRegurgitate is MONAI_Regurgitator_TargetFilter (VERIFIED filter,
	// read from the decompilation): the nearest monster (not the caster) that
	// is dead or dying with corpse flag bit 1 set and bit 9 clear, not in
	// the state mask 0x150, not converted, of a type flagged edible, within the
	// squared radius of the query.
	FBXScanRegurgitate
	// FBXScanPriestAlly is MONAI_ZakarumPriest_AllyFilter (VERIFIED filter): a
	// living monster of classes 0xEB or 0xEE (the priest's zealot allies) that
	// is allied to the caster, within the squared radius, below 61 percent
	// life; the one with the lowest life percent wins and Count counts all.
	FBXScanPriestAlly
	// FBXScanLinkedClass counts the units of Query.Class (a class linked to
	// the caster by the spawn chain) within radius 25 in mode DD (0xc), the
	// VileMother's brood limit.
	FBXScanLinkedClass
	// FBXScanWoundedAlly is the HighPriest heal scan (0x5df240, VERIFIED): a
	// living allied monster within Radius2 (squared) whose life percent is
	// below LifeBelow; the lowest life percent wins.
	FBXScanWoundedAlly
)

// FBXScanQuery is one scan request.
type FBXScanQuery struct {
	Kind    FBXScanKind
	Radius2 int // squared radius when the filter has one
	Class   int // monster class for FBXScanLinkedClass
	// LifeBelow is the life percent bound of FBXScanWoundedAlly.
	LifeBelow int
}

// FBXScanResult is what a scan found.
type FBXScanResult struct {
	Found bool
	T     Target
	// Count is the number of matching units where the exe counts them.
	Count int
	// Found2/T2 is a second result (the Overseer scan reports an ally and an
	// enemy).
	Found2 bool
	T2     Target
}

// FBXScanner is an optional Senses extension implementing the scans above.
// Without it every scan finds nothing.
type FBXScanner interface {
	FBXScan(b *Brain, q FBXScanQuery) FBXScanResult
}

func fbxScan(c *Ctx, q FBXScanQuery) FBXScanResult {
	if s, ok := c.W.(FBXScanner); ok {
		return s.FBXScan(c.B, q)
	}

	return FBXScanResult{}
}

// FBXFlagger is an optional Actor extension for the unit flag word (+0xc4):
// the exe sets bits 0xe or 0x20000 and clears 0xe from AI code. The meaning of
// the bits is UNVERIFIED (0x20000 is set by traps and Trapped Souls on every
// think, 0xe by burrowed/horde units).
type FBXFlagger interface {
	FBXSetFlags(b *Brain, set, clear uint32)
}

func fbxFlags(c *Ctx, set, clear uint32) {
	if f, ok := c.W.(FBXFlagger); ok {
		f.FBXSetFlags(c.B, set, clear)
	}
}

// FBXSpawnCells is MONSTER_AreSpawnerCellsFree at the unit's own position:
// true when a monster can be created there. Fallback: true.
type FBXSpawnCells interface {
	FBXSpawnCellsFree(b *Brain) bool
}

func fbxSpawnCellsFree(c *Ctx) bool {
	if s, ok := c.W.(FBXSpawnCells); ok {
		return s.FBXSpawnCellsFree(c.B)
	}

	return true
}

// FBXLineOfSight is COLLISION_CanTraceLineBetweenUnits(self, t, mask) == 0 as
// used by MONAI_IsNotImmune (mask 4): true when the straight line is BLOCKED.
// Fallback: false (not blocked).
type FBXLineOfSight interface {
	FBXBlocked(b *Brain, t Target, mask int) bool
}

func fbxBlocked(c *Ctx, t Target, mask int) bool {
	if s, ok := c.W.(FBXLineOfSight); ok {
		return s.FBXBlocked(c.B, t, mask)
	}

	return false
}

// FBXSkillRanger gives skills.txt record field +0x158 of the monster's skill
// slot (the siege weapons compare their edge distance with it; the column is
// UNVERIFIED but behaves as the skill's reach). Fallback: the monster's aggro
// radius.
type FBXSkillRanger interface {
	FBXSkillRange(b *Brain, slot int) (int, bool)
}

func fbxSkillRange(c *Ctx, slot int) int {
	if s, ok := c.W.(FBXSkillRanger); ok {
		if r, ok := s.FBXSkillRange(c.B, slot); ok {
			return r
		}
	}

	return c.B.Profile.Aggro()
}

// fbxReachable is MONAI_PickChaseOffsetByEdgeDistance: three collision probes
// (mask 0x805) around the target; true means all are clear. Uses the
// TargetReachability extension of postcheck.go; fallback true.
func fbxReachable(c *Ctx, t Target) bool {
	if r, ok := c.W.(TargetReachability); ok {
		return r.Reachable(c.B, t)
	}

	return true
}

// fbxSkill reports whether monstats skill slot i (0-based) is set (the exe
// tests Skill_n >= 0, sometimes > 0; the id 0 is "Attack" and never a monster
// skill here).
func fbxSkill(b *Brain, i int) bool { return b.Profile.Skills[i].Used() }

// fbxWalk is MONAI_WalkToTarget(target, reach): a failed request falls back to
// a 10 frame sleep, as every AI helper does (monster-ai-2.md 0).
func fbxWalk(c *Ctx, t Target, reach int) {
	if !c.WalkTo(t, reach) {
		c.Sleep(10)
	}
}

// fbxRunSpeed is the movement speed boost the exe derives from monstats
// (record shorts at +0x32 and +0x34, read here as the walk and run
// velocities: UNVERIFIED which columns): run*100/walk - 100. clampLow selects
// the Zealot/Vampire clamp (below 1 becomes 0, above 119 becomes 120) over
// the pet clamp (above 99 becomes 100, negative kept; 100 when walk < 1).
func fbxRunSpeed(p *Profile, zealotClamp bool) int {
	if p.Walk < 1 {
		if zealotClamp {
			return 0
		}

		return 100
	}

	v := p.Run*100/p.Walk - 100

	if zealotClamp {
		if v < 1 {
			return 0
		}

		if v > 119 {
			return 120
		}

		return v
	}

	if v > 99 {
		return 100
	}

	return v
}

// fbxSelf is the monster as a target (ID 0), for casts "at itself" (the exe
// passes the unit itself, e.g. Tentacle's second skill).
func fbxSelf(b *Brain) Target { return Target{ID: b.ID, X: b.X, Y: b.Y, Size: b.Size} }

// fbxStop ends the think without scheduling anything: the exe simply returns
// and no event is queued, so the monster never thinks again until something
// else (a hit, a forced state) wakes it.
func fbxStop(c *Ctx) {
	c.acted = true
	c.B.Wake = waitForever
}
