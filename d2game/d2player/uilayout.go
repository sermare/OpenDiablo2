package d2player

import "fmt"

// Interface layout of the original game, for the 640x480 and 800x600 modes.
//
// Everything here is a number read from the original executable (Game.exe 1.14b)
// and written down in the reverse-engineering notes (ui-layout.md: addresses and
// numbers only). "verified" values were read in the decompiler; the functions are
// named in the comments. The original draws a picture at the position of its
// BOTTOM-left corner; rectangles below are given as left/top/width/height.
//
// How the modes relate: the original keeps two screen sizes (DAT_00710f48 = width,
// DAT_00710f4c = height). The bottom control bar and its hit areas are laid out
// from the width/2 and the height. All panels (inventory, stats, skill tree, ...)
// are laid out for 640x480 and moved by (80, 60) in the 800x600 mode (UI_DrawInterface
// 0x452580 sets the offset DAT_0079a8e0/e4 to (0x50, -60); a picture at y bottom
// H-0xe0+oy has its top at H-480+60... see PanelTop).

// Mode is one of the two screen modes of the original.
type Mode struct{ W, H int }

// The two modes of the original game.
var (
	Mode640 = Mode{640, 480} //nolint:gochecknoglobals // constants
	Mode800 = Mode{800, 600} //nolint:gochecknoglobals // constants
)

// UIRect is one named rectangle of the interface in screen pixels.
type UIRect struct {
	Panel, Name string
	X, Y, W, H  int
}

// String formats the rectangle the way the layout log and the golden file do.
func (r UIRect) String() string {
	return fmt.Sprintf("%s %s %d %d %d %d", r.Panel, r.Name, r.X, r.Y, r.W, r.H)
}

// PanelOffset is the offset of the panel pictures: (0,0) at 640x480, (80,60) at 800x600
// (UI_DrawInterface 0x452580: DAT_0079a8e0 = 0x50, DAT_0079a8e4 = -60 when the mode is 800x600).
func (m Mode) PanelOffset() (x, y int) {
	if m == Mode800 {
		return 80, 60
	}

	return 0, 0
}

// LeftPanelX is the left edge of the panels that open on the left (stats, quests, party,
// stash, cube, trade, mercenary): the offset itself (UI_DrawCharacterStatsPanel 0x4a43f0).
func (m Mode) LeftPanelX() int {
	x, _ := m.PanelOffset()

	return x
}

// RightPanelX is the left edge of the panels that open on the right (inventory, skill tree):
// W - offset - 0x140 (FUN_0048b150 0x48b150, FUN_004a7220 0x4a7220: 400 at 800x600, 320 at 640x480).
func (m Mode) RightPanelX() int {
	x, _ := m.PanelOffset()

	return m.W - x - 0x140
}

// PanelTop is the top edge of the four pictures of a panel. The pictures are drawn at
// y = H - 0xe0 + oy (upper row, 256 high) and y = H - 0x30 + oy (lower row, 176 high); a picture
// stands on its bottom edge, so the upper row starts at H - 0xe0 - 256 + oy = 60 at 800x600 and 0 at 640x480.
func (m Mode) PanelTop() int {
	_, y := m.PanelOffset()

	return m.H - 0xe0 - 256 + y
}

// panelShiftY and panelShiftX move the contents of the panels that were laid out before this audit
// (their coordinates were tuned against pictures that stood 4 pixels too low, and 1 pixel too far
// right for the panels on the right) onto the original's picture position: the pictures of a panel now
// stand at (LeftPanelX|RightPanelX, PanelTop) = (80|400, 60), like UI_DrawCharacterStatsPanel 0x4a43f0 and
// FUN_0048b150 0x48b150 draw them. Verified on the add-stat buttons: the old coordinates plus this shift
// are exactly the original's (205,136) button and (202,170) socket.
const (
	panelShiftY = -4
	panelShiftX = -1
)

// Panel picture rows: the upper row is 256 high, the lower 176 high; the two columns are 256 and 64 wide.
const (
	panelUpperRowH = 256
	panelLowerRowH = 176
	panelColAW     = 256
	panelColBW     = 64
	panelWidth     = panelColAW + panelColBW
)

// PanelPictures returns the four pictures of the panel whose left edge is x:
// upper-left, upper-right, lower-left, lower-right (the first drawn picture of each of the
// functions named above).
func (m Mode) PanelPictures(x int) [4]UIRect {
	top := m.PanelTop()

	return [4]UIRect{
		{"panel", "upper_left", x, top, panelColAW, panelUpperRowH},
		{"panel", "upper_right", x + panelColAW, top, panelColBW, panelUpperRowH},
		{"panel", "lower_left", x, top + panelUpperRowH, panelColAW, panelLowerRowH},
		{"panel", "lower_right", x + panelColAW, top + panelUpperRowH, panelColBW, panelLowerRowH},
	}
}

// inclusive builds the rectangle of the pixels that a hit test with <= bounds accepts.
func inclusive(panel, name string, x0, y0, x1, y1 int) UIRect {
	return UIRect{panel, name, x0, y0, x1 - x0 + 1, y1 - y0 + 1}
}

// exclusive builds the rectangle of the pixels that a hit test with < bounds accepts.
func exclusive(panel, name string, x0, y0, x1, y1 int) UIRect {
	return UIRect{panel, name, x0 + 1, y0 + 1, x1 - x0 - 1, y1 - y0 - 1}
}

// HUDRects returns the rectangles of the always visible interface of the original for the mode,
// all of them read from the draw and hit-test functions of the original:
//
//	UI_DrawControlPanelBgCenter 0x494860 (the bar pictures), UI_DrawLifeOrb 0x4933d0, UI_DrawManaOrb 0x493560,
//	UI_DrawExperienceBar 0x495380, UI_DrawRunWalkButton 0x4938f0, UI_DrawStaminaBar 0x493a20,
//	UI_DrawMiniPanelButton 0x493c20, FUN_004945b0 (orb hover), UI_IsMouseOverStat/SkillButton800 0x4a2cb0/0x4a2d60,
//	UI_IsPointOverBeltArea 0x4952a0, FUN_00493020 + UI_IsMouseOverSkillIcon 0x4a5470 (skill icons).
//
// Names ending in .hit are hit-test areas, the others are drawn pictures. Only the 800x600 values
// are checked on screen; the 640x480 values come from the same formulas.
func (m Mode) HUDRects() []UIRect {
	w2, h := m.W/2, m.H

	rs := []UIRect{
		// the pieces of the bar of the 800x600 mode (800ctrlpnl7.dc6): 117x104 orb holders, 128x55 and 86x55 pieces
		{"hud", "bar_left_orb", 0, h - 104, 117, 104},
		{"hud", "bar_piece1", w2 - 0xeb, h - 55, 128, 55},
		{"hud", "bar_piece2", w2 - 0x6b, h - 55, 128, 55},
		{"hud", "bar_piece3", w2 + 0x15, h - 55, 128, 55},
		{"hud", "bar_piece4", w2 + 0x95, h - 55, 86, 55},
		{"hud", "bar_right_orb", m.W - 0x75, h - 104, 117, 104},

		// the skill icons: 48x48 at x = 117 and W - 0xa5, standing on the screen bottom
		inclusive("hud", "left_skill.hit", 117, h-0x30, 117+0x30, h),
		inclusive("hud", "right_skill.hit", m.W-0xa5, h-0x30, m.W-0xa5+0x30, h),

		// orbs: the fill is 80 wide and at most 80 high, standing at y = H - 13; hover areas from FUN_004945b0
		{"hud", "life_fill", 0x1d, h - 0xd - 80, 80, 80},
		{"hud", "mana_fill", m.W - 0x6f, h - 0xd - 80, 80, 80},
		inclusive("hud", "life.hit", 0x1e, h-0x4b, 0x1e+0x50, h-0xf),
		inclusive("hud", "mana.hit", m.W-0x6f, h-0x4b, m.W-0x1f, h-0xf),

		// experience bar: the hover area and the two 1 pixel lines (119 pixels at most) at y = H - 0x26
		inclusive("hud", "experience.hit", w2-0x92, h-0x2b, w2-0x17, h-0x22),
		{"hud", "experience_line", w2 - 0x90, h - 0x26, 0x77, 2},

		// stamina bar: hover area; the bar is 102 x 18
		inclusive("hud", "stamina.hit", w2-0x7f, h-0x1b, w2-0x19, h-9),
		{"hud", "stamina_bar", w2 - 0x7f, h - 0x1b, 0x66, 0x12},

		// run/walk button (runbutton.dc6, 16x20, standing at y = H - 10) and its hover area
		{"hud", "run_button", w2 - 0x91, h - 10 - 20, 16, 20},
		inclusive("hud", "run_button.hit", w2-0x91, h-0x1c, w2-0x80, h-8),

		// the mini panel toggle (menubutton.dc6, 15x24, standing at y = H - 0x10) and its hit area
		{"hud", "minipanel_toggle", w2 - 8, h - 0x10 - 24, 15, 24},
		inclusive("hud", "minipanel_toggle.hit", w2-8, h-0x27, w2+5, h-0xd),

		// the level up buttons (level.dc6, 30x30, standing at y = H - 8) and their hit areas
		{"hud", "stat_button", w2 - 0xc2, h - 8 - 30, 30, 30},
		exclusive("hud", "stat_button.hit", w2-0xc2, h-0x2a, w2-0xa0, h-8),
		{"hud", "skill_button", w2 + 0xa3, h - 8 - 30, 30, 30},
		exclusive("hud", "skill_button.hit", w2+0xa3, h-0x2a, w2+0xc5, h-8),

		// the belt: the hit area of the closed belt
		inclusive("hud", "belt.hit", w2+0x17, h-0x27, w2+0x91, h-10),
	}

	return rs
}

// MiniPanelRects returns the rectangles of the mini panel of the original (FUN_0047b510 0x47b510,
// FUN_0047a6a0 0x47a6a0): the strip picture standing at y = H - 0x2f and the buttons (20x20, 21 apart)
// standing at y = H - 0x32. singlePlayer selects the 7 button strip (156 wide) over the 8 button one
// (176 wide). shift is 0 when no panel is open, -1 when a panel on the right is open (the strip moves
// to the left) and 3 when one on the left is open (it moves to the right).
func (m Mode) MiniPanelRects(singlePlayer bool, shift int) []UIRect {
	w2, h := m.W/2, m.H

	stripW, count := 176, 8
	first := w2 - 0x54
	container := w2 - 0x54 - 3

	if singlePlayer {
		stripW, count = 156, 7
		first = w2 - 0x4a
		container = w2 - 0x4a - 3
	}

	switch shift {
	case -1:
		first = w2 - 0xca
		container = w2 - 0xcd
	case 3:
		if singlePlayer {
			first = w2 + 0x39
			container = w2 + 0x38 - 2
		} else {
			first = w2 + 0x23
			container = w2 + 0x23 - 2
		}
	}

	rs := []UIRect{{"minipanel", "strip", container, h - 0x2f - 26, stripW, 26}}

	for i := 0; i < count; i++ {
		rs = append(rs, UIRect{"minipanel", fmt.Sprintf("button%d", i), first + i*0x15, h - 0x32 - 20, 20, 20})
	}

	return rs
}
