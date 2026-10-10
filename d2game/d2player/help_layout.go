package d2player

import "fmt"

// Layout of the help screen of the original (Game.exe 1.14b), numbers only, read from the decompiler:
//
//	UI_DrawHelpScreen 0x492b00 (mode switch), UI_DrawHelpBackground800 0x4909d0 / ...640 0x490880 (the border
//	picture frames), UI_DrawHelpLegendRows 0x490b10 (eight yellow bullets), UI_DrawHelpLeaderLines800 0x490ef0 /
//	...640 0x490b70 (white dot + two one-pixel vertical lines), UI_DrawHelpCloseButton 0x491340 and the click test
//	UI_HandleHelpScreenClick 0x492b60.
//
// Every picture is placed by its BOTTOM-left corner (x, bottom y) in screen pixels; the help screen does not use the
// panel offset. The text x positions of the labels are register arguments that the decompiler dropped (UNVERIFIED),
// only their y (the baseline argument) is known, see HelpTextBaselines800.

// HelpFrame is one frame of the border picture: frame index and the bottom-left corner.
type HelpFrame struct{ Frame, X, Bottom int }

// HelpFrames800 are the seven frames of 800helpborder (UI_DrawHelpBackground800).
var HelpFrames800 = []HelpFrame{ //nolint:gochecknoglobals // data
	{0, 0x0, 0x100}, {1, 0x0, 0x200}, {2, 0x42, 0x14}, {3, 0x142, 0x14}, {4, 0x217, 0x100}, {5, 0x30c, 0x100}, {6, 0x30c, 0x200},
}

// HelpFrames640 returns the eight frames of the 640x480 border (UI_DrawHelpBackground640); the lower row stands on H - 0x30.
func HelpFrames640(h int) []HelpFrame {
	return []HelpFrame{
		{0, 0x0, 0x100}, {1, 0x100, 0x100}, {2, 0x0, h - 0x30}, {3, 0x100, h - 0x30},
		{4, 0x140, 0x100}, {5, 0x240, 0x100}, {6, 0x140, h - 0x30}, {7, 0x240, h - 0x30},
	}
}

// Help legend: eight yellow bullets at x 0x68, bottoms 0x49 + 0x14 * i (both modes).
const (
	helpLegendX      = 0x68
	helpLegendFirstY = 0x49
	helpLegendPitch  = 0x14
	helpLegendRows   = 8
)

// HelpLegendBullet returns the bottom-left corner of legend bullet i.
func HelpLegendBullet(i int) (x, bottom int) {
	return helpLegendX, helpLegendFirstY + helpLegendPitch*i
}

// HelpLeader is one callout marker: the white dot (frame 0) stands on (DotX, DotBottom); two vertical one-pixel
// lines run at x DotX+3 and DotX+4 from Y1 to Y2 (the original passes the coordinates to a line routine with both
// colour arguments 0xff).
type HelpLeader struct{ DotX, DotBottom, Y1, Y2 int }

// HelpLeaders800 are the eleven leader lines of UI_DrawHelpLeaderLines800, in the original's order.
var HelpLeaders800 = []HelpLeader{ //nolint:gochecknoglobals // data
	{0x37, 0x217, 0x1e6, 0x214}, {0x82, 0x234, 0x1bd, 0x231}, {0xd9, 0x23f, 0x177, 0x23c}, {0x104, 0x249, 0x20e, 0x246},
	{0x136, 0x249, 0x1e6, 0x246}, {0x16d, 0x235, 0x20d, 0x232}, {0x1bd, 0x21c, 0x1c3, 0x219}, {0x212, 0x239, 0x20d, 0x236},
	{0x23c, 0x23f, 0x177, 0x23c}, {0x29e, 0x234, 0x1bd, 0x231}, {0x2e4, 0x217, 0x1e6, 0x214},
}

// HelpLeaders640 are the nine leader lines of UI_DrawHelpLeaderLines640.
var HelpLeaders640 = []HelpLeader{ //nolint:gochecknoglobals // data
	{0x33, 0x19e, 0x16d, 0x19b}, {0x7f, 0x1bc, 0x145, 0x1b9}, {0xb2, 0x1d0, 0x195, 0x1cd}, {0xe3, 0x1d0, 0x16d, 0x1cd},
	{0x11c, 0x1bd, 0x195, 0x1ba}, {0x15e, 0x19f, 0x145, 0x19b}, {0x1bd, 0x1c2, 0x195, 0x1be}, {0x1f7, 0x1bd, 0x145, 0x1b9},
	{0x246, 0x19f, 0x16d, 0x19b},
}

// HelpLeaderAt returns the 800x600 leader whose dot stands on (x, bottom).
func HelpLeaderAt(x, bottom int) (HelpLeader, bool) {
	for _, l := range HelpLeaders800 {
		if l.DotX == x && l.DotBottom == bottom {
			return l, true
		}
	}

	return HelpLeader{}, false
}

// Close button: the 32x32 picture (frame 10, 11 while pressed) stands on its bottom-left corner; the click test
// accepts x in [X, X+0x20] and y in [0x18, 0x38] (both inclusive, UI_HandleHelpScreenClick).
const (
	helpClose800X = 0x2ad
	helpClose640X = 0x20d
	helpCloseY    = 0x38 // bottom edge
	helpCloseSize = 32
)

// HelpCloseRect is the close button as a top-left rectangle.
func (m Mode) HelpCloseRect() UIRect {
	x := helpClose640X
	if m == Mode800 {
		x = helpClose800X
	}

	return UIRect{"help", "close", x, helpCloseY - helpCloseSize, helpCloseSize, helpCloseSize}
}

// HelpTextBaselines800 are the y arguments of the text calls of UI_DrawHelpScreenColumnA (the x is lost, UNVERIFIED):
// the title line, the rows of the bulleted list and the callout texts, in the order of the calls.
var HelpTextBaselines800 = []int{ //nolint:gochecknoglobals // data
	0x11, 0x49, 0x4e, 0x62, 0x76, 0x8a, 0x9e, 0xb2, 0xc6, 0xda,
	0x1e1, 0x19a, 0x1a9, 0x1b8, 0x177, 0x1fa, 0x209, 0x1e1, 0x1f9, 0x208, 0x191, 0x1a0, 0x1af, 0x1be, 0x208, 0x177, 0x19a, 0x1a9, 0x1b8, 0x1e1,
}

// HelpLayoutLines renders the help layout for the golden file, one "mode rect" line each.
func HelpLayoutLines() []string {
	var out []string

	add := func(m Mode, name string, x, y, w, h int) {
		out = append(out, fmt.Sprintf("%dx%d %s", m.W, m.H, UIRect{"help", name, x, y, w, h}))
	}

	for _, f := range HelpFrames800 {
		add(Mode800, fmt.Sprintf("frame%d", f.Frame), f.X, f.Bottom, 0, 0)
	}

	for _, f := range HelpFrames640(Mode640.H) {
		add(Mode640, fmt.Sprintf("frame%d", f.Frame), f.X, f.Bottom, 0, 0)
	}

	for _, m := range []Mode{Mode800, Mode640} {
		r := m.HelpCloseRect()
		add(m, "close", r.X, r.Y, r.W, r.H)

		for i := 0; i < helpLegendRows; i++ {
			x, b := HelpLegendBullet(i)
			add(m, fmt.Sprintf("bullet%d", i), x, b, 0, 0)
		}
	}

	for i, l := range HelpLeaders800 {
		add(Mode800, fmt.Sprintf("dot%d", i), l.DotX, l.DotBottom, 0, 0)
		add(Mode800, fmt.Sprintf("line%d", i), l.DotX+3, l.Y1, 2, l.Y2-l.Y1)
	}

	for i, l := range HelpLeaders640 {
		add(Mode640, fmt.Sprintf("dot%d", i), l.DotX, l.DotBottom, 0, 0)
		add(Mode640, fmt.Sprintf("line%d", i), l.DotX+3, l.Y1, 2, l.Y2-l.Y1)
	}

	return out
}
