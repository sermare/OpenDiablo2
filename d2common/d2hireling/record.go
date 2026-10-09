package d2hireling

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// NumSkills is the number of Skill slots per record.
const NumSkills = 6

// SkillSlot is one SkillN/ModeN/ChanceN/ChancePerLevelN/LevelN/LvlPerLvlN group.
type SkillSlot struct {
	Name           string
	Mode           int
	Chance         int
	ChancePerLevel int
	Level          int
	LvlPerLvl      int
}

// Used reports whether the slot holds a skill.
func (s SkillSlot) Used() bool { return s.Name != "" }

// Record is one row of hireling.txt.
type Record struct {
	Hireling, SubType string
	ID, Class, Act    int
	Difficulty        int // 1 normal, 2 nightmare, 3 hell
	Level             int
	Seller            int
	NameFirst         string
	NameLast          string
	Gold, ExpPerLvl   int
	HP, HPPerLvl      int
	Defense, DefLvl   int
	Str, StrLvl       int
	Dex, DexLvl       int
	AR, ARLvl         int
	Share             int
	DmgMin, DmgMax    int
	DmgLvl            int
	Resist, ResistLvl int
	HireDesc          string
	DefaultChance     int
	Skills            [NumSkills]SkillSlot
}

// Table is hireling.txt. Rows of one Id are consecutive and sorted by Level.
type Table struct {
	Rows []*Record
}

// Parse reads the tab separated text of hireling.txt (header row first).
// Empty cells are zero; unknown columns are ignored.
func Parse(r io.Reader) (*Table, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	if !sc.Scan() {
		return nil, errors.New("hireling: empty table")
	}

	cols := map[string]int{}
	for i, h := range strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t") {
		cols[h] = i
	}

	for _, need := range []string{"Id", "Class", "Level", "Seller", "Gold"} {
		if _, ok := cols[need]; !ok {
			return nil, fmt.Errorf("hireling: missing column %q", need)
		}
	}

	t := &Table{}

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		f := strings.Split(line, "\t")
		str := func(n string) string {
			if i, ok := cols[n]; ok && i < len(f) {
				return strings.TrimSpace(f[i])
			}

			return ""
		}
		num := func(n string) int { v, _ := strconv.Atoi(str(n)); return v }

		rec := &Record{
			Hireling: str("Hireling"), SubType: str("SubType"), ID: num("Id"), Class: num("Class"),
			Act: num("Act"), Difficulty: num("Difficulty"), Level: num("Level"), Seller: num("Seller"),
			NameFirst: str("NameFirst"), NameLast: str("NameLast"), Gold: num("Gold"), ExpPerLvl: num("Exp/Lvl"),
			HP: num("HP"), HPPerLvl: num("HP/Lvl"), Defense: num("Defense"), DefLvl: num("Def/Lvl"),
			Str: num("Str"), StrLvl: num("Str/Lvl"), Dex: num("Dex"), DexLvl: num("Dex/Lvl"),
			AR: num("AR"), ARLvl: num("AR/Lvl"), Share: num("Share"),
			DmgMin: num("Dmg-Min"), DmgMax: num("Dmg-Max"), DmgLvl: num("Dmg/Lvl"),
			Resist: num("Resist"), ResistLvl: num("Resist/Lvl"),
			HireDesc: str("HireDesc"), DefaultChance: num("DefaultChance"),
		}

		for i := 0; i < NumSkills; i++ {
			n := strconv.Itoa(i + 1)
			rec.Skills[i] = SkillSlot{
				Name: str("Skill" + n), Mode: num("Mode" + n), Chance: num("Chance" + n),
				ChancePerLevel: num("ChancePerLevel" + n), Level: num("Level" + n), LvlPerLvl: num("LvlPerLvl" + n),
			}
		}

		t.Rows = append(t.Rows, rec)
	}

	return t, sc.Err()
}

// Find is HIRE_FindRecordByIdAndLevel (V): the row of that Id with the
// highest base Level <= level. A level below the first row uses the first row
// (U: the exe's loader asks with level 1 and so must get one).
func (t *Table) Find(id, level int) *Record {
	var best, first *Record

	for _, r := range t.Rows {
		if r.ID != id {
			continue
		}

		if first == nil {
			first = r
		}

		if r.Level <= level {
			best = r
		}
	}

	if best == nil {
		return first
	}

	return best
}

// Candidates returns the rows an offer for a seller and difficulty (1-based)
// can use: the first row of that (seller, difficulty) and every later row
// with the same Level (the SubType variants), V.
func (t *Table) Candidates(seller, difficulty int) []*Record {
	var out []*Record

	for _, r := range t.Rows {
		if r.Seller != seller || r.Difficulty != difficulty {
			continue
		}

		if len(out) > 0 && r.Level != out[0].Level {
			continue
		}

		out = append(out, r)
	}

	return out
}

// Sellers lists the vendor NPC classes that hire out mercenaries.
func (t *Table) Sellers() []int {
	seen := map[int]bool{}

	var out []int

	for _, r := range t.Rows {
		if !seen[r.Seller] {
			seen[r.Seller] = true
			out = append(out, r.Seller)
		}
	}

	return out
}
