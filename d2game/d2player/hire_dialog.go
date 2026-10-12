package d2player

import (
	"fmt"
	"strings"
)

// The text of the hire list (UI_OpenMercHireDialog 0x4b24c0, read in Game.exe 1.14b). The dialog is titled by string
// 0xd24 ("Your Gold: %d     Hire which Mercenary?", the number is the sum of the gold stats 0xe and 0xf: carried and
// stashed gold), shows one entry per offered mercenary and, when none is left, string 0xd25. An entry is built as
// "<name> - Lvl: NN  Life: NN  Def: NN  Cost: NN" (the labels are strings 0xd28, 0xd26, 0xd27 and 0xd29, the numbers use
// "%2u"; the exe pads the labels with a bounded copy of 20 characters, which does not change the text) and a second
// line holding the HireDesc label of the row. The keys below are the string.tbl keys of those ids.
const (
	hireKeyTitle = "ItemDesc1s" // 0xd24
	hireKeyEmpty = "ItemDesc1t" // 0xd25
	hireKeyLife  = "HP"         // 0xd26
	hireKeyDef   = "AC"         // 0xd27
	hireKeyLevel = "Level"      // 0xd28
	hireKeyCost  = "Cost"       // 0xd29
)

// hireDescKeys maps the HireDesc column of hireling.txt to the string key of the second line (table at 0x6ef9c0, ids
// 0xd2b, 0xd2c, then the expansion ids of Combat, Defensive, Offensive, then 0xd31/0xd30/0xd2f). The three
// expansion entries are 0x2b52 Combat, 0x2b50 Defensive, 0x2b51 Offensive (patch string table texts, VERIFIED to equal
// the StrMercEx15 / 12 / 14 texts used below).
var hireDescKeys = map[string]string{ //nolint:gochecknoglobals // static
	"farw": "strhirespecial1", "carw": "strhirespecial2", "comb": "StrMercEx15", "def": "StrMercEx12",
	"off": "StrMercEx14", "fire": "strhirespecial7", "cold": "strhirespecial6", "ltng": "strhirespecial5",
}

// HireDialogTitle formats the title of the hire list for the gold of the hero (carried plus stashed).
func HireDialogTitle(tr func(string) string, gold int) string {
	return strings.Replace(tr(hireKeyTitle), "%d", fmt.Sprint(gold), 1)
}

// ReviveLine fills the "Resurrect %s: %d" string with the name and the price ("Resurrect name: price" if the text has no
// placeholders, e.g. a missing translation). The exe uses it (key hireresurrect2, id 0x58a8) in UI_BuildNpcMenu 0x4b1050 for the merc revive row
// (arguments: the mercenary name, the price; VERIFIED, w44).
func ReviveLine(format, name string, cost int) string {
	if !strings.Contains(format, "%s") || !strings.Contains(format, "%d") {
		return fmt.Sprintf("Resurrect %s: %d", name, cost)
	}

	return fmt.Sprintf(format, name, cost)
}

// HireDialogEmpty is the text shown when no mercenary is left to hire.
func HireDialogEmpty(tr func(string) string) string { return tr(hireKeyEmpty) }

// HireDialogLine is the first line of an entry of the hire list.
func HireDialogLine(tr func(string) string, name string, level, life, def, cost int) string {
	return fmt.Sprintf("%s - %s: %2d  %s: %2d  %s: %2d  %s: %2d", name,
		tr(hireKeyLevel), level, tr(hireKeyLife), life, tr(hireKeyDef), def, tr(hireKeyCost), cost)
}

// HireDialogDesc is the second line of an entry: the label of the row's HireDesc ("" when it has none).
func HireDialogDesc(tr func(string) string, hireDesc string) string {
	k, ok := hireDescKeys[hireDesc]
	if !ok {
		return ""
	}

	return tr(k)
}
