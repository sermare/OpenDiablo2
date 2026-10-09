package d2hero

import (
	"errors"
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// DefaultMaxSkillLevel is the hard cap of a skill's own points when skills.txt
// leaves maxlvl empty (every class skill of the stock tables says 20).
const DefaultMaxSkillLevel = 20

// SkillPointsPerLevel and StatPointsPerLevel are what a level-up grants.
const (
	SkillPointsPerLevel = 1
	StatPointsPerLevel  = 5
)

// Reasons a skill point cannot be spent.
var (
	ErrNoPoints   = errors.New("no unused skill points")
	ErrWrongClass = errors.New("skill does not belong to this class")
	ErrLevelCap   = errors.New("skill is at its maximum level")
)

// SkillReqError says which prerequisite of a skill is missing.
type SkillReqError struct {
	Skill   string
	MinLvl  int    // character level needed (0 when a skill or stat is missing)
	Missing string // name of the required skill that has no points
	Stat    string // "strength", "dexterity", "energy" or "vitality" when a stat is too low
	StatMin int    // the stat value needed
}

func (e *SkillReqError) Error() string {
	if e.Stat != "" {
		return fmt.Sprintf("%s needs %d %s", e.Skill, e.StatMin, e.Stat)
	}

	if e.Missing != "" {
		return fmt.Sprintf("%s needs a point in %s", e.Skill, e.Missing)
	}

	return fmt.Sprintf("%s needs character level %d", e.Skill, e.MinLvl)
}

// SkillPool is what the spending rules need to know about a hero.
type SkillPool struct {
	Class  string // class token of skills.txt "charclass": "sor", "ama", ...
	Level  int    // character level
	Unused int    // unused skill points
	// Base stats compared with skills.txt reqstr/reqdex/reqint/reqvit.
	Str, Dex, Int, Vit int
	// Points returns the points spent in the skill with this name (0 if none).
	Points func(name string) int
}

// CanSpend checks the rules for putting one more point into rec. VERIFIED
// against Game.exe 1.14b (server handler 0x549bc0 for packet 0x3b; checks
// 0x56a440, 0x645b90, 0x645ce0, 0x645b20 and SKILLS_GetMaxSkillLevel): unused
// points left; the skill belongs to the hero's class; the character level
// reaches reqlevel PLUS the points already in the skill (every point, not only
// the first); each of reqskill1-3 has at least one base point; strength,
// energy, dexterity and vitality reach reqstr/reqint/reqdex/reqvit; and the
// base points are below maxlvl (20 when the column is empty). The client only
// greys the icon with the same rules; the server decides.
func CanSpend(rec *d2records.SkillRecord, pool SkillPool) error {
	switch {
	case rec == nil:
		return errors.New("unknown skill")
	case pool.Unused < 1:
		return ErrNoPoints
	case rec.Charclass == "" || !strings.EqualFold(rec.Charclass, pool.Class):
		return ErrWrongClass
	}

	have := pool.Points(rec.Skill)

	if need := rec.Reqlevel + have; pool.Level < need {
		return &SkillReqError{Skill: rec.Skill, MinLvl: need}
	}

	for _, req := range []string{rec.Reqskill1, rec.Reqskill2, rec.Reqskill3} {
		if req != "" && pool.Points(req) < 1 {
			return &SkillReqError{Skill: rec.Skill, Missing: req}
		}
	}

	for _, st := range []struct {
		name      string
		have, min int
	}{
		{"strength", pool.Str, rec.Reqstr}, {"energy", pool.Int, rec.Reqint},
		{"dexterity", pool.Dex, rec.Reqdex}, {"vitality", pool.Vit, rec.Reqvit},
	} {
		if st.have < st.min {
			return &SkillReqError{Skill: rec.Skill, Stat: st.name, StatMin: st.min}
		}
	}

	max := rec.Maxlvl
	if max <= 0 {
		max = DefaultMaxSkillLevel
	}

	if have >= max {
		return ErrLevelCap
	}

	return nil
}

// PoolOf describes a hero for CanSpend. classToken is the skills.txt charclass.
func PoolOf(skills map[int]*HeroSkill, stats *HeroStatsState, classToken string) SkillPool {
	return SkillPool{
		Class: classToken, Level: stats.Level, Unused: stats.SkillPoints,
		Str: stats.Strength, Dex: stats.Dexterity, Int: stats.Energy, Vit: stats.Vitality,
		Points: func(name string) int {
			for _, s := range skills {
				if s != nil && s.SkillRecord != nil && strings.EqualFold(s.Skill, name) {
					return s.SkillPoints
				}
			}

			return 0
		},
	}
}

// SpendSkillPoint puts one unused point into the skill, if the rules allow.
func SpendSkillPoint(skills map[int]*HeroSkill, stats *HeroStatsState, classToken string, id int) error {
	s := skills[id]
	if s == nil || s.SkillRecord == nil {
		return errors.New("hero does not have this skill slot")
	}

	if err := CanSpend(s.SkillRecord, PoolOf(skills, stats, classToken)); err != nil {
		return err
	}

	s.SetPoints(s.SkillPoints + 1)
	stats.SkillPoints--

	return nil
}

// GrantLevelUp applies levels level-ups: the level, the unused skill and stat
// points a level-up grants (one skill point and five stat points). Experience
// is the caller's business.
func GrantLevelUp(stats *HeroStatsState, levels int) {
	if levels <= 0 {
		return
	}

	stats.Level += levels
	stats.SkillPoints += SkillPointsPerLevel * levels
	stats.StatsPoints += StatPointsPerLevel * levels
}
