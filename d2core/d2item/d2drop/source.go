package d2drop

import "fmt"

// This file chooses the treasure class of a drop: for a killed monster
// (ITEMGEN_DropMonsterTreasure 5a4000 and ITEMGEN_DropFromTreasureClass
// 558d80) and for an opened chest (583a60 with the class table of 65c580).
// All of it is VERIFIED against the emulated game (oracle goldens
// testdata/monster.json), except where a comment says otherwise.

// Monster type flags of the monster unit data (unit data + 0x16). Bit 0x02
// marks a super unique (its index is in unit data + 0x26).
const (
	MonsterFlagChampion = 0x04
	MonsterFlagUnique   = 0x08
)

// MonStats flag bits (record byte +0xc) that stop the monster level upgrade of
// its treasure class. 0x04 is "NoRatio"; the column behind 0x40 is UNVERIFIED.
const (
	MonStatsNoRatio      = 0x04
	monStatsNoUpgradeBit = 0x40
)

// MonsterTreasureInput describes a killed monster.
type MonsterTreasureInput struct {
	Difficulty int // 0 normal, 1 nightmare, 2 hell
	Flags      int // MonsterFlag* of the monster's unit data
	// TCs holds monstats.txt TreasureClass1..4 per difficulty: [d][0] normal,
	// [d][1] champion, [d][2] unique, [d][3] quest. Empty means none.
	TCs [3][4]string
	// SuperUnique is the TC/TC(N)/TC(H) of the monster's SuperUniques.txt row.
	// nil when the monster is not a super unique, or its index has no row (the
	// game then falls back to the unique class TreasureClass3).
	SuperUnique *[3]string
	// IsSuperUnique is true when the unit data has MonsterFlagSuperUnique set
	// and a super unique index (not -1).
	IsSuperUnique bool
	// QuestID and QuestCP are TCQuestId and TCQuestCP of monstats.txt.
	QuestID, QuestCP int
	// KillerIsPlayer is true when the killer, after resolving summons and
	// mercenaries to their owner, is a player. QuestStates holds the three
	// quest checks of the game (state 15, state 1, and the TCQuestCP state);
	// the quest class only drops while none of them is set.
	KillerIsPlayer bool
	QuestStates    [3]bool
}

// MonsterTreasureClass picks the treasure class a monster drops from. An empty
// result means the monster drops nothing.
func MonsterTreasureClass(in MonsterTreasureInput) string {
	d := clampInt(in.Difficulty, 0, 2)

	var name string

	switch {
	case in.IsSuperUnique && in.SuperUnique != nil:
		name = in.SuperUnique[d]
	case in.IsSuperUnique:
		name = in.TCs[d][2]
	case in.Flags&MonsterFlagChampion != 0:
		name = in.TCs[d][1]
	case in.Flags&MonsterFlagUnique != 0:
		name = in.TCs[d][2]
	default:
		name = in.TCs[d][0]
	}

	if in.QuestID > 0 && in.TCs[d][3] != "" && in.KillerIsPlayer &&
		!in.QuestStates[0] && !in.QuestStates[1] && !in.QuestStates[2] {
		name = in.TCs[d][3]
	}

	return name
}

// MonsterUpgradeLevel is the level ITEMGEN_DropFromTreasureClass hands to the
// level-group upgrade (Context.UpgradeLevel): the monster's level in an
// expansion game above normal difficulty, for monsters whose MonStats flags
// allow it; otherwise 0 (no upgrade).
func MonsterUpgradeLevel(expansion bool, difficulty int, isMonster bool, monStatsFlags, level int) int {
	if !expansion || difficulty <= 0 || !isMonster || monStatsFlags&(MonStatsNoRatio|monStatsNoUpgradeBit) != 0 {
		return 0
	}

	return level
}

// ChestActLevels are the first and last level id of each act, the range the
// chest tier is computed from (data of Game.exe, VERIFIED).
var ChestActLevels = [5][2]int{{2, 37}, {41, 73}, {76, 102}, {104, 108}, {109, 136}}

// ChestTier is the tier (0, 1, 2 for the classes A, B, C) of a chest in an
// area of the given monster level: the monster level range of the act, from
// its first to its last level, is cut into thirds.
func ChestTier(area, first, last int) int {
	span := last - first
	if span < 0 {
		span = -span
	}

	third := (span + 1) / 3

	switch {
	case area < first+third:
		return 0
	case area >= first+2*third:
		return 2
	default:
		return 1
	}
}

// ChestClassName is the name of the chest class "Act N[ (N)| (H)] Chest A|B|C";
// difficulty, act (0 based) and tier are clamped to their ranges.
func ChestClassName(difficulty, act, tier int) string {
	suffix := [3]string{"", " (N)", " (H)"}[clampInt(difficulty, 0, 2)]

	return fmt.Sprintf("Act %d%s Chest %c", clampInt(act, 0, 4)+1, suffix, 'A'+clampInt(tier, 0, 2))
}

// Chest is the treasure class of an opened chest.
type Chest struct {
	Class string
	// Tier (0..2) is what the game hands the roller as its quality level
	// (Context.QualityLevel): chests roll quality as if their level were the
	// tier, whereas the items get the area level as item level (VERIFIED).
	Tier int
}

// ChestTreasureClass chooses the class of a chest in a level of the given act
// (0 based act, area = the level's monster level for this difficulty and game
// type). monLevel returns the monster level of any level id (Levels.txt
// MonLvl1..3 or the Ex columns); it is asked for the act's first and last level.
func ChestTreasureClass(difficulty, act, area int, monLevel func(levelID int) int) Chest {
	act = clampInt(act, 0, 4)
	tier := ChestTier(area, monLevel(ChestActLevels[act][0]), monLevel(ChestActLevels[act][1]))

	return Chest{Class: ChestClassName(difficulty, act, tier), Tier: tier}
}

func clampInt(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	}

	return v
}
