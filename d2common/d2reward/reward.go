// Package d2reward turns the EffectReward requests of the quest system into
// state changes and into the rules of the item rewards (Larzuk's sockets,
// Anya's personalisation, Charsi's imbue), plus the quest items monsters drop
// (the Hellforge Hammer, the Mephisto Soulstone). It is pure: the engine
// applies the returned Outcome to the hero and its items.
//
// Evidence: the reward names are those of d2quest (generic.go reward()). The
// item rules (LarzukRange, CanPersonalize, CanImbue, ImbueLevel) were read in
// Game.exe 1.14b, handler TRADE_ServerHandleNpcMenuAction 0x577b70 (npc class
// 0x9a Charsi, 0x1ff Larzuk, 0x200 Anya) and its helpers; see the comments for
// addresses. The numbers that come from the item tables (MaxSock1/25/40 of
// ItemTypes.txt, gemsockets and the size of the base item) are passed in by the
// engine.
package d2reward

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// Outcome is what applying a reward asks of the engine.
type Outcome struct {
	StatPoints  int // add to the hero's unspent stat points
	LifeBonus   int // permanent life (Potion of Life)
	ResistBonus int // permanent bonus to all four resistances (Malah's scroll)
	// Pending says the reward waits for the hero to choose an item.
	PendingSocket      bool
	PendingPersonalize bool
	Hire               string // "ironwolves" or "barbarians": the mercenaries become hirable
	UnlockDifficulty   bool
	GameComplete       bool
	Log                string
}

// State is the reward state of one hero (per game; the engine may persist it).
type State struct {
	StatPoints         int
	LifeBonus          int
	ResistBonus        int
	SocketPending      int // Larzuk's socketing is owed
	PersonalizePending int // Anya's personalisation is owed
	Hired              map[string]bool
	DifficultyUnlocked bool
	Complete           bool
}

// Apply handles one reward effect.
func (s *State) Apply(e d2quest.Effect) Outcome {
	var o Outcome

	switch e.Code {
	case "stat-points":
		s.StatPoints += e.Value
		o.StatPoints = e.Value
	case "life-boost":
		s.LifeBonus += e.Value
		o.LifeBonus = e.Value
	case "resist-bonus":
		s.ResistBonus += e.Value
		o.ResistBonus = e.Value
	case "socket-quest":
		s.SocketPending++
		o.PendingSocket = true
	case "personalize":
		s.PersonalizePending++
		o.PendingPersonalize = true
	case "hire-ironwolves", "hire-barbarians":
		who := strings.TrimPrefix(e.Code, "hire-")
		if s.Hired == nil {
			s.Hired = map[string]bool{}
		}

		s.Hired[who] = true
		o.Hire = who
	case "unlock-difficulty":
		s.DifficultyUnlocked = true
		o.UnlockDifficulty = true
	case "game-complete":
		s.Complete = true
		o.GameComplete = true
	default:
		o.Log = fmt.Sprintf("reward %q is not known", e.Code)

		return o
	}

	o.Log = fmt.Sprintf("%s value=%d", e.Code, e.Value)

	return o
}

// ---- Larzuk ----

// SocketItem is what the socket rule needs to know about an item.
type SocketItem struct {
	Code        string
	ItemLevel   int
	Sockets     int // already present
	Gems        int // items socketed into it
	Quality     int // item quality ids (1 low .. 9 tempered)
	BaseSockets int // gemsockets of the base item (0: cannot be socketed)
	MaxSock1    int // ItemTypes.txt MaxSock1 (ilvl 1-25)
	MaxSock25   int // MaxSock25 (26-40)
	MaxSock40   int // MaxSock40 (41+)
	Width       int // inventory width of the base item in cells
	Height      int // inventory height of the base item in cells
	Quest       bool
	NonSellable bool // item flag 0x1000
	Gold        bool // item type 4
}

// ErrNotSocketable explains a refusal.
type ErrNotSocketable string

func (e ErrNotSocketable) Error() string { return string(e) }

// maxSocketsForLevel is ITEM_GetMaxSocketsForLevel (0x62bd70, VERIFIED): the
// MaxSock1/25/40 of the item type chosen by item level (<= 25, <= 40, above),
// limited by the base record's gemsockets (byte +0x138).
func maxSocketsForLevel(it SocketItem) int {
	m := it.MaxSock1

	switch {
	case it.ItemLevel > 40:
		m = it.MaxSock40
	case it.ItemLevel > 25:
		m = it.MaxSock25
	}

	if it.BaseSockets < m {
		m = it.BaseSockets
	}

	if m < 0 {
		m = 0
	}

	return m
}

// sizeCap is the quality cap of ITEM_ClampSocketCountBySize (0x62be00,
// VERIFIED): the cells of the base item (width*height, at most 6), then magic
// items at most 4, rare 2, set and unique 1, crafted and tempered 3.
func sizeCap(it SocketItem) int {
	c := it.Width * it.Height
	if c > 6 {
		c = 6
	}

	limit := map[int]int{4: 4, 5: 1, 6: 2, 7: 1, 8: 3, 9: 3}[it.Quality]
	if limit != 0 && c > limit {
		c = limit
	}

	return c
}

// LarzukRange is the socket count range Larzuk gives (class 0x1ff in 0x577b70,
// ITEM_CanAddSocketsAtNpc 0x62c8d0, VERIFIED). Difficulty plays no part (the
// old 3/4/6 cap of this package was a guess and is gone). The wanted count is
// the level maximum for low, normal and superior items, 1..min(max,2) (random)
// for magic items and 1 for set, rare, unique, crafted and tempered; it is then
// limited by the cells of the item and the quality cap (sizeCap) and by the
// level maximum. Refused for gold, non-sellable and quest items, for items
// that already have sockets or gems, and when the level maximum is 0.
func LarzukRange(it SocketItem) (lo, hi int, err error) {
	switch {
	case it.Gold, it.NonSellable:
		return 0, 0, ErrNotSocketable("this item cannot be socketed")
	case it.Quest:
		return 0, 0, ErrNotSocketable("quest items cannot be socketed")
	case it.Sockets > 0 || it.Gems > 0:
		return 0, 0, ErrNotSocketable("the item already has sockets")
	}

	mx := maxSocketsForLevel(it)
	if mx <= 0 {
		return 0, 0, ErrNotSocketable("the item type allows no sockets at this item level")
	}

	wantLo, wantHi := mx, mx

	switch {
	case it.Quality == 4:
		wantLo, wantHi = 1, mx
		if wantHi > 2 {
			wantHi = 2
		}
	case it.Quality >= 5 && it.Quality <= 9:
		wantLo, wantHi = 1, 1
	}

	capacity := sizeCap(it)
	if mx <= capacity {
		capacity = mx
	}

	clamp := func(w int) int {
		if w < 1 {
			w = 1
		}

		if w < capacity {
			return w
		}

		return capacity
	}

	lo, hi = clamp(wantLo), clamp(wantHi)
	if hi < 1 {
		return 0, 0, ErrNotSocketable("the item is too small for sockets")
	}

	return lo, hi, nil
}

// LarzukSockets returns the sockets Larzuk gives. roll(n) returns a number in
// [0, n) and is only used for magic items (nil takes the largest count).
func LarzukSockets(it SocketItem, roll func(n int) int) (int, error) {
	lo, hi, err := LarzukRange(it)
	if err != nil {
		return 0, err
	}

	if roll == nil || hi == lo {
		return hi, nil
	}

	return lo + roll(hi-lo+1), nil
}

// ---- Anya ----

// PersonalizeItem is what the personalisation rule needs.
type PersonalizeItem struct {
	Nameable     bool // the "nameable" column of the base item
	Personalized bool // item flag 0x1000000
	NonSellable  bool // item flag 0x1000
	Gold         bool // item type 4
	Excluded     bool // quivers (types 5, 6) and player body parts (type 7)
}

// CanPersonalize says whether Anya may name the item (class 0x200 in 0x577b70,
// TRADE_IsItemSellableCheckB 0x62c800, VERIFIED): the base record's nameable
// column decides, whatever the quality; gold, quivers, body parts,
// non-sellable and already named items are refused. (One more refusal, a base
// record byte tested by ITEM_GetBaseRecordByteChecked, is not decoded.)
func CanPersonalize(it PersonalizeItem) error {
	switch {
	case it.Personalized:
		return ErrNotSocketable("the item is already personalised")
	case it.Gold, it.NonSellable, it.Excluded:
		return ErrNotSocketable("this item cannot be personalised")
	case !it.Nameable:
		return ErrNotSocketable("this kind of item cannot be personalised")
	}

	return nil
}

// PersonalName is the name shown on a personalised item (UNVERIFIED format).
func PersonalName(hero, item string) string {
	hero = strings.TrimSpace(hero)
	if hero == "" {
		return item
	}

	return hero + "'s " + item
}

// ---- quest item drops ----

// QuestDrop names a quest item a monster drops.
type QuestDrop struct {
	Code string
	Why  string
}

// Super uniques and bosses (display labels of the engine) that drop quest
// items, UNVERIFIED: the original gives the soulstone with Mephisto's death
// and the hammer with Hephasto the Armorer's.
const (
	LabelHephasto = "Hephasto the Armorer"
	LabelMephisto = "Mephisto"
)

// DropsFor returns the quest items a killed monster leaves. has reports what
// the hero already carries; done says the Hellforge quest is complete (nothing
// drops then, so a second kill cannot give a second soulstone).
func DropsFor(label string, has func(code string) bool, hellforgeDone bool) []QuestDrop {
	if hellforgeDone {
		return nil
	}

	switch {
	case strings.EqualFold(label, LabelHephasto) && !has(d2quest.ItemHellforgeHammer):
		return []QuestDrop{{d2quest.ItemHellforgeHammer, "Hephasto's Hellforge Hammer"}}
	case strings.EqualFold(label, LabelMephisto) && !has(d2quest.ItemMephistoSoulstone):
		return []QuestDrop{{d2quest.ItemMephistoSoulstone, "Mephisto's Soulstone"}}
	}

	return nil
}

// HellforgeSmash says what smashing the Soulstone with the Hammer does: the
// quest system deletes the stone (d2quest A4Q3); the Hammer stays so the forge
// can be used again (UNVERIFIED). ok is false without both items.
func HellforgeSmash(has func(code string) bool) (consume []string, ok bool) {
	if !has(d2quest.ItemMephistoSoulstone) || !has(d2quest.ItemHellforgeHammer) {
		return nil, false
	}

	return []string{d2quest.ItemMephistoSoulstone}, true
}
