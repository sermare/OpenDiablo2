package diablo2item

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

var (
	errUnknownMonster = errors.New("unknown monster")
	errUnknownLevel   = errors.New("unknown level")
)

// MonsterDropOptions describe a killed monster. The embedded DropOptions
// supply the seed, players, magic find, gold find and drop cap; its ILvl,
// UpgradeLevel, Classic and quality level fields are filled in from the
// monster.
type MonsterDropOptions struct {
	DropOptions
	Monster     string // MonStats.txt id ("fallen1")
	Flags       int    // d2drop.MonsterFlagChampion / MonsterFlagUnique
	SuperUnique string // SuperUniques.txt key, empty if the monster is not a super unique
	Difficulty  int    // 0 normal, 1 nightmare, 2 hell
	Expansion   bool   // Lord of Destruction game
	Level       int    // the monster's level
	// KillerIsPlayer and QuestStates: see d2drop.MonsterTreasureInput.
	KillerIsPlayer bool
	QuestStates    [3]bool
}

// MonsterTreasureClass picks the treasure class a monster drops from (see
// d2drop.MonsterTreasureClass); the empty string means it drops nothing.
func (f *ItemFactory) MonsterTreasureClass(o MonsterDropOptions) (string, error) {
	rec := f.asset.Records
	ms := rec.Monster.Stats[o.Monster]

	if ms == nil {
		return "", fmt.Errorf("%w: %q", errUnknownMonster, o.Monster)
	}

	in := d2drop.MonsterTreasureInput{
		Difficulty: o.Difficulty, Flags: o.Flags, KillerIsPlayer: o.KillerIsPlayer, QuestStates: o.QuestStates,
		TCs: [3][4]string{
			{ms.TreasureClassNormal, ms.TreasureClassChampionNormal, ms.TreasureClass3UniqueNormal, ms.TreasureClassQuestNormal},
			{ms.TreasureClassNightmare, ms.TreasureClassChampionNightmare, ms.TreasureClass3UniqueNightmare, ms.TreasureClassQuestNightmare},
			{ms.TreasureClassHell, ms.TreasureClassChampionHell, ms.TreasureClass3UniqueHell, ms.TreasureClassQuestHell},
		},
	}

	in.QuestID, _ = strconv.Atoi(ms.TreasureClassQuestTriggerId)
	in.QuestCP, _ = strconv.Atoi(ms.TreasureClassQuestCompleteId)

	if o.SuperUnique != "" {
		in.IsSuperUnique = true

		if su := rec.Monster.Unique.Super[o.SuperUnique]; su != nil {
			in.SuperUnique = &[3]string{su.TreasureClassNormal, su.TreasureClassNightmare, su.TreasureClassHell}
		}
	}

	return d2drop.MonsterTreasureClass(in), nil
}

// MonsterDrops rolls what a monster drops: the class choice of
// MonsterTreasureClass, the monster level upgrade of the class (expansion,
// above normal difficulty), the monster level as item level.
func (f *ItemFactory) MonsterDrops(o MonsterDropOptions) (*DropResult, error) {
	name, err := f.MonsterTreasureClass(o)
	if err != nil || name == "" {
		return &DropResult{}, err
	}

	flags := 0
	if ms := f.asset.Records.Monster.Stats[o.Monster]; ms != nil && ms.IgnoreMonLevelTxt {
		flags |= d2drop.MonStatsNoRatio
	}

	opts := o.DropOptions
	opts.ILvl = o.Level
	opts.Classic = !o.Expansion
	opts.Difficulty = o.Difficulty
	opts.UpgradeLevel = d2drop.MonsterUpgradeLevel(o.Expansion, o.Difficulty, true, flags, o.Level)

	return f.DropAll(name, opts)
}

// ChestDropOptions describe an opened chest.
type ChestDropOptions struct {
	DropOptions
	LevelID    int // Levels.txt id of the area the chest stands in
	Difficulty int
	Expansion  bool
}

// ChestDrops rolls what a chest drops: the class "Act N Chest A/B/C" follows
// from the area's monster level within its act, the items get the area's
// monster level as item level, and the quality roll uses the tier (0..2) as
// its level (VERIFIED, see d2drop.ChestTreasureClass).
func (f *ItemFactory) ChestDrops(o ChestDropOptions) (*DropResult, error) {
	class, opts, err := f.chestSetup(o)
	if err != nil {
		return nil, err
	}

	return f.DropAll(class, opts)
}

// ChestLoot is ChestDrops keeping the drop order of items and gold. It also
// returns the treasure class that was rolled.
func (f *ItemFactory) ChestLoot(o ChestDropOptions) (*Loot, string, error) {
	class, opts, err := f.chestSetup(o)
	if err != nil {
		return nil, "", err
	}

	loot, err := f.DropLoot(class, opts, 0)

	return loot, class, err
}

func (f *ItemFactory) chestSetup(o ChestDropOptions) (class string, opts DropOptions, err error) {
	levels := f.asset.Records.Level.Details

	area := levels[o.LevelID]
	if area == nil {
		return "", opts, fmt.Errorf("%w: %d", errUnknownLevel, o.LevelID)
	}

	monLevel := func(id int) int {
		if l := levels[id]; l != nil {
			return monsterLevelOf(l, o.Difficulty, o.Expansion)
		}

		return 1
	}

	chest := d2drop.ChestTreasureClass(o.Difficulty, area.Act, monLevel(o.LevelID), monLevel)

	opts = o.DropOptions
	opts.ILvl = monLevel(o.LevelID)
	opts.Classic = !o.Expansion
	opts.Difficulty = o.Difficulty
	opts.QualityLevel, opts.UseQualityLevel = chest.Tier, true

	if opts.MaxDrops == 0 {
		opts.MaxDrops = d2drop.DefaultMaxDrops
	}

	return chest.Class, opts, nil
}

// monsterLevelOf is the monster level of an area (Levels.txt MonLvl1..3, or the
// Ex columns in an expansion game); 1 for a difficulty out of range.
func monsterLevelOf(l *d2records.LevelDetailRecord, difficulty int, expansion bool) int {
	var byDiff [3]int

	if expansion {
		byDiff = [3]int{l.MonsterLevelNormalEx, l.MonsterLevelNightmareEx, l.MonsterLevelHellEx}
	} else {
		byDiff = [3]int{l.MonsterLevelNormal, l.MonsterLevelNightmare, l.MonsterLevelHell}
	}

	if difficulty < 0 || difficulty > 2 {
		return 1
	}

	return byDiff[difficulty]
}
