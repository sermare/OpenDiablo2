package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

func widgetRect(panel, name string, w interface {
	GetPosition() (int, int)
	GetSize() (int, int)
}) UIRect {
	x, y := w.GetPosition()
	ww, hh := w.GetSize()

	return UIRect{panel, name, x, y, ww, hh}
}

// spriteBox is the rectangle a sprite occupies (a sprite stands on its bottom edge).
func spriteBox(panel, name string, s *d2ui.Sprite) UIRect {
	x, y := s.GetPosition()
	w, h := s.GetCurrentFrameSize()

	return UIRect{panel, name, x, y - h, w, h}
}

// artRect is the first picture of a panel drawn at (x, top) from frame frameIdx of its sprite.
func artRect(panel, name string, s *d2ui.Sprite, frameIdx, x, top int) UIRect {
	if s == nil {
		return UIRect{panel, name, x, top, 0, 0}
	}

	w, h, _ := s.GetFrameSize(frameIdx)

	return UIRect{panel, name, x, top, w, h}
}

// UILayoutPanels lists the panels UILayout knows.
var UILayoutPanels = []string{"hud", "minipanel", "belt", "character", "inventory", "skills", "quest", "stash", "cube", "npcmenu", "trade", "waypoint", "party"} //nolint:gochecknoglobals // list

// UILayout returns the rectangles of what the interface really places for a panel, taken from the widgets and
// the constants they are placed with (not from uilayout.go's expected values), for the OD2_AUTOUILAYOUT log and its
// golden file. The panel must be open already for the panel entries.
func (g *GameControls) UILayout(panel string) []UIRect {
	var rs []UIRect

	reg := func(n string, t actionableType) {
		r := g.actionableRegions[t].rect
		rs = append(rs, UIRect{"hud", n, r.Left, r.Top, r.Width, r.Height})
	}

	switch panel {
	case "hud":
		reg("left_skill.hit", leftSkill)
		reg("right_skill.hit", rightSkill)
		reg("life.hit", hpGlobe)
		reg("mana.hit", manaGlobe)
		reg("experience.hit", xp)
		reg("stamina.hit", stamina)

		h := g.hud
		rs = append(rs,
			widgetRect("hud", "run_button", h.runButton),
			widgetRect("hud", "stat_button", h.addStatsButton),
			widgetRect("hud", "skill_button", h.addSkillButton),
			widgetRect("hud", "stamina_bar", h.widgetStamina),
			widgetRect("hud", "experience_bar", h.widgetExperience),
			widgetRect("hud", "minipanel_toggle", h.miniPanel.menuButton),
		)
		gx, gy := h.healthGlobe.globe.sprite.GetPosition()
		rs = append(rs, UIRect{"hud", "life_fill", gx, gy - globeHeight, globeWidth, globeHeight})
		gx, gy = h.manaGlobe.globe.sprite.GetPosition()
		rs = append(rs, UIRect{"hud", "mana_fill", gx, gy - globeHeight, globeWidth, globeHeight})
	case "minipanel":
		m := g.hud.miniPanel
		rs = append(rs, spriteBox("minipanel", "strip", m.container))

		for i, b := range m.buttons {
			rs = append(rs, widgetRect("minipanel", fmt.Sprintf("button%d", i), b))
		}
	case "character":
		s := g.heroStatsPanel
		rs = append(rs,
			artRect("character", "art_upper_left", s.panel, statsPanelTopLeft, statsPanelOffsetX, statsPanelOffsetY),
			UIRect{"character", "close", heroStatsCloseButtonX, heroStatsCloseButtonY, 32, 32},
		)

		for i, y := range []int{140, 202, 288, 350} {
			rs = append(rs, UIRect{"character", fmt.Sprintf("add_stat%d", i), 205, y + panelShiftY, 30, 30})
		}

		rs = append(rs, s.layoutRects()...)
	case "inventory":
		i := g.inventory
		l, t, r, b := i.grid.Bounds()
		rs = append(rs,
			artRect("inventory", "art_upper_left", i.panel, frameInventoryTopLeft, i.originX, i.originY+inventoryArtTop),
			UIRect{"inventory", "grid", l, t, r - l, b - t},
		)
	case "skills":
		s := g.skilltree
		rs = append(rs, UIRect{"skills", "art_origin", s.originX, s.originY, 0, 0})
		rs = append(rs, s.layoutRects()...)
	case "quest":
		rs = append(rs,
			UIRect{"quest", "art_origin", questLogOffsetX, questLogOffsetY, 0, 0},
			UIRect{"quest", "close", questLogCloseButtonX, questLogCloseButtonY, 32, 32},
		)
		rs = append(rs, g.questLog.layoutRects()...)
	case "belt":
		rs = append(rs, g.belt.layoutRects()...)
	case "waypoint":
		rs = append(rs, g.Waypoints.layoutRects()...)
	case "trade":
		rs = append(rs, g.Trade.layoutRects()...)
	case "npcmenu":
		rs = append(rs, g.NPCMenu.layoutRects()...)
	case "party":
		rs = append(rs, UIRect{"party", "art_origin", partyPanelOffsetX, partyPanelOffsetY, 0, 0},
			UIRect{"party", "close", partyPanelCloseButtonX, partyPanelCloseButtonY, 32, 32})
	case "stash", "cube":
		p := g.stash
		if panel == "cube" {
			p = g.cube
		}

		l, t, r, b := p.grid.Bounds()
		rs = append(rs,
			UIRect{panel, "art_origin", p.originX, panelArtTop, 0, 0},
			UIRect{panel, "grid", l, t, r - l, b - t},
			UIRect{panel, "close", p.originX + panelCloseOffset, panelCloseY, 32, 32},
		)
	}

	return rs
}
