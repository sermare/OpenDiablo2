package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Host side of the Baal crab AI (d2common/d2monster/ai_baal.go and
// ai_baal_hooks.go): the situation inputs, the clone spawn and the retreat
// teleport placement. Rules and statuses are in ai_baal_hooks.go; notes in
// d2-re-notes/gaps-slice-C.md.

const (
	worldstoneChamberLevel = 0x84 // levels.txt id checked by the table input (VERIFIED immediate)
	baalCloneKey           = "baalclone"
	baalCrabAI             = "baalcrab"
	baalCloneExeClass      = 0x23a // exe class of the clone (VERIFIED immediate, see FindStat fallback)
	baalCloneRing          = 6
	baalTeleportRing       = 4
)

// baalAnchor returns the anchor command point of the crab (its spawn point).
func baalAnchor(b *d2monster.Brain) (x, y int, ok bool) {
	a := b.FindCommand(d2monster.CmdAnchor)
	if a == nil {
		return 0, 0, false
	}

	return a.X, a.Y, true
}

// heroDangerous reports a hero whose left or right skill is one Baal avoids.
func heroDangerous(p *d2mapentity.Player) bool {
	for _, hs := range []*d2hero.HeroSkill{p.LeftSkill, p.RightSkill} {
		if hs != nil && hs.SkillRecord != nil && d2monster.BaalDangerousSkill(hs.ID, hs.SkillPoints) {
			return true
		}
	}

	return false
}

// BaalSituation implements d2monster.BaalSituationHook for the inputs the
// engine knows: the anchor distances, the hero count, the dangerous-skill
// test on the current target, the Worldstone Chamber test and the clones
// alive. The threat score (Score) and TargetBlocked are not computed
// (UNVERIFIED helpers), so Buff stays at its base weight.
func (d *Director) BaalSituation(b *d2monster.Brain, s *d2monster.BaalSituation) {
	u := d.unitOf(b)
	if u == nil {
		return
	}

	s.Clones = d.liveSummons(b.ID, baalCloneKey)

	var heroes, near int

	ax, ay, hasAnchor := baalAnchor(b)

	for _, id := range d.targetIDs() {
		p := d.targets[id]
		if p == nil || !d.targetable(p) {
			continue
		}

		heroes++

		x, y := playerSubtile(p)
		if d2monster.Distance(b.X-x, b.Y-y) <= d2monster.BaalFarDist {
			near++
		}

		if hasAnchor && b.LevelID == worldstoneChamberLevel && d2monster.Distance(ax-x, ay-y) <= d2monster.BaalFarDist {
			s.Worldstone = true
		}
	}

	if near > 0 {
		s.Targets = near
	}

	s.Nearby = heroes

	if hasAnchor {
		s.Far, s.VeryFar = d2monster.BaalHomeFlags(d2monster.Distance(b.X-ax, b.Y-ay))
	}

	if b.HasTarget {
		if p := d.playerFor(b.TargetID); p != nil {
			s.Flag1 = heroDangerous(p)
		}
	}
}

// PlaceNear implements d2monster.PlacementChecker: the nearest free cell to
// the hop destination (UNVERIFIED stand-in for the exe's room test).
func (d *Director) PlaceNear(_ *d2monster.Brain, p d2monster.Point) (d2monster.Point, bool) {
	if d.fp == nil {
		return d2monster.Point{}, false
	}

	q, ok := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: p.X, Y: p.Y}, baalTeleportRing)
	if !ok {
		return d2monster.Point{}, false
	}

	return d2monster.Point{X: q.X, Y: q.Y}, true
}

// baalMayClone is the host bound on live clones: the AI limit and the
// Director's summon cap (HostSummonCap).
func baalMayClone(live int) bool {
	limit := d2monster.BaalCloneLimit
	if c := HostSummonCap(SummonNest, baalCrabAI); c < limit {
		limit = c
	}

	return live < limit
}

// SpawnBaalClone implements d2monster.Cloner (FUN_005fba40): a clone placed
// within +-12 subtiles of Baal's target (or of Baal), with a third of his life,
// linked to him as a minion.
func (d *Director) SpawnBaalClone(b *d2monster.Brain) bool {
	u := d.unitOf(b)
	if u == nil || d.fp == nil || d.asset == nil {
		return false
	}

	if !baalMayClone(d.liveSummons(b.ID, baalCloneKey)) {
		return false
	}

	st := d.FindStat(baalCloneKey)
	if st == nil {
		st = d.statByID[baalCloneExeClass]
	}

	if st == nil {
		return false
	}

	cx, cy := b.X, b.Y
	if b.HasTarget {
		if x, y, ok := d.targetPos(u, b.TargetID); ok {
			cx, cy = x, y
		}
	}

	dx, dy := d2monster.BaalCloneOffset(uint32(b.Roll(24)), uint32(b.Roll(24)))

	p, ok := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: cx + dx, Y: cy + dy}, baalCloneRing)
	if !ok {
		return false
	}

	m, err := d.spawn(st, p.X, p.Y, nil)
	if err != nil {
		return false
	}

	nu := d.byEntity[m.ID()]
	if nu == nil {
		return false
	}

	nu.summoner = b.ID
	b.AddMinion(nu.b)
	nu.b.Wake = d.frame + 15

	hp, max := d2monster.BaalCloneStats(u.m.Vitals.HP, u.m.Vitals.MaxHP)
	nu.m.Vitals.HP, nu.m.Vitals.MaxHP = hp, max
	d.Counters.Summoned++
	d.emit("summon", "BAAL %s cloned id=%d pos=(%d,%d) hp=%d/%d", u.m.Label(), nu.b.ID, p.X, p.Y, hp, max)

	return true
}
