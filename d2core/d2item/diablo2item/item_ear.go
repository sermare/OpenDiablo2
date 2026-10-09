package diablo2item

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// earCode is the misc.txt code of the player ear.
const earCode = "ear"

// heroClassNames are the class names of an ear, in the order of the game's
// classes (amazon ... assassin).
var heroClassNames = [...]string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}

// EarInfo is what an ear remembers of the player it was cut from.
type EarInfo struct {
	Name  string `json:"name"`
	Class int    `json:"class"` // 0 amazon .. 6 assassin
	Level int    `json:"level"`
}

// Label is the ear's name: the player's name, with the level and class.
func (e *EarInfo) Label() string {
	class := "?"
	if e.Class >= 0 && e.Class < len(heroClassNames) {
		class = heroClassNames[e.Class]
	}

	return fmt.Sprintf("%s (Level %d %s)", e.Name, e.Level, class)
}

// Ear returns what the item remembers of the player when it is an ear.
func (i *Item) Ear() *EarInfo {
	return i.ear
}

// NewEar makes the ear a hardcore player kill gives (d2combat.PvPKillGivesEar):
// a player ear (misc.txt "ear") of the victim, with the victim's name, class
// and level. The item level is the victim's level.
func (f *ItemFactory) NewEar(name string, class, level int) (*Item, error) {
	icr := f.asset.Records.Item.All[earCode]
	if icr == nil {
		return nil, fmt.Errorf("%w: %q", errUnknownItemCode, earCode)
	}

	// the item creator does not make ears (their name and level come from the
	// player, see d2drop.Creator), so the ear is built directly
	item := &Item{
		factory: f, CommonCode: earCode, TypeCode: icr.Type, itemLevel: level, quality: d2drop.QualityNormal,
		ear: &EarInfo{Name: name, Class: class, Level: level},
	}
	item.SetSeed(int64(f.nextSeed()))
	item.init()
	item.Identify()

	return item, nil
}
