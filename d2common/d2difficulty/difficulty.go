package d2difficulty

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Level is a difficulty: 0 Normal, 1 Nightmare, 2 Hell. It has the value of
// the game's difficulty byte (game+0x6d).
type Level int

// The three difficulties.
const (
	Normal Level = iota
	Nightmare
	Hell
	Count
)

// String returns the English name.
func (l Level) String() string {
	switch l {
	case Normal:
		return "Normal"
	case Nightmare:
		return "Nightmare"
	case Hell:
		return "Hell"
	}

	return fmt.Sprintf("Difficulty(%d)", int(l))
}

// Clamp forces a value into 0..2 like the game does (VERIFIED, 0x6551e0).
func Clamp(l int) Level {
	if l < 0 {
		return Normal
	}

	if l > int(Hell) {
		return Hell
	}

	return Level(l)
}

// Pick returns the value of the difficulty out of the three columns.
func Pick(l Level, normal, nightmare, hell int) int {
	return [3]int{normal, nightmare, hell}[Clamp(int(l))]
}

// Row is one row of DifficultyLevels.txt. Columns that a table does not have
// (classic has fewer) are zero.
type Row struct {
	Name                      string
	ResistPenalty             int
	DeathExpPenalty           int
	UberCodeOddsNormal        int
	UberCodeOddsGood          int
	UltraCodeOddsNormal       int
	UltraCodeOddsGood         int
	MonsterSkillBonus         int
	MonsterFreezeDivisor      int
	MonsterColdDivisor        int
	AiCurseDivisor            int
	LifeStealDivisor          int
	ManaStealDivisor          int
	UniqueDamageBonus         int
	ChampionDamageBonus       int
	HireableBossDamagePercent int
	MonsterCEDamagePercent    int
	StaticFieldMin            int
	GambleRare                int
	GambleSet                 int
	GambleUnique              int
	GambleUber                int
	GambleUltra               int
	ExtraUniqueMonsters       int
}

// Table is the three rows of DifficultyLevels.txt.
type Table [Count]Row

// Row returns the row of a difficulty.
func (t *Table) Row(l Level) *Row { return &t[Clamp(int(l))] }

// Default is the 1.14b table (patch_d2.mpq DifficultyLevels.txt, VERIFIED by
// extraction from this install). ExtraUniqueMonsters comes from d2exp.mpq
// (0, 1, 2) and the Hell life/mana steal divisors are 3 in patch_d2 (d2exp
// has 2); the patch wins.
var Default = Table{
	{Name: "Normal", ResistPenalty: 0, DeathExpPenalty: 0, MonsterFreezeDivisor: 1, MonsterColdDivisor: 1,
		AiCurseDivisor: 1, LifeStealDivisor: 1, ManaStealDivisor: 1, UniqueDamageBonus: 90, ChampionDamageBonus: 90,
		HireableBossDamagePercent: 50, MonsterCEDamagePercent: 50, GambleRare: 10000, GambleSet: 100,
		GambleUnique: 50, GambleUber: 90, GambleUltra: 33},
	{Name: "Nightmare", ResistPenalty: -40, DeathExpPenalty: 5, UberCodeOddsNormal: 10, UberCodeOddsGood: 20,
		MonsterSkillBonus: 3, MonsterFreezeDivisor: 2, MonsterColdDivisor: 2, AiCurseDivisor: 2, LifeStealDivisor: 2,
		ManaStealDivisor: 2, UniqueDamageBonus: 75, ChampionDamageBonus: 75, HireableBossDamagePercent: 35,
		MonsterCEDamagePercent: 35, StaticFieldMin: 33, GambleRare: 10000, GambleSet: 100, GambleUnique: 50,
		GambleUber: 90, GambleUltra: 33, ExtraUniqueMonsters: 1},
	{Name: "Hell", ResistPenalty: -100, DeathExpPenalty: 10, UberCodeOddsNormal: 20, UberCodeOddsGood: 40,
		UltraCodeOddsNormal: 30, UltraCodeOddsGood: 40, MonsterSkillBonus: 7, MonsterFreezeDivisor: 4,
		MonsterColdDivisor: 4, AiCurseDivisor: 4, LifeStealDivisor: 3, ManaStealDivisor: 3, UniqueDamageBonus: 66,
		ChampionDamageBonus: 66, HireableBossDamagePercent: 25, MonsterCEDamagePercent: 20, StaticFieldMin: 50,
		GambleRare: 10000, GambleSet: 100, GambleUnique: 50, GambleUber: 90, GambleUltra: 33,
		ExtraUniqueMonsters: 2},
}

// Classic is the Diablo II (non expansion) table (d2data.mpq, VERIFIED).
var Classic = Table{
	{Name: "Normal", MonsterFreezeDivisor: 1, MonsterColdDivisor: 1, AiCurseDivisor: 1},
	{Name: "Nightmare", ResistPenalty: -20, DeathExpPenalty: 5, UberCodeOddsNormal: 10, UberCodeOddsGood: 20,
		MonsterSkillBonus: 3, MonsterFreezeDivisor: 2, MonsterColdDivisor: 2, AiCurseDivisor: 2},
	{Name: "Hell", ResistPenalty: -50, DeathExpPenalty: 10, UberCodeOddsNormal: 20, UberCodeOddsGood: 40,
		MonsterSkillBonus: 7, MonsterFreezeDivisor: 4, MonsterColdDivisor: 4, AiCurseDivisor: 4},
}

// ParseTable reads a DifficultyLevels.txt (tab separated, header line). Rows
// are matched by position (Normal, Nightmare, Hell). Unknown columns are
// ignored, missing ones stay zero.
func ParseTable(data []byte) (Table, error) {
	var t Table

	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1<<16), 1<<20)

	if !sc.Scan() {
		return t, fmt.Errorf("d2difficulty: empty table")
	}

	header := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
	n := 0

	for sc.Scan() && n < int(Count) {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		cells := strings.Split(line, "\t")
		row := &t[n]

		for i, h := range header {
			if i >= len(cells) {
				break
			}

			if h == "Name" {
				row.Name = cells[i]

				continue
			}

			f := row.field(h)
			if f == nil {
				continue
			}

			v, err := strconv.Atoi(strings.TrimSpace(cells[i]))
			if err != nil {
				return t, fmt.Errorf("d2difficulty: row %d column %s: %v", n, h, err)
			}

			*f = v
		}

		n++
	}

	if n != int(Count) {
		return t, fmt.Errorf("d2difficulty: %d rows, want 3", n)
	}

	return t, nil
}

func (r *Row) field(name string) *int {
	switch name {
	case "ResistPenalty":
		return &r.ResistPenalty
	case "DeathExpPenalty":
		return &r.DeathExpPenalty
	case "UberCodeOddsNormal":
		return &r.UberCodeOddsNormal
	case "UberCodeOddsGood":
		return &r.UberCodeOddsGood
	case "UltraCodeOddsNormal":
		return &r.UltraCodeOddsNormal
	case "UltraCodeOddsGood":
		return &r.UltraCodeOddsGood
	case "MonsterSkillBonus":
		return &r.MonsterSkillBonus
	case "MonsterFreezeDivisor":
		return &r.MonsterFreezeDivisor
	case "MonsterColdDivisor":
		return &r.MonsterColdDivisor
	case "AiCurseDivisor":
		return &r.AiCurseDivisor
	case "LifeStealDivisor":
		return &r.LifeStealDivisor
	case "ManaStealDivisor":
		return &r.ManaStealDivisor
	case "UniqueDamageBonus":
		return &r.UniqueDamageBonus
	case "ChampionDamageBonus":
		return &r.ChampionDamageBonus
	case "HireableBossDamagePercent":
		return &r.HireableBossDamagePercent
	case "MonsterCEDamagePercent":
		return &r.MonsterCEDamagePercent
	case "StaticFieldMin":
		return &r.StaticFieldMin
	case "GambleRare":
		return &r.GambleRare
	case "GambleSet":
		return &r.GambleSet
	case "GambleUnique":
		return &r.GambleUnique
	case "GambleUber":
		return &r.GambleUber
	case "GambleUltra":
		return &r.GambleUltra
	case "ExtraUniqueMonsters":
		return &r.ExtraUniqueMonsters
	}

	return nil
}

// ---------------------------------------------------------------------------
// Monster scaling

// ScaleStat is MATH_MulDiv(base, ratio, 100): truncating integer division
// (VERIFIED, 0x47f2c0; the 64-bit path is only taken for huge operands).
func ScaleStat(base, ratioPct int) int { return int(int64(base) * int64(ratioPct) / 100) }

// MonLvlIndex is the monlvl.txt row used for a monster level: the level
// clamped to the last row (VERIFIED, 0x6551e0). ok is false for a negative
// level or an empty table, where the game returns "no stats".
func MonLvlIndex(level, rows int) (idx int, ok bool) {
	if level < 0 || rows <= 0 {
		return 0, false
	}

	if level > rows-1 {
		level = rows - 1
	}

	return level, true
}

// MonsterStat is one scaled stat: the monstats raw value for noRatio classes,
// otherwise the monlvl.txt value times the monstats ratio in percent.
func MonsterStat(noRatio bool, monlvl, ratioPct, raw int) int {
	if noRatio {
		return raw
	}

	return ScaleStat(monlvl, ratioPct)
}

// ---------------------------------------------------------------------------
// Penalties

// ResistPenalty is the resistance added to the player's resistances: the
// ResistPenalty column in Lord of Destruction (0, -40, -100), -20/-50 in
// classic. Damage resist and magic resist are exempt (VERIFIED, 0x579b10).
func ResistPenalty(classic bool, l Level) int {
	if classic {
		return Classic.Row(l).ResistPenalty
	}

	return Default.Row(l).ResistPenalty
}

// DeathExpPenalty is the percent of the experience span of the current level
// lost on death (VERIFIED column: 0, 5, 10).
func DeathExpPenalty(l Level) int { return Default.Row(l).DeathExpPenalty }

// ColdDivisor divides the length of the chill a player puts on monsters, and
// FreezeDivisor the freeze length: MonsterColdDivisor / MonsterFreezeDivisor.
func ColdDivisor(l Level) int { return Default.Row(l).MonsterColdDivisor }

// FreezeDivisor see ColdDivisor.
func FreezeDivisor(l Level) int { return Default.Row(l).MonsterFreezeDivisor }

// ---------------------------------------------------------------------------
// Items

// UpgradesTreasureClass reports whether a monster's treasure class is moved
// along its level group by the monster level: only in the expansion, above
// Normal, for monsters (VERIFIED, ITEMGEN_DropFromTreasureClass 0x558d80).
func UpgradesTreasureClass(expansion bool, l Level, killedIsMonster bool) bool {
	return expansion && l != Normal && killedIsMonster
}

// MaxSockets caps the sockets of a rolled item: 3 in Normal, 4 in Nightmare,
// 6 in Hell (VERIFIED, ITEMGEN_RollSockets).
func MaxSockets(l Level, wanted int) int {
	limit := Pick(l, 3, 4, 6)
	if wanted > limit {
		return limit
	}

	return wanted
}

// ChestSuffix is the suffix of "Act N (N) Chest A"-style treasure class names
// ("", " (N)", " (H)").
func ChestSuffix(l Level) string {
	switch Clamp(int(l)) {
	case Nightmare:
		return " (N)"
	case Hell:
		return " (H)"
	}

	return ""
}

// MercDifficulty is the 1-based difficulty column of hireling.txt
// (VERIFIED, hirelings.md).
func MercDifficulty(l Level) int { return int(Clamp(int(l))) + 1 }
