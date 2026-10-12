package d2player

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// Extra tooltip text of the skill tree icon tooltip builder UI_ShowSkillTreeIconTooltip 0x4ec180. VERIFIED strings
// and conditions (wave 37):
//
//   - itemOnly: the hero has no skill points in the skill (hard level 0) but items give it levels (effective > 0).
//     Then " (item)" (StrSkill117, 0x1112) follows the "Current Skill Level: n" text, and the line
//     "You have been granted this skill by an item" (StrSkill118, 0x1113) is added.
//   - The line "You have not learned this skill yet" (StrSkill119, 0x1114) is added while the hero has not put a
//     point into ANY skill of his class yet (UI_CheckSkillTooltipUnlocked 0x4ea3a0; the answer is remembered for the
//     rest of the session once it was true).
//   - The Next Level block is left out when the HARD level (points without items) has reached the global maximum
//     level (SKILLS_GetMaxSkillLevel 0x4a6ff0), not the effective level.
//
// VERIFIED placement (UI_ShowSkillTreeIconTooltip 0x4ec180 builds one string with CRT_WcsCat): after the description
// text come, in this order, "\n\n" + 0x1113 (item granted) and "\n\n" + 0x1114 (not learned); 0xf9e is the newline
// string. So each footer line is preceded by one blank line.
const (
	keyItemMark     = "StrSkill117"
	keyItemGranted  = "StrSkill118"
	keyNotLearned   = "StrSkill119"
	itemMarkDefault = " (item)"
)

// SkillItemOnly reports a skill that exists only through items.
func SkillItemOnly(base, effective int) bool { return base == 0 && effective > 0 }

// SkillNextBlockHidden reports whether the Next Level block is skipped (hard level at the maximum).
func SkillNextBlockHidden(base, maxLevel int) bool { return maxLevel > 0 && base >= maxLevel }

// SkillTooltipFooter returns the extra lines for the icon tooltip (see above); tr translates a string key
// (returning the key when unknown).
func SkillTooltipFooter(tr func(string) string, base, effective int, anyLearned bool) []string {
	get := func(key, fallback string) string {
		if v := tr(key); v != "" && v != key {
			return v
		}

		return fallback
	}

	var out []string

	if SkillItemOnly(base, effective) {
		out = append(out, "", get(keyItemGranted, "You have been granted this skill by an item"))
	}

	if !anyLearned {
		out = append(out, "", get(keyNotLearned, "You have not learned this skill yet"))
	}

	return out
}

// MarkItemLevel appends the item mark to the "Current Skill Level" line of lines when the skill is item-only.
func MarkItemLevel(tr func(string) string, lines []string, currentPrefix string, base, effective int) {
	if !SkillItemOnly(base, effective) {
		return
	}

	mark := itemMarkDefault
	if v := tr(keyItemMark); v != "" && v != keyItemMark {
		mark = v
	}

	for i, l := range lines {
		if strings.HasPrefix(l, currentPrefix) {
			lines[i] = l + mark

			return
		}
	}
}

// skillTreeTooltip is the tooltip of a skill tree icon: skillTooltip plus the item notes above.
func skillTreeTooltip(asset *d2asset.AssetManager, sk *d2hero.HeroSkill, effective int, bar *d2hero.SkillBar,
	keyName func(int) string, skills map[int]*d2hero.HeroSkill, stats *d2hero.HeroStatsState) string {
	text := skillTooltip(asset, sk, effective, bar, keyName, skills, stats)
	lines := strings.Split(text, "\n")
	tr := func(key string) string { return asset.TranslateString(key) }

	MarkItemLevel(tr, lines, label(tr, keyCurrentLevel, "Current Skill Level: "), sk.SkillPoints, effective)
	lines = append(lines, SkillTooltipFooter(tr, sk.SkillPoints, effective, anyClassSkillLearned(skills))...)

	return strings.Join(lines, "\n")
}

// anyClassSkillLearned reports whether any of the hero's skills holds a point.
func anyClassSkillLearned(skills map[int]*d2hero.HeroSkill) bool {
	for _, s := range skills {
		if s != nil && s.SkillPoints > 0 {
			return true
		}
	}

	return false
}
