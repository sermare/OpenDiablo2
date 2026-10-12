package d2player

import (
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// textAnchor is where a text label really stands on the screen, as the pair the original's text tables use: the
// horizontal centre of the drawn text and the bottom edge of its line (the original's text y is the bottom of the
// line, and its tables hold the x1..x2 box the text is centred in). w and h of the result are 0.
// VERIFIED (w47, 0x4a43f0 and 0x486ab0): the original centres each caption in its box x1..x2 when the text is narrower.
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

	for _, e := range []struct {
		name string
		l    *d2ui.Label
	}{{"statpoints.caption1", s.pointsCaption1}, {"statpoints.caption2", s.pointsCaption2}, {"statpoints.value", s.remainingPoints}} {
		if e.l != nil {
			rs = append(rs, textAnchor("character", "text."+e.name, e.l))
		}
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
		// the sockets are created in quest order, the log places them by socket (LogPlacement): name them by grid
		// position, row by row, so slotN is the Nth socket of the 3x2 grid
		boxes := make([]UIRect, 0, len(q.sockets))
		for _, sock := range q.sockets {
			boxes = append(boxes, spriteBox("quest", "", sock))
		}

		sort.SliceStable(boxes, func(a, b int) bool {
			if boxes[a].Y != boxes[b].Y {
				return boxes[a].Y < boxes[b].Y
			}

			return boxes[a].X < boxes[b].X
		})

		for i, b := range boxes {
			b.Name = fmt.Sprintf("slot%d", i)
			rs = append(rs, b)
		}
	}

	return rs
}
