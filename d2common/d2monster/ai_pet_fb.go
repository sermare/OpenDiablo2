package d2monster

// Faithful ports of the summoned-minion thinkers: NecroPet 0x5e3c30 (with
// MONAI_Pet_ThinkMelee 0x5e3770 and MONAI_Pet_ThinkCaster 0x5e3a00), Vines
// 0x5eb7c0, Totem 0x5ecb10 and Raven 0x5ebd20. They supersede the UNVERIFIED
// simplified versions of ai_pet.go (this file's init runs later).
//
// The exe's path-following helpers (MONAI_FollowLeaderStep, a large switch of
// path probes) and the pet bookkeeping are optional host interfaces (FBN
// prefix); without them a fallback built on the existing PetWorld is used.

// FBNPetStepper is MONAI_FollowLeaderStep(unit, mode, ?, boost, maxDist):
// modes 0 follow, 1 hover, 2 shuffle, 3 teleport next to the leader, 4/5 nudge.
// It returns true when it issued a move. UNVERIFIED mode meanings beyond that.
type FBNPetStepper interface {
	FBNFollowLeader(b *Brain, mode, boost, maxDist int) bool
}

// FBNPetCounter is PETS_CountAllPets for the leader (SelectPetActionByRange
// widens the leash by half of it, capped at 36).
type FBNPetCounter interface {
	FBNPetCount(b *Brain) int
}

// FBNTargetStates answers state/stat questions about a target unit (Vines
// avoid frozen-state (2) or cold-immune (stat 0x2d == 100) foes).
type FBNTargetStates interface {
	FBNTargetHasState(b *Brain, t Target, state int) bool
	FBNTargetStat(b *Brain, t Target, stat int) int
}

// FBNLeaderTeleporter is SERVER_MoveUnitToLevelPosition onto the leader's
// position (the Totem's recall). Returns true on success.
type FBNLeaderTeleporter interface {
	FBNTeleportToLeader(b *Brain) bool
}

// FBNSkillValue is SKILLS_GetSecondBasePlusPerLevelValue of the owner's skill
// 1 at its level: the number of pecks a Raven gets. Fallback 3.
type FBNSkillValue interface {
	FBNRavenPecks(b *Brain) int
}

func init() {
	fbnReg("NecroPet", thinkNecroPetB)
	fbnReg("Vines", thinkVinesB)
	fbnReg("Totem", thinkTotemB)
	fbnReg("Raven", thinkRavenB)
}

// fbnLeader is MONAI_GetLeaderUnit: the linked leader brain, else the owner
// the host reports (PetWorld/MercWorld Owner).
func fbnLeader(c *Ctx) (OwnerInfo, bool) {
	if l := c.B.Leader; l != nil {
		return OwnerInfo{Target: Target{ID: l.ID, X: l.X, Y: l.Y, Size: l.Size, IsPlayer: false},
			Mode: l.Mode, LevelID: l.LevelID}, true
	}

	if w, ok := c.W.(interface {
		Owner(b *Brain) (OwnerInfo, bool)
	}); ok {
		return w.Owner(c.B)
	}

	return OwnerInfo{}, false
}

func fbnInTown(c *Ctx) bool {
	if t, ok := c.W.(TownChecker); ok {
		return t.InTown(c.B)
	}

	return false
}

// fbnFollow is MONAI_FollowLeaderStep; the fallback walks to the owner and
// teleports for mode 3.
func fbnFollow(c *Ctx, l OwnerInfo, mode, boost, maxDist int) bool {
	if s, ok := c.W.(FBNPetStepper); ok {
		return s.FBNFollowLeader(c.B, mode, boost, maxDist)
	}

	if mode == 3 {
		if w, ok := c.W.(interface{ Teleport(b *Brain) bool }); ok && w.Teleport(c.B) {
			c.Sleep(5)

			return true
		}

		return false
	}

	if mode == 5 {
		return c.Wander(3)
	}

	if EdgeDistance(c.B.X-l.X, c.B.Y-l.Y, c.B.Size) > 8 {
		return c.WalkTo(l.Target, 8)
	}

	return false
}
