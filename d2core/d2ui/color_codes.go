package d2ui

import "strings"

// Colour tokens for the three original colours the engine had no token for.
// The original recolours text through 13 palette-shift rows of the act PL2
// (PAL_LoadPaletteAndPl2 0x4f87c0 copies 13*256 bytes from file offset 0x6b627;
// D2WIN_DrawFormattedText 0x4ff270 passes selector-'0' as the row). The RGB values
// below are the palette colour that the row maps the brightest font index (255)
// to, read from the shipped pal.pl2 files: acts 1, 3 and 5 agree; act 2 has tan
// (196,180,136), act 4 has dark green (0,124,0). Purple is the same in all acts.
const (
	ColorTokenTan       ColorToken = "[tan]"
	ColorTokenDarkGreen ColorToken = "[darkgreen]"
	ColorTokenPurple    ColorToken = "[purple]"
)

const (
	colorTanAlpha       = 0xd8_b8_64_ff // row 7, act 1/3/5 (216,184,100)
	colorDarkGreenAlpha = 0x18_64_08_ff // row 10, act 1/2/3/5 (24,100,8)
	colorPurpleAlpha    = 0xa4_20_fc_ff // row 11, all acts (164,32,252)
)

// colorCodeIntro is the byte that starts a colour code in the original's
// strings (0xFF, "y with diaeresis" in Latin-1) followed by 'c' and one
// selector character.
const colorCodeIntro = "\xffc"

// colorCodeIntroUTF8 is the same intro after the string went through a
// Latin-1 to UTF-8 conversion.
const colorCodeIntroUTF8 = "ÿc"

// ColorCodeTokens maps the selector character of an original colour code
// ("\xffc3" is blue) to the engine's token. Verified against the quality ->
// colour rule of INV_DrawGroundItemLabels (magic 3, set 2, rare 9, unique 4,
// crafted 8, socketed/ethereal 5; inventory-trade.md) and the well known
// string table usage; the order is the original's 13 text colours, of which
// the 13th (selector '<') is not used by the string tables and not mapped.
var ColorCodeTokens = map[byte]ColorToken{
	'0': ColorTokenWhite,
	'1': ColorTokenRed,
	'2': ColorTokenGreen,
	'3': ColorTokenBlue,
	'4': ColorTokenGold,
	'5': ColorTokenGrey,
	'6': ColorTokenBlack,
	'7': ColorTokenTan,
	'8': ColorTokenOrange,
	'9': ColorTokenYellow,
	':': ColorTokenDarkGreen,
	';': ColorTokenPurple,
}

// ConvertColorCodes rewrites the original's inline colour codes into the
// engine's "[token]" form that Label understands. Text before the first code
// is white (the original's default). An unknown selector is dropped, the text
// keeping the colour in effect.
func ConvertColorCodes(s string) string {
	s = strings.ReplaceAll(s, colorCodeIntroUTF8, colorCodeIntro)
	if !strings.Contains(s, colorCodeIntro) {
		return s
	}

	var b strings.Builder

	for len(s) > 0 {
		i := strings.Index(s, colorCodeIntro)
		if i < 0 {
			b.WriteString(s)
			break
		}

		b.WriteString(s[:i])

		s = s[i+len(colorCodeIntro):]
		if len(s) == 0 {
			break
		}

		if tok, ok := ColorCodeTokens[s[0]]; ok {
			b.WriteString(string(tok))
		}

		s = s[1:]
	}

	out := b.String()
	if !strings.HasPrefix(out, "[") {
		out = string(ColorTokenWhite) + out
	}

	return out
}
