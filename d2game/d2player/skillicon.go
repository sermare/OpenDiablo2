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

	skillIconXOff  = 346 + panelShiftX
	skillIconYOff  = 59 + panelShiftY
	skillIconDistX = 69
	skillIconDistY = 68
)

func newSkillIcon(ui *d2ui.UIManager,
	baseSprite *d2ui.Sprite,
	l d2util.LogLevel,
	skill *d2hero.HeroSkill) *skillIcon {
	base := d2ui.NewBaseWidget(ui)
	label := ui.NewLabel(d2resource.Font16, d2resource.PaletteSky)

	x := skillIconXOff + skill.SkillColumn*skillIconDistX
	y := skillIconYOff + skill.SkillRow*skillIconDistY

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
