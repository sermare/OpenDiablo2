package d2boss

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Table-driven Act 5 endgame rules. The numbers come from the player's own
// tables (SuperUniques.txt, MonStats.txt, Levels.txt of patch_d2); nothing of
// them is committed. The loaders below are used by tests (D2_TABLES) and may be
// used by an engine that wants to replace the built-in Waves.

// SlotRiteOfPassage is the quest record slot of Rite of Passage; levels.txt
// "QuestFlagEx" 39 on levels 128..132 (Worldstone Keep levels 1-3, Throne of
// Destruction, Worldstone Chamber) is this slot: the levels are closed until
// the quest is done (UNVERIFIED in the exe: only the table column is read).
const SlotRiteOfPassage = 39

// Levels of the Act 5 endgame (levels.txt Id; the names are the table's).
const (
	LevelNihlathakEntrance = 121 // "Temple Entrance" (Nihlathak's Temple)
	LevelHallsOfVaught     = 124 // "Temple Boss"; Nihlathak's super unique is placed here (UNVERIFIED)
	LevelWorldstoneKeep1   = 128
	LevelWorldstoneKeep3   = 130
)

// Difficulty indices of the per-difficulty columns.
const (
	DiffNormal = iota
	DiffNightmare
	DiffHell
)

type tsv struct {
	col  map[string]int
	rows [][]string
}

func readTSV(r io.Reader) (*tsv, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)

	t := &tsv{col: map[string]int{}}

	for first := true; sc.Scan(); {
		f := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
		if first {
			for i, h := range f {
				if _, dup := t.col[h]; !dup {
					t.col[h] = i
				}
			}

			first = false

			continue
		}

		t.rows = append(t.rows, f)
	}

	return t, sc.Err()
}

func (t *tsv) get(row []string, name string) string {
	i, ok := t.col[name]
	if !ok || i >= len(row) {
		return ""
	}

	return row[i]
}

func (t *tsv) num(row []string, name string) int {
	n, _ := strconv.Atoi(t.get(row, name))

	return n
}

// MonsterInfo is a monstats row as far as the endgame rules need it.
type MonsterInfo struct {
	Class    int // hcIdx column = exe class
	ID       string
	AI       string
	Level    [3]int // Level, Level(N), Level(H)
	MinGroup int
	MaxGroup int
}

// ReadMonsters reads monstats.txt keyed by lower-case Id.
func ReadMonsters(r io.Reader) (map[string]MonsterInfo, error) {
	t, err := readTSV(r)
	if err != nil {
		return nil, err
	}

	out := map[string]MonsterInfo{}

	for _, row := range t.rows {
		id := strings.ToLower(t.get(row, "Id"))
		if id == "" || id == "expansion" {
			continue
		}

		c, err := strconv.Atoi(t.get(row, "hcIdx"))
		if err != nil {
			continue
		}

		out[id] = MonsterInfo{Class: c, ID: id, AI: t.get(row, "AI"),
			Level:    [3]int{t.num(row, "Level"), t.num(row, "Level(N)"), t.num(row, "Level(H)")},
			MinGroup: t.num(row, "MinGrp"), MaxGroup: t.num(row, "MaxGrp")}
	}

	return out, nil
}

// SuperUniqueInfo is a SuperUniques.txt row.
type SuperUniqueInfo struct {
	Row      int // 0-based row, the super unique id of the engine
	Key      string
	Class    string
	HcIdx    int
	MinGroup int
	MaxGroup int
	TC       [3]string
}

// ReadSuperUniques reads SuperUniques.txt in file order.
func ReadSuperUniques(r io.Reader) ([]SuperUniqueInfo, error) {
	t, err := readTSV(r)
	if err != nil {
		return nil, err
	}

	var out []SuperUniqueInfo

	for _, row := range t.rows {
		if t.get(row, "Superunique") == "" {
			continue
		}

		out = append(out, SuperUniqueInfo{Row: len(out), Key: t.get(row, "Superunique"), Class: strings.ToLower(t.get(row, "Class")),
			HcIdx: t.num(row, "hcIdx"), MinGroup: t.num(row, "MinGrp"), MaxGroup: t.num(row, "MaxGrp"),
			TC: [3]string{t.get(row, "TC"), t.get(row, "TC(N)"), t.get(row, "TC(H)")}})
	}

	return out, nil
}

// WavesFromTables builds the five throne waves from the tables: the rows
// named "Baal Subject 1..5" of SuperUniques.txt, the class of the row resolved
// through MonStats.txt, the group from MaxGrp (= MinGrp).
func WavesFromTables(sup []SuperUniqueInfo, mons map[string]MonsterInfo) ([5]Wave, error) {
	var out [5]Wave

	for i := range out {
		key := fmt.Sprintf("Baal Subject %d", i+1)
		found := false

		for _, s := range sup {
			if s.Key != key {
				continue
			}

			m, ok := mons[s.Class]
			if !ok {
				return out, fmt.Errorf("%s: class %q not in monstats", key, s.Class)
			}

			out[i] = Wave{Super: s.Row, Class: m.Class, Group: s.MaxGroup, Name: fmt.Sprintf("%s (%s)", key, s.Class)}
			found = true

			break
		}

		if !found {
			return out, fmt.Errorf("%s not in SuperUniques", key)
		}
	}

	return out, nil
}

// LevelInfo is a levels.txt row as far as the endgame rules need it.
type LevelInfo struct {
	ID        int
	Name      string
	QuestFlag int    // slot that must be done to enter (0 = none)
	MonLvl    [3]int // MonLvl1Ex..3Ex (expansion)
}

// ReadLevels reads Levels.txt keyed by level id.
func ReadLevels(r io.Reader) (map[int]LevelInfo, error) {
	t, err := readTSV(r)
	if err != nil {
		return nil, err
	}

	out := map[int]LevelInfo{}

	for _, row := range t.rows {
		id, err := strconv.Atoi(t.get(row, "Id"))
		if err != nil {
			continue
		}

		li := LevelInfo{ID: id, Name: t.get(row, "Name"), QuestFlag: t.num(row, "QuestFlagEx")}
		for d, n := range [3]string{"MonLvl1Ex", "MonLvl2Ex", "MonLvl3Ex"} {
			li.MonLvl[d] = t.num(row, n)
		}

		out[id] = li
	}

	return out, nil
}

// Open reports whether a hero may enter the level, done telling which quest
// slots are finished (levels.txt QuestFlagEx).
func (l LevelInfo) Open(done func(slot int) bool) bool {
	return l.QuestFlag == 0 || done(l.QuestFlag)
}

// WaveMonsterLevel is the level of a monster class in a difficulty (MonStats
// Level / Level(N) / Level(H)).
func WaveMonsterLevel(mons map[string]MonsterInfo, id string, diff int) int {
	if diff < 0 || diff > 2 {
		return 0
	}

	return mons[strings.ToLower(id)].Level[diff]
}
