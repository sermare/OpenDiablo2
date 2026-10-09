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
	MinLvl  int    // character level needed (0 when a skill is missing)
	Missing string // name of the required skill that has no points
}

func (e *SkillReqError) Error() string {
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
	// Points returns the points spent in the skill with this name (0 if none).
	Points func(name string) int
}

// CanSpend checks the rules for putting one more point into rec: unused points
// left, the skill is one of the class' skills, below its maximum, the character
// level reaches skills.txt reqlevel and each of reqskill1-3 has a point. (The
// reqlevel/reqskill rule is the documented skills.txt semantics; the original
// binary's check was not read.)
func CanSpend(rec *d2records.SkillRecord, pool SkillPool) error {
	switch {
	case rec == nil:
		return errors.New("unknown skill")
	case pool.Unused < 1:
		return ErrNoPoints
	case rec.Charclass == "" || !strings.EqualFold(rec.Charclass, pool.Class):
		return ErrWrongClass
	}

	max := rec.Maxlvl
	if max <= 0 {
		max = DefaultMaxSkillLevel
	}

	if pool.Points(rec.Skill) >= max {
		return ErrLevelCap
	}

	if pool.Level < rec.Reqlevel {
		return &SkillReqError{Skill: rec.Skill, MinLvl: rec.Reqlevel}
	}

	for _, req := range []string{rec.Reqskill1, rec.Reqskill2, rec.Reqskill3} {
		if req != "" && pool.Points(req) < 1 {
			return &SkillReqError{Skill: rec.Skill, Missing: req}
		}
	}

	return nil
}

// PoolOf describes a hero for CanSpend. classToken is the skills.txt charclass.
func PoolOf(skills map[int]*HeroSkill, stats *HeroStatsState, classToken string) SkillPool {
	return SkillPool{
		Class: classToken, Level: stats.Level, Unused: stats.SkillPoints,
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
