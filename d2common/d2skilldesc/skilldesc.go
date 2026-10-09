// Package d2skilldesc formats the text lines of a skill tooltip from the
// skilldesc.txt columns. It is pure: strings and numbers in, strings out.
//
// Status: the line kinds below are the ones whose output is fully determined
// by the table data (text A, a number, text B). The meaning of the other
// descline types (the original's switch in UI\SkillDesc.cpp, 0x4e6fd0, which
// the Ghidra instance could not be asked about in this pass) is UNVERIFIED and
// those lines are not produced.
package d2skilldesc

import "strconv"

// Line kinds (the descline/dsc2line/dsc3line column) this package formats.
// UNVERIFIED against the binary: derived from the shipped skilldesc.txt, where
// kind 2 rows read "Damage: " + n + " percent" (shown with a sign, as bonuses
// are), kind 3 "Ranged attacks slowed to " + n + " percent" and kind 7
// n + " arrows" (no sign).
const (
	KindSignedValue = 2 // texta + "+n" + textb
	KindValue       = 3 // texta + n + textb
	KindCount       = 7 // n + texta (the text follows the number)
)

// FormatLine renders one description line. ok is false for kinds that are not
// modelled, so the caller can skip them rather than print wrong text.
func FormatLine(kind int, texta, textb string, value int) (line string, ok bool) {
	n := strconv.Itoa(value)

	switch kind {
	case KindSignedValue:
		if value >= 0 {
			n = "+" + n
		}

		return texta + n + textb, true
	case KindValue:
		return texta + n + textb, true
	case KindCount:
		return n + texta, true
	}

	return "", false
}

// ManaCost formats the "Mana Cost: n" line from the "str mana" label and the
// cost in the engine's 8.8 fixed point (d2combat.ManaCost). The game shows
// whole mana, so the fraction is dropped (UNVERIFIED rounding). A free skill
// has no line.
func ManaCost(label string, cost88 int) (string, bool) {
	if cost88 <= 0 || label == "" {
		return "", false
	}

	return label + strconv.Itoa(cost88>>8), true
}
