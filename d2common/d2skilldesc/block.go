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
// Lines2 is dsc2line1..4 (a second block of the same shape) and Lines3 is
// dsc3line1..7 (the synergy/bonus block). UNVERIFIED: that the blocks are
// shown in this order and that Lines2 shares Lines' per-level behaviour. The
// binary's line dispatcher was not read: 0x4e6fd0 is SKILLDESC_DamLineFunc_15,
// the damage-line function of descdam 15 (Dragon Talon), not the line switch.
// It does confirm descdam is a function index and that a damage line is
// physical (weapon scaled by a byte/128) + elemental min/max.
type Desc struct {
	DescDam int // descdam: index of the skill's damage-line function, 0 = none
	Lines   []Row
	Lines2  []Row
	Lines3  []Row
}

// Further line kinds, all inferred from the shipped skilldesc.txt only
// (UNVERIFIED wording and signs).
const (
	// KindSignedNeg is kind 5 (calc "-edmn" style): a signed value.
	KindSignedNeg = 5
	// KindSynergyHead is dsc3 kind 40: a header "texta textb" (Sksyn + skill).
	KindSynergyHead = 40
)

var (
	// texta + n + textb, e.g. 12 "inner sight": StrSkill20 + ln34.
	plainKinds = map[int]bool{4: true, 6: true, 12: true, 19: true, 31: true, 37: true, 57: true}
	// texta + "a-b" + textb; the original may word it "a to b".
	rangeKinds = map[int]bool{16: true, 38: true, 43: true, 52: true}
	// dsc3 synergy: "texta: +n% textb" (63, 67, 71).
	synKinds = map[int]bool{63: true, 67: true, 71: true}
)

// Block renders rows. tr translates a text column (returning the key when it
// is unknown; empty stays empty), eval evaluates a calc source at the level
// being shown. Unmodelled kinds are skipped.
func Block(rows []Row, tr func(string) string, eval func(string) int) []string {
	var out []string

	for _, r := range rows {
		if line, ok := RowLine(r, tr, eval); ok {
			out = append(out, line)
		}
	}

	return out
}

// RowLine renders one row.
func RowLine(r Row, tr func(string) string, eval func(string) int) (string, bool) {
	a, b := tr(r.TextA), tr(r.TextB)
	va, vb := eval(r.CalcA), eval(r.CalcB)

	switch {
	case r.Kind == KindSignedValue || r.Kind == KindValue || r.Kind == KindCount:
		return FormatLine(r.Kind, a, b, va)
	case r.Kind == KindSignedNeg:
		return FormatLine(KindSignedValue, a, b, va)
	case plainKinds[r.Kind]:
		return FormatLine(KindValue, a, b, va)
	case rangeKinds[r.Kind]:
		return a + strconv.Itoa(va) + "-" + strconv.Itoa(vb) + b, true
	case synKinds[r.Kind]:
		return a + ": +" + strconv.Itoa(va) + "% " + b, true
	case r.Kind == KindSynergyHead:
		if a == "" && b == "" {
			return "", false
		}

		return strings.TrimSpace(strings.TrimRight(a, " ") + " " + b), true
	}

	return "", false
}

// Damage formats the descdam line from the skill's own damage in whole points.
// UNVERIFIED: the original also adds the weapon part scaled by SrcDam/128.
// No damage gives no line.
func Damage(label string, minDmg, maxDmg int) (string, bool) {
	if maxDmg <= 0 || label == "" {
		return "", false
	}

	if minDmg == maxDmg {
		return label + strconv.Itoa(maxDmg), true
	}

	return label + strconv.Itoa(minDmg) + " to " + strconv.Itoa(maxDmg), true
}

// NextLevel renders the "Next Level" block: the label, then the damage line
// and the main and second blocks evaluated by evalNext (the calc evaluator at
// level+1). It is empty when nothing would be shown or when the skill is at
// its cap (maxLvl > 0).
func NextLevel(label string, d Desc, level, maxLvl int, dmg string,
	tr func(string) string, evalNext func(string) int) []string {
	if maxLvl > 0 && level >= maxLvl {
		return nil
	}

	var body []string
	if dmg != "" {
		body = append(body, dmg)
	}

	body = append(body, Block(d.Lines, tr, evalNext)...)
	body = append(body, Block(d.Lines2, tr, evalNext)...)

	if len(body) == 0 {
		return nil
	}

	return append([]string{label}, body...)
}
