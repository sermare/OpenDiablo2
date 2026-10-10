// Package d2reward turns the EffectReward requests of the quest system into
// state changes and into the rules of the item rewards (Larzuk's sockets,
// Anya's personalisation), plus the quest items monsters drop (the Hellforge
// Hammer, the Mephisto Soulstone). It is pure: the engine applies the returned
// Outcome to the hero and its items.
//
// Evidence: the reward names are those of d2quest (generic.go reward()). The
// rules below come from the community description of the 1.14 rewards and are
// UNVERIFIED against the binary (quests-2.md only covers Acts 1 and 2): the
// socket count, the eligibility lists and the personalised name. The numbers
// that come from the item tables (MaxSock1/25/40 of ItemTypes.txt and
// gemsockets of the base item) are passed in by the engine.
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
	Sockets     int  // already present
	BaseSockets int  // gemsockets of the base item (0: cannot be socketed)
	MaxSock1    int  // ItemTypes.txt MaxSock1 (ilvl 1-25)
	MaxSock25   int  // MaxSock25 (26-40)
	MaxSock40   int  // MaxSock40 (41+)
	Quest       bool // a quest item (never socketed)
}

// ErrNotSocketable explains a refusal.
type ErrNotSocketable string

func (e ErrNotSocketable) Error() string { return string(e) }

// difficultyCap is the limit the tables name for sockets on drops (3, 4, 6);
// UNVERIFIED that Larzuk obeys it, so it is the less generous reading.
var difficultyCap = [3]int{3, 4, 6}

// LarzukSockets returns the number of sockets Larzuk gives the item in the
// difficulty (0..2). The item must have a socketable base, no sockets yet and
// not be a quest item. The count is the ilvl bracket maximum of the item type,
// limited by the base item and by the difficulty cap (UNVERIFIED rule; the
// original may roll a lower number).
func LarzukSockets(it SocketItem, difficulty int) (int, error) {
	switch {
	case it.Quest:
		return 0, ErrNotSocketable("quest items cannot be socketed")
	case it.Sockets > 0:
		return 0, ErrNotSocketable("the item already has sockets")
	case it.BaseSockets <= 0:
		return 0, ErrNotSocketable("the base item has no socket slots")
	}

	max := it.MaxSock1

	switch {
	case it.ItemLevel > 40:
		max = it.MaxSock40
	case it.ItemLevel > 25:
		max = it.MaxSock25
	}

	if it.BaseSockets < max {
		max = it.BaseSockets
	}

	if difficulty < 0 {
		difficulty = 0
	}

	if difficulty > 2 {
		difficulty = 2
	}

	if c := difficultyCap[difficulty]; c < max {
		max = c
	}

	if max <= 0 {
		return 0, ErrNotSocketable("the item type allows no sockets at this item level")
	}

	return max, nil
}

// ---- Anya ----

// PersonalizeItem is what the personalisation rule needs.
type PersonalizeItem struct {
	MagicOrBetter bool // magic, rare, set, unique or crafted
	Nameable      bool // the "nameable" column of the base item
	Personalized  bool
}

// CanPersonalize says whether Anya may name the item: it must be at least
// magic or nameable by the tables, and not named yet.
func CanPersonalize(it PersonalizeItem) error {
	switch {
	case it.Personalized:
		return ErrNotSocketable("the item is already personalised")
	case it.MagicOrBetter || it.Nameable:
		return nil
	}

	return ErrNotSocketable("only magic, rare, set and unique items can be personalised")
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
