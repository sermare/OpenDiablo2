package d2monster

// Baal hooks that the crab AI (ai_baal.go) needs from the world: the
// retreat teleport of action 0xe, the "dangerous ground skill" test of the
// weight tables (AiBaal.cpp 0x5fb230 / 0x5fb280) and the home-distance inputs.
//
// Sources: Game.exe 1.14b read-only. MONAI_BaalCrabThinkHelper 0x5fbd50 case
// 0xe (teleport) and case 0xf, MONAI_SpawnMinionInheritingLeaderHp 0x5fba40
// (the clone). Statuses are given per rule.

// SkillBaalTeleport is skills.txt row 329 "Baal Teleport" (srvdofunc 98,
// monanim S1), the skill in the Skill5 slot of baalcrab (VERIFIED table).
const SkillBaalTeleport = 329

// baalTeleportStep is the length of the teleport hop in subtiles (VERIFIED
// immediate 0x19 in case 0xe of 0x5fbd50).
const baalTeleportStep = 25

// BaalTeleportDest is the destination of action 0xe (VERIFIED arithmetic): the
// point 25 subtiles from the crab on the line from the target through the
// crab, i.e. a hop AWAY from the target. dist is the unit distance (0 is read
// as 1, like the exe). Integer division truncates toward zero.
func BaalTeleportDest(self, target Point, dist int) Point {
	if dist == 0 {
		dist = 1
	}

	return Point{
		X: (self.X-target.X)*baalTeleportStep/dist + self.X,
		Y: (self.Y-target.Y)*baalTeleportStep/dist + self.Y,
	}
}

// PlacementChecker is an optional World extension: the room test the exe makes
// on the hop destination (MONAI_SpawnAtRandomPointInRoom 0x5xxxxx, which may
// also nudge the point). It returns the point to use; ok false means no free
// spot and the crab does a plain 5-frame wait (UNVERIFIED: the nudging
// radius is unknown).
type PlacementChecker interface {
	PlaceNear(b *Brain, p Point) (Point, bool)
}

// baalTeleportAway is case 0xe. With no target the exe re-enters the switch
// with action 0 (a 5-frame wait). The class 0x2c5 special case that casts at
// the target position instead is not ported (UNVERIFIED which unit it is).
func baalTeleportAway(c *Ctx) {
	b := c.B
	t := c.Target

	if t == nil || !b.Profile.Skills[slot5].Used() {
		c.Sleep(5)

		return
	}

	d := Distance(b.X-t.X, b.Y-t.Y)
	dest := BaalTeleportDest(Point{b.X, b.Y}, Point{t.X, t.Y}, d)

	if pc, ok := c.W.(PlacementChecker); ok {
		var ok2 bool
		if dest, ok2 = pc.PlaceNear(b, dest); !ok2 {
			c.Sleep(5)

			return
		}
	}

	if sc, ok := c.W.(SkillCaster); ok && sc.CastSkillAt(b, SkillBaalTeleport, b.Profile.Skills[slot5].Mode, dest) {
		c.busy()

		return
	}

	c.Sleep(5)
}

// BaalDangerousSkill is AiBaal 0x5fb230: a hero's skill that makes Baal avoid
// the area. Ids are skills.txt (59 Blizzard and 56 Meteor at any level, 51
// Fire Wall above level 3, 27 Immolation Arrow above level 7). VERIFIED
// numbers; the skill NAMES are read from the standard tables.
func BaalDangerousSkill(id, level int) bool {
	switch id {
	case 59, 56:
		return true
	case 51:
		return level > 3
	case 27:
		return level > 7
	}

	return false
}

// BaalHome holds the anchor-distance thresholds of the tables: Far above 0x4b
// (75) and VeryFar above 100 (VERIFIED in the study notes).
const (
	BaalFarDist     = 0x4b
	BaalVeryFarDist = 100
)

// BaalHomeFlags returns the Far and VeryFar inputs from the distance to the
// anchor.
func BaalHomeFlags(distAnchor int) (far, veryFar bool) {
	return distAnchor > BaalFarDist, distAnchor > BaalVeryFarDist
}

// BaalCloneLimit is the most clones alive at once. The weight (2 - clones)*10
// reaches 0 at 2 (VERIFIED); the exe also refuses when the leader group
// reports something through the callback at 0x5fba20 (UNVERIFIED which
// condition), which this cap approximates.
const BaalCloneLimit = 2

// BaalCloneStats is the life the clone gets (VERIFIED): a third of the
// leader's max and current life; the clone's regeneration stat is cleared.
func BaalCloneStats(curHP, maxHP int) (hp, max int) { return curHP / 3, maxHP / 3 }

// BaalCloneOffset maps two draws of the monster's own seed to the clone's
// position relative to the target (or to Baal when there is none): each axis
// is draw%24 - 12 (VERIFIED).
func BaalCloneOffset(rx, ry uint32) (dx, dy int) { return int(rx%24) - 12, int(ry%24) - 12 }
