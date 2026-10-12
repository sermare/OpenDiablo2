package d2gamescreen

import (
	"fmt"
	"strings"
)

// The death overlay texts (UI_LoadYouDiedSprites 0x44e710, VERIFIED w50): below the youdied pictures the exe
// prints, for a hardcore game, string 0x13e8 and otherwise string 0x13e6 formatted with stat 0xaf (the gold
// the death took, only when it is not 0), then, outside hardcore when the panel flag byte is set, string
// 0x13e7. The pictures themselves (UI\<lang>\youdiedhardcore / youdiedinst) are not drawn yet.
const (
	deathGoldStringID     = 0x13e6 // "Death takes its toll of %d Gold"
	deathExpStringID      = 0x13e7 // "You have lost experience"
	deathHardcoreStringID = 0x13e8 // "Your deeds of valor will be remembered"
)

// deathOverlayText builds the text lines of the death overlay from the three strings; goldLost is stat 0xaf.
// expLost reports whether the experience penalty took anything (UNVERIFIED stand-in for the exe's panel flag).
func deathOverlayText(hardcore bool, goldLost int, expLost bool, goldFmt, expText, hardcoreText string) string {
	if hardcore {
		return hardcoreText
	}

	var lines []string

	if goldLost > 0 {
		lines = append(lines, strings.Replace(goldFmt, "%d", fmt.Sprint(goldLost), 1))
	}

	if expLost {
		lines = append(lines, expText)
	}

	return strings.Join(lines, "\n")
}
