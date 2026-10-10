package d2ui

import "sync/atomic"

// colorBlind switches the colour tokens of item names to a palette that stays
// apart for the common kinds of colour blindness: the Okabe-Ito set (Okabe and
// Ito, "Color Universal Design"), chosen because its colours differ in
// lightness as well as hue. The original palette separates magic (blue) from
// set (green) and rare (yellow) from unique (gold) by hue alone, which
// red-green and blue-yellow colour-blind players cannot do.
var colorBlind atomic.Bool

// SetColorBlind turns the colour-blind friendly item colours on or off.
func SetColorBlind(on bool) { colorBlind.Store(on) }

// ColorBlind reports whether the colour-blind friendly colours are on.
func ColorBlind() bool { return colorBlind.Load() }

var colorBlindPalette = map[ColorToken]uint32{
	ColorTokenBlue:   0x56_b4_e9_ff, // magic: sky blue
	ColorTokenYellow: 0xf0_e4_42_ff, // rare: yellow
	ColorTokenGreen:  0x00_9e_73_ff, // set: bluish green
	ColorTokenGold:   0xe6_9f_00_ff, // unique: orange
	ColorTokenOrange: 0xd5_5e_00_ff, // crafted: vermillion
	ColorTokenGrey:   0xb0_b0_b0_ff, // socketed/ethereal: light grey (readable on dark)
}
