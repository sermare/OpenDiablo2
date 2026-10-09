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
	"ThornHulk": kindMelee, "PinHead": kindMelee, "MaggotLarva": kindMelee, "Regurgitator": kindMelee,
	"VileDog": kindMelee, "VileMother": kindMelee, "DeathMauler": kindMelee, "PutridDefiler": kindMelee,
	"ReanimatedHorde": kindMelee, "SiegeBeast": kindMelee, "Overseer": kindMelee, "BloodLord": kindMelee,
	"FrozenHorror": kindMelee, "QuillMother": kindMelee, "Ancient": kindMelee, "ZakarumZealot": kindMelee,
	"CorruptLancer": kindMelee, "ElementalBeast": kindMelee, "DarkWanderer": kindMelee,
	"7TIllusion": kindMelee, "Trap-Melee": kindMelee, "SuicideMinion": kindMelee, "FlyingScimitar": kindFlyer,
	"UberBaal": kindMelee, "ShadowWarrior": kindMelee, "Spirit": kindMelee, "TrappedSoul": kindMelee,

	// casters / ranged
	"OblivionKnight": kindCaster, "Vampire": kindCaster, "SuccubusWitch": kindCaster, "FingerMage": kindCaster,
	"ZakarumPriest": kindCaster, "HighPriest": kindCaster, "Imp": kindCaster, "MinionSpawner": kindCaster,
	"Nihlathak": kindCaster, "ShadowMaster": kindCaster, "ShadowMasterNoInit": kindCaster, "Hydra": kindTurret,

	// flyers
	"BloodHawk": kindFlyer, "Mosquito": kindFlyer, "WillOWisp": kindFlyer, "Raven": kindPet,
	"BladeCreeper": kindFlyer, "MaggotEgg": kindInert,

	// stationary shooters
	"GargoyleTrap": kindTurret, "EvilHole": kindTurret, "Trap-Missile": kindTurret,
	"Trap-RightArrow": kindTurret, "Trap-LeftArrow": kindTurret, "Trap-Poison": kindTurret,
	"Trap-Nova": kindTurret, "DesertTurret": kindTurret, "ArcaneTower": kindTurret, "SiegeTower": kindTurret,
	"Catapult": kindTurret, "CatapultSpotter": kindTurret, "AssassinSentry": kindTurret,
	"DeathSentry": kindTurret, "SandMaggotQueen": kindTurret,

	// summoned pets: follow the owner like a mercenary
	"NecroPet": kindPet, "DruidWolf": kindPet, "DruidBear": kindPet,

	// non-combat by design
	"Npc": kindInert, "NpcOutOfTown": kindInert, "NpcStationary": kindInert, "Towner": kindInert,
	"Vendor": kindInert, "GoodNpcRanged": kindInert, "TownRogue": kindInert, "Navi": kindInert,
	"JarJar": kindInert, "NpcBarb": kindInert, "Wussie": kindInert, "Buffy": kindInert,
	"Sarcophagus": kindInert, "HellMeteor": kindInert, "FoulCrowNest": kindInert, "MosquitoNest": kindInert,
	"InvisoSpawner": kindInert, "GenericSpawner": kindInert, "BoneWall": kindInert, "InvisoPet": kindInert,
	"Totem": kindInert, "Vines": kindInert, "CycleOfLife": kindInert, "AncientStatue": kindInert,
}

// DeliberatelyUnported are monai names left to the "idle" stand-in because
// other work owns them (the Baal-wave tentacles and the Frog Demon phases).
var DeliberatelyUnported = []string{"Tentacle", "TentacleHead", "FrogDemon"}

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
			register(name, TargetStandard, thinkGenericMelee)
		case kindCaster:
			register(name, TargetStandard, thinkGenericCaster)
		case kindFlyer:
			register(name, TargetStandard, thinkGenericFlyer)
		case kindTurret:
			register(name, TargetStandard, thinkGenericTurret)
		case kindPet:
			if h, ok := Lookup("Hireable"); ok {
				register(name, h.TargetMode, h.Think)
			} else {
				register(name, TargetNone, thinkInert)
			}
		default:
			register(name, TargetNone, thinkInert)
		}
	}
}

func thinkInert(c *Ctx) { c.Sleep(100) }

// thinkGenericMelee chases and attacks; A2 is used a quarter of the time when
// the class has a second attack skill slot to name (it is merely tried).
func thinkGenericMelee(c *Ctx) {
	b, t := c.B, *c.Target

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
	b, t := c.B, *c.Target
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
	b, t := c.B, *c.Target

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
	b, t := c.B, *c.Target

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
