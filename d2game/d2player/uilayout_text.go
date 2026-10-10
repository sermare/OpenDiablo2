package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// textAnchor is where a text label really stands on the screen, as the pair the original's text tables use: the
// horizontal centre of the drawn text and the bottom edge of its line (the original's text y is the bottom of the
// line, and its tables hold the x1..x2 box the text is centred in). w and h of the result are 0.
// UNVERIFIED: that the original centres every caption in its box (it is how the table reads).
func textAnchor(panel, name string, l *d2ui.Label) UIRect {
	x, y := l.GetPosition()
	w, h := l.GetSize()

	left := x

	switch l.Alignment {
	case d2ui.HorizontalAlignCenter:
		left = x - w/2
	case d2ui.HorizontalAlignRight:
		left = x - w
	case d2ui.HorizontalAlignLeft:
	}

	return UIRect{panel, name, left + w/2, y + h, 0, 0}
}

// layoutRects returns the text anchors of the character panel: its captions (label table 0x723a5c of the original)
// and the value boxes (0x723b70).
func (s *HeroStatsPanel) layoutRects() []UIRect {
	var rs []UIRect

	for _, c := range s.staticLabelSpecs() {
		l := s.uiManager.NewLabel(c.font, d2resource.PaletteStatic)
		if l == nil {
			continue
		}

		if c.centerAlign {
			l.Alignment = d2ui.HorizontalAlignCenter
		}

		l.SetText(c.txt)
		l.SetPosition(c.x, c.y+panelShiftY)
		rs = append(rs, textAnchor("character", "text."+c.name, l))
	}

	s.setStatValues()

	values := []struct {
		name string
		l    *d2ui.Label
	}{
		{"level", s.labels.Level}, {"experience", s.labels.Experience}, {"nextlevel", s.labels.NextLevelExp},
		{"strength", s.labels.Strength}, {"dexterity", s.labels.Dexterity}, {"defense", s.labels.Defense},
		{"vitality", s.labels.Vitality}, {"maxstamina", s.labels.MaxStamina}, {"stamina", s.labels.Stamina},
		{"maxlife", s.labels.MaxHealth}, {"life", s.labels.Health}, {"energy", s.labels.Energy},
		{"maxmana", s.labels.MaxMana}, {"mana", s.labels.Mana},
		{"fire", s.labels.Resist[0]}, {"cold", s.labels.Resist[1]}, {"lightning", s.labels.Resist[2]},
		{"poison", s.labels.Resist[3]},
	}

	for _, v := range values {
		rs = append(rs, textAnchor("character", "value."+v.name, v.l))
	}

	return rs
}

// layoutRects returns the picture rectangles of the quest log as placed: the act tabs and the six quest slots of
// the shown act (UI_DrawQuestLogPanel 0x49fc00, slot table 0x7231a8).
func (s *QuestLog) layoutRects() []UIRect {
	var rs []UIRect

	for i, t := range s.tab {
		if t.sprite == nil {
			continue
		}

		rs = append(rs, spriteBox("quest", fmt.Sprintf("tab%d", i), t.sprite))
	}

	if q := s.quests[s.selectedTab]; q != nil {
		for i, sock := range q.sockets {
			rs = append(rs, spriteBox("quest", fmt.Sprintf("slot%d", i), sock))
		}
	}

	return rs
}
