package d2ui

import "strings"

// Colour tokens for the three original colours the engine had no token for.
// The RGB values are UNVERIFIED approximations (the original recolours text
// through 13 palette-shift rows of the PL2, not through RGB).
const (
	ColorTokenTan       ColorToken = "[tan]"
	ColorTokenDarkGreen ColorToken = "[darkgreen]"
	ColorTokenPurple    ColorToken = "[purple]"
)

const (
	colorTanAlpha       = 0xc4_a4_68_ff // UNVERIFIED
	colorDarkGreenAlpha = 0x00_80_00_ff // UNVERIFIED
	colorPurpleAlpha    = 0xae_00_ff_ff // UNVERIFIED
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
