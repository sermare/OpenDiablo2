package d2reward

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// The reward NPCs: who takes an item from the cursor, and the rules of the two
// rewards that reward.go does not cover (Charsi's imbue and Akara's reset).
// The mechanism (the item is dropped on the NPC) and the rules below are
// VERIFIED in Game.exe 1.14b (0x577b70, classes 0x9a, 0x1ff, 0x200): the item
// is on the player's cursor, the quest record bit (state 1) of the reward must
// be set (Charsi flag 3, Larzuk 0x23, Anya 0x26), the item is cloned or
// rebuilt and put back into the inventory (dropped on the ground when it does
// not fit); on a refusal the cursor item is dropped. Charsi's rare keeps
// ethereal and personalisation.

// ItemKind is the reward an NPC performs on an item.
type ItemKind string

// The item rewards.
const (
	KindSocket      ItemKind = "socket"      // Larzuk
	KindPersonalize ItemKind = "personalize" // Anya (Drehya)
	KindImbue       ItemKind = "imbue"       // Charsi
)

// ItemKindOf returns the reward the NPC class performs on an item dropped on it.
func ItemKindOf(class int) (ItemKind, bool) {
	switch class {
	case d2quest.NPCLarzuk:
		return KindSocket, true
	case d2quest.NPCDrehya:
		return KindPersonalize, true
	case d2quest.NPCCharsi:
		return KindImbue, true
	}

	return "", false
}

// NPCOf is the class that performs the reward.
func NPCOf(k ItemKind) int {
	switch k {
	case KindSocket:
		return d2quest.NPCLarzuk
	case KindPersonalize:
		return d2quest.NPCDrehya
	case KindImbue:
		return d2quest.NPCCharsi
	}

	return 0
}

// Verb is the label of the menu row that asks for the item (a row of this
// fork: the original shows no row, the player drops an item on the NPC).
func (k ItemKind) Verb() string {
	switch k {
	case KindSocket:
		return "Add Sockets"
	case KindPersonalize:
		return "Personalize"
	case KindImbue:
		return "Imbue"
	}

	return string(k)
}

// Owed says how many of each item reward the hero is still owed.
type Owed struct {
	Sockets, Personalize int
	Imbue                bool
}

// Of reports whether the reward of the kind is waiting.
func (o Owed) Of(k ItemKind) bool {
	switch k {
	case KindSocket:
		return o.Sockets > 0
	case KindPersonalize:
		return o.Personalize > 0
	case KindImbue:
		return o.Imbue
	}

	return false
}

// OwedFrom combines the reward state with the imbue flag.
func OwedFrom(s *State, imbue bool) Owed {
	if s == nil {
		return Owed{Imbue: imbue}
	}

	return Owed{Sockets: s.SocketPending, Personalize: s.PersonalizePending, Imbue: imbue}
}

// ---- Charsi: imbue ----

// ImbueItem is what the imbue rule needs to know about an item.
type ImbueItem struct {
	WeaponOrArmor bool
	// Quality uses the item quality ids (1 low, 2 normal, 3 superior, 4 magic,
	// 5 set, 6 rare, 7 unique, 8 crafted).
	Quality     int
	Quest       bool
	Gems        int  // items socketed into it
	Socketed    bool // item flag 0x800 (it has sockets)
	Throwable   bool
	NonSellable bool // item flag 0x1000
	Gold        bool // item type 4
}

// CanImbue says whether Charsi may imbue the item (class 0x9a in 0x577b70,
// TRADE_IsItemSellableCheckA 0x62c700, VERIFIED): low, normal and superior
// quality only (magic and every better quality are refused), no socketed
// item (flag 0x800 or gems inside), no quest item, no throwing weapon, no gold
// and no non-sellable item. WeaponOrArmor stands for the base-record flag the
// original tests (ITEM_TestBitfield1Flag1; UNVERIFIED which bit it is).
func CanImbue(it ImbueItem) error {
	switch {
	case it.Gold, it.NonSellable:
		return ErrNotSocketable("this item cannot be imbued")
	case it.Quest:
		return ErrNotSocketable("quest items cannot be imbued")
	case !it.WeaponOrArmor:
		return ErrNotSocketable("only weapons and armour can be imbued")
	case it.Throwable:
		return ErrNotSocketable("throwing weapons cannot be imbued")
	case it.Quality < 1 || it.Quality > 3:
		return ErrNotSocketable("only low, normal and superior items can be imbued")
	case it.Gems > 0 || it.Socketed:
		return ErrNotSocketable("a socketed item cannot be imbued")
	}

	return nil
}

// ImbueLevel is the item level of the rare Charsi makes (VERIFIED, class 0x9a
// in 0x577b70): ITEMGEN_GetDropBaseLevel(player) is the character level (at
// least 1), plus 4 when above 5.
func ImbueLevel(heroLevel int) int {
	if heroLevel < 1 {
		heroLevel = 1
	}

	if heroLevel > 5 {
		return heroLevel + 4
	}

	return heroLevel
}

// ---- Akara: reset ----

// Allocation is the spent attribute and skill points of a hero.
type Allocation struct {
	Strength, Dexterity, Vitality, Energy int
	SkillPoints                           int // points spent in skills
}

// Respec returns what Akara's reset gives back: the attribute points above the
// class base and all spent skill points (SKILL_ServerRespecAllSkills and
// STATS_ServerResetAttributesToClassBase, names from the notes, unverified).
func Respec(now, base Allocation) (statPoints, skillPoints int) {
	for _, p := range [][2]int{{now.Strength, base.Strength}, {now.Dexterity, base.Dexterity},
		{now.Vitality, base.Vitality}, {now.Energy, base.Energy}} {
		if p[0] > p[1] {
			statPoints += p[0] - p[1]
		}
	}

	return statPoints, now.SkillPoints
}
