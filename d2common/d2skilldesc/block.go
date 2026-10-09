package d2skilldesc

import (
	"strconv"
	"strings"
)

// Row is one description line of a skilldesc.txt row (descline/dsc2line/dsc3line
// n with its texta, textb, calca and calcb columns). Calcs are source strings
// the caller evaluates.
type Row struct {
	Kind         int
	TextA, TextB string
	CalcA, CalcB string
}

// Desc groups the rows of a skill: Lines is descline1..6 (the main block),
// Lines2 is dsc2line1..4 and Lines3 is dsc3line1..7 (the synergy block).
//
// VERIFIED (Game.exe 1.14b, ~/git/d2-re-notes/verify-skilldesc.md): the three
// blocks are line indexes 0..5, 6..9 and 10..16 of one list; the per-line kind
// switch is the 75-entry jump table at 0x4ec048 reached from 0x4eae9b inside
// the function at 0x4eade0, which the tooltip builders 0x4ec6d0 (current-level
// text) and 0x4ec180 (adds the Next Level block) call once per line index.
// The builders append bottom-up (the last thing appended is the skill name),
// so on screen, top to bottom: name and description, the dsc2 block, the
// "Current Skill Level: n" label with the descline block, the "Next Level"
// (or "First Level") label with the descline block of level+1, then the dsc3
// block (evaluated at level+1 in both builders).
type Desc struct {
	DescDam int // descdam: index of the skill's damage-line function, 0 = none
	Lines   []Row
	Lines2  []Row
	Lines3  []Row
}

// Line kinds, VERIFIED against the jump table at 0x4ec048 (entry = kind-1).
const (
	// KindMana is kind 1 (0x4eaea2 -> 0x4e5720): the "str mana" label and the
	// skill's mana cost at the level, not a texta/calc line. A skill with
	// mana and lvlmana both 0 has no line.
	KindMana = 1
	// KindSignedValue is kind 2 (0x4eaeda -> 0x4e4f40, flag 1): texta, the
	// value with "+" when it is not negative, textb. Value 0 gives no line.
	KindSignedValue = 2
	// KindValue is kind 3 (0x4eaef9 -> 0x4e4f40, flag 0): texta, value, textb.
	KindValue = 3
	// KindSignedText is kind 4 (0x4eaf18 -> 0x4e4a30, flag 1): texta and the
	// signed value; textb is not printed.
	KindSignedText = 4
	// KindText is kind 5 (0x4eaf36 -> 0x4e4a30, flag 0): texta and the value.
	KindText = 5
	// KindSignedPrefix is kind 6 (0x4eaf54 -> 0x4e4cc0, flag 1): the signed
	// value, then texta.
	KindSignedPrefix = 6
	// KindCount is kind 7 (0x4eaf72 -> 0x4e4cc0, flag 0): value, then texta.
	KindCount = 7
	// KindDuration is kind 12 (0x4eb10b -> 0x4eaa50): texta, the calc in game
	// frames shown as seconds (25 per second, one decimal when not whole),
	// then " second" (exactly 1) or " seconds" (StrSkill15/16). textb, when
	// set, is written before the line. Zero gives no line.
	KindDuration = 12
	// KindRadius is kind 19 (0x4eb30b -> 0x4e53d0): texta, calc*2/3 with one
	// decimal (at least 1), then " yard" (exactly 1) or " yards"
	// (StrSkill36/26). Kind 37 (0x4eb77c -> 0x4e5520) is the same with calc/3.
	KindRadius = 19
	// KindRadiusThird is kind 37, see KindRadius.
	KindRadiusThird = 37
	// KindRange is kind 38 (0x4eb798 -> 0x4e5020, flag 0): texta, "a-b", textb;
	// a == b gives the single-value line of kind 3.
	KindRange = 38
	// KindSynergyHead is dsc3 kind 40 (0x4eb7f0): textb is substituted into the
	// "%s" of texta (Sksyn: "%s Receives Bonuses From:"). Empty texta gives
	// no line.
	KindSynergyHead = 40
	// KindSynergy is dsc3 kind 63 (0x4ebcd8 -> 0x4e96b0, flags 1,1): texta,
	// ": ", the signed value, "%", " ", textb. Value 0 gives no line.
	KindSynergy = 63
)

var (
	// UNVERIFIED (not read in the binary; taken from the shipped table):
	// texta + "a-b" + textb; kind 38 is the verified one of these.
	rangeKinds = map[int]bool{16: true, 43: true, 52: true}
	// UNVERIFIED: plain texta + n + textb (31 goes through 0x4eb645 and a
	// divisor, 57 was not read).
	plainKinds = map[int]bool{31: true, 57: true}
	// UNVERIFIED: dsc3 synergy variants of kind 63 (67, 71 not read).
	synKinds = map[int]bool{67: true, 71: true}
)

// Block renders rows without a mana line (kind 1 rows are skipped).
func Block(rows []Row, tr func(string) string, eval func(string) int) []string {
	return BlockMana(rows, tr, eval, nil)
}

// BlockMana renders rows in order. mana, when non-nil, renders the kind-1 line
// of the level being shown. tr translates a text column (returning the key
// when it is unknown; empty stays empty), eval evaluates a calc source at the
// level being shown. Unmodelled kinds are skipped.
func BlockMana(rows []Row, tr func(string) string, eval func(string) int, mana func() (string, bool)) []string {
	var out []string

	for _, r := range rows {
		if r.Kind == KindMana {
			if mana == nil {
				continue
			}

			if line, ok := mana(); ok {
				out = append(out, line)
			}

			continue
		}

		if line, ok := RowLine(r, tr, eval); ok {
			out = append(out, line)
		}
	}

	return out
}

// signed formats n with "%d" or, when plus is set and n >= 0, "+%d".
func signed(n int, plus bool) string {
	if plus && n >= 0 {
		return "+" + strconv.Itoa(n)
	}

	return strconv.Itoa(n)
}

// tenths formats whole.tenth, or just whole when tenth is 0.
func tenths(whole, tenth int) string {
	if tenth == 0 {
		return strconv.Itoa(whole)
	}

	return strconv.Itoa(whole) + "." + strconv.Itoa(tenth)
}

// RowLine renders one row.
func RowLine(r Row, tr func(string) string, eval func(string) int) (string, bool) {
	a, b := tr(r.TextA), tr(r.TextB)
	va, vb := eval(r.CalcA), eval(r.CalcB)

	switch {
	case r.Kind == KindSignedValue || r.Kind == KindValue:
		if va == 0 {
			return "", false
		}

		return a + signed(va, r.Kind == KindSignedValue) + b, true
	case r.Kind == KindSignedText || r.Kind == KindText:
		if va == 0 {
			return "", false
		}

		return a + signed(va, r.Kind == KindSignedText), true
	case r.Kind == KindSignedPrefix || r.Kind == KindCount:
		if va == 0 {
			return "", false
		}

		return signed(va, r.Kind == KindSignedPrefix) + a, true
	case r.Kind == KindDuration:
		return durationLine(a, b, va, tr)
	case r.Kind == KindRadius || r.Kind == KindRadiusThird:
		return radiusLine(a, b, va, r.Kind == KindRadius, tr)
	case r.Kind == KindRange:
		if va == vb {
			return RowLine(Row{Kind: KindValue, TextA: r.TextA, TextB: r.TextB, CalcA: r.CalcA}, tr, eval)
		}

		return a + strconv.Itoa(va) + "-" + strconv.Itoa(vb) + b, true
	case plainKinds[r.Kind]:
		return a + strconv.Itoa(va) + b, true
	case rangeKinds[r.Kind]:
		return a + strconv.Itoa(va) + "-" + strconv.Itoa(vb) + b, true
	case r.Kind == KindSynergy:
		if va == 0 {
			return "", false
		}

		return synergyLine(a, b, va), true
	case synKinds[r.Kind]:
		return a + ": +" + strconv.Itoa(va) + "% " + b, true
	case r.Kind == KindSynergyHead:
		if r.TextA == "" {
			return "", false
		}

		return strings.Replace(a, "%s", b, 1), true
	}

	return "", false
}

// synergyLine is the dsc3 kind-63 line.
func synergyLine(a, b string, v int) string {
	head := ""
	if a != "" {
		head = a + ": "
	}

	return head + signed(v, true) + "% " + b
}

// durationLine is kind 12: frames to seconds, 0x4eaa50.
func durationLine(a, b string, frames int, tr func(string) string) (string, bool) {
	whole := frames / 25
	tenth := (frames - whole*25) * 10 / 25

	if whole == 0 && tenth == 0 {
		return "", false
	}

	unit := tr("StrSkill16")
	if whole == 1 && tenth == 0 {
		unit = tr("StrSkill15")
	}

	return b + a + tenths(whole, tenth) + unit, true
}

// radiusLine is kinds 19 and 37: v*2/3 or v/3 yards with one decimal,
// 0x4e53d0 and 0x4e5520.
func radiusLine(a, b string, v int, twoThirds bool, tr func(string) string) (string, bool) {
	scaled := v * 10 / 3
	if twoThirds {
		scaled = v * 20 / 3
	}

	whole, tenth := scaled/10, scaled%10
	if whole < 1 {
		whole = 1
	}

	unit := tr("StrSkill26")
	if whole == 1 {
		unit = tr("StrSkill36")
	}

	return b + a + tenths(whole, tenth) + unit, true
}

// Damage formats a damage line from the skill's own damage in whole points:
// label (the StrSkill4 "Damage: " text), then "a-b", or "n" when both are
// equal. VERIFIED wording and format for the generic damage lines (kind 9
// 0x4e9140 and the elemental kind 10 0x4e8f90 both use "%d-%d"); kind 9 writes
// "+n" when min and max are equal. UNVERIFIED: the original builds damage from
// the rows of kinds 8-11 and 13 (descdam function 0x4e8ed0) at their row
// position, not as one synthesized line; no damage gives no line.
func Damage(label string, minDmg, maxDmg int) (string, bool) {
	if maxDmg <= 0 || label == "" {
		return "", false
	}

	if minDmg == maxDmg {
		return label + strconv.Itoa(maxDmg), true
	}

	return label + strconv.Itoa(minDmg) + "-" + strconv.Itoa(maxDmg), true
}

// CurrentLevel is the "Current Skill Level: n" label line (StrSkill2, 0x109e,
// built at 0x4ec3dd/0x4ec5d0): the label then the level, no separator.
func CurrentLevel(label string, level int) string {
	return label + strconv.Itoa(level)
}

// NextLevel renders the "Next Level" block: the label (StrSkill1 "Next Level",
// or StrSkill17 "First Level" when the skill has no points yet, chosen by the
// caller), then the damage line and the main block evaluated at level+1. The
// second (dsc2) block is NOT part of it: the builder at 0x4ec180 evaluates
// descline indexes 0..5 only for this block. mana renders the kind-1 line of
// level+1. It is empty when nothing would be shown or when the skill is at its
// cap (maxLvl > 0).
func NextLevel(label string, d Desc, level, maxLvl int, dmg string,
	tr func(string) string, evalNext func(string) int, mana func() (string, bool)) []string {
	if maxLvl > 0 && level >= maxLvl {
		return nil
	}

	var body []string
	if dmg != "" {
		body = append(body, dmg)
	}

	body = append(body, BlockMana(d.Lines, tr, evalNext, mana)...)

	if len(body) == 0 {
		return nil
	}

	return append([]string{label}, body...)
}
