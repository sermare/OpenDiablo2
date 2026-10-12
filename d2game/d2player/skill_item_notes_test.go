package d2player

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

func TestSkillItemNotes(t *testing.T) {
	tr := func(k string) string { return k } // unknown keys: the English fallbacks are used

	if !SkillItemOnly(0, 3) || SkillItemOnly(1, 4) || SkillItemOnly(0, 0) {
		t.Fatal("item only")
	}

	if !SkillNextBlockHidden(20, 20) || SkillNextBlockHidden(19, 20) || SkillNextBlockHidden(5, 0) {
		t.Fatal("next block")
	}

	got := SkillTooltipFooter(tr, 0, 2, true)
	if want := []string{"", "You have been granted this skill by an item"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}

	got = SkillTooltipFooter(tr, 0, 0, false)
	if want := []string{"", "You have not learned this skill yet"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}

	if len(SkillTooltipFooter(tr, 3, 3, true)) != 0 {
		t.Fatal("a learned skill has no footer")
	}

	lines := []string{"Name", "Current Skill Level: 2", "x"}
	MarkItemLevel(tr, lines, "Current Skill Level: ", 0, 2)

	if lines[1] != "Current Skill Level: 2 (item)" {
		t.Fatalf("got %q", lines[1])
	}

	lines = []string{"Current Skill Level: 2"}
	MarkItemLevel(tr, lines, "Current Skill Level: ", 1, 2)

	if lines[0] != "Current Skill Level: 2" {
		t.Fatal("no mark with hard points")
	}
}

func TestAnyClassSkillLearned(t *testing.T) {
	m := map[int]*d2hero.HeroSkill{1: {}, 2: nil}
	if anyClassSkillLearned(m) {
		t.Fatal("none learned")
	}

	m[3] = &d2hero.HeroSkill{SkillPoints: 1}
	if !anyClassSkillLearned(m) {
		t.Fatal("one learned")
	}
}
