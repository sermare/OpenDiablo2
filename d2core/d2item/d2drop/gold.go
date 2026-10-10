package d2drop

// Gold and magic find helpers. VERIFIED against the emulated game (goldens
// testdata/gold.json and testdata/goldmul.json).

// GoldCode is the item code of gold in the treasure class tables.
const GoldCode = "gld"

// GoldAmount is the amount of a freshly created gold item
// (InitItemBaseStats 555b10): rand(5*ilvl) + ilvl on the item's own
// generator, at least 1. A forced amount (> 0, the request's +0x54 field)
// replaces the result, but the generator is stepped regardless. ilvl is the
// item level of the gold.
func GoldAmount(rng RNG, ilvl, forced int) int {
	amount := int(rng.Roll(int32(5*ilvl))) + ilvl
	if amount <= 0 {
		amount = 1
	}

	if forced > 0 {
		amount = forced
	}

	return amount
}

// ScaleGoldMul applies the "mul=N" multiplier of a treasure class entry to a
// gold amount: amount*mul >> 8 (so mul=256 is the identity). The game does
// this only for entries that have a multiplier (Drop.Mul != 0).
func ScaleGoldMul(amount, mul int) int {
	return int(int32(amount*mul) >> 8)
}

// ApplyGoldFind is ITEMGEN_ApplyGoldFindBonus (556a10): gold * (100 +
// killer's gold find + owner's gold find) / 100, truncated toward zero. The
// owner is the master of a summon or mercenary (0 if there is none). Nothing
// is changed when there is no killer or no gold item; callers skip the call
// in that case.
func ApplyGoldFind(gold, killerGoldFind, ownerGoldFind int) int {
	return int(int32(gold) * int32(100+killerGoldFind+ownerGoldFind) / 100)
}

// MagicFindOf is ITEMGEN_GetMagicFind (556630): the magic find stat (0x50) of
// the dropper plus that of its owner. Only players and monsters (unit types 0
// and 1) count; other units give 0.
func MagicFindOf(unitType, own, owner int) int {
	if unitType < 0 || unitType > 1 {
		return 0
	}

	return own + owner
}
