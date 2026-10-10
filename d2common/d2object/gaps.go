package d2object

// Rules of the object gaps closed by the second object audit (feat/objects-gaps). Every function names the
// address of Game.exe it follows; a rule that was read from the binary is VERIFIED, one that was guessed is
// UNVERIFIED. Numbers only: no decompiled code.

// Rand is the generator the rules draw from. Roll returns a number in [0, n) and 0 for n <= 1 (the exe's
// RAND_RollSeedModulo). *Roller implements it.
type Rand interface{ Roll(n int) int }

// ---------------------------------------------------------------------------------------------------------
// Exploding barrel (OperateFn 7, 0x5821c0 -> 0x5820c0 -> 0x5de7f0)
// ---------------------------------------------------------------------------------------------------------

// Explosion constants. VERIFIED in OBJECT_OperateExplodeDamageAndSetMode (0x5821c0) and
// OBJECT_ServerDamageUnitsAroundObject (0x5820c0).
const (
	// ExplosionRadius is the radius in subtiles: UNIT_IsWithinRadiusOfPoint(unit, barrel, 3), a Euclidean
	// test dx*dx + dy*dy <= r*r on the unit's subtile position (0x642cd0).
	ExplosionRadius = 3
	// ExplosionChainDistance: another unexploded barrel (objects.txt id 11, mode 0) closer than this by
	// UnitDistance chain-explodes (0x5820c0, UNIT_GetDistanceToUnit < 3).
	ExplosionChainDistance = 3
	// ExplodingBarrelID is the objects.txt row the chain test compares the class with.
	ExplodingBarrelID = 11
	// ExplosionMinHitChance is the floor of the percent chance a unit is hurt (0x5de7f0: values below 0x41 are
	// raised to 0x41).
	ExplosionMinHitChance = 0x41
	// ExplosionBaseHitChance is the constant (0x7d) added to the chance.
	ExplosionBaseHitChance = 0x7d
)

// WithinRadius is UNIT_IsWithinRadiusOfPoint (0x642cd0): the squared distance of the offsets is at most r*r.
func WithinRadius(dx, dy, r int) bool { return dx*dx+dy*dy <= r*r }

// UnitDistance is UNIT_GetDistanceToUnit (0x642b10), the octagonal distance in subtiles: both axis offsets
// are shortened by half of each unit's size, floored at 0, then (2*max + min) / 2. sizeA and sizeB are the
// units' SizeX values.
func UnitDistance(dx, dy, sizeA, sizeB int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	cut := -(sizeB / 2) - sizeA/2
	dx, dy = dx+cut, dy+cut

	if dx < 0 {
		dx = 0
	}

	if dy < 0 {
		dy = 0
	}

	if dx <= dy {
		return (dx + dy*2) / 2
	}

	return (dy + dx*2) / 2
}

// ExplosionTarget is what the damage roll of 0x5de7f0 reads from the unit that may be hurt. Life8 is the
// current life in the exe's 8.8 fixed point (stat 6).
type ExplosionTarget struct {
	Level, Dex, Defense, Life8 int
}

// ExplosionOutcome is one resolved explosion packet.
type ExplosionOutcome struct {
	Hit     bool
	Chance  int // percent chance, after the floor
	Roll    int // the 0..99 roll compared with Chance
	Damage8 int // damage in 8.8 fixed point, 0 on a miss
}

// ExplosionHitChance is the percent chance of 0x5de7f0, VERIFIED:
//
//	r      = Roll(level >> 2)                  (target level, stat 0xc)
//	chance = 2 * (r - 5*(dex >> 1)) - defense + 0x7d,  at least 0x41
//
// where dex is stat 2 and defense stat 0x1f of the target. (The exe adds the level and subtracts it again in
// an 8 bit register; the result is r unless level + r overflows a byte.) The roll consumes the barrel's seed.
func ExplosionHitChance(level, dex, defense int, r Rand) int {
	roll := r.Roll(level >> 2)
	chance := (((roll+level)&0xff)-5*(dex>>1)-level)*2 - defense + ExplosionBaseHitChance

	if chance < ExplosionMinHitChance {
		chance = ExplosionMinHitChance
	}

	return chance
}

// ExplosionDamage8 is the damage of one packet in 8.8 fixed point (0x5de780, VERIFIED): with L the target's
// current life in 8.8, lo = max(L >> 5, 1) and hi = max(L >> 3, lo + 1); the damage is
// damagePct * (Roll(hi - lo + 0x100) + lo) / 100, damagePct being the objects.txt Damage byte (100 for every
// row of 1.14b). So a barrel takes roughly 3% to 12.5% of the unit's CURRENT life.
func ExplosionDamage8(life8, damagePct int, r Rand) int {
	lo := life8 >> 5
	if lo < 1 {
		lo = 1
	}

	hi := life8 >> 3
	if hi < lo+1 {
		hi = lo + 1
	}

	return damagePct * (r.Roll(hi-lo+0x100) + lo) / 100
}

// RollExplosion resolves one explosion packet on a target: the hit chance (one roll), the percent roll and,
// on a hit, the damage roll, in this order on the barrel's seed (0x5de7f0). Whether the target stands in a town
// or behind a wall (PATH_CanTraceLineIgnoringBothUnits 0x804) is the caller's test.
func RollExplosion(r Rand, t ExplosionTarget, damagePct int) ExplosionOutcome {
	out := ExplosionOutcome{Chance: ExplosionHitChance(t.Level, t.Dex, t.Defense, r)}
	out.Roll = r.Roll(100)

	if out.Roll < out.Chance {
		out.Hit = true
		out.Damage8 = ExplosionDamage8(t.Life8, damagePct, r)
	}

	return out
}

// Fixed8Points turns an 8.8 damage into whole hit points. The exe keeps the fraction in the stat; the engine
// keeps integer life, so the damage is rounded down but a hit that landed costs at least 1 point
// (UNVERIFIED rounding).
func Fixed8Points(d8 int) int {
	if d8 <= 0 {
		return 0
	}

	if p := d8 >> 8; p > 0 {
		return p
	}

	return 1
}

// ---------------------------------------------------------------------------------------------------------
// Locked and trapped containers (init 0x54db00 / 0x54da20, operate 0x583e70, spawn handlers 0x580410)
// ---------------------------------------------------------------------------------------------------------

// ObjectInitFn values (objects.txt InitFn, the table at 0x72f578) that roll a container variant. VERIFIED.
const (
	InitVariant     = 2 // OBJINIT_RollRandomVariantByLevel: only the spawn handler ("trap") roll
	InitVariantLock = 3 // OBJINIT_RollVariantWithHighFlag: the same roll plus the lock bit
)

// TrapChancePct is the percent chance that a container carries a spawn handler (the exe's trapped chest),
// from the level's MonLvl1 column (normal, classic column, LevelsTxt wMonLvl1): monLvl/8 + 5 (0x54da20).
func TrapChancePct(monLvl int) int { return monLvl/8 + 5 }

// LockChancePct is the percent chance that a Lockable container is locked: monLvl/2 + 8 (0x54db00).
func LockChancePct(monLvl int) int { return monLvl/2 + 8 }

// TrapHandlerMax is the exclusive upper bound of the rolled handler: SEED_RollRangeFromSeed(1, 9), so the
// handler is 1..8 with equal odds (0x54da7d).
const TrapHandlerMax = 9

// ChestInit is what the exe stores in the object data byte at +4 when a container is created: the low seven
// bits are the spawn handler (0 for none), bit 7 is the lock.
type ChestInit struct {
	Locked  bool
	Handler int
}

// RollChestInit follows OBJINIT_RollVariantWithHighFlag / OBJINIT_RollRandomVariantByLevel. initFn is the
// objects.txt InitFn; other values leave the container plain. The draws are: percent < TrapChancePct, then
// (if it fired) the handler 1..8, then, for InitFn 3 and a Lockable row, percent < LockChancePct. The
// original draws from the level's seed; the engine passes a per object generator (UNVERIFIED: the exact
// stream position).
func RollChestInit(initFn int, lockable bool, monLvl int, r Rand) ChestInit {
	var out ChestInit

	if initFn != InitVariant && initFn != InitVariantLock {
		return out
	}

	if r.Roll(100) < TrapChancePct(monLvl) {
		out.Handler = 1 + r.Roll(TrapHandlerMax-1)
	}

	if initFn == InitVariantLock && lockable && r.Roll(100) < LockChancePct(monLvl) {
		out.Locked = true
	}

	return out
}

// SpawnKind is what a spawn handler does when its container is opened.
type SpawnKind int

// Spawn kinds.
const (
	SpawnNone    SpawnKind = iota
	SpawnTrap              // a trap monster of Class is created on the container (handlers 1-4, 6)
	SpawnFire              // two "fire" objects (objects.txt 162) next to the container (handlers 5, 7)
	SpawnMonster           // one or two monsters of the level (handlers 8, 9)
)

// SpawnHandler is one row of the handler table at 0x730228.
type SpawnHandler struct {
	Kind SpawnKind
	// Class is the monstats id of the trap monster ("a trap" rows: 330 chain lightning, 326 firebolt, 329
	// poison, 369 nova).
	Class int
}

// spawnHandlers is VERIFIED from the table at 0x730228 and the four wrappers 0x580390..0x5803f0 (each loads the
// monster class and calls 0x580310): 1 -> 330, 2 -> 326, 3 -> 329, 4 -> 369, 5 -> fire pair, 6 -> 326,
// 7 -> fire pair, 8 and 9 -> level monsters.
var spawnHandlers = [...]SpawnHandler{
	{}, {SpawnTrap, 330}, {SpawnTrap, 326}, {SpawnTrap, 329}, {SpawnTrap, 369}, {SpawnFire, 0}, {SpawnTrap, 326},
	{SpawnFire, 0}, {SpawnMonster, 0}, {SpawnMonster, 0},
}

// HandlerFor returns the spawn handler of a rolled handler number (0 or out of range: none).
func HandlerFor(n int) SpawnHandler {
	if n <= 0 || n >= len(spawnHandlers) {
		return SpawnHandler{}
	}

	return spawnHandlers[n]
}

// SpawnDelayFrames is how long after the container opens its handler runs: OBJECT_ServerRunSpawnHandler
// schedules the object event 4 at +0x23 frames (0x580410).
const SpawnDelayFrames = 0x23

// LevelMonsterCount is how many monsters handler 8/9 creates: 1 + (seed & 1) (0x5801d0).
func LevelMonsterCount(r Rand) int { return 1 + r.Roll(2) }

// Container drop rules of OBJECT_OperateLootContainer (0x583e70), VERIFIED for the plain containers (every
// loot container except objects.txt row 397, whose tiered drop is not modelled):
//
//   - a locked container needs a key (INV_ConsumeOneKey 0x55cf90: the first item of type 0x29 "key" in the
//     inventory loses one of its quantity or is removed); without one nothing opens;
//   - the treasure class is rolled once, twice if it was locked;
//   - the roll happens only when Roll(100) > 24 (75%), always when the container was locked.
const (
	// EmptyChestThreshold: an unlocked container drops only when Roll(100) > 0x18, so about a quarter open empty.
	EmptyChestThreshold = 0x18
	// KeyItemType is the itemtypes.txt row of keys (0x29 = 41).
	KeyItemType = 0x29
	// GemItemType is the itemtypes.txt row of gems (0x14 = 20).
	GemItemType = 0x14
)

// ContainerDropRolls returns how many times the container's treasure class is rolled when it is opened.
func ContainerDropRolls(locked bool, r Rand) int {
	pass := r.Roll(100) > EmptyChestThreshold || locked
	if !pass {
		return 0
	}

	if locked {
		return 2
	}

	return 1
}

// Small container drops. VERIFIED: fn 3 (urns, 0x5845d0) and fn 5 (barrels, 0x5847b0) drop their chest
// treasure only when Roll(100) < 0x15 (21%). A normal barrel also creates a level monster when
// Roll(10000) >= 0x2000 (about 18%), after the hero's skill hit it.
const (
	SmallDropPercent   = 0x15
	BarrelMonsterFloor = 0x2000
)

// SmallContainerDrops reports whether an urn/barrel drops its treasure.
func SmallContainerDrops(r Rand) bool { return r.Roll(100) < SmallDropPercent }

// BarrelSpawnsMonster reports whether a barrel hit releases a level monster (0x5847b0).
func BarrelSpawnsMonster(r Rand) bool { return r.Roll(10000) >= BarrelMonsterFloor }

// ---------------------------------------------------------------------------------------------------------
// Magic shrines (table at 0x6e2b48)
// ---------------------------------------------------------------------------------------------------------

// StormLifeLoss is the Storm Shrine's life loss per unit (0x580cb0, VERIFIED): Arg0 percent of the CURRENT
// life, in whole points (the exe takes life >> 8 first), for every living monster and every living hero
// found within Arg1 (2000) subtiles of the shrine.
func StormLifeLoss(life, argPct int) int {
	if life <= 0 || argPct <= 0 {
		return 0
	}

	return argPct * life / 100
}

// MissileShot is a missile the object code creates: the offsets are relative to the shrine (Storm) or
// absolute targets around it (potion shrines), see the constructors.
type MissileShot struct {
	MissileID int // missiles.txt id
	DX, DY    int
}

// StormMissileID is missiles.txt row 62 (FireBall, VERIFIED id 0x3e); the shrines.txt text says "shoots
// fireballs".
const StormMissileID = 0x3e

// StormMissiles is the 4x4 grid of 0x580cb0: x and y each take 5, -10, 15, -20, x in the outer loop.
// The meaning of the two numbers (offsets from the shrine in subtiles) is UNVERIFIED.
func StormMissiles() []MissileShot {
	v := [4]int{5, -10, 15, -20}
	out := make([]MissileShot, 0, 16)

	for _, x := range v {
		for _, y := range v {
			out = append(out, MissileShot{MissileID: StormMissileID, DX: x, DY: y})
		}
	}

	return out
}

// MissileLevel is the level the shrine missiles are created with: the activator's level / 5 clamped to
// 1..8 (0x580cb0, 0x580ff0, 0x581320).
func MissileLevel(heroLevel int) int {
	n := heroLevel / 5
	if n < 1 {
		return 1
	}

	if n > 8 {
		return 8
	}

	return n
}

// PotionShrine describes the Exploding and the Poison shrine (0x580ff0 / 0x581320, VERIFIED).
type PotionShrine struct {
	ItemCode  string // misc.txt code of the dropped potion
	MissileID int
}

// The two potion shrines: "opm" (Exploding Potion) with missile 45 (ExplosivePotion), and "gpm" (Choking
// Gas Potion) with missile 48 (ChokingGasPotion).
var (
	ExplodingShrine = PotionShrine{ItemCode: "opm", MissileID: 0x2d}
	PoisonShrine    = PotionShrine{ItemCode: "gpm", MissileID: 0x30}
)

// PotionCount is how many potions the shrine puts on the ground around the activator: Arg0 + Roll(Arg1 -
// Arg0) (0x580ff0), so 5..9 for the shrines.txt rows 5/10. Each potion has quantity 1.
func PotionCount(arg0, arg1 int, r Rand) int {
	n := arg1 - arg0
	if n < 1 {
		n = 0
	}

	return arg0 + r.Roll(n)
}

// PotionMissiles are the six missiles the potion shrines fire, from the shrine to six points around it:
// (-6,6) (-6,-6) (0,6) (0,-6) (6,6) (6,-6) (0x580ff0), owner the activator.
func (p PotionShrine) PotionMissiles() []MissileShot {
	pts := [6][2]int{{-6, 6}, {-6, -6}, {0, 6}, {0, -6}, {6, 6}, {6, -6}}
	out := make([]MissileShot, 0, len(pts))

	for _, q := range pts {
		out = append(out, MissileShot{MissileID: p.MissileID, DX: q[0], DY: q[1]})
	}

	return out
}

// GemChoice is one inventory gem as the Gem Upgrade shrine sees it.
type GemChoice struct {
	Code       string // misc.txt code
	BetterCode string // misc.txt BetterGem, "" or "non" for none
}

// FreshGemCodes are the six chipped gems the shrine gives when no gem can be upgraded, in the order of the
// roll (0x580b50, VERIFIED): diamond, ruby, emerald, sapphire, topaz, amethyst.
var FreshGemCodes = [6]string{"gcw", "gcr", "gcg", "gcb", "gcy", "gcv"}

// GemUpgrade is OBJECT_ShrineGenerateItemsFromInventoryCodes (0x580b50, VERIFIED): the loose gems of the
// activator's inventory (item type 0x14, not socketed ones) are walked in inventory order; the first whose
// BetterGem is not "non" gives one new gem of the better code, dropped next to the activator, and the old gem
// STAYS in the inventory. When no gem qualifies, a random chipped gem is created instead. The returned code
// is the item to create.
func GemUpgrade(inv []GemChoice, r Rand) (code string, upgraded bool) {
	for _, g := range inv {
		if g.BetterCode != "" && g.BetterCode != "non" {
			return g.BetterCode, true
		}
	}

	return FreshGemCodes[r.Roll(len(FreshGemCodes))], false
}

// ---------------------------------------------------------------------------------------------------------
// Wells (0x5837b0, init 0x550ba0, refill event 0x57f410)
// ---------------------------------------------------------------------------------------------------------

// WellCharges is the number of pulses a fresh well holds: Parm2 * 2 (init 0x550ba0, VERIFIED; Parm2 is 1 for
// every well of 1.14b, so 2).
func WellCharges(parm2 int) int { return parm2 * 2 }

// WellMode is the object mode after the charge counter becomes c (0x5837b0 and 0x57f410): the mode changes
// only when c is a multiple of Parm2 (c <= 2*Parm2), and is 2 - c/Parm2: 0 full, 1 half, 2 empty. It reports
// ok=false when the mode does not change at that count.
func WellMode(c, parm2 int) (mode int, ok bool) {
	if parm2 < 1 || c < 0 || c > WellCharges(parm2) || c%parm2 != 0 {
		return 0, false
	}

	return 2 - c/parm2, true
}

// WellRefillFrames is the delay of the refill event scheduled after every pulse that did something:
// Parm0 + 1 frames (0x5837b0). Each event returns one charge (0x57f410).
func WellRefillFrames(parm0 int) int { return parm0 + 1 }

// ---------------------------------------------------------------------------------------------------------
// Weapon racks and armor stands (OperateFn 20 / 19, 0x582050 / 0x581fe0 -> 0x557610 / 0x5574c0 -> 0x5540b0 / 0x553f60)
// ---------------------------------------------------------------------------------------------------------

// A rack does not roll a treasure class. VERIFIED: it picks ONE random base item of the weapons.txt (rack) or
// armor.txt (stand) table, uniformly among the eligible rows, and creates it at the item level
// monsterLevel-1 (not below 1) with the creation code choosing the quality, then the object goes to mode 2.

// RackBase is a row of armor.txt or weapons.txt as the picker (0x553ef0) sees it.
type RackBase struct {
	Code      string
	QLvl      int  // the "level" column (record byte +0xfd)
	Rarity    int  // record byte +0xfc
	Spawnable bool // byte +0x133
	Quest     bool // byte +0x12a
	Expansion bool // version >= 100 (word +0xf6)
}

// RackItemLevel is the item level of a rack's item: the area's monster level minus one, not below 1
// (0x5574c0 / 0x557610).
func RackItemLevel(monLvl int) int {
	if monLvl > 1 {
		return monLvl - 1
	}

	return monLvl
}

// ActOfLevelID is DRLG_GetActFromLevelId: the act (1..5) of a levels.txt id.
func ActOfLevelID(id int) int {
	switch {
	case id >= 109:
		return 5
	case id >= 103:
		return 4
	case id >= 75:
		return 3
	case id >= 40:
		return 2
	}

	return 1
}

// RackMaxCandidates is the size of the candidate list in 0x553f60.
const RackMaxCandidates = 0x3ff

// RackEligible follows ITEMGEN_IsBaseItemEligible (0x553ef0) and the version test of the picker: the row
// must be spawnable, not a quest item and have a level not above the item level (at least 1); then, unless
// forced, a row whose rarity exceeds the act by d is kept only when Roll(d) is 0 - the exe passes the ITEM
// LEVEL to DRLG_GetActFromLevelId there (a quirk kept here); a classic game excludes expansion rows.
func RackEligible(b RackBase, ilvl int, classic bool, r Rand) bool {
	if ilvl < 1 {
		ilvl = 1
	}

	if !b.Spawnable || b.Quest || b.QLvl > ilvl {
		return false
	}

	if d := b.Rarity - ActOfLevelID(ilvl); d > 0 && r.Roll(d) != 0 {
		return false
	}

	return !classic || !b.Expansion
}

// PickRackBase draws the rack item: the eligible rows in table order (at most RackMaxCandidates), then one
// uniform pick. ok is false when nothing is eligible. rows must be in the table's file order; the engine
// has to sort them when its records are a map (UNVERIFIED: that order changes which row a seed picks, not
// the odds).
func PickRackBase(rows []RackBase, ilvl int, classic bool, r Rand) (code string, ok bool) {
	var cand []string

	for _, b := range rows {
		if len(cand) < RackMaxCandidates && RackEligible(b, ilvl, classic, r) {
			cand = append(cand, b.Code)
		}
	}

	if len(cand) == 0 {
		return "", false
	}

	return cand[r.Roll(len(cand))], true
}
