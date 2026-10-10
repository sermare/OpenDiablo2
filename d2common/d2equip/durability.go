package d2equip

// Durability loss in combat, from ITEM_RollDurabilityLoss (0x557d90) and its
// caller (0x57b380), VERIFIED unless marked.
//
// When the hero is hit one of the worn armor pieces is chosen by weight
// (head 3, torso 5, right hand 4, left hand 4, belt 2, feet 2, gloves 2; the
// hand slots count only when they hold armor, i.e. a shield), and that piece
// then loses one durability point with a chance of 10%. When the hero hits
// with a weapon, the weapon loses one point with a chance of 4%.
//
// VERIFIED at 0x557d90: the chance is 10 for items of ItemTypes id 0x32 (Any
// Armor) and 4 for items of id 0x2d (Weapon); a throwable weapon (the Throwable
// column) uses 10 in an expansion game and in a classic game never loses
// durability at all. Anything that is neither armor nor weapon never loses
// durability. This overrides the note in itemgen.md that had armor and weapon
// the other way round.

// Chances in percent (the game rolls seed % 100 < chance).
const (
	ChanceArmor  = 10
	ChanceWeapon = 4
	// ChanceThrown is the chance of throwable weapons in an expansion game (VERIFIED, 0x557d90).
	ChanceThrown = 10
)

// RepairNumerator is the part of an item's full repair cost that is charged
// (VERIFIED, TRADE_CalcItemPrice 0x62f100, repair mode): the price is
// numerator * fullPrice / maxDur. A normal item is charged maxDur-cur points.
// An item with the replenish stat (0xfc) is measured against maxDur-1: it is
// free once cur >= maxDur-1, otherwise the numerator is maxDur-1 (not
// maxDur-1-cur). Items without durability, or at full durability, are free.
func RepairNumerator(maxDur, cur int, replenish bool) int {
	if maxDur <= 0 || maxDur <= cur {
		return 0
	}

	if replenish {
		if maxDur-1 <= cur {
			return 0
		}

		return maxDur - 1
	}

	return maxDur - cur
}

// ArmorWeight is one row of the table at 0x730148.
type ArmorWeight struct {
	Loc    Loc
	Weight int
}

// ArmorWeights is the table of the pieces that can be damaged, in table order.
var ArmorWeights = []ArmorWeight{
	{LocHead, 3}, {LocTorso, 5}, {LocRightHand, 4}, {LocLeftHand, 4}, {LocBelt, 2}, {LocFeet, 2}, {LocGloves, 2},
}

// TotalArmorWeight sums the weights of the candidate pieces.
func TotalArmorWeight(candidate func(Loc) bool) int {
	n := 0

	for _, w := range ArmorWeights {
		if candidate(w.Loc) {
			n += w.Weight
		}
	}

	return n
}

// PickArmorPiece chooses the piece that takes the hit. start is a random number
// in [0, len(ArmorWeights)) and roll one in [0, TotalArmorWeight): the game
// walks the table from start, wrapping, subtracting the weight of each candidate
// from the roll until it falls inside one (the ranges of the two random calls
// are UNVERIFIED, the caller decompiles to RAND_RollSeedModulo with hidden
// arguments). It returns false when no piece can be damaged.
func PickArmorPiece(start, roll int, candidate func(Loc) bool) (Loc, bool) {
	total := TotalArmorWeight(candidate)
	if total <= 0 {
		return LocNone, false
	}

	n := len(ArmorWeights)
	i := ((start % n) + n) % n
	roll %= total

	for {
		w := ArmorWeights[i]
		if candidate(w.Loc) {
			if roll < w.Weight {
				return w.Loc, true
			}

			roll -= w.Weight
		}

		i = (i + 1) % n
	}
}

// CanLoseDurability is FUN_00629b00: the item has a durability (max > 0, not
// "nodurability") and is not indestructible. Ethereal items do lose durability
// (they cannot be repaired, TRADE_CheckItemRepairable).
func (it *Item) CanLoseDurability() bool {
	return it.MaxDurability > 0 && !it.NoDurability && !it.Indestructible
}

// Broken reports 0 durability on an item that has one.
func (it *Item) Broken() bool {
	return it.MaxDurability > 0 && !it.NoDurability && it.Durability <= 0
}

// RollLoss applies one durability roll (percent in [0,100)) with the given
// chance. It returns the new durability and whether a point was lost; the
// durability never goes below 0 and is clamped to the maximum.
func RollLoss(it *Item, percent, chance int) (dur int, lost bool) {
	dur = it.Durability

	if !it.CanLoseDurability() || percent >= chance || dur <= 0 {
		return dur, false
	}

	dur--
	if dur < 0 {
		dur = 0
	}

	if dur > it.MaxDurability {
		dur = it.MaxDurability
	}

	return dur, true
}

// RollWear is the skill-use durability loss of ITEM_ReduceDurabilityOrConsumeOnSkillUse (0x5d9580,
// VERIFIED, Impale): with percent (seed % 100) below chance, durability drops by amount; when that would
// leave 0 or less the item breaks (durability 0, broke true). Non-durable items never change.
func RollWear(it *Item, percent, chance, amount int) (dur int, broke, lost bool) {
	dur = it.Durability
	if !it.CanLoseDurability() || percent >= chance {
		return dur, false, false
	}

	if n := dur - amount; n > 0 {
		return n, false, true
	}

	return 0, true, true
}

// PropertiesOff reports whether a worn item's properties are off because of its
// condition. VERIFIED for armor: when the next point would bring it below 1
// the game sets the broken flag (0x100) and removes the item's stat list
// (0x55d660), and the activation pass never switches a broken item on again.
// For weapons the same function takes the other branch (0x557d90 only zeroes
// the stat and sends the update, no broken flag), so a weapon at 0 durability
// may keep its properties: UNVERIFIED, the damage code was not read. We treat broken weapons like broken armor (the
// conservative choice, matching the community statement that nothing works).
func PropertiesOff(it *Item) bool { return it.Broken() }
