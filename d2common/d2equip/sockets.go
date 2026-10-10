package d2equip

import (
	"strconv"
	"strings"
)

// Socket and gem rules. VERIFIED from the binary: ITEMGEN_RollSockets
// (0x554c60) rolls only for quality >= normal (2), non stackable items whose
// base has an inventory, 33% (rand(100) < 33); the count is capped per
// difficulty (normal 3, nightmare 4, hell 6) and, for LoD items, is
// initseed % max + 1 (classic: min(max, 3), helms at most 2), then clamped by
// ITEM_ClampSocketCountBySize (0x62be00) to the inventory area (invwidth *
// invheight, at most 6). The max is ITEM_GetMaxSocketsForLevel (0x62bd70): the
// smaller of the base gemsockets (record +0x138) and the ItemTypes MaxSock1 /
// MaxSock25 / MaxSock40 column of the item's type row, picked by item level
// (ilvl <= 25, ilvl <= 40, above). Plate Mail's gemsockets of 2 is plain
// armor.txt data (base column), not a cap derived from ItemTypes.

// Item level limits of the ItemTypes MaxSock columns (VERIFIED, 0x62bd70:
// "ilvl <= 25" and "ilvl <= 40"); maxSockMid is the first level of the last column.
const (
	maxSockLow = 25
	maxSockMid = 41
)

// SocketCapByDifficulty is the hard cap on rolled sockets (VERIFIED, 0x554c60):
// 3 in normal, 4 in nightmare, 6 in hell.
func SocketCapByDifficulty(difficulty int) int {
	switch {
	case difficulty <= 0:
		return 3
	case difficulty == 1:
		return 4
	}

	return 6
}

// MaxSocketsByLevel picks the MaxSock1/25/40 value of an item type for an
// item level (VERIFIED borders, 0x62bd70).
func (t *Type) MaxSocketsByLevel(ilvl int) int {
	switch {
	case ilvl <= maxSockLow:
		return t.MaxSock1
	case ilvl < maxSockMid:
		return t.MaxSock25
	}

	return t.MaxSock40
}

// MaxSockets is the most sockets a generated item of base b can roll at an
// item level in a difficulty: the base gemsockets, the type's level limit
// (the type's own row; ItemTypes repeats the columns on every socketable row)
// and the difficulty cap.
func (t *Types) MaxSockets(b Base, ilvl, difficulty int) int {
	n := b.GemSockets

	if ty := t.Get(b.Type); ty != nil && ty.MaxSock1+ty.MaxSock25+ty.MaxSock40 > 0 {
		if m := ty.MaxSocketsByLevel(ilvl); m < n {
			n = m
		}
	}

	if c := SocketCapByDifficulty(difficulty); c < n {
		n = c
	}

	if n < 0 {
		n = 0
	}

	return n
}

// SocketCount is the number of sockets a successful socket roll gives
// (VERIFIED, 0x554c60): LoD itemseed % max + 1; classic min(max, 3) with helms
// at most 2. max <= 0 gives 0.
func SocketCount(max int, seed uint32, classic, helm bool) int {
	if max <= 0 {
		return 0
	}

	if classic {
		if helm && max > 2 {
			max = 2
		}

		if max > 3 {
			max = 3
		}

		return max
	}

	return int(seed%uint32(max)) + 1
}

// SocketSlot is where a socketed gem or rune acts, picked by the item it sits in.
type SocketSlot int

// The column groups of Gems.txt (and the values of the gemapplytype column).
const (
	SlotWeapon SocketSlot = 0 // weaponMod*
	SlotArmor  SocketSlot = 1 // helmMod* (used for helms and body armor alike)
	SlotShield SocketSlot = 2 // shieldMod*
)

// SlotOf returns the Gems.txt column group for an item type: weapons use the
// weapon mods, shields the shield mods, other armor the helm (armor) mods.
func (t *Types) SlotOf(itemType string) (SocketSlot, bool) {
	switch {
	case t.IsA(itemType, "weap"):
		return SlotWeapon, true
	case t.IsA(itemType, "shld"):
		return SlotShield, true
	case t.IsA(itemType, "armo"):
		return SlotArmor, true
	}

	return 0, false
}

// GemMod is one property of a gem for one slot.
type GemMod struct {
	Code     string
	Param    string
	Min, Max int
}

// Gem is one Gems.txt row.
type Gem struct {
	Name, Code string
	Mods       [3][]GemMod // indexed by SocketSlot
}

// Gems is Gems.txt keyed by code.
type Gems map[string]Gem

// ParseGems reads Gems.txt (gems and runes).
func ParseGems(data []byte) (Gems, error) {
	rows, col, err := readTSV(data)
	if err != nil {
		return nil, err
	}

	out := Gems{}

	for _, r := range rows {
		code := cell(r, col, "code")
		if code == "" {
			continue
		}

		g := Gem{Name: cell(r, col, "name"), Code: code}

		for slot, prefix := range []string{"weaponmod", "helmmod", "shieldmod"} {
			for i := 1; i <= 3; i++ {
				p := prefix + strconv.Itoa(i)
				if mc := cell(r, col, p+"code"); mc != "" {
					g.Mods[slot] = append(g.Mods[slot], GemMod{
						Code: mc, Param: cell(r, col, p+"param"),
						Min: num(r, col, p+"min"), Max: num(r, col, p+"max"),
					})
				}
			}
		}

		out[strings.TrimSpace(code)] = g
	}

	return out, nil
}

// ModsFor returns the properties a gem gives in a slot.
func (g Gem) ModsFor(s SocketSlot) []GemMod {
	if s < 0 || int(s) >= len(g.Mods) {
		return nil
	}

	return g.Mods[s]
}
