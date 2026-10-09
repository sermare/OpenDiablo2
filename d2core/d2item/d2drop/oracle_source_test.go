package d2drop

import (
	"strconv"
	"testing"
)

// Goldens of the treasure class choice, produced by running ITEMGEN_DropMonsterTreasure,
// ITEMGEN_DropFromTreasureClass and the chest code of the real game over fake
// units and synthetic tables (class ids are made-up numbers).

type monCase struct {
	D   int       `json:"d"`
	X   int       `json:"x"`
	FL  int       `json:"fl"`
	SI  int       `json:"si"`
	QID int       `json:"qid"`
	QCP int       `json:"qcp"`
	K   string    `json:"k"`
	PL  int       `json:"pl"`
	QS  [3]bool   `json:"qs"`
	MS  [3][4]int `json:"ms"`
	TC  *int      `json:"tc"`
}

type upCase struct {
	D, X, FL, UT, ML int
	LVL              *int `json:"lvl"`
}

type chestCase struct {
	D    int    `json:"d"`
	X    int    `json:"x"`
	Act  int    `json:"act"`
	Area int    `json:"area"`
	Lo   int    `json:"lo"`
	Hi   int    `json:"hi"`
	A    []int  `json:"a"`
	TC   string `json:"tc"`
}

type sourceGolden struct {
	Mon   []monCase   `json:"mon"`
	Up    []upCase    `json:"up"`
	Chest []chestCase `json:"chest"`
	Acts  [5][2]int   `json:"acts"`
}

func TestOracleMonsterTreasureClass(t *testing.T) {
	var g sourceGolden

	readGolden(t, "monster.json", &g)

	name := func(v int) string {
		if v == 0 {
			return ""
		}

		return strconv.Itoa(v)
	}

	bad := 0

	for _, c := range g.Mon {
		in := MonsterTreasureInput{
			Difficulty: c.D, Flags: c.FL, QuestID: c.QID, QuestCP: c.QCP,
			KillerIsPlayer: c.K == "player" && c.PL != 0, QuestStates: c.QS,
		}

		for d := range in.TCs {
			for k := range in.TCs[d] {
				in.TCs[d][k] = name(c.MS[d][k])
			}
		}

		// super unique rows 0..2 exist (class ids 0x200 + 16*index + difficulty + 1)
		if c.FL&0x02 != 0 && c.SI >= 0 {
			in.IsSuperUnique = true

			if c.SI < 3 {
				in.SuperUnique = &[3]string{name(0x200 + c.SI*16 + 1), name(0x200 + c.SI*16 + 2), name(0x200 + c.SI*16 + 3)}
			}
		}

		want := ""
		if c.TC != nil {
			want = name(*c.TC)
		}

		if got := MonsterTreasureClass(in); got != want {
			bad++

			if bad <= 5 {
				t.Errorf("case %+v: got %q, real %q", c, got, want)
			}
		}
	}

	t.Logf("%d monsters, %d mismatches", len(g.Mon), bad)
}

func TestOracleMonsterUpgradeLevel(t *testing.T) {
	var g sourceGolden

	readGolden(t, "monster.json", &g)

	bad := 0

	for _, c := range g.Up {
		want := 0
		if c.LVL != nil {
			want = *c.LVL
		}

		if got := MonsterUpgradeLevel(c.X != 0, c.D, c.UT == 1, c.FL, c.ML); got != want {
			bad++

			if bad <= 5 {
				t.Errorf("case %+v: got %d, real %d", c, got, want)
			}
		}
	}

	t.Logf("%d monsters, %d mismatches", len(g.Up), bad)
}

func TestOracleChest(t *testing.T) {
	var g sourceGolden

	readGolden(t, "monster.json", &g)

	if g.Acts != ChestActLevels {
		t.Errorf("act level table %v, real %v", ChestActLevels, g.Acts)
	}

	bad := 0

	for _, c := range g.Chest {
		first, last := ChestActLevels[c.Act][0], ChestActLevels[c.Act][1]
		got := ChestTreasureClass(c.D, c.Act, c.Area, func(id int) int {
			switch id {
			case first:
				return c.Lo
			case last:
				return c.Hi
			}

			return 1
		})

		if got.Class != c.TC || got.Tier != c.A[3] {
			bad++

			if bad <= 5 {
				t.Errorf("case %+v: got %+v", c, got)
			}
		}
	}

	t.Logf("%d chests, %d mismatches", len(g.Chest), bad)
}
