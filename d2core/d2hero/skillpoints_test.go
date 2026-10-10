package d2hero

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func sorcRecords() map[int]*HeroSkill {
	rec := func(id int, name string, lvl int, r1, r2 string, max int) *HeroSkill {
		return &HeroSkill{SkillRecord: &d2records.SkillRecord{
			ID: id, Skill: name, Charclass: "sor", Reqlevel: lvl, Reqskill1: r1, Reqskill2: r2, Maxlvl: max,
		}}
	}

	return map[int]*HeroSkill{
		36: rec(36, "Fire Bolt", 1, "", "", 20),
		47: rec(47, "Fire Ball", 12, "Fire Bolt", "", 20),
		51: rec(51, "Fire Wall", 18, "Blaze", "", 20),
		52: rec(52, "Blaze", 18, "Fire Bolt", "", 20),
		56: rec(56, "Meteor", 24, "Fire Ball", "Fire Wall", 20),
		3:  {SkillRecord: &d2records.SkillRecord{ID: 3, Skill: "Throw", Charclass: ""}},
		99: rec(99, "Capped", 1, "", "", 2),
	}
}

func TestSpendSkillPoint(t *testing.T) {
	skills := sorcRecords()
	stats := &HeroStatsState{Level: 1, SkillPoints: 40}

	type step struct {
		name  string
		id    int
		level int // character level before the step; 0 keeps it
		want  error
		isReq string // expected missing prerequisite, or "lvl" for the level requirement
	}

	steps := []step{
		{name: "first point in a level 1 skill", id: 36},
		{name: "level requirement", id: 47, isReq: "lvl"},
		{name: "prerequisite skill missing", id: 51, level: 18, isReq: "Blaze"},
		{name: "level ok, reqskill1 ok", id: 47, level: 12},
		{name: "second point needs one more character level (reqlevel + points)", id: 47, isReq: "lvl"},
		{name: "second point at level 13", id: 47, level: 13},
		{name: "two prerequisites, one missing", id: 56, level: 24, isReq: "Fire Wall"},
		{name: "other class skill", id: 3, want: ErrWrongClass},
		{name: "cap point 1", id: 99},
		{name: "cap point 2", id: 99},
		{name: "cap reached", id: 99, want: ErrLevelCap},
	}

	for _, st := range steps {
		if st.level != 0 {
			stats.Level = st.level
		}

		before := stats.SkillPoints
		err := SpendSkillPoint(skills, stats, "sor", st.id)

		switch {
		case st.want != nil:
			if !errors.Is(err, st.want) {
				t.Errorf("%s: err = %v, want %v", st.name, err, st.want)
			}
		case st.isReq != "":
			var re *SkillReqError
			if !errors.As(err, &re) {
				t.Errorf("%s: err = %v, want a SkillReqError", st.name, err)
				break
			}

			if (st.isReq == "lvl") != (re.Missing == "") || (re.Missing != "" && re.Missing != st.isReq) {
				t.Errorf("%s: %v", st.name, err)
			}
		default:
			if err != nil {
				t.Errorf("%s: unexpected error %v", st.name, err)
			}
		}

		if err != nil && stats.SkillPoints != before {
			t.Errorf("%s: a refused spend changed the unused points (%d -> %d)", st.name, before, stats.SkillPoints)
		}
	}

	if skills[36].SkillPoints != 1 || skills[47].SkillPoints != 2 || skills[99].SkillPoints != 2 {
		t.Errorf("points: bolt=%d ball=%d capped=%d", skills[36].SkillPoints, skills[47].SkillPoints, skills[99].SkillPoints)
	}

	// the copy a save keeps follows
	if skills[47].Shallow == nil || skills[47].Shallow.SkillPoints != 2 {
		t.Errorf("Shallow not updated: %+v", skills[47].Shallow)
	}

	if stats.SkillPoints != 40-5 {
		t.Errorf("unused points = %d, want 35", stats.SkillPoints)
	}
}

func TestSpendNeedsUnusedPoints(t *testing.T) {
	skills := sorcRecords()
	stats := &HeroStatsState{Level: 30}

	if err := SpendSkillPoint(skills, stats, "sor", 36); !errors.Is(err, ErrNoPoints) {
		t.Fatalf("err = %v, want ErrNoPoints", err)
	}
}

func TestGrantLevelUp(t *testing.T) {
	tests := []struct {
		levels                int
		wantLvl, wantSP, want int
	}{
		{0, 5, 2, 10},
		{1, 6, 3, 15},
		{3, 8, 5, 25},
		{-1, 5, 2, 10},
	}

	for _, tt := range tests {
		s := &HeroStatsState{Level: 5, SkillPoints: 2, StatsPoints: 10}
		GrantLevelUp(s, tt.levels)

		if s.Level != tt.wantLvl || s.SkillPoints != tt.wantSP || s.StatsPoints != tt.want {
			t.Errorf("levels=%d: level=%d sp=%d stat=%d", tt.levels, s.Level, s.SkillPoints, s.StatsPoints)
		}
	}
}
