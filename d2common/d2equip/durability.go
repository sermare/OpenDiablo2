package d2equip

// Durability loss in combat, from ITEM_RollDurabilityLoss (0x557d90) and its
// caller (0x57b380), VERIFIED unless marked.
//
// When the hero is hit one of the worn armor pieces is chosen by weight
// (head 3, torso 5, right hand 4, left hand 4, belt 2, feet 2, gloves 2; the
// hand slots count only when they hold armor, i.e. a shield), and that piece
// then loses one durability point with a chance of 10%. When the hero hits
// with a weapon, the weapon loses one point with a chance of 4% (10% for the
// item types whose ItemTypes record has the byte at +0x10 set; what that byte
// means is UNVERIFIED, Types do not carry it, see ChanceThrown).

// Chances in percent (the game rolls seed % 100 < chance).
const (
	ChanceArmor  = 10
	ChanceWeapon = 4
	// ChanceThrown is used by weapons of the types flagged at ItemTypes +0x10 (UNVERIFIED which).
	ChanceThrown = 10
)

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

// PropertiesOff reports whether a worn item's properties are off because of its
// condition. VERIFIED for armor: at 0 durability the game sets the broken flag
// (0x100) and removes the item's stat list (0x55d660), and the activation pass
// never switches a broken item on again. For weapons the same function takes the
// other branch (0x557d90 only zeroes the stat and sends the update), so a weapon
// at 0 durability may keep its properties: UNVERIFIED either way, and the
// damage code was not read. We treat broken weapons like broken armor (the
// conservative choice, matching the community statement that nothing works).
func PropertiesOff(it *Item) bool { return it.Broken() }
