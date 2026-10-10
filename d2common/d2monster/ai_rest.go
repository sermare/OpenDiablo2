package d2monster

// Stand-in archetypes for every monai.txt AI that has no decompiled port yet,
// so that no common monster idles. They are NOT faithful ports: they read no
// aip column (the per-AI meaning of the columns is not recorded for these) and
// only give each family its recognisable behaviour (chase and bite, cast from a
// distance and back off, fly and strafe, stay put and shoot). Every entry is
// UNVERIFIED and replaceable: a real port registers the same name from a file
// sorted after this one, or removes the name from the table below.
//
// Deliberately inert (non-combat) AIs are registered too, with a long sleep,
// so tools can tell "no combat by design" from "not ported" (Implemented).

type genericKind int

const (
	kindMelee genericKind = iota
	kindCaster
	kindFlyer
	kindTurret
	kindInert
	kindPet
)

// genericAIs maps monai names to their stand-in archetype.
var genericAIs = map[string]genericKind{
	// melee chasers
	"Baboon": kindMelee, "ClawViper": kindMelee, "ClawViperEx": kindMelee, "Arach": kindMelee,
	"MaggotLarva":  kindMelee,
	"DeathMauler":  kindMelee,
	"BloodLord":    kindMelee,
	"FrozenHorror": kindMelee, "Ancient": kindMelee,
	"CorruptLancer": kindMelee, "ElementalBeast": kindMelee,
	"7TIllusion": kindMelee, "FlyingScimitar": kindFlyer,

	// casters / ranged
	"FingerMage":    kindCaster,
	"MinionSpawner": kindCaster,
	"Hydra":         kindTurret,

	// flyers
	"BloodHawk": kindFlyer, "Mosquito": kindFlyer,
	"MaggotEgg": kindInert,

	// stationary shooters
	"GargoyleTrap": kindTurret,
	"ArcaneTower":  kindTurret,
	"Catapult":     kindTurret,

	// non-combat by design
	"GoodNpcRanged": kindInert,
	"Buffy":         kindInert,
	"HellMeteor":    kindInert, "FoulCrowNest": kindInert, "MosquitoNest": kindInert,
	"BoneWall": kindInert, "InvisoPet": kindInert,
	"AncientStatue": kindInert,
}

// DeliberatelyUnported are monai names left to the "idle" stand-in because
// other work owns them. Empty since Tentacle, TentacleHead and FrogDemon were
// ported (ai_fb_3.go).
var DeliberatelyUnported []string

// GenericAIs lists the stand-in names, for tests and tooling.
func GenericAIs() []string {
	out := make([]string, 0, len(genericAIs))
	for n := range genericAIs {
		out = append(out, n)
	}

	return out
}

func init() {
	for name, kind := range genericAIs {
		if _, taken := Lookup(name); taken {
			continue
		}

		switch kind {
		case kindMelee:
			register(name, standInMode(name), thinkGenericMelee)
		case kindCaster:
			register(name, standInMode(name), thinkGenericCaster)
		case kindFlyer:
			register(name, standInMode(name), thinkGenericFlyer)
		case kindTurret:
			register(name, standInMode(name), thinkGenericTurret)
		case kindPet:
			if h, ok := Lookup("Hireable"); ok {
				register(name, h.TargetMode, h.Think)
			} else {
				register(name, TargetNone, thinkInert)
			}
		default:
			register(name, standInMode(name), thinkInert)
		}
	}
}

func thinkInert(c *Ctx) { c.Sleep(100) }

// genericTarget is the target of a stand-in think function. AIs whose exe
// target mode is 0 or 2 (BladeCreeper, the sentries, Hydra...) get no target
// from the tick, so like the exe's own-scan AIs they look one up with
// MONAI_GetAttackTargetAndDistance; it reports false when there is none.
func (c *Ctx) genericTarget() (Target, bool) {
	if c.Target != nil {
		return *c.Target, true
	}

	t, d, ok := c.W.AttackTarget(c.B)
	if !ok {
		return Target{}, false
	}

	c.Target, c.Dist, c.InRange = &t, d, c.W.InRange(c.B, t, d)

	return t, true
}

// standInMode is the target mode a stand-in registers with: the verified
// exe table entry of its monai name (monster-ai.md), TargetStandard when the
// name is not in the table.
func standInMode(name string) int {
	if m, ok := AITargetMode(name); ok {
		return m
	}

	return TargetStandard
}

// thinkGenericMelee chases and attacks; A2 is used a quarter of the time when
// the class has a second attack skill slot to name (it is merely tried).
func thinkGenericMelee(c *Ctx) {
	b := c.B
	t, ok := c.genericTarget()
	if !ok {
		c.Sleep(25)

		return
	}

	if c.InRange {
		if b.Chance(70) {
			if b.Chance(25) {
				c.Attack(ModeAttack2, t)
			} else {
				c.Attack(ModeAttack1, t)
			}

			return
		}

		c.Sleep(6)

		return
	}

	if !c.WalkTo(t, meleeReach) {
		c.Sleep(10)
	}
}

// thinkGenericCaster fires its monstats skills from a distance, backs off when
// pressed and otherwise closes in.
func thinkGenericCaster(c *Ctx) {
	b := c.B
	t, ok := c.genericTarget()
	if !ok {
		c.Sleep(25)

		return
	}
	p := b.Profile

	if c.Dist < 4 && b.Chance(30) && c.WalkAway(t, 8) {
		return
	}

	if c.Dist <= 20 {
		var used []int

		for i := range p.Skills {
			if p.Skills[i].Used() {
				used = append(used, i)
			}
		}

		if len(used) > 0 && b.Chance(45) {
			c.Cast(used[b.Roll(len(used))], t)

			return
		}

		if c.InRange && b.Chance(50) {
			c.Attack(ModeAttack1, t)

			return
		}
	}

	if c.Dist > 12 && c.WalkTo(t, 12) {
		return
	}

	if b.Chance(30) && c.Circle(t, 3) {
		return
	}

	c.Sleep(10)
}

// thinkGenericFlyer rushes the target, strikes, and strafes around it.
func thinkGenericFlyer(c *Ctx) {
	b := c.B
	t, ok := c.genericTarget()
	if !ok {
		c.Sleep(25)

		return
	}

	b.Airborne = !c.InRange

	if c.InRange {
		if b.Chance(60) {
			c.Attack(ModeAttack1, t)

			return
		}

		if c.Circle(t, 4) {
			return
		}

		c.Sleep(5)

		return
	}

	c.SetSpeed(100)

	if !c.RunTo(t, meleeReach) {
		c.Sleep(8)
	}
}

// thinkGenericTurret never moves: it fires its first skill (or A1) at a target
// in range and watches otherwise.
func thinkGenericTurret(c *Ctx) {
	b := c.B
	t, ok := c.genericTarget()
	if !ok {
		c.Sleep(25)

		return
	}

	if c.Dist <= b.Profile.Aggro() && b.Chance(60) {
		if b.Profile.Skills[slot1].Used() {
			c.Cast(slot1, t)
		} else {
			c.Attack(ModeAttack1, t)
		}

		return
	}

	c.Sleep(15)
}
