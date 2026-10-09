package d2statlist

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
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
	Str, Dex, Vit, Ene int // with item bonuses

	MaxLife, MaxMana, MaxStamina int

	Defense      int
	AttackRating int
	DamageMin    int // physical damage of the main hand (or bare hands)
	DamageMax    int
	BlockPct     int // block chance with shield and dexterity, capped at 75
	Resist       [4]int
	MaxResist    [4]int // the cap (75 plus max resist bonuses, at most 95)
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
	LifeSteal      int
	ManaSteal      int

	FasterAttack, FasterCast, FasterHit, FasterBlock, FasterRun int
	MagicFind, GoldFind                                         int
	AllSkills                                                   int

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
	setCount := map[int]int{}

	for _, it := range active {
		if it.SetID != 0 {
			setCount[it.SetID]++
		}
	}

	list := NewList()
	armor := 0

	for i := range active {
		it := &active[i]

		armor += it.DefenseOf(h.Level)

		for _, p := range it.ownProps() {
			if !itemSpecific(p.ID) {
				list.Add(p.ID, p.Param, p.Value)
			}
		}

		for tier, props := range it.SetLists {
			if setCount[it.SetID] >= tier+2 {
				list.AddProps(props)
			}
		}

		for _, p := range env.gemProps(it) {
			list.Add(p.ID, p.Param, p.Value)
		}
	}

	if env.Sets != nil {
		for id, n := range setCount {
			list.AddProps(env.Sets.SetBonus(id, n))
		}
	}

	// per-level stats fold into their targets
	for _, pl := range perLevelTable {
		if v := list.Get(pl.stat); v != 0 {
			list.Add(pl.target, 0, v*int64(h.Level)>>uint(pl.shift))
		}
	}

	t := Totals{Stats: list}
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
	t.MaxLife, t.MaxMana, t.MaxStamina = int(life>>8), int(mana>>8), int(stam>>8)

	t.Defense = d2combat.Defense(armor, t.Dex, 0)

	toHit := int(list.Get(StatToHit))
	t.AttackRating = d2combat.PlayerAttackRating(toHit, t.Dex, h.Class.ToHitFactor)
	t.AttackRating += t.AttackRating * int(list.Get(StatToHitPct)) / 100

	t.DamageMin, t.DamageMax = weaponDamage(active, list, t.Str, t.Dex)

	shield := shieldOf(active)
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
	t.LifeSteal = int(list.Get(StatLifeSteal))
	t.ManaSteal = int(list.Get(StatManaSteal))
	t.FasterAttack = int(list.Get(StatFasterAttack))
	t.FasterCast = int(list.Get(StatFasterCast))
	t.FasterHit = int(list.Get(StatFasterHit))
	t.FasterBlock = int(list.Get(StatFasterBlock))
	t.FasterRun = int(list.Get(StatFasterMove))
	t.MagicFind = int(list.Get(StatMagicFind))
	t.GoldFind = int(list.Get(StatGoldFind))
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
	var w *Item

	for i := range active {
		if active[i].Slot == SlotRightHand && active[i].Weapon != nil {
			w = &active[i]
		}
	}

	baseMin, baseMax, strB, dexB := 1, 2, 0, 0
	edMin, edMax := int64(0), int64(0)

	if w != nil {
		wb := w.Weapon
		baseMin, baseMax = wb.Min, wb.Max

		if wb.TwoHanded && wb.TwoMax > 0 {
			baseMin, baseMax = wb.TwoMin, wb.TwoMax
		}

		if w.Ethereal {
			baseMin, baseMax = EtherealDamage(baseMin), EtherealDamage(baseMax)
		}

		strB, dexB = wb.StrBonus, wb.DexBonus
		props := w.ownProps()
		edMin, edMax = sumID(props, StatMinDmgPct), sumID(props, StatMaxDmgPct)
	}

	bonus := int64(str*strB/100+dex*dexB/100) + list.Get(StatDamagePct)
	min = int((int64(baseMin)*(100+edMin)/100 + list.Get(StatMinDamage)) * (100 + bonus) / 100)
	max = int((int64(baseMax)*(100+edMax)/100 + list.Get(StatMaxDamage)) * (100 + bonus) / 100)

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
