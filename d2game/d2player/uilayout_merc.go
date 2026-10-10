package d2player

import "fmt"

// Layout of the mercenary panel (the "hireling" screen). All numbers were read
// read-only from Game.exe 1.14b:
//
//	UI_DrawMercenaryPanel 0x48ee50: the four pictures at the left panel position (x = offset,
//	  y = H - 0xe0 / H - 0x30 with the panel offset), the 32x32 close button at
//	  (offset x + 0x110, bottom H - 0x3f - offset y);
//	UI_DrawMercNameHeader: name text at (offset x + 0xf, 0xd6 + offset y), cut to 0xa0 (160) pixels wide;
//	UI_DrawMercStatLabels: table 0x6dab08, 12 entries of 14 bytes {x1 int, x2 int, y int, string id u16};
//	UI_DrawMercStatValues: table 0x721648, 12 entries of 20 bytes {flag, x, y, w, stat};
//	UI_DrawMercEmptySlotIcons / INV_*: the four body slots are the "Hireling" (640x480) and
//	  "Hireling2" (800x600) records of Inventory.txt (rArm = weapon, torso, lArm = shield, head).
//
// Panel-relative numbers are shifted by the panel offset ((80,60) at 800x600, (0,0) at 640x480).
// Second pass (disassembly + memory read): the label table is {x1, x2, y, id}, a label rect's W is x2 - x1
// (0 = left-anchored caption) and the name x is offset + 0xf. Value table 0x721648 is {flag, x1, y, x2, stat};
// the anchor of its flag == 0 rows (left or right edge) is UNVERIFIED. The panel pictures themselves are a stand-in (the character panel
// pictures), the original loads its own file (DAT_007b4f10) that this layout does not name.

const (
	mercCloseDX     = 0x110 // close button x from the panel's left edge (800 and 640)
	mercCloseBottom = 0x3f  // close button bottom: H - 0x3f - offset y
	mercCloseSize   = 32
	mercNameDX      = 0xf  // name header x from the panel left (asm 0x4869a0: EDX = offset x + 0xf)
	mercNameY       = 0xd6 // name header y (panel relative)
	mercNameMaxW    = 0xa0 // the name is shortened until it is at most this wide
	mercValueBoxW   = 48   // width of the left and resist value boxes (table field w)
)

// MercSlotRect is a body slot of the mercenary panel in Inventory.txt (Hireling for 640x480,
// Hireling2 for 800x600): left, top, width, height in screen pixels of the mode.
type MercSlotRect struct {
	Name       string
	X, Y, W, H int
}

// mercSlots640 and mercSlots800 are the Inventory.txt records "Hireling" and "Hireling2".
var mercSlots640 = []MercSlotRect{ //nolint:gochecknoglobals // data
	{"head", 135, 8, 54, 51}, {"torso", 133, 77, 56, 82}, {"weapon", 20, 47, 55, 112}, {"shield", 251, 47, 55, 112},
}

var mercSlots800 = []MercSlotRect{ //nolint:gochecknoglobals // data
	{"head", 215, 68, 54, 51}, {"torso", 213, 137, 56, 82}, {"weapon", 100, 107, 55, 112}, {"shield", 331, 107, 55, 112},
}

// MercSlots returns the body slots for the mode.
func (m Mode) MercSlots() []MercSlotRect {
	if m == Mode800 {
		return mercSlots800
	}

	return mercSlots640
}

// mercLabel is a row of the label table 0x6dab08: 12 records of 14 bytes {x1 int, x2 int, y int, string id u16}
// (read from the executable's memory; the earlier version of this file read the table one field off). x2 == 0 means a
// plain left-anchored caption at x1; otherwise UI_DrawTextLine draws it between x1 and x2 (centred, exact rounding
// UNVERIFIED). UI_DrawMercStatLabels adds the panel offset to x2 before the zero test, so at 800x600 every record
// takes the x2 branch with a degenerate range for x2 == 0; that quirk is NOT reproduced (UNVERIFIED how it looks).
type mercLabel struct {
	Name   string
	X1, X2 int
	Y      int
	ID     int
}

var mercLabels = []mercLabel{ //nolint:gochecknoglobals // data
	{"experience", 15, 0, 236, 4058}, {"level", 145, 0, 236, 4057}, {"next_level", 200, 0, 236, 4059},
	{"strength", 15, 0, 281, 4060}, {"dexterity", 15, 0, 305, 4062}, {"damage", 15, 0, 329, 4061},
	{"defense", 15, 0, 353, 4064},
	{"fire", 180, 245, 281, 4071}, {"cold", 180, 245, 305, 4072}, {"lightning", 180, 245, 329, 4073}, {"poison", 180, 245, 353, 4074},
	{"life", 180, 0, 214, 4068},
}

// mercValue is a row of the value table 0x721648: flag, x, y, box width and the item stat shown.
type mercValue struct {
	Name       string
	Flag, X, Y int
	W          int
	Stat       int
}

var mercValues = []mercValue{ //nolint:gochecknoglobals // data
	{"experience", 0, 120, 254, 0, 13}, {"level", 0, 175, 254, 0, 12}, {"next_level", 0, 305, 254, 0, 30},
	{"strength", 0, 154, 282, 48, 0}, {"dexterity", 0, 154, 306, 48, 2}, {"damage", 0, 154, 330, 48, 21},
	{"defense", 0, 154, 354, 48, 31},
	{"fire", 0, 309, 282, 48, 39}, {"cold", 0, 309, 306, 48, 43}, {"lightning", 0, 309, 330, 48, 41}, {"poison", 0, 309, 354, 48, 45},
	{"life", 210, 305, 215, 0, 6},
}

// MercPanelRects lists the rectangles of the mercenary panel for the mode: the four pictures, the close
// button, the body slots, the name header and the label and value cells. A zero width or height means a
// point (text anchor). Label x is -1 where the table has no x (see the file comment); the label's raw
// field c is in W.
func (m Mode) MercPanelRects() []UIRect {
	ox, oy := m.PanelOffset()
	names := [4]string{"upper_left", "upper_right", "lower_left", "lower_right"}

	var rs []UIRect

	for i, p := range m.PanelPictures(m.LeftPanelX()) {
		rs = append(rs, UIRect{"merc", "art_" + names[i], p.X, p.Y, p.W, p.H})
	}

	rs = append(rs, UIRect{"merc", "close", ox + mercCloseDX, m.H - mercCloseBottom - oy - mercCloseSize, mercCloseSize, mercCloseSize})

	for _, s := range m.MercSlots() {
		rs = append(rs, UIRect{"merc", "slot_" + s.Name, s.X, s.Y, s.W, s.H})
	}

	// UI_DrawMercNameHeader 0x4868f0: left anchor at x = offset + 0xf (asm 0x4869a0), y = 0xd6 + offset.
	rs = append(rs, UIRect{"merc", "name", ox + mercNameDX, mercNameY + oy, mercNameMaxW, 0})

	for _, l := range mercLabels {
		w := 0
		if l.X2 != 0 {
			w = l.X2 - l.X1
		}

		rs = append(rs, UIRect{"merc", "label_" + l.Name, l.X1 + ox, l.Y + oy, w, 0})
	}

	for _, v := range mercValues {
		rs = append(rs, UIRect{"merc", "value_" + v.Name, v.X + ox, v.Y + oy, v.W, 0})
	}

	return rs
}

// MercSlotAt returns the name of the body slot under a screen point ("" for none).
func (m Mode) MercSlotAt(x, y int) string {
	for _, s := range m.MercSlots() {
		if x >= s.X && x < s.X+s.W && y >= s.Y && y < s.Y+s.H {
			return s.Name
		}
	}

	return ""
}

// MercValueStat is the item stat id a value row shows (see the table comment), by row name; ok false for an
// unknown name. Used by the panel to choose what to print and by the golden test.
func MercValueStat(name string) (stat int, ok bool) {
	for _, v := range mercValues {
		if v.Name == name {
			return v.Stat, true
		}
	}

	return 0, false
}

// MercLabelString is the string.tbl id of a label row.
func MercLabelString(name string) (id int, ok bool) {
	for _, l := range mercLabels {
		if l.Name == name {
			return l.ID, true
		}
	}

	return 0, false
}

// MercLayoutLines renders MercPanelRects for the golden file: one "mode rect" line each.
func MercLayoutLines() []string {
	var out []string

	for _, m := range []Mode{Mode800, Mode640} {
		for _, r := range m.MercPanelRects() {
			out = append(out, fmt.Sprintf("%dx%d %s", m.W, m.H, r))
		}
	}

	return out
}
