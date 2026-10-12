package d2statlist

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
)

// Stat ids skills use (ItemStatCost.txt, verified by the D2_TABLES oracle test).
const (
	StatSkillStaminaPct     = 162 // skill_staminapercent, op 1 on maxstamina
	StatSkillPassiveStamPct = 163 // skill_passive_staminapercent, op 1 on maxstamina
	StatSkillArmorPct       = 171 // skill_armor_percent, defense percent from skills
	StatArmorOverridePct    = 182 // armor_override_percent, last term of GetDefense (0x6225a0)
)

// Class is the part of charstats.txt the derivations use. The "per" values
// are in quarters (8 means 2 per point) exactly as in the table.
type Class struct {
	ID int // 0 Amazon, 1 Sorceress, 2 Necromancer, 3 Paladin, 4 Barbarian, 5 Druid, 6 Assassin
	InitStr, InitDex, InitVit, InitEne,
	InitStamina int
	HpAdd           int // charstats "hpadd"
	LifePerVit      int
	ManaPerEne      int
	StaminaPerVit   int
	LifePerLevel    int
	ManaPerLevel    int
	StaminaPerLevel int
	ToHitFactor     int
	BlockFactor     int
}

// BaseMax returns the maximum life, mana and stamina a character has from
// its class, level and allocated attributes alone, before any item:
//
//	life    = hpadd + initVit + (level-1)*lifePerLevel/4 + (vit-initVit)*lifePerVit/4
//	mana    = initEne         + (level-1)*manaPerLevel/4 + (ene-initEne)*manaPerEne/4
//	stamina = initStamina     + (level-1)*stamPerLevel/4 + (vit-initVit)*stamPerVit/4
//
// Verified against the real level 94 Sorceress save (mana 221 and stamina 525
// match exactly; life matches plus the 20 life of the Act 3 Potion of Life
// quest reward, which the save adds to the stored value), and against the
// starting life of every class (hpadd + initVit = 40 for a Sorceress).
func (c Class) BaseMax(level, vit, ene int) (life, mana, stamina int) {
	if level < 1 {
		level = 1
	}

	life = c.HpAdd + c.InitVit + ((level-1)*c.LifePerLevel+(vit-c.InitVit)*c.LifePerVit)/4
	mana = c.InitEne + ((level-1)*c.ManaPerLevel+(ene-c.InitEne)*c.ManaPerEne)/4
	stamina = c.InitStamina + ((level-1)*c.StaminaPerLevel+(vit-c.InitVit)*c.StaminaPerVit)/4

	return life, mana, stamina
}

// Hero is the hero without its items.
type Hero struct {
	Class      Class
	Level      int
	Str, Dex   int // allocated attributes, as stored in the save (no items)
	Vit, Ene   int
	BaseLife   int // stored maximum life without items
	BaseMana   int
	BaseStam   int
	Difficulty int // 0 normal, 1 nightmare, 2 hell
	// Classic selects the Diablo II (non expansion) resist penalties -20/-50
	// instead of the Lord of Destruction -40/-100.
	Classic bool
}

// Totals is everything derived from the hero and its equipment.
type Totals struct {
	// SetPieces counts the worn pieces per set (1-based Sets.txt row).
	SetPieces map[int]int

	Str, Dex, Vit, Ene int // with item bonuses

	MaxLife, MaxMana, MaxStamina int

	Defense      int
	AttackRating int
	// AttackRatingFlat is AttackRating before the percent (stat 0x77): COMBAT_RollToHit (0x57b9c0) adds that
	// percent itself, so the skill engine takes this and the percent apart.
	AttackRatingFlat int
	DamageMin        int // physical damage of the main hand (or bare hands)
	DamageMax        int
	// Weapon is the percent free weapon numbers for the skill descriptor (see WeaponParts).
	Weapon WeaponParts
	// GearDamagePct and GearToHitPct are stats 25 and 119 from the items only (the skill engine's own states
	// supply the skills' share).
	GearDamagePct, GearToHitPct int
	// ShieldWorn is set when the left hand holds a shield (item type 0x33, the clause of the defense colour rule).
	ShieldWorn bool
	BlockPct   int // block chance with shield and dexterity, capped at 75
	Resist     [4]int
	MaxResist  [4]int // the cap (75 plus max resist bonuses, at most 95)
	// ResistShown is Resist after the difficulty penalty and the cap, which
	// is what the character panel prints.
	ResistShown [4]int
	PhysResist  int // damageresist (cap 50)
	MagicResist int
	// DamageReduction and MagicReduction are the flat "damage reduced by N"
	// stats (34, 35).
	DamageReduction int
	MagicReduction  int

	DeadlyStrike   int // percent
	CriticalStrike int
	CrushingBlow   int
	OpenWounds     int
	SlowTarget     int // item_slow (150): slows the target of a hit by N percent

	// DamageToMana is item_damagetomana (114): percent of the damage a hit did
	// to the wearer that is paid back as mana (event func 13, 0x5bd7c0).
	DamageToMana int
	// AttackerTakes and AttackerTakesLight are item_attackertakesdamage (78)
	// and item_attackertakeslightdamage (128): flat physical / lightning
	// damage dealt back to a melee attacker (event funcs 6 and 10).
	AttackerTakes, AttackerTakesLight int
	// AbsorbPct and AbsorbFlat are the item absorb stats (142..149) per
	// element in the order fire, lightning, magic, cold.
	AbsorbPct, AbsorbFlat [4]int
	LifeSteal             int
	ManaSteal             int

	FasterAttack, FasterCast, FasterHit, FasterBlock, FasterRun int
	// ArmorSpeed is the sum of the armor.txt "speed" column over the worn armor pieces (body armor 5 or 10, large
	// shields 5 or 10, the rest 0). The exe puts -speed on each piece as stat 0x43 (ITEMGEN_InitItemBaseStats
	// 0x555b10, VERIFIED) and adds the stat to the walk/run percent, so the movement percent is
	// 100 - ArmorSpeed (+50 running) + the diminished faster run/walk.
	ArmorSpeed int
	// BodyArmorSpeed is the speed column of the body armor (slot torso, 0 when none): PLAYER_CalcStaminaDrainPerStep
	// (0x57d220, VERIFIED) multiplies the run drain by BodyArmorSpeed/10 + 1, so a heavy body armor (speed 10)
	// doubles the stamina drain. StaminaDrainPct is stat 0x9a (item_staminadrainpct), read by the same function.
	BodyArmorSpeed  int
	StaminaDrainPct int
	// StaminaRecoveryPct is stat 0x1c (staminarecoverybonus): PLAYER_ServerRegenStaminaTick (0x57e4f0, VERIFIED)
	// adds that percent to the regeneration step, and at 1000 or more lets a running unit regenerate.
	StaminaRecoveryPct int
	// WeaponSpeed is the weapon speed modifier (weapons.txt "speed") of the weapon in the right hand: stat 0x44 of
	// the attack rate is 100 - WeaponSpeed + skill speed, before the diminished IAS is added (VERIFIED, 0x624150).
	WeaponSpeed         int
	MagicFind, GoldFind int
	AllSkills           int

	// Natural regeneration inputs (see d2herostats/regen.go): ManaRecoveryPct
	// is stat 27 (regenerate mana %, items and the mana shrine), ManaRecoveryRaw
	// stat 26 (raw 1/256 mana per frame) and LifeRegen stat 74 (replenish
	// life, raw 1/256 life per frame).
	ManaRecoveryPct int
	ManaRecoveryRaw int
	LifeRegen       int

	// Stats is the summed stat list of all active items (without the
	// item-specific enhanced defense/damage percent).
	Stats *List `json:"-"`
}

// Index of Resist: fire, cold, lightning, poison.
const (
	ResFire = iota
	ResCold
	ResLight
	ResPoison
)

var resistStats = [4][2]int{
	ResFire:   {StatFireResist, StatMaxFireRes},
	ResCold:   {StatColdResist, StatMaxColdRes},
	ResLight:  {StatLightResist, StatMaxLightRes},
	ResPoison: {StatPoisonResist, StatMaxPoisonRes},
}

// perLevelTable lists the per-level stats (ItemStatCost op 2, op param 1 or
// 3): stat id -> (target stat, shift). The armor and damage entries are
// handled with their item. Checked against the real ItemStatCost by the
// optional D2_TABLES test.
var perLevelTable = []struct{ stat, target, shift int }{
	{216, StatMaxHP, 3}, {217, StatMaxMana, 3},
	{220, StatStrength, 3}, {221, StatDexterity, 3}, {222, StatEnergy, 3}, {223, StatVitality, 3},
	{224, StatToHit, 1}, {225, StatToHitPct, 1},
	{226, StatColdMax, 3}, {227, StatFireMax, 3}, {228, StatLightMax, 3}, {229, StatPoisonMax, 3},
	{230, StatColdResist, 3}, {231, StatFireResist, 3}, {232, StatLightResist, 3}, {233, StatPoisonResist, 3},
	{238, 78, 3}, {239, StatGoldFind, 3}, {240, StatMagicFind, 3}, {241, StatStamRecovery, 3},
	{242, StatMaxStamina, 3}, {247, StatCrushing, 3}, {248, StatOpenWounds, 3}, {249, 137, 3},
	{250, StatDeadlyStrike, 3},
}

// itemSpecific stats never reach the hero's list: they act on their item.
func itemSpecific(id int) bool {
	switch id {
	case StatArmorPct, StatMaxDmgPct, StatMinDmgPct, StatArmorClass,
		statArmorPerLevel, statArmorPctPerLvl, 218, 219:
		return true
	}

	return false
}

// Env carries optional tables.
type Env struct {
	Gems GemTable
	Sets SetTable

	// Skill is what the hero's skills put on its stat list: passive skills
	// (passivestat1..5), the active aura and timed buffs (aurastat1..6), in
	// ItemStatCost ids with the calc already evaluated. It is merged into the
	// same list as the item stats (all stats are summed per (id, parameter);
	// the percent stats 76/77/162/163/171 are summed first and applied once).
	//
	// Verified: aurastat1..6 are stored into the buff's statlist in column
	// order, zero values skipped (0x5c4c60), and the unit's stat is the sum of
	// its lists; percent stats are read as one summed stat (GetDefense adds 171
	// and 16 before the multiply, 0x6225a0). The passive path of true passives
	// (Iron Skin, Resist Fire...) is NOT located in the exe: 0x5c4d70 is only
	// called from the buff casts, so callers must not pass passives (U).
	Skill *List
}

// Compute aggregates the active items onto the hero and derives the totals.
// items may contain the whole inventory: only equipped slots 1-10 and
// charms are active (weapon switch slots are not).
func Compute(h Hero, items []Item, env *Env) Totals {
	if env == nil {
		env = &Env{}
	}

	if h.Level < 1 {
		h.Level = 1
	}

	active := activeItems(items)
	setCount, setPieces := countSetPieces(active, env.Sets)

	// the stat lists form the exe's tree: the hero's list with one list per
	// worn item, per set bonus tier and per gem group attached under it (tree.go)
	root := NewTree(Owner{Type: 0, ID: 1}, FlagBaseUnit)
	armor := 0

	attach := func(flags ListFlags, props []Prop) {
		if len(props) == 0 {
			return
		}

		node := NewTree(Owner{Type: 4}, flags)
		node.AddProps(props)
		node.Attach(root, itemSpecific)
	}

	for i := range active {
		it := &active[i]

		armor += it.DefenseOf(h.Level)

		attach(0, it.ownProps())

		for tier, props := range it.SetLists {
			if setListActive(it, tier, setCount[it.SetID], setPieces[it.SetID], env.Sets) {
				attach(0, props) // active tier: summed like an item list (a 0x2000 list in the exe is kept apart and adds nothing on attach)
			}
		}

		attach(0, env.gemProps(it))
	}

	if env.Sets != nil {
		ids := make([]int, 0, len(setCount))
		for id := range setCount {
			ids = append(ids, id)
		}

		sort.Ints(ids)

		for _, id := range ids {
			attach(0, env.Sets.SetBonus(id, setCount[id]))
		}
	}

	list := root.List()

	// the item only part of two percent stats: the skill engine adds the skills' own part itself
	gear := NewList()
	gear.Merge(list)

	// skill sourced stats (passives, aura, buffs) join the item stats
	list.Merge(env.Skill)

	// per-level stats fold into their targets
	for _, pl := range perLevelTable {
		if v := list.Get(pl.stat); v != 0 {
			list.Add(pl.target, 0, v*int64(h.Level)>>uint(pl.shift))
		}
	}

	t := Totals{Stats: list, SetPieces: setCount}
	t.Str = h.Str + int(list.Get(StatStrength))
	t.Dex = h.Dex + int(list.Get(StatDexterity))
	t.Vit = h.Vit + int(list.Get(StatVitality))
	t.Ene = h.Ene + int(list.Get(StatEnergy))

	// life, mana and stamina in 8.8 fixed point so a quarter point survives
	// until the percent bonus is applied (op 9/8: stat bonus * per/4, op 11: percent)
	vitB, eneB := int64(t.Vit-h.Vit), int64(t.Ene-h.Ene)
	life := int64(h.BaseLife+int(list.Get(StatMaxHP)))<<8 + vitB*int64(h.Class.LifePerVit)*64
	mana := int64(h.BaseMana+int(list.Get(StatMaxMana)))<<8 + eneB*int64(h.Class.ManaPerEne)*64
	stam := int64(h.BaseStam+int(list.Get(StatMaxStamina)))<<8 + vitB*int64(h.Class.StaminaPerVit)*64

	life = life * (100 + list.Get(StatMaxHPPct)) / 100
	mana = mana * (100 + list.Get(StatMaxManaPct)) / 100
	// skill_staminapercent (162, Sanctuary-like, Battle Orders) and
	// skill_passive_staminapercent (163, Vigor): ItemStatCost op 1 on maxstamina.
	stam = stam * (100 + list.Get(StatSkillStaminaPct) + list.Get(StatSkillPassiveStamPct)) / 100
	t.MaxLife, t.MaxMana, t.MaxStamina = int(life>>8), int(mana>>8), int(stam>>8)

	// GetDefense (0x6225a0, verified): the percent is stat 171
	// (skill_armor_percent: Iron Skin, Frozen Armor, Shout, Defiance...) plus
	// stat 16, which here is already inside the item defense.
	t.Defense = d2combat.DefenseOverride(armor, t.Dex, int(list.Get(StatSkillArmorPct)), int(list.Get(StatArmorOverridePct)))

	toHit := int(list.Get(StatToHit))
	t.AttackRating = d2combat.PlayerAttackRating(toHit, t.Dex, h.Class.ToHitFactor)
	t.AttackRatingFlat = t.AttackRating
	t.AttackRating += t.AttackRating * int(list.Get(StatToHitPct)) / 100

	t.DamageMin, t.DamageMax = weaponDamageClass(active, list, h.Class.ID, t.Str, t.Dex)
	t.Weapon = weaponParts(active, list, h.Class.ID, t.Str, t.Dex)
	t.GearDamagePct, t.GearToHitPct = int(gear.Get(StatDamagePct)), int(gear.Get(StatToHitPct))

	shield := shieldOf(active)

	for i := range active {
		if active[i].Slot == SlotLeftHand && active[i].Weapon == nil && !active[i].Charm {
			t.ShieldWorn = true
		}
	}
	t.BlockPct = d2combat.PlayerBlockChance(d2combat.PlayerBlockInput{
		HasShield: shield != nil, ToBlock: shieldBlock(shield) + int(list.Get(StatToBlock)),
		ClassBlockFactor: h.Class.BlockFactor, Dex: t.Dex, Level: h.Level, IncludeDex: true,
	})

	penalty := d2combat.LoDResistPenalty(h.Difficulty)
	if h.Classic {
		penalty = d2combat.ClassicResistPenalty(h.Difficulty)
	}

	for i, ids := range resistStats {
		t.Resist[i] = int(list.Get(ids[0]))
		t.MaxResist[i] = d2combat.DefaultMaxResist + int(list.Get(ids[1]))

		if t.MaxResist[i] > d2combat.MaxResistCeiling {
			t.MaxResist[i] = d2combat.MaxResistCeiling
		}

		t.ResistShown[i] = d2combat.EffectiveResist(d2combat.ResistInput{
			Resist: t.Resist[i], MaxResistBonus: int(list.Get(ids[1])), HasMaxResist: true,
			DifficultyPenalty: penalty,
		})
	}

	t.PhysResist = d2combat.EffectiveResist(d2combat.ResistInput{
		Resist: int(list.Get(StatDamageResist)), IsPhysical: true, NoDifficultyPenalty: true,
	})
	t.MagicResist = d2combat.EffectiveResist(d2combat.ResistInput{
		Resist: int(list.Get(StatMagicResist)), MaxResistBonus: int(list.Get(StatMaxMagicRes)),
		HasMaxResist: true, NoDifficultyPenalty: true,
	})

	t.DamageReduction = int(list.Get(StatNormalReduce))
	t.MagicReduction = int(list.Get(StatMagicReduce))
	t.DeadlyStrike = int(list.Get(StatDeadlyStrike))
	t.CriticalStrike = int(list.Get(StatCritical))
	t.CrushingBlow = int(list.Get(StatCrushing))
	t.OpenWounds = int(list.Get(StatOpenWounds))
	t.SlowTarget = int(list.Get(StatSlowTarget))
	t.DamageToMana = int(list.Get(StatDamageToMana))
	t.AttackerTakes = int(list.Get(StatAttackerTakes))
	t.AttackerTakesLight = int(list.Get(StatAttackerTakesLight))

	for i := 0; i < 4; i++ {
		t.AbsorbPct[i] = int(list.Get(StatAbsorbFirePct + 2*i))
		t.AbsorbFlat[i] = int(list.Get(StatAbsorbFirePct + 2*i + 1))
	}
	t.LifeSteal = int(list.Get(StatLifeSteal))
	t.ManaSteal = int(list.Get(StatManaSteal))
	t.FasterAttack = int(list.Get(StatFasterAttack))
	t.FasterCast = int(list.Get(StatFasterCast))
	t.FasterHit = int(list.Get(StatFasterHit))
	t.FasterBlock = int(list.Get(StatFasterBlock))
	t.FasterRun = int(list.Get(StatFasterMove))
	t.StaminaDrainPct = int(list.Get(StatStaminaDrain))
	t.StaminaRecoveryPct = int(list.Get(StatStamRecovery))

	for i := range active {
		if active[i].Charm {
			continue
		}

		switch {
		case active[i].Weapon == nil:
			t.ArmorSpeed += active[i].BaseSpeed

			if active[i].Slot == SlotTorso {
				t.BodyArmorSpeed = active[i].BaseSpeed
			}
		case active[i].Slot == SlotRightHand:
			t.WeaponSpeed = active[i].BaseSpeed
		}
	}
	t.MagicFind = int(list.Get(StatMagicFind))
	t.GoldFind = int(list.Get(StatGoldFind))
	t.ManaRecoveryPct = int(list.Get(StatManaRecovery))
	t.ManaRecoveryRaw = int(list.Get(StatManaRecRaw))
	t.LifeRegen = int(list.Get(StatHPRegen))
	t.AllSkills = int(list.Get(StatAllSkills))

	return t
}

// SkillBonus returns the +skills the equipment gives to the skills of tab
// (0..2) of the hero's class: +all skills, +class skills and +tab skills.
// The tab stat 188 carries class*8+tab as its parameter (verified in the
// real save: a Sorceress "+1 fire skills" charm has parameter 8).
func (t *Totals) SkillBonus(class, tab int) int {
	return int(t.Stats.Get(StatAllSkills) + t.Stats.GetParam(StatClassSkills, class) +
		t.Stats.GetParam(StatSkillTab, class*8+tab))
}

// SingleSkillBonus returns the bonus to one skill id (stat 107 and 97
// carry the skill id as parameter).
func (t *Totals) SingleSkillBonus(skillID int) int {
	return int(t.Stats.GetParam(StatSingleSkill, skillID) + t.Stats.GetParam(StatNonClass, skillID))
}

// CombinedCritical is the chance (percent) that a hit is critical when deadly
// strike and critical strike roll independently (both double physical damage;
// verified OR structure in COMBAT_BuildAttackerDamage).
func (t *Totals) CombinedCritical() int {
	return 100 - (100-clampPct(t.DeadlyStrike))*(100-clampPct(t.CriticalStrike))/100
}

func clampPct(v int) int {
	if v < 0 {
		return 0
	}

	if v > 100 {
		return 100
	}

	return v
}

func activeItems(items []Item) []Item {
	var out []Item

	for _, it := range items {
		if it.Broken {
			continue
		}

		if it.Charm || (it.Slot >= SlotHead && it.Slot <= SlotGloves) {
			out = append(out, it)
		}
	}

	return out
}

func shieldOf(active []Item) *Item {
	for i := range active {
		if active[i].Slot == SlotLeftHand && active[i].BaseBlock > 0 {
			return &active[i]
		}
	}

	return nil
}

func shieldBlock(s *Item) int {
	if s == nil {
		return 0
	}

	return s.BaseBlock
}

// weaponDamage computes the physical damage of the main hand:
//
//	min = (baseMin*(100+EDmin)/100 + addedMin) * (100 + bonus)/100
//
// where bonus is strength*strbonus/100 + dexterity*dexbonus/100 plus the
// damage percent stat (25). Community formula (UNVERIFIED in the binary:
// the roll helper's operands were hidden in the decompile); the floor
// min>=1, max>=2 is verified (COMBAT_RollPhysicalDamage).
func weaponDamage(active []Item, list *List, str, dex int) (min, max int) {
	return weaponDamageClass(active, list, -1, str, dex)
}

// WeaponParts are the weapon numbers the skill engine needs apart from the finished range of Totals.DamageMin /
// DamageMax: the unit damage stats 0x15/0x16 (0x9f/0xa0 for a throwing weapon) BEFORE the percent pool of
// MISSILE_BuildDamageDescriptor (0x64ca60, VERIFIED wave 24) and that pool's weapon part. The descriptor scales
// these stats by the skill's SrcDam and adds stat 0x19, the weapon mastery and the weapon's strength / dexterity
// bonus to ONE percent, so the range of DamageMin / DamageMax (which already holds that percent) would count it twice.
type WeaponParts struct {
	Armed bool // a weapon sits in the right hand
	// Min and Max are base damage (two handed range when it is used two handed) raised by the weapon's own
	// enhanced damage (stats 17/18 of its properties), ethereal included, plus the added damage stats 21/22.
	Min, Max int
	// ThrowMin and ThrowMax are the same for the thrown range (weapons.txt minmisdam / maxmisdam, stats 0x9f/0xa0);
	// Throwing is set for a weapon that has one. UNVERIFIED: the added damage 21/22 and the enhanced damage are
	// applied to the thrown range like to the melee range.
	ThrowMin, ThrowMax int
	Throwing           bool
	// BonusPct is the weapon's strength and dexterity bonus in percent (ITEM_GetStrBonus / ITEM_GetDexBonus
	// times the attribute, divided by 100, 0x64ca60).
	BonusPct int
	// Flat is the weapon's own item_normaldamage (stat 0x6f, the "+N to damage" property dmg): added after the
	// percent pool in the character screen box (0x4e5860 reads STATS_GetUnitFullStat of the weapon item) and before
	// it in the hit roll (0x579120). VERIFIED wave 45; the skill engine does not use it yet.
	Flat int
}

func weaponDamageClass(active []Item, list *List, class, str, dex int) (min, max int) {
	wp := weaponParts(active, list, class, str, dex)
	bonus := int64(wp.BonusPct) + list.Get(StatDamagePct)

	if !wp.Armed {
		bonus += int64(str) // bare hands: strength counts as percent (VERIFIED 0x579120)
	}

	if bonus < -90 { // VERIFIED 0x579120 and 0x4e5860: the percent is floored at -90
		bonus = -90
	}

	// The character screen box (SKILLDESC_ComputeWeaponDamageRange 0x4e5860, VERIFIED): the unit range is floored at
	// 1 / 2 BEFORE the percent pool, then
	//	value = base * (100 + pool) / 100 + item_normaldamage of the weapon
	// (the flat "+N to damage" is added after the percent in this description path; the hit roll 0x579120 adds it
	// before the percent).
	baseMin, baseMax := int64(wp.Min), int64(wp.Max)
	if baseMin < 1 {
		baseMin = 1
	}

	if baseMax < 2 {
		baseMax = 2
	}

	min = int(baseMin*(100+bonus)/100) + wp.Flat
	max = int(baseMax*(100+bonus)/100) + wp.Flat

	if min < 1 {
		min = 1
	}

	if max < 2 {
		max = 2
	}

	if max < min {
		max = min
	}

	return min, max
}

// weaponParts is the percent free part of weaponDamage. class is the hero class id (4 = Barbarian) or -1.
// INV_ClassifyHandWeaponMode (0x63e3e0, VERIFIED wave 24) picks the two handed range (stats 0x17/0x18): a two
// handed weapon always, a weapon of the 1or2handed column only for a Barbarian (INV_PlayerCanUseOneOrTwoHanded
// 0x62a350) with the other hand empty; with both hands filled such a weapon is used one handed.
func weaponParts(active []Item, list *List, class, str, dex int) WeaponParts {
	var w *Item

	hasLeft := false

	for i := range active {
		if active[i].Slot == SlotRightHand && active[i].Weapon != nil {
			w = &active[i]
		}

		if active[i].Slot == SlotLeftHand {
			hasLeft = true
		}
	}

	baseMin, baseMax := 1, 2
	edMin, edMax := int64(0), int64(0)

	var wp WeaponParts

	if w != nil {
		wb := w.Weapon
		wp.Armed = true
		baseMin, baseMax = wb.Min, wb.Max

		two := wb.TwoHanded
		if class == 4 && wb.OneOrTwo {
			two = !hasLeft
		}

		if two && wb.TwoMax > 0 {
			baseMin, baseMax = wb.TwoMin, wb.TwoMax
		}

		thrMin, thrMax := wb.MisMin, wb.MisMax

		if w.Ethereal {
			baseMin, baseMax = EtherealDamage(baseMin), EtherealDamage(baseMax)
			thrMin, thrMax = EtherealDamage(thrMin), EtherealDamage(thrMax)
		}

		wp.BonusPct = str*wb.StrBonus/100 + dex*wb.DexBonus/100
		props := w.ownProps()
		edMin, edMax = sumID(props, StatMinDmgPct), sumID(props, StatMaxDmgPct)
		wp.Flat = int(sumID(props, StatNormalDamage))

		if wb.MisMax > 0 {
			wp.Throwing = true
			wp.ThrowMin = int(int64(thrMin)*(100+edMin)/100 + list.Get(StatMinDamage))
			wp.ThrowMax = int(int64(thrMax)*(100+edMax)/100 + list.Get(StatMaxDamage))
		}
	}

	wp.Min = int(int64(baseMin)*(100+edMin)/100 + list.Get(StatMinDamage))
	wp.Max = int(int64(baseMax)*(100+edMax)/100 + list.Get(StatMaxDamage))

	return wp
}

// pieceTable is optionally implemented by a SetTable that knows the members
// of its sets (SetDefs does).
type pieceTable interface {
	pieceInfo(setID, row int) (idx, addFunc int, ok bool)
}

// countSetPieces is INV_CountEquippedSetMates (62a4e0): per set, the number of
// DISTINCT worn pieces (the game ORs 1<<index of each piece). Pieces whose
// index is unknown count once per item. The mask is also returned (bit =
// position in the set) for the sets whose members are known.
func countSetPieces(active []Item, tab SetTable) (count map[int]int, mask map[int]uint32) {
	count = map[int]int{}
	mask = map[int]uint32{}
	seen := map[[2]int]bool{}
	pt, _ := tab.(pieceTable)

	for i := range active {
		it := &active[i]
		if it.SetID == 0 {
			continue
		}

		if it.SetPiece > 0 {
			key := [2]int{it.SetID, it.SetPiece}
			if seen[key] {
				continue
			}

			seen[key] = true

			if pt != nil {
				if idx, _, ok := pt.pieceInfo(it.SetID, it.SetPiece-1); ok && idx < 32 {
					mask[it.SetID] |= 1 << uint(idx)
				}
			}
		}

		count[it.SetID]++
	}

	return count, mask
}

// setListActive says whether tier list `tier` of a set item is attached.
// VERIFIED (INV_ApplySetItemStateList 6663c0): add func 2 attaches the first
// (distinct pieces - 1) lists; add func 1 attaches list k when the k-th OTHER
// piece of the set is worn (the item's own position is skipped).
func setListActive(it *Item, tier, count int, mask uint32, tab SetTable) bool {
	if pt, ok := tab.(pieceTable); ok && it.SetPiece > 0 {
		if own, addFunc, found := pt.pieceInfo(it.SetID, it.SetPiece-1); found && addFunc == 1 {
			m := tier
			if m >= own {
				m++
			}

			return m < 32 && mask&(1<<uint(m)) != 0
		}
	}

	return count >= tier+2
}
