package d2skilldesc

import "strconv"

// The original has no single "Damage" line. Damage appears where a skilldesc
// row of kind 8 (attack rating), 9 (physical), 10 (elemental), 11 (cold or
// poison length) or 13 (summon life) sits. All numbers below are VERIFIED
// against Game.exe 1.14b (see ~/git/d2-re-notes/skilldesc-damage.md).

// elemKeys are the string keys of the element labels, from the helper at
// 0x4e3fa0 (skills.txt EType 1 fire, 2 lightning, 3 magic, 4 cold, 5 poison):
// "Fire Damage: " (0x10a1) and so on.
var elemKeys = map[int]string{
	1: "StrSkill5", 2: "StrSkill7", 3: "StrSkill39", 4: "StrSkill6", 5: "StrSkill8",
}

// ElemKey is the label key of an element type, "" when the type has none.
func ElemKey(etype int) string { return elemKeys[etype] }

// damageRow renders rows of kinds 8-11 and 13. handled is false for every
// other kind.
func damageRow(r Row, a, b string, va, vb int, tr func(string) string, ctx *Ctx) (line string, ok, handled bool) {
	switch r.Kind {
	case KindToHit:
		if ctx.ToHit == nil {
			return "", false, true
		}

		// 0x4e9380 -> 0x4e4f40 flag 1: label, "+n" when not negative, " percent".
		v := ctx.ToHit()
		if v == 0 {
			return "", false, true
		}

		return tr("StrSkill10") + signed(v, true) + tr("StrSkill23"), true, true
	case KindPhysDamage:
		if ctx.Phys == nil {
			return "", false, true
		}

		lo, hi := ctx.Phys()
		line, ok = PhysDamageLine(a+b, tr("StrSkill4"), lo, hi, va, vb)

		return line, ok, true
	case KindElemDamage:
		if ctx.Elem == nil {
			return "", false, true
		}

		lo, hi, et := ctx.Elem()
		line, ok = ElemDamageLine(a+b, tr(ElemKey(et)), lo, hi)

		return line, ok, true
	case KindElemLength:
		if ctx.Elem == nil || ctx.ElemLen == nil {
			return "", false, true
		}

		_, _, et := ctx.Elem()

		key := map[int]string{4: "StrSkill13", 5: "StrSkill14"}[et]
		if key == "" {
			return "", false, true
		}

		line, ok = durationLine(tr(key), "", ctx.ElemLen(), tr)

		return line, ok, true
	case KindLife:
		if ctx.Life == nil {
			return "", false, true
		}

		avg, has := ctx.Life()
		if !has {
			return "", false, true
		}

		line, ok = LifeLine(a+b, tr("StrSkill42"), avg, va, vb)

		return line, ok, true
	}

	return "", false, false
}

// scalePct is v + v*pct/100 + flat, the way 0x4e9140 and 0x4e8ed0 combine a
// base value with the row's calcA (percent bonus) and calcB (flat bonus).
func scalePct(v, pct, flat int) int { return v + v*pct/100 + flat }

// PhysDamageLine is the kind-9 line (0x4e9140). lo and hi are the skill's
// physical minimum and maximum in whole points (weapon part included, 8.8
// fixed point shifted down by 8); each end becomes lo + lo*pct/100 + flat
// with pct = calcA and flat = calcB. Both ends 0 gives no line. Different
// ends print "a-b"; equal ends print "+n" ("n" when negative). prefix is
// texta+textb, label the "Damage: " text (StrSkill4, 0x10a0).
func PhysDamageLine(prefix, label string, lo, hi, pct, flat int) (string, bool) {
	lo, hi = scalePct(lo, pct, flat), scalePct(hi, pct, flat)
	if lo == 0 && hi == 0 {
		return "", false
	}

	if lo != hi {
		return prefix + label + strconv.Itoa(lo) + "-" + strconv.Itoa(hi), true
	}

	return prefix + label + signed(lo, true), true
}

// ElemDamageLine is the kind-10 line (0x4e9080 -> 0x4e8f90): the skill's
// elemental range in whole points, "a-b" or "n" when equal (no plus sign),
// preceded by texta+textb and the element label. Both ends 0, or an element
// without a label, give no line. The calcs are not used.
func ElemDamageLine(prefix, label string, lo, hi int) (string, bool) {
	if (lo == 0 && hi == 0) || label == "" {
		return "", false
	}

	if lo != hi {
		return prefix + label + strconv.Itoa(lo) + "-" + strconv.Itoa(hi), true
	}

	return prefix + label + strconv.Itoa(lo), true
}

// LifeLine is the kind-13 line (0x4eb140 -> 0x4e8ed0): "Life: " (StrSkill42,
// 0x10c6) and avg + avg*pct/100 + flat, where avg is the mean of the summoned
// monster's minimum and maximum life and pct, flat are calcA and calcB. A
// result of 0 gives no line. The number is plain (0x4e4a30 flag 0).
func LifeLine(prefix, label string, avg, pct, flat int) (string, bool) {
	v := scalePct(avg, pct, flat)
	if v == 0 {
		return "", false
	}

	return prefix + label + strconv.Itoa(v), true
}

// durationRangeLine is kind 16 (0x4e4680): two frame counts, each split into
// whole seconds (v/25) and tenths ((v%25)*10/25), printed as "a-b" with the
// "Duration: " label and the plural unit; "a.x-b.y" when either end has a
// tenth. 0 at both ends gives no line. prefix (textb) comes first.
func durationRangeLine(prefix, label string, x, y int, tr func(string) string) (string, bool) {
	wx, tx := x/25, (x%25)*10/25
	wy, ty := y/25, (y%25)*10/25

	if wx == 0 && tx == 0 && wy == 0 {
		return "", false
	}

	num := strconv.Itoa(wx) + "-" + strconv.Itoa(wy)
	if tx != 0 || ty != 0 {
		num = strconv.Itoa(wx) + "." + strconv.Itoa(tx) + "-" + strconv.Itoa(wy) + "." + strconv.Itoa(ty)
	}

	return prefix + label + num + tr("StrSkill16"), true
}
