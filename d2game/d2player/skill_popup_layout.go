package d2player

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// popupPages is the number of skilldesc.txt "SkillPage" values the popup has a
// row for: 0 general skills, 1-3 the three skill trees of the class, 4 other
// (item) skills.
const popupPages = skillListsLength

// attackSkillID is the skills.txt id of Attack, the one general skill the popup always offers.
const attackSkillID = 0

// popupCell is one icon of the skill popup.
type popupCell struct {
	Skill *d2hero.HeroSkill
	// Row is the row counted from the bottom (the row nearest the skill
	// buttons is 0) and Col the position in the row counted from the screen
	// edge the popup is anchored to: for the left popup column 0 is the leftmost
	// icon, for the right popup the rightmost.
	Row, Col int
	// X and Y are the top-left corner on the screen.
	X, Y int
}

// popupVisible reports whether a skill belongs in the popup of a button: the
// hero can use it there (learned, not passive, "leftskill" for the left
// button), skilldesc.txt lists it (ListRow -1 hides, e.g. Kick) and it has a
// page the popup has a row for.
func popupVisible(s *d2hero.HeroSkill, left bool) bool {
	if !d2hero.Selectable(s, left) || s.SkillDescriptionRecord == nil {
		return false
	}

	// The general skills other than Attack (Throw, Unsummon, the scrolls and books) exist only while the
	// hero has the item or minion they belong to; the engine does not model that yet, so they are hidden
	// (UNVERIFIED how the original decides, see session-core.md section 6).
	if s.Charclass == "" && s.ID != attackSkillID {
		return false
	}

	return s.ListRow >= 0 && s.SkillPage >= 0 && s.SkillPage < popupPages
}

// layoutSkillPopup places the skills the popup of a button shows. The rows are
// the skilldesc.txt pages (only pages that have a skill get a row, stacked
// upwards from the button); inside a row the icons follow the skill tree:
// SkillRow, then SkillColumn, then id. Icons are 48 px apart, the left popup
// grows to the right from leftPanelStartX and the right popup to the left from
// rightPanelEndX (as the original lays out its popup: spacing 0x30, direction
// by hand; the original's per-column de-duplication is not reproduced).
func layoutSkillPopup(skills map[int]*d2hero.HeroSkill, left bool) []popupCell {
	var pages [popupPages][]*d2hero.HeroSkill

	for _, s := range skills {
		if popupVisible(s, left) {
			pages[s.SkillPage] = append(pages[s.SkillPage], s)
		}
	}

	var cells []popupCell

	row := 0

	for _, list := range pages {
		if len(list) == 0 {
			continue
		}

		sort.Slice(list, func(a, b int) bool {
			x, y := list[a], list[b]
			if x.SkillRow != y.SkillRow {
				return x.SkillRow < y.SkillRow
			}

			if x.SkillColumn != y.SkillColumn {
				return x.SkillColumn < y.SkillColumn
			}

			return x.ID < y.ID
		})

		for col, s := range list {
			c := popupCell{Skill: s, Row: row, Col: col, Y: skillPanelOffsetY - row*skillIconHeight}

			if left {
				c.X = leftPanelStartX + col*skillIconWidth
			} else {
				c.X = rightPanelEndX - (col+1)*skillIconWidth
			}

			cells = append(cells, c)
		}

		row++
	}

	return cells
}

// cellAt returns the cell under a screen position, or nil.
func cellAt(cells []popupCell, x, y int) *popupCell {
	for i := range cells {
		c := &cells[i]
		if x >= c.X && x < c.X+skillIconWidth && y >= c.Y && y < c.Y+skillIconHeight {
			return c
		}
	}

	return nil
}
