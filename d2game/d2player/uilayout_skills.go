package d2player

import "fmt"

// layoutRects returns what the skill tree places: the three tab buttons, the close button of each tab, the unspent
// points count and every icon of the class (page, row and column of its skilldesc record in the name).
func (s *skillTree) layoutRects() []UIRect {
	var rs []UIRect

	for i, t := range s.tab {
		if t.button != nil {
			rs = append(rs, widgetRect("skills", fmt.Sprintf("tab%d", i+1), t.button))
		}

		rs = append(rs, UIRect{"skills", fmt.Sprintf("close_tab%d", i+1), t.closeButtonPosX, skillCloseButtonY, 32, 32})
	}

	rs = append(rs, textAnchor("skills", "text.points", s.remainingPoints))

	for _, si := range s.skillIcons {
		l, t, r, b := iconRect(si)
		name := fmt.Sprintf("icon_p%d_r%d_c%d", si.skill.SkillPage, si.skill.SkillRow, si.skill.SkillColumn)
		rs = append(rs, UIRect{"skills", name, l, t, r - l, b - t})
	}

	return rs
}
