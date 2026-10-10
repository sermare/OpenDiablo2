// Package d2act3 holds the rules of the Act 3 content that is played in the running game and not only
// in the quest tables: which quest object gives which item, the Council of Travincal, Khalim's Flail,
// the Compelling Orb and Mephisto's red portal. It is pure data and small functions; the game screen
// (d2game/d2gamescreen/act3_live.go) carries the results out.
//
// Status of the rules: the objects, their ids and levels are OBSERVED in the real DS1 presets and
// objects.txt (the Khalim chests are in Spider Cavern, Flayer Dungeon 3 and Sewers 2; the Compelling
// Orb has no lookup row, so no DS1 carries it and the game creates it at the stairs of the Durance).
// Which Council member carries the Flail, the delay of the red portal and the smashing with a carried
// (not equipped) Will are UNVERIFIED.
package d2act3

import (
	"errors"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// objects.txt ids of the Act 3 quest objects.
const (
	ObjLamEsenTome    = 193 // OperateFn 28
	ObjGidbinnAltar   = 251 // OperateFn 0, scenery of the altar
	ObjGidbinn        = 252 // OperateFn 31: the blade on its altar
	ObjSewerStairs    = 366 // OperateFn 44, not selectable
	ObjSewerLever     = 367 // OperateFn 45
	ObjMephistoBridge = 341 // OperateFn 4, not selectable (the bridge before the lair)
	ObjHellgate       = 342 // OperateFn 46: the red portal out of Mephisto's lair
	ObjDuranceStairs  = 386 // "stairs of the Compelling Orb", OperateFn 50, a dummy in the Travincal preset
	ObjCompellingOrb  = 404 // OperateFn 53
	ObjChestHeart     = 405 // Khalim's Heart, OperateFn 57
	ObjChestBrain     = 406 // Khalim's Brain, OperateFn 59
	ObjChestEye       = 407 // Khalim's Eye, OperateFn 58
)

// Levels of the Act 3 quest content (Levels.txt ids).
const (
	LevelSewers1      = 92
	LevelSpiderCavern = 85
	LevelFlayerDung3  = 91 // Flayer Dungeon 3 (the "treasure" preset of the second dungeon)
	LevelSewers2      = 93
	LevelRuinedTemple = 94
	LevelTravincal    = d2level.LevelTravincal
	LevelDurance3     = d2level.LevelDurance3
)

// FortressStart is the level the red portal leads to: the Pandemonium Fortress, the start level of Act 4.
const FortressStart = 103

// HellgateDelayFrames is the delay (frames of 25 per second) between Mephisto's death and the red portal
// (the exe starts a quest timer of 0xc after the kill, d2boss.hellgateDelay; the unit is UNVERIFIED).
const HellgateDelayFrames = 300

// QuestObject says what a quest object of Act 3 does when the hero operates it.
type QuestObject struct {
	ID    int
	Name  string
	Item  string // the quest item the object gives, "" for none
	Level int    // the level the object is placed in (0: several or none)
	Note  string
}

var questObjects = []QuestObject{
	{ID: ObjChestEye, Name: "Khalim's Eye chest", Item: d2quest.ItemKhalimEye, Level: LevelSpiderCavern},
	{ID: ObjChestBrain, Name: "Khalim's Brain chest", Item: d2quest.ItemKhalimBrain, Level: LevelFlayerDung3},
	{ID: ObjChestHeart, Name: "Khalim's Heart chest", Item: d2quest.ItemKhalimHeart, Level: LevelSewers2},
	{ID: ObjLamEsenTome, Name: "Lam Esen's Tome", Item: d2quest.ItemLamEsenTome, Level: LevelRuinedTemple,
		Note: "UNVERIFIED: the tome is the object, the item is what the quest picks up"},
	{ID: ObjGidbinn, Name: "Gidbinn", Item: d2quest.ItemGidbinn,
		Note: "UNVERIFIED: the blade is taken from its altar like a chest item"},
	{ID: ObjSewerLever, Name: "Sewer lever", Level: LevelSewers1,
		Note: "UNVERIFIED: pulling it makes the stairs to Sewers 2 (the Heart) usable"},
	{ID: ObjCompellingOrb, Name: "Compelling Orb", Level: LevelTravincal,
		Note: "smashed with Khalim's Will; no item"},
}

// IsQuestObject reports whether an objects.txt id is operated by the Act 3 quest code.
func IsQuestObject(id int) bool {
	_, ok := Object(id)

	return ok
}

// Object returns the description of an Act 3 quest object.
func Object(id int) (QuestObject, bool) {
	for _, o := range questObjects {
		if o.ID == id {
			return o, true
		}
	}

	return QuestObject{}, false
}

// Objects lists all quest objects (for tests and logs).
func Objects() []QuestObject { return append([]QuestObject(nil), questObjects...) }

// CanSmashOrb says whether the Compelling Orb can be smashed: the hero must have Khalim's Will.
// (The original wants the Will in the hand; the engine accepts it in the inventory. UNVERIFIED.)
func CanSmashOrb(has func(code string) bool) bool { return has(d2quest.ItemKhalimWill) }

// The three Council members of Travincal (SuperUniques.txt, Normal; Bremm Sparkfist, Wyand Voidfinger
// and Maffer Dragonhand guard the Durance of Hate 3).
var councilTravincal = []string{"Ismail Vilehand", "Geleb Flamefinger", "Toorc Icefist"}

// CouncilOfTravincal returns the names of the three Council members of Travincal.
func CouncilOfTravincal() []string { return append([]string(nil), councilTravincal...) }

// IsTravincalCouncil reports whether a monster label is one of the Council members of Travincal.
func IsTravincalCouncil(label string) bool {
	for _, n := range councilTravincal {
		if strings.EqualFold(n, label) {
			return true
		}
	}

	return false
}

// IsCouncilClass reports whether a monstats class is one of the three Council classes (345..347).
func IsCouncilClass(class int) bool {
	return class == d2quest.NPCCouncilA || class == d2quest.NPCCouncilB || class == d2quest.NPCCouncilC
}

// QuestKillClass is the monster class the quest system sees for a kill. The Blackened Temple counts
// three kills of the Council classes; every pack of a Council member brings followers of the same class,
// so a follower must not count (the quest asks for the three members). A follower is reported as class 0.
func QuestKillClass(class int, isSuperUnique bool) int {
	if IsCouncilClass(class) && !isSuperUnique {
		return 0
	}

	return class
}

// FlailDrop says whether the death of a monster leaves Khalim's Flail: the first Council member of
// Travincal that dies while the hero has neither the Flail nor the Will. UNVERIFIED: the original
// drops it from the Council; which member is not recorded.
func FlailDrop(label string, has func(code string) bool) (string, bool) {
	if !IsTravincalCouncil(label) || has(d2quest.ItemKhalimFlail) || has(d2quest.ItemKhalimWill) {
		return "", false
	}

	return d2quest.ItemKhalimFlail, true
}

// HellgateOpen says whether the red portal of the lair is there: Mephisto is dead.
func HellgateOpen(mephistoDead bool) bool { return mephistoDead }

// ErrSewerStairsHidden is returned for the stairs from Sewers 1 to Sewers 2 while the lever is not pulled.
var ErrSewerStairsHidden = errors.New("warp: the stairs to the Lower Kurast Sewers 2 are hidden until the lever is pulled")

// CheckSewerStairs applies the Heart quest rule to a stair from level `from` to level `to`: the sewer lever
// (object 367) and the sewer stairs (366) are "for the Act 3 sewer quest" in objects.txt; the stairs down
// work once the lever was pulled. UNVERIFIED (the objects and their OperateFns 44/45 are observed; that the
// original gates the stairs this way is recalled, not read from the exe).
func CheckSewerStairs(from, to int, leverPulled bool) error {
	if from == LevelSewers1 && to == LevelSewers2 && !leverPulled {
		return ErrSewerStairsHidden
	}

	return nil
}
