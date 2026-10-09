package d2statlist

// Equipment slots of a .d2s file (Item.Equipped).
const (
	SlotHead      = 1
	SlotAmulet    = 2
	SlotTorso     = 3
	SlotRightHand = 4
	SlotLeftHand  = 5
	SlotRingRight = 6
	SlotRingLeft  = 7
	SlotBelt      = 8
	SlotFeet      = 9
	SlotGloves    = 10
	SlotSwapRight = 11 // weapon switch, inactive
	SlotSwapLeft  = 12
)

const (
	statArmorPerLevel  = 214
	statArmorPctPerLvl = 215
	perLevelShift      = 3
)

// WeaponBase is the table damage of a weapon (weapons.txt).
type WeaponBase struct {
	Min, Max       int // one handed damage (mindam/maxdam)
	TwoMin, TwoMax int // two handed damage (2handmindam/2handmaxdam)
	TwoHanded      bool
	StrBonus       int // strbonus column, percent per point of strength (usually 100)
	DexBonus       int // dexbonus column
	Ranged         bool
	Throwing       bool
}

// SocketedItem is an item socketed into another: a gem, rune or jewel.
type SocketedItem struct {
	Code  string
	Props []Prop // jewels carry their own properties; gems and runes have none
}

// Item is one piece of equipment (or a charm) as the stat list sees it.
type Item struct {
	Code string
	Slot int // equipment slot, 0 for a charm in the inventory
	// Charm marks an inventory charm: active while it is in the inventory.
	Charm    bool
	Ethereal bool
	// Broken: durability 0. A broken item gives nothing (UNVERIFIED for the
	// base defense, community knowledge for the bonuses).
	Broken bool

	// Defense is the base defense of an armor piece: the rolled value a .d2s
	// stores (already includes the ethereal bonus, UNVERIFIED), or the roll
	// from armor.txt minac..maxac for generated items.
	Defense int
	// BaseBlock is the block chance of the base shield (armor.txt "block").
	BaseBlock int
	Weapon    *WeaponBase

	Props []Prop
	// SetID identifies the set (0 none). SetLists[i] is the bonus list of
	// tier i, active when at least i+2 pieces of the set are equipped.
	SetID         int
	SetLists      [][]Prop
	Runeword      bool
	RunewordProps []Prop
	Sockets       []SocketedItem
}

// allProps returns the item's own properties (no set tiers): base
// properties, runeword properties and socketed jewels.
func (it *Item) ownProps() []Prop {
	out := append([]Prop(nil), it.Props...)
	out = append(out, it.RunewordProps...)

	for _, s := range it.Sockets {
		out = append(out, s.Props...)
	}

	return out
}

// AllProps returns the item's own properties (base, runeword and socketed
// jewel properties), without set tiers.
func (it *Item) AllProps() []Prop { return it.ownProps() }

// sumID sums a stat over props.
func sumID(props []Prop, id int) int64 {
	var s int64

	for _, p := range props {
		if p.ID == id {
			s += p.Value
		}
	}

	return s
}

// DefenseOf returns the defense an armor piece contributes: its base
// defense raised by the item's enhanced defense percent, plus flat defense
// bonuses (stat 31) that enhanced defense does NOT multiply. The "ED applies
// to base defense only" rule is community knowledge (UNVERIFIED in the
// binary). Per level armor (stats 214, 215) uses the hero level.
func (it *Item) DefenseOf(clvl int) int {
	if it.Broken {
		return 0
	}

	props := it.ownProps()
	ed := sumID(props, StatArmorPct)
	flat := sumID(props, StatArmorClass)

	// item_armor_perlevel / item_armorpercent_perlevel: op param 3 in ItemStatCost
	flat += sumID(props, statArmorPerLevel) * int64(clvl) >> perLevelShift
	ed += sumID(props, statArmorPctPerLvl) * int64(clvl) >> perLevelShift

	base := int64(it.Defense)

	return int(base+base*ed/100) + int(flat)
}

// EtherealDefense is the defense roll of a generated ethereal armor:
// minac..maxac raised by 50% (rounded down). itemgen.md section 3 notes the
// 3/2 factor in the item parser; whether a saved item already contains it is
// UNVERIFIED, so saved items are used as stored.
func EtherealDefense(roll int) int { return roll * 3 / 2 }

// EtherealDamage applies the ethereal +50% to a weapon's table damage.
func EtherealDamage(v int) int { return v * 3 / 2 }

// ItemDefenseRoll is the defense of a freshly generated armor: roll within
// [minac, maxac] (r01 in [0,1) picks the value), +50% if ethereal.
func ItemDefenseRoll(minAC, maxAC int, r01 float64, ethereal bool) int {
	if maxAC < minAC {
		maxAC = minAC
	}

	v := minAC + int(r01*float64(maxAC-minAC+1))
	if v > maxAC {
		v = maxAC
	}

	if ethereal {
		v = EtherealDefense(v)
	}

	return v
}
