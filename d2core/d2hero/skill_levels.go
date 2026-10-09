package d2hero

import (
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// ItemSkillBonusOf reads the +skill stats (127 all skills, 83 class skills,
// 188 skill tab, 107/97 single skill) out of the hero's stat totals, the same
// list hero_totals.go builds from the equipment and charms. skillID and page
// (skilldesc SkillPage, 1-based) select the single-skill and tab entries
// that matter for one skill. A nil totals gives no bonus.
func ItemSkillBonusOf(t *d2statlist.Totals, hero d2enum.Hero, skillID, page int) d2skill.ItemSkillBonus {
	var b d2skill.ItemSkillBonus

	if t == nil || t.Stats == nil {
		return b
	}

	class, ok := D2SClassOf(hero)
	if !ok {
		return b
	}

	b.All = int(t.Stats.Get(d2statlist.StatAllSkills))
	b.Class = int(t.Stats.GetParam(d2statlist.StatClassSkills, int(class)))

	if page >= 1 && page <= 3 {
		b.Tab = map[int]int{page: int(t.Stats.GetParam(d2statlist.StatSkillTab, int(class)*8+page-1))}
	}

	b.Single = map[int]int{skillID: t.SingleSkillBonus(skillID)}

	return b
}

// TreeLabel is the level label of a skill tree icon. VERIFIED (UI_DrawSkillTreeIcon
// 0x4a86b0, caller 0x4a8980): the number is the effective level (base points plus
// the item bonus, the SKILL_GetTotalLevel includeBonus=1 value) and it is
// drawn whenever the effective level or the base points are non-zero; the
// colour is blue (palette colour 3) when the items add levels, red (1) when
// they subtract, white otherwise. The tree click itself only works with the
// base points (they decide the cap and the cost).
func TreeLabel(base, effective int) (text, colorToken string, shown bool) {
	if base == 0 && effective == 0 {
		return "", "", false
	}

	text = strconv.Itoa(effective)

	switch {
	case effective > base:
		return text, "[blue]", true
	case effective < base:
		return text, "[red]", true
	}

	return text, "", true
}

// EffectiveSkillLevel is the level of a skill a cast, a calc and the skill
// tooltip see: the points of the tree plus the +skills of the hero's items,
// clamped to d2skill.MaxLevelCap. The skill tree icon shows it too (TreeLabel).
// Without +skill items it equals the base points.
func EffectiveSkillLevel(st *HeroStatsState, hero d2enum.Hero, sk *HeroSkill) int {
	if sk == nil {
		return 0
	}

	if st == nil || st.Totals == nil || sk.SkillRecord == nil {
		return sk.SkillPoints
	}

	page := 0
	if sk.SkillDescriptionRecord != nil {
		page = sk.SkillPage
	}

	b := ItemSkillBonusOf(st.Totals, hero, sk.ID, page)
	ps := &d2skill.Skill{ID: sk.ID, CharClass: sk.Charclass}

	return d2skill.EffectiveLevel(sk.SkillPoints, b, ps, ClassCodeOfHero(hero), page)
}
