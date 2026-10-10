package d2skill

// ItemSkillBonus is the +skill stats a unit's equipment grants, by the
// ItemStatCost stat they come from.
type ItemSkillBonus struct {
	All    int         // item_allskills (127)
	Class  int         // item_addclassskills (83), the unit's own class
	Tab    map[int]int // item_addskill_tab (188), by skill page 1..3 of the unit's class
	Single map[int]int // item_singleskill (107) / item_nonclassskill (97), by skill id
}

// MaxLevelCap is the upper clamp of a skill's effective level: the MaxLvl row
// of Experience.txt (99 in every class column of the stock tables).
const MaxLevelCap = 99

// EffectiveLevel is the level a calc and a cast see for a skill: the points
// put into it plus the item bonuses (GetSkillLevel with includeBonus=1,
// 0x645680). The all-skills, class and tab stats only apply to skills of the
// unit's own class (page is the skilldesc SkillPage of sk, 1-based);
// single-skill stats apply to any skill.
//
// VERIFIED (Game.exe 1.14b, 0x645680 -> 0x610ae0): the sum is clamped to
// [0, MaxLevelCap]. The bound is the character level cap read from the
// experience table (Experience.txt MaxLvl row, 99), NOT the skills.txt maxlvl
// column, so +skills lift a level-20 skill above 20. Synergy calcs must use
// the base points (blvl), not this value.
func EffectiveLevel(base int, b ItemSkillBonus, sk *Skill, unitClass string, page int) int {
	lvl := base

	if sk != nil {
		if sk.CharClass == unitClass {
			lvl += b.All + b.Class + b.Tab[page]
		}

		lvl += b.Single[sk.ID]
	}

	if lvl < 0 {
		return 0
	}

	if lvl > MaxLevelCap {
		return MaxLevelCap
	}

	return lvl
}
