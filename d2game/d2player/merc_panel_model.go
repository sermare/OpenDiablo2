package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
)

// MercView is what the mercenary panel shows: the numbers of the merc's unit (stats with the gear
// applied, as UI_DrawMercStatValues reads them from the unit). The game fills it from the monster
// director; the panel and its tests need no game data.
type MercView struct {
	Name         string
	Level        int
	Exp, NextExp int // NextExp is 0 at the top level
	Str, Dex     int
	DmgMin       int
	DmgMax       int
	Defense      int
	Resist       [4]int // fire, cold, lightning, poison
	HP, MaxHP    int
	Dead         bool
}

// MercRowOrder lists the value rows of the panel in the order of the tables of the original.
var MercRowOrder = []string{ //nolint:gochecknoglobals // static
	"experience", "level", "next_level", "strength", "dexterity", "damage", "defense",
	"fire", "cold", "lightning", "poison", "life",
}

// Values gives the text of every value row: plain numbers, the damage as "min-max" and the life as
// "current/max" (the original prints the damage with "%d-%d"; the life format string is at 0x6dadb0 and
// is read as "%d / %d": UNVERIFIED).
func (v MercView) Values() map[string]string {
	hp := v.HP
	if v.Dead {
		hp = 0
	}

	return map[string]string{
		"experience": fmt.Sprint(v.Exp),
		"level":      fmt.Sprint(v.Level),
		"next_level": fmt.Sprint(v.NextExp),
		"strength":   fmt.Sprint(v.Str),
		"dexterity":  fmt.Sprint(v.Dex),
		"damage":     fmt.Sprintf("%d-%d", v.DmgMin, v.DmgMax),
		"defense":    fmt.Sprint(v.Defense),
		"fire":       fmt.Sprint(v.Resist[0]),
		"cold":       fmt.Sprint(v.Resist[1]),
		"lightning":  fmt.Sprint(v.Resist[2]),
		"poison":     fmt.Sprint(v.Resist[3]),
		"life":       fmt.Sprintf("%d/%d", hp, v.MaxHP),
	}
}

// MercSlotLoc maps a body slot of the panel to the body location of the rules (Inventory.txt rArm is the
// weapon hand, lArm the shield hand).
func MercSlotLoc(slot string) (d2equip.Loc, bool) {
	switch slot {
	case "head":
		return d2equip.LocHead, true
	case "torso":
		return d2equip.LocTorso, true
	case "weapon":
		return d2equip.LocRightHand, true
	case "shield":
		return d2equip.LocLeftHand, true
	}

	return 0, false
}
