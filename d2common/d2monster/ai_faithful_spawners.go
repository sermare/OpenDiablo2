package d2monster

// Faithful ports of four spawner/caster AIs that ran on generic stand-ins:
// EvilHole 0x5fa590, HighPriest 0x5df2a0, GenericSpawner 0x5e5110 (pre-hook
// 0x5e50f0) and InvisoSpawner 0x5def70 (all read with the read-only Ghidra
// tools; rules in d2-re-notes/ai-spawners.md). The file name sorts before
// ai_rest.go so these register first and the stand-ins skip them.
//
// Scratch[0..1] are AiGeneral +0x14/+0x18; Brain.SpawnClass is +0x3c.
// Things the world must do (units created, cells probed) are optional host
// interfaces; without them the corresponding step fails, like a blocked cell.

// SpawnerHost creates the units the spawner AIs lay.
type SpawnerHost interface {
	// SpawnHoleMinion is EvilHole's spawn: the minion class of the hole at
	// the hole's own position (0x63fff0 case BaseId 0x141, then the spawn
	// core with flags 2/0x42 and the unit flags 0x4020000). It reports
	// whether a unit was created.
	SpawnHoleMinion(b *Brain) bool
	// SpawnRoomUnit is InvisoSpawner's spawn: a unit of the level-scaled
	// class at a random point of the room (0x54ba70 + 0x5b0b60). It reports
	// whether a unit was created.
	SpawnRoomUnit(b *Brain) bool
	// PickGenericSpawnClass is MONAI_GenericSpawner_PickSpawnClass 0x5e4fb0:
	// the class the hut lays, drawn from the unit's own seed (b.Seed).
	PickGenericSpawnClass(b *Brain) (class int, ok bool)
}

// MissileRanger gives the Range of the monster's MissS1 missile, 0 when the
// class has none (HighPriest).
type MissileRanger interface {
	MissileS1Range(b *Brain) int
}

// Constants of the four ports.
const (
	evilHoleNearDist      = 5
	evilHoleWait          = 20
	highPriestHealDistSq  = 2500 // 0x9c4
	highPriestHealBelow   = 75   // 0x4b
	highPriestSkillGap    = 100
	genericSpawnerRange   = 20
	genericSpawnerWait    = 20
	invisoSpawnerWait     = 15
	genericSpawnerSkillID = 167 // 0xa7
)

// highPriestOffsets is the 4-entry table at 0x6e490c.
var highPriestOffsets = [4][2]int{{-5, -5}, {5, -5}, {5, 5}, {-5, 5}}

func init() {
	register("EvilHole", standInMode("EvilHole"), thinkEvilHole)
	register("HighPriest", standInMode("HighPriest"), thinkHighPriest)
	register("GenericSpawner", standInMode("GenericSpawner"), thinkGenericSpawner)
	register("InvisoSpawner", standInMode("InvisoSpawner"), thinkInvisoSpawner)
}

// FaithfulSpawners lists the names ported in this file.
func FaithfulSpawners() []string {
	return []string{"EvilHole", "HighPriest", "GenericSpawner", "InvisoSpawner"}
}

// setMode is MONAI_QueueModeAtPoint(mode, 0, 0) for a unit without target.
func (c *Ctx) setMode(m Mode) { c.Attack(m, c.self()) }

// thinkEvilHole is MONAI_Think_EvilHole (VERIFIED). The state is the unit's
// mode: NU -> S3 -> S4, in S4 one minion every aip2 frames while the
// counter S1 (started at aip1) lasts, then the death mode.
func thinkEvilHole(c *Ctx) {
	b := c.B
	s := &b.Scratch

	if s[0] < 1 {
		s[0] = b.AIP(2) + c.frame()
		s[1] = b.AIP(1)
	}

	switch {
	case b.Mode == ModeNeutral:
		if c.Dist > evilHoleNearDist {
			c.Sleep(5)

			return
		}

		c.setMode(ModeSkill3)
		c.Sleep(evilHoleWait)
	case b.Mode == ModeSkill3:
		c.setMode(ModeSkill4)
		c.Sleep(evilHoleWait)
	case b.Mode == ModeSkill4 && s[1] > 0:
		if s[0] < c.frame() {
			s[0] = b.AIP(2) + c.frame()

			if h, ok := c.W.(SpawnerHost); ok && h.SpawnHoleMinion(b) {
				s[1]--
			}
		}

		c.Sleep(b.AIP(2))
	default:
		c.setMode(ModeDying)
	}
}

// thinkHighPriest is MONAI_Think_HighPriest (VERIFIED). Scratch[0] is the
// engaged flag, Scratch[1] the frame of the next special skill.
// aip1 open-attack %, aip2 heal %, aip3 heal gap, aip4 skill1 %, aip5 S1
// missile %, aip6 retreat %, aip7 melee %, aip8 close distance.
func thinkHighPriest(c *Ctx) {
	b, t := c.B, faTarget(c)
	s := &b.Scratch
	frame := c.frame()

	if s[0] == 0 {
		if c.InRange {
			if b.AIP(1) <= b.Roll(100) {
				c.WalkAway(t, 6)

				return
			}

			s[0] = 1
			c.Attack(ModeAttack1, t)

			return
		}

		if c.highPriestOpening(t, frame) {
			return
		}
	}

	s[0] = 1

	if !c.InRange {
		if c.Dist > 5 || b.AIP(7) <= b.Roll(100) {
			if b.Roll(100) < b.AIP(6) {
				s[0] = 0
				c.Sleep(10)

				return
			}

			if b.Roll(100) > 0x45 {
				c.Wander(12)

				return
			}

			c.WalkTo(t, 7)

			return
		}

		c.Attack(ModeSkill1, t)

		return
	}

	if b.AIP(7) <= b.Roll(100) {
		if b.Roll(100) < b.AIP(6) {
			c.WalkAway(t, 6)
			s[0] = 0

			return
		}

		if b.Roll(100) > 0x59 {
			c.Sleep(10)

			return
		}

		c.Attack(ModeAttack1, t)

		return
	}

	c.Attack(ModeSkill1, t)
}

// highPriestOpening is the not-in-range part of the opening phase; it returns
// true when it ended the tick.
func (c *Ctx) highPriestOpening(t Target, frame int) bool {
	b := c.B
	s := &b.Scratch

	if c.skill(slotB) && s[1] < frame && b.Roll(100) < b.AIP(2) {
		r := fbxScan(c, FBXScanQuery{Kind: FBXScanWoundedAlly, Radius2: highPriestHealDistSq,
			LifeBelow: highPriestHealBelow})
		if r.Found {
			s[1] = frame + b.AIP(3)
			c.Cast(slotB, r.T)

			return true
		}
	}

	if c.skill(slotA) && s[1] < frame && c.Dist < b.AIP(8) && b.Roll(100) < b.AIP(4) {
		o := highPriestOffsets[b.Seed.Step()&3]
		c.Cast(slotA, Target{X: t.X + o[0], Y: t.Y + o[1]})
		s[1] = frame + highPriestSkillGap

		return true
	}

	if b.AIP(5) > 0 {
		if mr, ok := c.W.(MissileRanger); ok {
			if rng := mr.MissileS1Range(b); rng > 0 && c.Dist < rng-2 && b.Roll(100) < b.AIP(5) {
				c.Attack(ModeSkill1, t)

				return true
			}
		}
	}

	if b.Roll(100) < 0x50 {
		if c.Dist <= b.AIP(8) {
			c.Circle(t, 3)
		} else {
			c.WalkTo(t, 6)
		}

		return true
	}

	return false
}

// thinkGenericSpawner is MONAI_Think_GenericSpawner with pre-hook 0x5e50f0
// (VERIFIED). Scratch[0] last lay frame, Scratch[1] units laid, SpawnClass
// the chosen class (-1 = not chosen). aip1 lay gap, aip3 lay limit.
func thinkGenericSpawner(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0], s[1], b.SpawnClass = c.frame(), 0, -1 })

	if c.Target != nil {
		if b.SpawnClass == -1 {
			ok := false
			if h, hok := c.W.(SpawnerHost); hok {
				b.SpawnClass, ok = h.PickGenericSpawnClass(b)
			}

			if !ok {
				b.SpawnClass = -1
				c.die()

				return
			}
		}

		if c.Dist <= genericSpawnerRange {
			if s[1] >= b.AIP(3) {
				c.die()

				return
			}

			gap := c.frame() - s[0]
			if gap < 0 {
				gap = -gap
			}

			if b.AIP(1) <= gap {
				s[0] = c.frame()

				if c.spawnSpace() {
					s[1]++
					c.castSkillID(genericSpawnerSkillID, *c.Target)

					return
				}
			}
		}
	}

	c.Sleep(genericSpawnerWait)
}

// thinkInvisoSpawner is MONAI_Think_InvisoSpawner (VERIFIED). Scratch[0] is
// the next spawn frame, Scratch[1] the units left. aip1 count, aip2 range,
// aip3 gap.
func thinkInvisoSpawner(c *Ctx) {
	b := c.B
	s := &b.Scratch

	if s[0] == 0 {
		s[1] = b.AIP(1)
	}

	if c.Dist <= b.AIP(2) {
		if s[1] < 1 {
			c.die()

			return
		}

		if s[0] <= c.frame() {
			if h, ok := c.W.(SpawnerHost); ok && h.SpawnRoomUnit(b) {
				s[1]--
				s[0] = b.AIP(3) + c.frame()
			}
		}
	}

	c.Sleep(invisoSpawnerWait)
}

// castSkillID casts a skill by id (the hut's 0xa7) at t: through RawCaster
// when the world has it, else through the monstats Skill1 slot.
func (c *Ctx) castSkillID(id int, t Target) {
	if r, ok := c.W.(RawCaster); ok {
		mode := ModeSkill1
		if c.skill(slotA) && c.B.Profile.Skills[slotA].Mode != 0 {
			mode = c.B.Profile.Skills[slotA].Mode
		}

		if r.CastSkillID(c.B, id, mode, &t, nil) {
			c.busy()

			return
		}
	} else if c.skill(slotA) {
		c.Cast(slotA, t)

		return
	}

	c.Sleep(10)
}
