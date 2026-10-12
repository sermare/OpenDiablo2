package d2player

import "image/color"

// Colours of the numbers on the character screen. VERIFIED in Game.exe: UI_DrawCharacterStatsPanel 0x4a43f0 picks
// a font colour index per value and passes it to D2WIN_DrawText. The index meaning is the D2 text palette:
// 0 white, 1 red, 2 green, 3 blue, 4 gold.
const (
	StatFontWhite = 0
	StatFontRed   = 1
	StatFontGreen = 2
	StatFontBlue  = 3
	StatFontGold  = 4
	// StatFontYellow is font colour 9 (lightning damage in the skill boxes); its RGB is an approximation.
	StatFontYellow = 9
)

// StatColour names the classification of an attribute value against its base.
type StatColour int

// The attribute value colours.
const (
	StatColourNormal    StatColour = iota // current == base
	StatColourIncreased                   // items or states raise it: blue
	StatColourDecreased                   // a penalty lowers it: red
)

// StatValueColour compares a displayed attribute to its base value (VERIFIED: current > base is blue, below is red).
func StatValueColour(current, base int) StatColour {
	switch {
	case current > base:
		return StatColourIncreased
	case current < base:
		return StatColourDecreased
	}

	return StatColourNormal
}

// FontIndex is the exe font colour index of a classification.
func (c StatColour) FontIndex() int {
	switch c {
	case StatColourIncreased:
		return StatFontBlue
	case StatColourDecreased:
		return StatFontRed
	}

	return StatFontWhite
}

// Stat ids of the rows the screen draws (UI row table 0x723b7c).
const (
	statIDLife       = 6
	statIDMaxLife    = 7
	statIDMana       = 8
	statIDMaxMana    = 9
	statIDStamina    = 10
	statIDMaxStamina = 11
)

// VitalColour is the colour of the life, mana and stamina numbers. VERIFIED: current life, current mana and
// current stamina (stat ids 6, 8, 10) are always white; the three maxima (7, 9, 11) are blue above the base value
// and red below it. Current life is shown as at least 1 while the hero is alive (not modelled here).
func VitalColour(statID, current, base int) int {
	switch statID {
	case statIDMaxLife, statIDMaxMana, statIDMaxStamina:
		return StatValueColour(current, base).FontIndex()
	}

	return StatFontWhite
}

// DefenseColour: blue when one of the defense-raising states is on (state mask 0x118, or a hand item of type 0x33
// together with state 0x65), red when a defense-lowering state is on (mask 0x130), red wins. Otherwise white.
// The value is drawn in the small font from 1000 up (or in the language id 6 build). VERIFIED 0x4a43f0.
func DefenseColour(raised, lowered bool) int {
	switch {
	case lowered:
		return StatFontRed
	case raised:
		return StatFontBlue
	}

	return StatFontWhite
}

// DefenseRaised is the blue condition of the defense: a state with the armblue column, or Holy Shield (state 0x65)
// while a shield (item type 0x33) is in the hand. VERIFIED 0x4a43f0.
func DefenseRaised(armBlue, shieldWorn, holyShield bool) bool {
	return armBlue || (shieldWorn && holyShield)
}

// ResistCap is the value above which a resistance is drawn gold: max resist stat + 75, at most 95 (0x4b and 0x5f).
func ResistCap(maxResist int) int {
	c := maxResist + 75
	if c > 95 {
		c = 95
	}

	return c
}

// ResistColour classifies a displayed resistance. VERIFIED 0x4a43f0: shown is the value after the difficulty penalty
// and clamped at -100; raised and lowered are the state masks of that element (blue / red). At or above the cap the
// number is gold unless a raising state is on; below zero it is red.
func ResistColour(shown, maxResist int, raised, lowered bool) int {
	col := StatFontWhite
	if raised {
		col = StatFontBlue
	}

	if lowered {
		col = StatFontRed
	}

	v := shown
	if v < -100 {
		v = -100
	}

	if v >= ResistCap(maxResist) {
		if col != StatFontBlue {
			return StatFontGold
		}
	}

	if v < 0 {
		return StatFontRed
	}

	return col
}

// ResistDisplayed is the number the screen prints: the value clamped to -100 .. ResistCap (VERIFIED: the number
// is the cap itself when the resistance reaches it).
func ResistDisplayed(shown, maxResist int) int {
	if shown < -100 {
		return -100
	}

	if c := ResistCap(maxResist); shown > c {
		return c
	}

	return shown
}

// StatFontRGBA is the tint for a font colour index (0 white, 1 red, 2 green, 3 blue, 4 gold); the shades match the
// text palette entries used by the other panels.
func StatFontRGBA(idx int) color.RGBA {
	switch idx {
	case StatFontRed:
		return color.RGBA{R: 0xdb, G: 0x3f, B: 0x3d, A: 0xff}
	case StatFontGreen:
		return color.RGBA{R: 0x00, G: 0xd0, B: 0x00, A: 0xff}
	case StatFontBlue:
		return color.RGBA{R: 105, G: 105, B: 255, A: 255}
	case StatFontGold:
		return color.RGBA{R: 255, G: 215, B: 0, A: 255}
	case StatFontYellow:
		return color.RGBA{R: 255, G: 255, B: 100, A: 255}
	}

	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}
