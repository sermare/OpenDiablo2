package d2boss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

func loadAct5(t *testing.T) ([]SuperUniqueInfo, map[string]MonsterInfo, map[int]LevelInfo) {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	dir := filepath.Join(root, "monsters", "patch_d2")

	open := func(name string) *os.File {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Skip(err)
		}

		t.Cleanup(func() { f.Close() })

		return f
	}

	sup, err := ReadSuperUniques(open("superuniques.txt"))
	if err != nil {
		t.Fatal(err)
	}

	mons, err := ReadMonsters(open("monstats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	lv, err := ReadLevels(open("levels.txt"))
	if err != nil {
		t.Fatal(err)
	}

	return sup, mons, lv
}

// The built-in Waves are exactly the rows "Baal Subject 1..5" of the tables.
func TestWavesMatchTables(t *testing.T) {
	sup, mons, _ := loadAct5(t)

	w, err := WavesFromTables(sup, mons)
	if err != nil {
		t.Fatal(err)
	}

	for i := range w {
		if w[i].Super != Waves[i].Super || w[i].Class != Waves[i].Class || w[i].Group != Waves[i].Group {
			t.Errorf("wave %d: table %+v, built-in %+v", i, w[i], Waves[i])
		}

		if sup[w[i].Super].MinGroup != sup[w[i].Super].MaxGroup {
			t.Errorf("wave %d: MinGrp != MaxGrp", i)
		}
	}
}

// The classes of the throne chain and the end bosses and their AIs.
func TestAct5ClassesAndAIs(t *testing.T) {
	_, mons, _ := loadAct5(t)

	for _, c := range []struct {
		id    string
		class int
		ai    string
	}{
		{"baalthrone", 543, "BaalThrone"},
		{"baalcrab", 544, "BaalCrab"},
		{"baalcrabstairs", 559, "BaalToStairs"},
		{"baalclone", 570, "BaalCrabClone"},
		{"nihlathakboss", d2quest.NPCNihlathakBoss, ""},
		{"ancientstatue1", d2quest.NPCAncientStatue1, "AncientStatue"},
		{"ancientstatue2", d2quest.NPCAncientStatue2, "AncientStatue"},
		{"ancientstatue3", d2quest.NPCAncientStatue3, "AncientStatue"},
		{"ancientbarb1", d2quest.NPCAncient1, "Ancient"},
		{"ancientbarb2", d2quest.NPCAncient2, "Ancient"},
		{"ancientbarb3", d2quest.NPCAncient3, "Ancient"},
		{"baalthrone", 543, "BaalThrone"},
	} {
		m, ok := mons[c.id]
		if !ok || m.Class != c.class || (c.ai != "" && !strings.EqualFold(m.AI, c.ai)) {
			t.Errorf("%s: %+v (want class %d ai %q)", c.id, m, c.class, c.ai)
		}
	}

	if d2quest.NPCBaalCrab != mons["baalcrab"].Class {
		t.Errorf("quest Baal class %d", d2quest.NPCBaalCrab)
	}
}

// Nihlathak is the super unique hcIdx 60 whose class is the quest's boss class.
func TestNihlathakSuperUnique(t *testing.T) {
	sup, mons, _ := loadAct5(t)

	for _, s := range sup {
		if s.Key == "Nihlathak Boss" {
			if s.HcIdx != 60 || mons[s.Class].Class != d2quest.NPCNihlathakBoss || s.MaxGroup != 0 {
				t.Errorf("%+v", s)
			}

			return
		}
	}

	t.Fatal("no Nihlathak Boss row")
}

// Monster levels of the wave leaders grow with the difficulty, and the leaders
// are within the throne level's monster levels' neighbourhood in Hell.
func TestWaveLevelsPerDifficulty(t *testing.T) {
	_, mons, lv := loadAct5(t)

	for i, w := range Waves {
		var id string

		for k, m := range mons {
			if m.Class == w.Class {
				id = k
			}
		}

		a, b, c := WaveMonsterLevel(mons, id, DiffNormal), WaveMonsterLevel(mons, id, DiffNightmare), WaveMonsterLevel(mons, id, DiffHell)
		if !(a > 0 && a < b && b < c) {
			t.Errorf("wave %d (%s): levels %d/%d/%d", i, id, a, b, c)
		}
	}

	if lv[LevelThrone].MonLvl[DiffHell] <= lv[LevelThrone].MonLvl[DiffNormal] {
		t.Errorf("throne level: %v", lv[LevelThrone].MonLvl)
	}
}

// Levels 128..132 need Rite of Passage; the rest of Act 5 does not.
func TestWorldstoneLevelsGatedByRite(t *testing.T) {
	_, _, lv := loadAct5(t)

	none := func(int) bool { return false }
	rite := func(s int) bool { return s == SlotRiteOfPassage }

	for id := LevelWorldstoneKeep1; id <= LevelWorldstone; id++ {
		l := lv[id]
		if l.QuestFlag != SlotRiteOfPassage || l.Open(none) || !l.Open(rite) {
			t.Errorf("level %d %q: %+v", id, l.Name, l)
		}
	}

	for _, id := range []int{d2quest.LevelArreatSummit, LevelNihlathakEntrance, LevelHallsOfVaught, 125, 126, 127} {
		if lv[id].QuestFlag != 0 {
			t.Errorf("level %d is gated: %+v", id, lv[id])
		}
	}

	if lv[LevelHallsOfVaught].Name != "Act 5 - Temple Boss" || lv[LevelNihlathakEntrance].Name != "Act 5 - Temple Entrance" {
		t.Errorf("temple levels: %q %q", lv[LevelNihlathakEntrance].Name, lv[LevelHallsOfVaught].Name)
	}

	if lv[d2quest.LevelThrone].Name != "Act 5 - Throne Room" {
		t.Errorf("throne level %q", lv[d2quest.LevelThrone].Name)
	}
}
