package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// sk builds a hero skill the way skills.txt and skilldesc.txt describe the
// Sorceress ones.
func sk(id int, name string, points, page, row, col, listRow int, leftOK, passive bool) *d2hero.HeroSkill {
	class := "sor"
	if id <= 1 { // Attack and Kick are general skills
		class = ""
	}

	return &d2hero.HeroSkill{
		SkillRecord:            &d2records.SkillRecord{ID: id, Skill: name, Charclass: class, Leftskill: leftOK, Passive: passive},
		SkillDescriptionRecord: &d2records.SkillDescriptionRecord{SkillPage: page, SkillRow: row, SkillColumn: col, ListRow: listRow},
		SkillPoints:            points,
	}
}

func sorcSkills() map[int]*d2hero.HeroSkill {
	list := []*d2hero.HeroSkill{
		sk(0, "Attack", 1, 0, 0, 0, 0, true, false),
		sk(1, "Kick", 1, 0, 0, 0, -1, false, false), // hidden: ListRow -1
		sk(47, "Fire Ball", 3, 1, 3, 2, 1, true, false),
		sk(36, "Fire Bolt", 1, 1, 1, 2, 1, true, false),
		sk(41, "Charged Bolt", 0, 2, 1, 2, 2, true, false), // not learned
		sk(42, "Static Field", 2, 2, 2, 1, 2, true, false),
		sk(54, "Teleport", 1, 2, 4, 3, 2, false, false), // right button only
		sk(37, "Warmth", 5, 3, 1, 1, 3, true, true),     // passive
		sk(59, "Blizzard", 1, 3, 5, 1, 3, true, false),
	}

	m := map[int]*d2hero.HeroSkill{}
	for _, s := range list {
		m[s.ID] = s
	}

	return m
}

func names(cells []popupCell) [][2]string {
	var out [][2]string
	for _, c := range cells {
		out = append(out, [2]string{string(rune('0' + c.Row)), c.Skill.Skill})
	}

	return out
}

func TestLayoutSkillPopupRightButton(t *testing.T) {
	cells := layoutSkillPopup(sorcSkills(), false)

	type want struct {
		name     string
		row, col int
		x, y     int
	}

	// rows are the pages that have skills (0, 1, 2, 3 -> rows 0..3), ordered inside by
	// SkillRow; the right popup is anchored at x=720 and grows to the left
	wants := []want{
		{"Attack", 0, 0, 720 - 48, 465},
		{"Fire Bolt", 1, 0, 720 - 48, 465 - 48},
		{"Fire Ball", 1, 1, 720 - 96, 465 - 48},
		{"Static Field", 2, 0, 720 - 48, 465 - 96},
		{"Teleport", 2, 1, 720 - 96, 465 - 96},
		{"Blizzard", 3, 0, 720 - 48, 465 - 144},
	}

	if len(cells) != len(wants) {
		t.Fatalf("got %d cells %v, want %d", len(cells), names(cells), len(wants))
	}

	for i, w := range wants {
		c := cells[i]
		if c.Skill.Skill != w.name || c.Row != w.row || c.Col != w.col || c.X != w.x || c.Y != w.y {
			t.Errorf("cell %d = %s row=%d col=%d at (%d,%d), want %s row=%d col=%d at (%d,%d)", i,
				c.Skill.Skill, c.Row, c.Col, c.X, c.Y, w.name, w.row, w.col, w.x, w.y)
		}
	}
}

func TestLayoutSkillPopupLeftButton(t *testing.T) {
	cells := layoutSkillPopup(sorcSkills(), true)

	// no Teleport (right only), no passive, no unlearned skill, no Kick
	var got []string
	for _, c := range cells {
		got = append(got, c.Skill.Skill)
	}

	want := []string{"Attack", "Fire Bolt", "Fire Ball", "Static Field", "Blizzard"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	// the left popup grows to the right from x=90
	if c := cells[2]; c.X != 90+48 || c.Y != 465-48 {
		t.Errorf("Fire Ball at (%d,%d), want (138,417)", c.X, c.Y)
	}

	if c := cellAt(cells, 100, 470); c == nil || c.Skill.Skill != "Attack" {
		t.Errorf("cellAt(100,470) = %v", c)
	}

	if c := cellAt(cells, 100, 200); c != nil {
		t.Errorf("cellAt above the popup = %v", c.Skill.Skill)
	}
}

func TestLayoutSkillPopupHidesItemSkills(t *testing.T) {
	// Throw, Unsummon, scrolls and books are general skills that depend on items/minions
	m := map[int]*d2hero.HeroSkill{
		0:   sk(0, "Attack", 1, 0, 0, 0, 0, true, false),
		2:   {SkillRecord: &d2records.SkillRecord{ID: 2, Skill: "Throw", Leftskill: true}, SkillDescriptionRecord: &d2records.SkillDescriptionRecord{}, SkillPoints: 1},
		219: {SkillRecord: &d2records.SkillRecord{ID: 219, Skill: "Scroll of Identify"}, SkillDescriptionRecord: &d2records.SkillDescriptionRecord{}, SkillPoints: 1},
	}

	cells := layoutSkillPopup(m, false)
	if len(cells) != 1 || cells[0].Skill.Skill != "Attack" {
		t.Fatalf("cells = %v", names(cells))
	}
}

func TestLayoutSkillPopupPages(t *testing.T) {
	// a page without a skill does not leave a gap; a page outside 0..4 is dropped
	m := map[int]*d2hero.HeroSkill{
		0:  sk(0, "Attack", 1, 0, 0, 0, 0, true, false),
		50: sk(50, "T3", 1, 3, 1, 1, 3, true, false),
		60: sk(60, "Odd", 1, 7, 1, 1, 1, true, false),
	}

	cells := layoutSkillPopup(m, false)
	if len(cells) != 2 || cells[1].Skill.Skill != "T3" || cells[1].Row != 1 {
		t.Fatalf("cells = %v", names(cells))
	}
}
