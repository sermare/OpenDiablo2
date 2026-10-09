// Package d2herostats holds the pure hero progression math: experience and
// level-ups, life/mana/stamina/attack rating/defense derived from the class,
// level and attributes, and the faster hit/cast/block/attack breakpoint
// tables. It has no engine dependencies and touches no game files.
//
// Verified against the real level 94 Sorceress save (see the tests, which
// need D2_TABLES and D2S_SAVE_JSON). Anything not verified is marked
// "unverified" in its comment. Level lookup, experience cap and level-up grants
// are VERIFIED against the exe (0x00610b10, 0x0057c510, 0x0056e770), see
// ~/git/d2-re-notes/verify-monster-hero.md.
package d2herostats

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Per-level-up grants (same for every class and difficulty).
const (
	StatPointsPerLevel  = 5
	SkillPointsPerLevel = 1
)

// ClassNames lists the Experience.txt class columns in charstats order.
var ClassNames = []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}

// ExpTable is one class's column of Experience.txt. Threshold[l] is the
// experience needed to BE level l+1 (row l of the file), so a character with
// experience e is level L when Threshold[L-1] <= e < Threshold[L]. Verified
// with the level 94 save: experience 2411280845 lies between rows 93 and 94.
type ExpTable struct {
	MaxLevel  int
	Threshold []int64 // index = level, Threshold[0] == 0; length MaxLevel+1
}

// ParseExperience parses Experience.txt (tab separated: a header, a MaxLvl row,
// then Level rows 0..) into a table per class name.
func ParseExperience(data []byte) (map[string]*ExpTable, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
	if len(lines) < 3 {
		return nil, errors.New("experience: too few rows")
	}

	head := strings.Split(lines[0], "\t")
	out := map[string]*ExpTable{}
	cols := map[string]int{}

	for i, h := range head {
		for _, c := range ClassNames {
			if h == c {
				cols[c] = i
				out[c] = &ExpTable{}
			}
		}
	}

	if len(cols) != len(ClassNames) {
		return nil, fmt.Errorf("experience: found %d of %d class columns", len(cols), len(ClassNames))
	}

	for _, line := range lines[1:] {
		f := strings.Split(line, "\t")
		if len(f) <= 1 || f[0] == "" {
			continue
		}

		for c, i := range cols {
			if i >= len(f) {
				return nil, fmt.Errorf("experience: short row %q", f[0])
			}

			v, err := strconv.ParseInt(f[i], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("experience: row %q column %s: %w", f[0], c, err)
			}

			if f[0] == "MaxLvl" {
				out[c].MaxLevel = int(v)
				continue
			}

			if _, err := strconv.Atoi(f[0]); err != nil {
				continue
			}

			out[c].Threshold = append(out[c].Threshold, v)
		}
	}

	for c, t := range out {
		if t.MaxLevel < 1 || len(t.Threshold) != t.MaxLevel+1 {
			return nil, fmt.Errorf("experience: %s has max level %d but %d rows", c, t.MaxLevel, len(t.Threshold))
		}
	}

	return out, nil
}

// LevelFor returns the level of a character with the given experience,
// at most MaxLevel (and at least 1). VERIFIED (0x00610b10): the exe counts
// consecutive rows whose threshold is <= experience, bounded by MaxLevel.
func (t *ExpTable) LevelFor(exp int64) int {
	lvl := 1

	for lvl < t.MaxLevel && exp >= t.Threshold[lvl] {
		lvl++
	}

	return lvl
}

// NextLevelExp is the experience needed for the next level (stat 0x1e, row
// "level" of the file, VERIFIED 0x0056e770), or the file's last row at the
// maximum level.
func (t *ExpTable) NextLevelExp(level int) int64 {
	if level < 1 {
		level = 1
	}

	if level >= t.MaxLevel {
		return t.Threshold[t.MaxLevel]
	}

	return t.Threshold[level]
}

// Progress is the experience state of a hero.
type Progress struct {
	Level       int
	Experience  int64
	StatPoints  int
	SkillPoints int
}

// ExpCap is the highest experience a character can hold: the exe clamps to
// the threshold of row MaxLevel-1 (VERIFIED 0x0057c510), i.e. the experience
// that first reaches the maximum level, not the file's last row.
func (t *ExpTable) ExpCap() int64 { return t.Threshold[t.MaxLevel-1] }

// AddExperience adds gained experience, clamped to ExpCap, and applies
// every level-up it causes (one skill and five stat points each). It returns
// the number of levels gained. Negative gains (death penalty) never lower the
// level here: the game keeps the level and only moves experience, which this
// does too (floored at the threshold of the current level).
func (t *ExpTable) AddExperience(p *Progress, gain int64) (levelsGained int) {
	if p.Level < 1 {
		p.Level = 1
	}

	p.Experience += gain

	if p.Experience > t.ExpCap() {
		p.Experience = t.ExpCap()
	}

	if gain < 0 {
		if floor := t.Threshold[p.Level-1]; p.Experience < floor {
			p.Experience = floor
		}

		return 0
	}

	for p.Level < t.MaxLevel && p.Experience >= t.Threshold[p.Level] {
		p.Level++
		p.StatPoints += StatPointsPerLevel
		p.SkillPoints += SkillPointsPerLevel
		levelsGained++
	}

	return levelsGained
}
