// Package d2skilldesc formats the text lines of a skill tooltip from the
// skilldesc.txt columns. It is pure: strings and numbers in, strings out.
//
// Status: the per-line kind switch of Game.exe 1.14b was located (jump table
// 0x4ec048, 75 kinds, dispatched at 0x4eae9b inside 0x4eade0); the kinds that
// are listed as VERIFIED in block.go follow it. The remaining kinds (8-11, 13,
// 14, 16-18, 20-36, 39, 41-62, 64-75 except those listed) are not modelled and
// produce no line. 0x4e6fd0 is only the damage function of descdam 15.
package d2skilldesc

import "strconv"

// FormatLine renders one text/number description line of kind 2, 3 or 7.
// VERIFIED (0x4e4f40 and 0x4e4cc0): kind 2 is texta, the value with a "+" when
// it is not negative, textb; kind 3 is texta, value, textb; kind 7 is the value
// then texta. A zero value gives no line in all three. ok is false for kinds
// that are not modelled here, so the caller can skip them.
func FormatLine(kind int, texta, textb string, value int) (line string, ok bool) {
	if value == 0 {
		return "", false
	}

	switch kind {
	case KindSignedValue:
		return texta + signed(value, true) + textb, true
	case KindValue:
		return texta + strconv.Itoa(value) + textb, true
	case KindCount:
		return strconv.Itoa(value) + texta, true
	}

	return "", false
}

// ManaCost formats the "Mana Cost: n" line (kind 1, 0x4e5720) from the "str
// mana" label (StrSkill3 "Mana Cost: ") and the cost in the engine's 8.8
// fixed point (d2combat.ManaCost). VERIFIED: costs up to 25 mana (0x1900) are
// shown with one decimal when they are not whole (Fire Bolt 2.5), the tenth
// being the fraction byte times 10 over 256, truncated; larger costs show
// whole mana. A free skill has no line. UNVERIFIED: the original only shows
// the decimal when manashift != 8 and the per-level step is at most 1 mana;
// that cannot differ for the shipped data (shift 8 costs are whole).
func ManaCost(label string, cost88 int) (string, bool) {
	if cost88 <= 0 || label == "" {
		return "", false
	}

	whole := cost88 >> 8
	if cost88 <= 0x1900 {
		if tenth := (cost88 & 0xff) * 10 >> 8; tenth != 0 {
			return label + tenths(whole, tenth), true
		}
	}

	if whole == 0 {
		return "", false
	}

	return label + strconv.Itoa(whole), true
}
