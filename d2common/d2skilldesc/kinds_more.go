package d2skilldesc

import "strings"

// More skilldesc line kinds, read from the handlers of the dispatch at 0x4eae9b
// (jump table 0x4ec048) in Game.exe 1.14b. Notes:
// ~/git/d2-re-notes/skilldesc-remaining-kinds.md. VERIFIED means the handler
// and its helper were read; the data hooks in Ctx are what the caller has to
// supply from the skill and missile records.
const (
	// KindPoisonOverTime is kind 14 (0x4eb16c -> 0x4eabd0): texta and textb
	// (when set) raw, then the element label of the skill (skills.txt EType)
	// with the per-frame elemental range multiplied by the element length,
	// "lo-hi" (or "n" when equal), and "over " (StrSkill63Patch, 0x2b0e) with
	// the length as seconds. Used by the poison skills (Poison Javelin,
	// Plague Javelin, Poison Dagger, Venom, Rabies, ...).
	KindPoisonOverTime = 14
	// KindPercentSigned is kind 20 (0x4eb33e -> 0x4e52e0, flag 1): texta,
	// "+n", " percent" (0x10b3), textb; kind 21 (0x4eb362) the same without
	// the plus. Zero gives no line. No shipped row uses them.
	KindPercentSigned = 20
	// KindPercent is kind 21, see KindPercentSigned.
	KindPercent = 21
	// KindAvgPerSecond is kind 22 (0x4eb366 -> 0x4e94b0 -> 0x4e93b0):
	// "Average " (StrSkill68) + element label + "lo-hi" (or "n") + " per
	// second" (StrSkill34). The range comes from the missile record named by
	// the skill (word at +0x3c), see Ctx.MissileDamage.
	KindAvgPerSecond = 22
	// KindMissileDuration is kind 23 (0x4eb397 -> 0x4eaa50 flag 0): texta and
	// the missile's frames, LevRange*level + Range (missile record words +0x98
	// and +0x96, missile id = word +0x3c of the skill record), as seconds.
	KindMissileDuration = 23
	// KindElemDamageLabelled is kind 24 (0x4eb3de -> 0x4e9080): textb first,
	// then the kind-10 elemental range with texta as the label instead of the
	// element label.
	KindElemDamageLabelled = 24
	// KindMastery is kind 51 (0x4ebafc -> 0x4e4990): texta used as a printf
	// format with calcA ("%d Percent Chance of Critical Strike"), newline.
	KindMastery = 51
	// KindRangePlus is kind 52 (0x4ebb18 -> 0x4e5020, flag 1): texta, "+a-b",
	// textb; a == b gives the kind-3 line.
	KindRangePlus = 52
	// KindDurationPlus is kind 57 (0x4ebbe0 -> 0x4eaa50 flag 1): kind 12 with
	// a plus sign (the tenths branch 0x4e4470 was not read; UNVERIFIED).
	KindDurationPlus = 57
	// KindRangeTextPlus is kind 58 (0x4ebbe9 -> 0x4e5110, flag 1) and kind 62
	// (0x4ebcb6, flag 0): texta, textb, then "lo-hi" (with a leading "+" for
	// 58) and a newline; lo == hi gives the kind-3 line. 58 has no shipped row.
	KindRangeTextPlus = 58
	// KindRangeText is kind 62, see KindRangeTextPlus.
	KindRangeText = 62
	// KindFormatNoZero is kind 66 (0x4ebd3a -> 0x4e9b60): like 51 but an
	// empty texta gives no line.
	KindFormatNoZero = 66
	// KindFractionSigned is kind 72 (0x4ebe67): "+a/b texta", needs a != 0 and
	// b > 0; kind 73 (0x4ebf0c) is "a/b texta" without the plus.
	KindFractionSigned = 72
	// KindFraction is kind 73, see KindFractionSigned.
	KindFraction = 73
)

// sprintfD is the swprintf of one %d (and %% for a literal percent).
func sprintfD(format string, v int) string {
	format = strings.Replace(format, "%d", signed(v, false), 1)

	return strings.ReplaceAll(format, "%%", "%")
}

// rangeText is 0x4e5110 / 0x4e5020: "lo-hi" with an optional plus on lo.
func rangeNum(lo, hi int, plus bool) string {
	p := ""
	if plus {
		p = "+"
	}

	return p + signed(lo, false) + "-" + signed(hi, false)
}

// moreRow renders the kinds of this file. handled is false for other kinds.
func moreRow(r Row, a, b string, va, vb int, tr func(string) string, ctx *Ctx,
	eval func(string) int) (line string, ok, handled bool) {
	switch r.Kind {
	case KindPoisonOverTime:
		if ctx.ElemOverTime == nil {
			return "", false, true
		}

		lo, hi, frames, et := ctx.ElemOverTime()
		label := tr(ElemKey(et))

		if label == "" {
			return "", false, true
		}

		dmg := a + b + label + signed(lo, false)
		if lo != hi {
			dmg = a + b + label + rangeNum(lo, hi, false)
		}

		// The handler appends the "over n seconds" line first and the damage
		// line second; the tooltip text box shows the later append above
		// (VERIFIED for the order of calls; the display direction is inferred
		// from descline order, U).
		over, has := durationLine(tr("StrSkill63Patch"), "", frames, tr)
		if !has {
			// 0x4eaa50 returns without text for a zero length.
			return dmg, true, true
		}

		return dmg + "\n" + over, true, true
	case KindPercentSigned, KindPercent:
		if va == 0 {
			return "", false, true
		}

		return a + signed(va, r.Kind == KindPercentSigned) + tr("StrSkill23") + b, true, true
	case KindAvgPerSecond:
		if ctx.MissileDamage == nil {
			return "", false, true
		}

		lo, hi, et := ctx.MissileDamage()
		label := tr(ElemKey(et))

		if (lo == 0 && hi == 0) || label == "" {
			return "", false, true
		}

		num := signed(lo, false)
		if lo != hi {
			num = rangeNum(lo, hi, false)
		}

		return tr("StrSkill68") + label + num + tr("StrSkill34"), true, true
	case KindMissileDuration:
		if ctx.MissileRange == nil {
			return "", false, true
		}

		line, ok = durationLine(a, "", ctx.MissileRange(), tr)

		return line, ok, true
	case KindElemDamageLabelled:
		if ctx.Elem == nil {
			return "", false, true
		}

		lo, hi, _ := ctx.Elem()
		// An element without a label would give no line in kind 10; here the
		// label is texta, so only the zero check applies.
		if lo == 0 && hi == 0 {
			return "", false, true
		}

		num := signed(lo, false)
		if lo != hi {
			num = rangeNum(lo, hi, false)
		}

		return b + a + num, true, true
	case KindMastery:
		if r.TextA == "" {
			return "", false, true
		}

		return sprintfD(a, va), true, true
	case KindFormatNoZero:
		if r.TextA == "" {
			return "", false, true
		}

		return sprintfD(a, va), true, true
	case KindRangePlus:
		if va == vb {
			line, ok = RowLineCtx(Row{Kind: KindValue, TextA: r.TextA, TextB: r.TextB, CalcA: r.CalcA}, tr, eval, ctx)

			return line, ok, true
		}

		return a + rangeNum(va, vb, true) + b, true, true
	case KindDurationPlus:
		line, ok = durationLine(a, "", va, tr)
		if ok {
			line = strings.Replace(line, a, a+"+", 1)
		}

		return line, ok, true
	case KindRangeTextPlus, KindRangeText:
		if va == vb {
			line, ok = RowLineCtx(Row{Kind: KindValue, TextA: r.TextA, TextB: r.TextB, CalcA: r.CalcA}, tr, eval, ctx)

			return line, ok, true
		}

		return a + b + rangeNum(va, vb, r.Kind == KindRangeTextPlus), true, true
	case KindFractionSigned, KindFraction:
		if va == 0 || vb <= 0 {
			return "", false, true
		}

		return signed(va, r.Kind == KindFractionSigned) + "/" + signed(vb, false) + " " + a, true, true
	}

	return "", false, false
}
