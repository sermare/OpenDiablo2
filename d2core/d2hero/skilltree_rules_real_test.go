package d2hero

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// readTSV reads a tab separated table of $D2_TABLES/skills/patch_d2 as rows
// keyed by column name; the test is skipped when the data is not there. No game
// data is stored in the repository.
func readTSV(t *testing.T, name string) []map[string]string {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	f, err := os.Open(filepath.Join(root, "skills", "patch_d2", name))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var header []string

	var rows []map[string]string

	for sc.Scan() {
		cols := strings.Split(sc.Text(), "\t")
		if header == nil {
			header = cols
			continue
		}

		r := map[string]string{}

		for i, c := range cols {
			if i < len(header) {
				r[header[i]] = c
			}
		}

		rows = append(rows, r)
	}

	return rows
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// realTrees builds the heroes' skills (class skills only) of the real tables,
// keyed by skill id, grouped by class token.
func realTrees(t *testing.T) map[string]map[int]*HeroSkill {
	t.Helper()

	desc := map[string]*d2records.SkillDescriptionRecord{}

	for _, r := range readTSV(t, "skilldesc.txt") {
		desc[r["skilldesc"]] = &d2records.SkillDescriptionRecord{
			Name: r["skilldesc"], SkillPage: atoi(r["SkillPage"]), SkillRow: atoi(r["SkillRow"]), SkillColumn: atoi(r["SkillColumn"]),
		}
	}

	trees := map[string]map[int]*HeroSkill{}

	for _, r := range readTSV(t, "skills.txt") {
		if r["charclass"] == "" || r["skill"] == "" {
			continue
		}

		rec := &d2records.SkillRecord{
			ID: atoi(r["Id"]), Skill: r["skill"], Charclass: r["charclass"], Reqlevel: atoi(r["reqlevel"]),
			Maxlvl: atoi(r["maxlvl"]), Reqskill1: r["reqskill1"], Reqskill2: r["reqskill2"], Reqskill3: r["reqskill3"],
			Passive: r["passive"] == "1", Leftskill: r["leftskill"] == "1",
		}

		d := desc[r["skilldesc"]]
		if d == nil {
			t.Fatalf("%s: skilldesc %q missing", rec.Skill, r["skilldesc"])
		}

		if trees[rec.Charclass] == nil {
			trees[rec.Charclass] = map[int]*HeroSkill{}
		}

		trees[rec.Charclass][rec.ID] = &HeroSkill{SkillRecord: rec, SkillDescriptionRecord: d}
	}

	return trees
}

// Tree layout rules of the stock 1.14 tables (verified against the table
// data; whether the executable draws exactly this grid is not checked): each
// class has 30 skills in 3 tabs (SkillPage 1..3) of 10, on a 6 row x 3 column
// grid with no two skills in one cell, every prerequisite is a skill of the
// same class and tab, and the class token the engine derives from the hero
// matches the charclass column.
func TestRealTreeLayout(t *testing.T) {
	trees := realTrees(t)

	heroes := []d2enum.Hero{
		d2enum.HeroAmazon, d2enum.HeroSorceress, d2enum.HeroNecromancer, d2enum.HeroPaladin,
		d2enum.HeroBarbarian, d2enum.HeroDruid, d2enum.HeroAssassin,
	}

	for _, h := range heroes {
		token := strings.ToLower(h.GetToken3())

		skills := trees[token]
		if len(skills) != 30 {
			t.Errorf("%s: %d class skills, want 30", token, len(skills))
			continue
		}

		perTab := map[int]int{}
		cells := map[[3]int]string{}
		byName := map[string]*HeroSkill{}

		for _, s := range skills {
			byName[s.Skill] = s
		}

		for _, s := range skills {
			perTab[s.SkillPage]++

			cell := [3]int{s.SkillPage, s.SkillRow, s.SkillColumn}
			if s.SkillPage < 1 || s.SkillPage > 3 || s.SkillRow < 1 || s.SkillRow > 6 || s.SkillColumn < 1 || s.SkillColumn > 3 {
				t.Errorf("%s/%s: cell %v outside 3 tabs x 6 rows x 3 columns", token, s.Skill, cell)
			}

			if other, dup := cells[cell]; dup {
				t.Errorf("%s: %s and %s share cell %v", token, s.Skill, other, cell)
			}

			cells[cell] = s.Skill

			if s.Maxlvl != 20 {
				t.Errorf("%s/%s: maxlvl %d, want 20", token, s.Skill, s.Maxlvl)
			}

			for _, req := range []string{s.Reqskill1, s.Reqskill2, s.Reqskill3} {
				if req == "" {
					continue
				}

				r := byName[req]
				if r == nil {
					t.Errorf("%s/%s: prerequisite %q is not a skill of the class", token, s.Skill, req)
					continue
				}

				if r.SkillPage != s.SkillPage {
					t.Errorf("%s/%s: prerequisite %q is on another tab", token, s.Skill, req)
				}
			}
		}

		for tab := 1; tab <= 3; tab++ {
			if perTab[tab] != 10 {
				t.Errorf("%s: tab %d has %d skills, want 10", token, tab, perTab[tab])
			}
		}
	}
}

// reqlevel follows the row of the tree: 1, 6, 12, 18, 24, 30 for rows 1..6.
// There is no per-point level column in skills.txt, so the character level gate
// applies to the first point only (the skill level is bounded by maxlvl 20 and
// the unused points). UNVERIFIED against the executable: whether the original
// tree also refuses later points below some character level.
func TestRealReqLevelByRow(t *testing.T) {
	want := map[int]int{1: 1, 2: 6, 3: 12, 4: 18, 5: 24, 6: 30}

	for token, skills := range realTrees(t) {
		for _, s := range skills {
			if s.Reqlevel != want[s.SkillRow] {
				t.Errorf("%s/%s: reqlevel %d on row %d, want %d", token, s.Skill, s.Reqlevel, s.SkillRow, want[s.SkillRow])
			}
		}
	}

	for _, r := range readTSV(t, "skills.txt") {
		for _, col := range []string{"lvlreq", "levelreq", "reqlvl"} {
			if _, has := r[col]; has {
				t.Fatalf("skills.txt has a %q column, the per-point gate needs modelling", col)
			}
		}

		break
	}
}

// Spot values of the shipped tables the rules were checked against.
func TestRealPrerequisites(t *testing.T) {
	trees := realTrees(t)

	tests := []struct {
		class, skill string
		reqlevel     int
		reqs         []string
	}{
		{"sor", "Fire Bolt", 1, nil},
		{"sor", "Fire Ball", 12, []string{"Fire Bolt"}},
		{"sor", "Teleport", 18, []string{"Telekinesis"}},
		{"sor", "Meteor", 24, []string{"Fire Ball", "Fire Wall"}},
		{"sor", "Hydra", 30, []string{"Enchant"}},
		{"nec", "Skeleton Mastery", 1, []string{"Raise Skeleton"}},
	}

	for _, tt := range tests {
		var got *HeroSkill

		for _, s := range trees[tt.class] {
			if s.Skill == tt.skill {
				got = s
			}
		}

		if got == nil {
			t.Errorf("%s not found", tt.skill)
			continue
		}

		var reqs []string

		for _, r := range []string{got.Reqskill1, got.Reqskill2, got.Reqskill3} {
			if r != "" {
				reqs = append(reqs, r)
			}
		}

		if got.Reqlevel != tt.reqlevel || strings.Join(reqs, ",") != strings.Join(tt.reqs, ",") {
			t.Errorf("%s: reqlevel %d reqs %v, want %d %v", tt.skill, got.Reqlevel, reqs, tt.reqlevel, tt.reqs)
		}
	}
}

// Replays the rules on every real tree: at character level 1 only the
// level 1 skills without prerequisites can start, with unlimited points and
// level 99 every skill can be raised to 20 and not beyond, a skill of another
// class is refused, and nothing ever lowers a skill (there is no respec API).
func TestRealSpendReplay(t *testing.T) {
	for token, skills := range realTrees(t) {
		ids := make([]int, 0, len(skills))
		for id := range skills {
			ids = append(ids, id)
		}

		sort.Ints(ids)

		// level 1: exactly the skills with reqlevel <= 1 and no prerequisite can start
		stats := &HeroStatsState{Level: 1, SkillPoints: 1000}

		for _, id := range ids {
			s := skills[id]
			err := SpendSkillPoint(skills, stats, token, id)

			startable := s.Reqlevel <= 1 && s.Reqskill1 == "" && s.Reqskill2 == "" && s.Reqskill3 == ""
			if startable && err != nil {
				t.Errorf("%s/%s: level 1 start refused: %v", token, s.Skill, err)
			}

			if !startable && err == nil && s.SkillPoints == 1 && !prereqsMetBefore(skills, s) {
				t.Errorf("%s/%s: started at level 1 without its requirements", token, s.Skill)
			}
		}

		// level 99: everything reaches 20 exactly
		for _, s := range skills {
			s.SetPoints(0)
		}

		stats = &HeroStatsState{Level: 99, SkillPoints: 1000}
		spent := 0

		for pass := 0; pass < 25; pass++ {
			for _, id := range ids {
				if SpendSkillPoint(skills, stats, token, id) == nil {
					spent++
				}
			}
		}

		for _, s := range skills {
			if s.SkillPoints != 20 {
				t.Errorf("%s/%s: %d points after the full replay, want 20", token, s.Skill, s.SkillPoints)
			}
		}

		if stats.SkillPoints != 1000-spent || spent != 30*20 {
			t.Errorf("%s: spent %d (unused %d), want 600", token, spent, stats.SkillPoints)
		}

		if err := SpendSkillPoint(skills, stats, token, ids[0]); !errors.Is(err, ErrLevelCap) {
			t.Errorf("%s: 21st point: %v, want ErrLevelCap", token, err)
		}

		// another class' skill is refused
		other := map[string]string{"sor": "ama", "ama": "sor", "nec": "pal", "pal": "nec", "bar": "dru", "dru": "bar", "ass": "bar"}[token]
		if err := SpendSkillPoint(skills, &HeroStatsState{Level: 99, SkillPoints: 5}, other, ids[0]); !errors.Is(err, ErrWrongClass) {
			t.Errorf("%s: spending as class %s: %v, want ErrWrongClass", token, other, err)
		}
	}
}

// prereqsMetBefore reports whether the skill's requirements hold in the
// current state ignoring the point just spent (used for the level 1 pass,
// where an earlier skill of the same loop may have unlocked a later one).
func prereqsMetBefore(skills map[int]*HeroSkill, s *HeroSkill) bool {
	if s.Reqlevel > 1 {
		return false
	}

	for _, req := range []string{s.Reqskill1, s.Reqskill2, s.Reqskill3} {
		if req == "" {
			continue
		}

		ok := false

		for _, o := range skills {
			if o.Skill == req && o.SkillPoints >= 1 {
				ok = true
			}
		}

		if !ok {
			return false
		}
	}

	return true
}
