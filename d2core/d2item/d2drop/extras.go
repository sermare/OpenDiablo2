package d2drop

// Ethereal and socket rolls of a freshly generated item, from
// ITEMGEN_RollEthereal (0x554d90) and ITEMGEN_RollSockets (0x554c60) with
// ITEM_ClampSocketCountBySize (0x62be00). VERIFIED against the binary, LoD
// (item format > 0) items only; the classic-format variants are not modelled.

const (
	// EtherealChance is the percent chance of an ethereal item (rand(100) < 5).
	EtherealChance = 5
	// SocketChance is the percent chance of a socketed item (rand(100) < 33).
	SocketChance = 33
	// MaxSocketCells is the most sockets the inventory size clamp allows.
	MaxSocketCells = 6
)

// EtherealInput describes the item for the ethereal roll.
type EtherealInput struct {
	WeaponOrArmor bool    // ItemTypes id 0x2d (Weapon) or 0x32 (Any Armor)
	Durability    bool    // the base has a durability (not nodurability)
	Quest         bool    // quest items are never ethereal
	Quality       Quality // final quality of the item
}

// EtherealEligible says whether the ethereal roll is made at all (and so
// consumes one generator step): weapons and armor with a durability, any
// quality but low quality and set, not quest items.
func EtherealEligible(in EtherealInput) bool {
	return in.WeaponOrArmor && in.Durability && !in.Quest &&
		in.Quality != QualityLow && in.Quality != QualitySet
}

// RollEthereal draws rand(100) < 5 when the item is eligible.
func RollEthereal(rng RNG, in EtherealInput) bool {
	if !EtherealEligible(in) {
		return false
	}

	return rng.Roll(100) < EtherealChance
}

// EtherealMaxDurability is the maximum (and current) durability of an
// ethereal item: half of the base, rounded down, plus one.
func EtherealMaxDurability(base int) int { return base/2 + 1 }

// SocketInput describes the item for the socket roll.
type SocketInput struct {
	Quality      Quality
	HasInventory bool   // the base can hold sockets (hasinv)
	Stackable    bool   // stackable items never get sockets
	MaxSockets   int    // ITEM_GetMaxSocketsForLevel: min(gemsockets, ItemTypes MaxSock by ilvl)
	Difficulty   int    // 0 normal, 1 nightmare, 2 hell
	InitSeed     uint32 // the item's initial seed (pItemData +0x10)
	Cells        int    // invwidth * invheight of the base
}

// SocketsEligible says whether the socket roll is made at all: normal
// quality or better, an item that can hold sockets, not stackable, and a
// level limit above zero. Low quality (1) never rolls.
func SocketsEligible(in SocketInput) bool {
	return in.Quality >= QualityNormal && in.HasInventory && !in.Stackable && in.MaxSockets > 0
}

// SocketDifficultyCap applies the per difficulty limit to a socket maximum:
// 3 in normal, 4 in nightmare, 6 in hell.
func SocketDifficultyCap(max, difficulty int) int {
	caps := [3]int{3, 4, 6}
	if difficulty >= 0 && difficulty < len(caps) && max > caps[difficulty] {
		return caps[difficulty]
	}

	return max
}

// RollSockets draws rand(100) < 33 for an eligible item and returns the
// number of sockets (0 for none): initseed % cap + 1, where cap is the
// difficulty capped maximum, clamped to the inventory area (at most 6 cells
// count) and to the level limit.
func RollSockets(rng RNG, in SocketInput) int {
	if !SocketsEligible(in) {
		return 0
	}

	limit := SocketDifficultyCap(in.MaxSockets, in.Difficulty)
	if limit <= 0 {
		return 0
	}

	if rng.Roll(100) >= SocketChance {
		return 0
	}

	return ClampSocketCount(int(in.InitSeed%uint32(limit))+1, in.Cells, in.MaxSockets)
}

// ClampSocketCount is ITEM_ClampSocketCountBySize for normal and superior
// items: the requested count (at least 1) limited by the inventory area (at
// most 6) and by the level limit. A base with no inventory area gets 0.
func ClampSocketCount(req, cells, levelMax int) int {
	if cells <= 0 {
		return 0
	}

	n := cells
	if n > MaxSocketCells {
		n = MaxSocketCells
	}

	if levelMax <= n {
		n = levelMax
	}

	if req < 1 {
		req = 1
	}

	if req < n {
		n = req
	}

	if n < 1 {
		return 0
	}

	return n
}
