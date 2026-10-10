package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	skillLabelXOffset = 49
	skillLabelYOffset = -4

	skillIconColumn1X = 415 // W - 80 - 0x131 (UI_DrawSkillTreeIcon 0x4a86b0)
	skillIconDistX    = 69  // the columns are 415, 484 and 553
)

// skillIconBottoms are the bottom edges of the six rows of the skill tree icons: H + e4 - {0x1a2, 0x15e, 0x11a, 0xd6,
// 0x91, 0x4d} (UI_GetSkillTreeTierY 0x4a7100). The pitch is 68 except between rows 4 and 5, where it is 69.
var skillIconBottoms = [6]int{122, 190, 258, 326, 395, 463} //nolint:gochecknoglobals // constant table

// skillIconPosition returns the bottom-left corner of the icon of a skill: row 1..6 and column 1..3 of its
// skilldesc record.
func skillIconPosition(row, column int) (x, y int) {
	if row < 1 || row > len(skillIconBottoms) {
		row = 1
	}

	return skillIconColumn1X + (column-1)*skillIconDistX, skillIconBottoms[row-1]
}

func newSkillIcon(ui *d2ui.UIManager,
	baseSprite *d2ui.Sprite,
	l d2util.LogLevel,
	skill *d2hero.HeroSkill) *skillIcon {
	base := d2ui.NewBaseWidget(ui)
	label := ui.NewLabel(d2resource.Font16, d2resource.PaletteSky)

	x, y := skillIconPosition(skill.SkillRow, skill.SkillColumn)

	res := &skillIcon{
		BaseWidget: base,
		sprite:     baseSprite,
		skill:      skill,
		lvlLabel:   label,
	}

	res.Logger = d2util.NewLogger()
	res.Logger.SetLevel(l)
	res.Logger.SetPrefix(logPrefix)

	res.SetPosition(x, y)

	return res
}

type skillIcon struct {
	*d2ui.BaseWidget
	lvlLabel *d2ui.Label
	sprite   *d2ui.Sprite
	skill    *d2hero.HeroSkill
	// effective returns base points plus the item bonus (nil: base points).
	effective func(*d2hero.HeroSkill) int

	*d2util.Logger
}

func (si *skillIcon) SetVisible(visible bool) {
	si.BaseWidget.SetVisible(visible)
	si.lvlLabel.SetVisible(visible)
}

func (si *skillIcon) renderSprite(target d2interface.Surface) {
	x, y := si.GetPosition()

	if err := si.sprite.SetCurrentFrame(si.skill.IconCel); err != nil {
		si.Errorf("Cannot set Frame %e", err)
		return
	}

	if si.skill.SkillPoints == 0 {
		target.PushSaturation(skillIconGreySat)
		defer target.Pop()

		target.PushBrightness(skillIconGreyBright)
		defer target.Pop()
	}

	si.sprite.SetPosition(x, y)
	si.sprite.Render(target)
}

func (si *skillIcon) renderSpriteLabel(target d2interface.Surface) {
	eff := si.skill.SkillPoints
	if si.effective != nil {
		eff = si.effective(si.skill)
	}

	text, col, shown := d2hero.TreeLabel(si.skill.SkillPoints, eff)
	if !shown {
		return
	}

	x, y := si.GetPosition()
	si.lvlLabel.SetText(col + text)
	si.lvlLabel.SetPosition(x+skillLabelXOffset, y+skillLabelYOffset)
	si.lvlLabel.Render(target)
}

func (si *skillIcon) Render(target d2interface.Surface) {
	si.renderSprite(target)
	si.renderSpriteLabel(target)
}

func (si *skillIcon) Advance(elapsed float64) error {
	return nil
}
