package d2difficulty

import (
	"os"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestClampPickString(t *testing.T) {
	for _, c := range []struct {
		in   int
		want Level
	}{{-3, Normal}, {0, Normal}, {1, Nightmare}, {2, Hell}, {9, Hell}} {
		if got := Clamp(c.in); got != c.want {
			t.Errorf("Clamp(%d)=%v want %v", c.in, got, c.want)
		}
	}

	if Pick(Hell, 1, 2, 3) != 3 || Pick(Nightmare, 1, 2, 3) != 2 || Pick(7, 1, 2, 3) != 3 {
		t.Error("Pick")
	}

	if Hell.String() != "Hell" || Level(5).String() != "Difficulty(5)" {
		t.Error("String")
	}
}

func TestScaleStat(t *testing.T) {
	// hand computations: truncating MulDiv(base, ratio, 100)
	for _, c := range []struct{ base, ratio, want int }{
		{830, 25, 207}, // 207.5 truncates
		{1107, 100, 1107},
		{7, 25, 1}, // 1.75 -> 1
		{3, 33, 0}, // 0.99 -> 0
		{216, 150, 324},
		{0, 500, 0},
		{2000000, 5000, 100000000}, // large operands use the 64-bit path
	} {
		if got := ScaleStat(c.base, c.ratio); got != c.want {
			t.Errorf("ScaleStat(%d,%d)=%d want %d", c.base, c.ratio, got, c.want)
		}
	}
}

func TestMonsterStat(t *testing.T) {
	if MonsterStat(true, 830, 25, 77) != 77 {
		t.Error("noRatio uses the raw monstats value")
	}

	if MonsterStat(false, 830, 25, 77) != 207 {
		t.Error("ratio path")
	}
}

func TestMonLvlIndex(t *testing.T) {
	for _, c := range []struct {
		level, rows, idx int
		ok               bool
	}{{0, 111, 0, true}, {110, 111, 110, true}, {300, 111, 110, true}, {-1, 111, 0, false}, {5, 0, 0, false}} {
		idx, ok := MonLvlIndex(c.level, c.rows)
		if idx != c.idx || ok != c.ok {
			t.Errorf("MonLvlIndex(%d,%d)=%d,%v", c.level, c.rows, idx, ok)
		}
	}
}

func TestMonsterLevel(t *testing.T) {
	if MonsterLevel(true, 40, 26) != 40 || MonsterLevel(false, 40, 26) != 26 || MonsterLevel(false, 40, 0) != 40 {
		t.Error("MonsterLevel")
	}
}

func TestPenaltiesAgreeWithCombat(t *testing.T) {
	for l := Normal; l < Count; l++ {
		if got, want := ResistPenalty(false, l), d2combat.LoDResistPenalty(int(l)); got != want {
			t.Errorf("LoD penalty %v: %d vs d2combat %d", l, got, want)
		}

		if got, want := ResistPenalty(true, l), d2combat.ClassicResistPenalty(int(l)); got != want {
			t.Errorf("classic penalty %v: %d vs d2combat %d", l, got, want)
		}
	}

	if DeathExpPenalty(Normal) != 0 || DeathExpPenalty(Nightmare) != 5 || DeathExpPenalty(Hell) != 10 {
		t.Error("death penalty")
	}

	if ColdDivisor(Hell) != 4 || FreezeDivisor(Nightmare) != 2 || FreezeDivisor(Normal) != 1 {
		t.Error("divisors")
	}
}

func TestItemRules(t *testing.T) {
	if UpgradesTreasureClass(true, Normal, true) || !UpgradesTreasureClass(true, Nightmare, true) ||
		UpgradesTreasureClass(false, Hell, true) || UpgradesTreasureClass(true, Hell, false) {
		t.Error("TC upgrade rule")
	}

	for _, c := range []struct {
		l      Level
		wanted int
		want   int
	}{{Normal, 6, 3}, {Nightmare, 6, 4}, {Hell, 6, 6}, {Hell, 2, 2}} {
		if got := MaxSockets(c.l, c.wanted); got != c.want {
			t.Errorf("MaxSockets(%v,%d)=%d want %d", c.l, c.wanted, got, c.want)
		}
	}

	if ChestSuffix(Normal) != "" || ChestSuffix(Nightmare) != " (N)" || ChestSuffix(Hell) != " (H)" {
		t.Error("ChestSuffix")
	}

	if MercDifficulty(Normal) != 1 || MercDifficulty(Hell) != 3 {
		t.Error("MercDifficulty")
	}
}

const patchTable = "Name\tResistPenalty\tDeathExpPenalty\tUberCodeOddsNormal\tUberCodeOddsGood\tUltraCodeOddsNormal\tUltraCodeOddsGood\tMonsterSkillBonus\tMonsterFreezeDivisor\tMonsterColdDivisor\tAiCurseDivisor\tLifeStealDivisor\tManaStealDivisor\tUniqueDamageBonus\tChampionDamageBonus\tHireableBossDamagePercent\tMonsterCEDamagePercent\tStaticFieldMin\tGambleRare\tGambleSet\tGambleUnique\tGambleUber\tGambleUltra\r\n" +
	"Normal\t0\t0\t0\t0\t0\t0\t0\t1\t1\t1\t1\t1\t90\t90\t50\t50\t0\t10000\t100\t50\t90\t33\r\n" +
	"Nightmare\t-40\t5\t10\t20\t0\t0\t3\t2\t2\t2\t2\t2\t75\t75\t35\t35\t33\t10000\t100\t50\t90\t33\r\n" +
	"Hell\t-100\t10\t20\t40\t30\t40\t7\t4\t4\t4\t3\t3\t66\t66\t25\t20\t50\t10000\t100\t50\t90\t33\r\n"

func TestParseTableMatchesDefault(t *testing.T) {
	got, err := ParseTable([]byte(patchTable))
	if err != nil {
		t.Fatal(err)
	}

	for l := Normal; l < Count; l++ {
		want := Default[l]
		want.ExtraUniqueMonsters = 0 // not a column of the patch table

		if got[l] != want {
			t.Errorf("%v: parsed %+v\nbuilt-in %+v", l, got[l], want)
		}
	}

	if _, err := ParseTable([]byte("Name\tResistPenalty\nNormal\tx\n")); err == nil {
		t.Error("bad number accepted")
	}

	if _, err := ParseTable(nil); err == nil {
		t.Error("empty table accepted")
	}
}

// D2_DIFFICULTYLEVELS=<patch_d2 DifficultyLevels.txt> checks the built-in table
// against the real file.
func TestRealTable(t *testing.T) {
	p := os.Getenv("D2_DIFFICULTYLEVELS")
	if p == "" {
		t.Skip("D2_DIFFICULTYLEVELS not set")
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ParseTable(data)
	if err != nil {
		t.Fatal(err)
	}

	for l := Normal; l < Count; l++ {
		want := Default[l]
		want.ExtraUniqueMonsters = got[l].ExtraUniqueMonsters

		if got[l] != want {
			t.Errorf("%v: real %+v built-in %+v", l, got[l], want)
		}
	}
}

func TestProgression(t *testing.T) {
	// the real sample: status 0x0d24 -> progression 13
	if Progression(0x0d24) != 13 {
		t.Fatal("Progression")
	}

	s := WithProgression(0x0d24, 5)
	if s != 0x0524 || Progression(s) != 5 || WithProgression(1, 99)>>8&0x1f != 0x1f {
		t.Errorf("WithProgression %#x", s)
	}
}

func finish(q *d2s.QuestRecord, acts int) {
	for _, a := range [][2]int{{1, 6}, {2, 6}, {3, 6}, {4, 2}, {5, 6}}[:acts] {
		q.SetCompleted(a[0], a[1])
	}
}

func TestUnlock(t *testing.T) {
	var q [3]d2s.QuestRecord

	if u := Unlocked(q, 0, true); u != [3]bool{true, false, false} {
		t.Errorf("fresh hero %v", u)
	}

	finish(&q[0], 4) // Diablo dead, Baal alive: expansion Normal is not finished
	if Unlocked(q, 0, true)[Nightmare] {
		t.Error("expansion: Nightmare needs Baal")
	}

	if !Unlocked(q, 0, false)[Nightmare] {
		t.Error("classic: Diablo ends Normal")
	}

	finish(&q[0], 5)
	if u := Unlocked(q, 0, true); u != [3]bool{true, true, false} {
		t.Errorf("Baal done %v", u)
	}

	if ProgressionOf(q, true) != 5 {
		t.Errorf("ProgressionOf %d", ProgressionOf(q, true))
	}

	finish(&q[1], 3)
	if ProgressionOf(q, true) != 8 || Unlocked(q, 0, true)[Hell] {
		t.Error("NM act 3")
	}

	finish(&q[1], 5)
	if u := Unlocked(q, 0, true); u != [3]bool{true, true, true} || Highest(u) != Hell {
		t.Errorf("both done %v", u)
	}

	// the stored progression alone also unlocks (quests of a stripped save)
	var none [3]d2s.QuestRecord
	if u := Unlocked(none, 5, true); !u[Nightmare] || u[Hell] {
		t.Errorf("progression 5: %v", u)
	}

	if u := Unlocked(none, 13, true); !u[Hell] { // the real level-94 sample
		t.Errorf("progression 13: %v", u)
	}

	if Allowed(Hell, none, 5, true) || !Allowed(Normal, none, 0, true) {
		t.Error("Allowed")
	}

	// acts count in order: Baal alone does not finish anything
	var skip [3]d2s.QuestRecord
	skip[0].SetCompleted(5, 6)
	if Finished(Normal, skip, 0, true) {
		t.Error("out of order")
	}
}
