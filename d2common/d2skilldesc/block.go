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

	// KindToHit is kind 8 (0x4eaf90 -> 0x4e9380 -> 0x4e4f40, flag 1): the
	// skill's attack-rating bonus (SKILL_GetToHitBonus 0x645da0) as
	// "To Attack Rating: +n percent" (StrSkill10, StrSkill23). 0 gives no line.
	KindToHit = 8
	// KindPhysDamage is kind 9 (0x4eafe9 -> 0x4e9140): texta, textb, then
	// "Damage: " (StrSkill4) and the skill's physical damage range, see
	// PhysDamageLine.
	KindPhysDamage = 9
	// KindElemDamage is kind 10 (0x4eb048 -> 0x4e9080 -> 0x4e8f90): texta,
	// textb, the element label and the skill's elemental range, see
	// ElemDamageLine.
	KindElemDamage = 10
	// KindElemLength is kind 11 (0x4eb0ab -> 0x4eab70): the cold ("Cold
	// Length: ", StrSkill13) or poison ("Poison Length: ", StrSkill14)
	// duration of the skill's elemental damage; other elements give no line.
	KindElemLength = 11
	// KindLife is kind 13 (0x4eb140 -> 0x4e8ed0): "Life: " (StrSkill42) and the
	// life of the summoned monster, see LifeLine. It is NOT a descdam
	// dispatcher.
	KindLife = 13
	// KindLabelText is kind 15 (0x4eb1cd): texta, ": ", textb.
	KindLabelText = 15
	// KindDurationRange is kind 16 (0x4eb211 -> 0x4e4680): "Duration: "
	// (StrSkill20), calcA and calcB as frames shown as seconds "a-b seconds",
	// each end with one decimal when not whole. textb comes first, texta is
	// not printed.
	KindDurationRange = 16
	// KindWrapped is kind 18 (0x4eb272): texta alone as a wrapped paragraph.
	KindWrapped = 18
	// KindWrapped2 is kind 25 (0x4eb429): texta immediately followed by textb
	// as one wrapped paragraph.
	KindWrapped2 = 25
	// KindCurseDuration is kind 31 (0x4eb645 -> 0x4eb3c2 -> 0x4eaa50): texta
	// and calcA frames divided by a per-difficulty divisor (field +0x1c of the
	// 0x58-byte record from 0x610fd0, applied when positive; U: AiCurseDiv of
	// DifficultyLevels.txt) shown as seconds.
	KindCurseDuration = 31
	// KindSynergySigned is dsc3 kind 67 (0x4ebd56 -> 0x4e96b0, flags plus=1,
	// percent=0): like kind 63 without the "%".
	KindSynergySigned = 67
	// KindFormatted is dsc3 kind 71 (0x4ebde5): texta, ": ", then textb used
	// as a printf format with the calcA value. Both texts must be set.
	KindFormatted = 71
)

var (
	// UNVERIFIED (not read in the binary; taken from the shipped table):
	// texta + "a-b" + textb; kind 38 is the verified one of these.
	rangeKinds = map[int]bool{43: true}
	// UNVERIFIED: plain texta + n + textb (none left; 57 is decoded).
	plainKinds = map[int]bool{}
)

// Ctx carries the per-skill, per-level values that rows of kinds 1, 8-11 and
// 13 need beyond the calc evaluator. Every hook is optional; a row whose hook
// is nil gives no line.
type Ctx struct {
	// Mana renders the kind-1 line of the level being shown.
	Mana func() (string, bool)
	// ToHit is the skill's attack-rating bonus percent (kind 8).
	ToHit func() int
	// Phys is the skill's physical damage range in whole points, including
	// the weapon part (SrcDam/128 of the weapon) (kind 9).
	Phys func() (lo, hi int)
	// Elem is the skill's elemental range in whole points and the skills.txt
	// element type 1..5 (kind 10); ElemLen its length in frames (kind 11).
	Elem    func() (lo, hi, etype int)
	ElemLen func() int
	// Life is the average life of the summoned monster (kind 13).
	Life func() (avg int, ok bool)
	// ElemOverTime is the skill's elemental damage over its length for kind
	// 14: lo and hi are the per-frame range times the element length >> 8,
	// frames the length (ELen), etype the skills.txt element.
	ElemOverTime func() (lo, hi, frames, etype int)
	// MissileDamage is the missile's average damage per second range and
	// element for kind 22 (UNVERIFIED how the range is computed).
	MissileDamage func() (lo, hi, etype int)
	// MissileRange is the missile's lifetime in frames, LevRange*level+Range,
	// for kind 23.
	MissileRange func() int
	// CurseDiv is the divisor of kind 31; values below 1 mean no division.
	CurseDiv int
}

// Block renders rows without a mana line (kind 1 rows are skipped).
func Block(rows []Row, tr func(string) string, eval func(string) int) []string {
	return BlockCtx(rows, tr, eval, nil)
}

// BlockMana renders rows in order. mana, when non-nil, renders the kind-1 line
// of the level being shown.
func BlockMana(rows []Row, tr func(string) string, eval func(string) int, mana func() (string, bool)) []string {
	return BlockCtx(rows, tr, eval, &Ctx{Mana: mana})
}

// BlockCtx renders rows in order. tr translates a text column (returning the
// key when it is unknown; empty stays empty), eval evaluates a calc source at
// the level being shown and ctx (may be nil) supplies the skill values.
// Unmodelled kinds are skipped. The game appends every line to one text
// buffer, so the caller shows them in the same box (VERIFIED: 0x4a86b0 calls
// one builder and draws once).
func BlockCtx(rows []Row, tr func(string) string, eval func(string) int, ctx *Ctx) []string {
	var out []string

	for _, r := range rows {
		if line, ok := RowLineCtx(r, tr, eval, ctx); ok {
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

// RowLine renders one row without skill context.
func RowLine(r Row, tr func(string) string, eval func(string) int) (string, bool) {
	return RowLineCtx(r, tr, eval, nil)
}

// RowLineCtx renders one row.
func RowLineCtx(r Row, tr func(string) string, eval func(string) int, ctx *Ctx) (string, bool) {
	a, b := tr(r.TextA), tr(r.TextB)
	va, vb := eval(r.CalcA), eval(r.CalcB)

	if ctx == nil {
		ctx = &Ctx{}
	}

	if line, ok, handled := damageRow(r, a, b, va, vb, tr, ctx); handled {
		return line, ok
	}

	if line, ok, handled := moreRow(r, a, b, va, vb, tr, ctx, eval); handled {
		return line, ok
	}

	switch {
	case r.Kind == KindMana:
		if ctx.Mana == nil {
			return "", false
		}

		return ctx.Mana()
	case r.Kind == KindLabelText:
		return a + ": " + b, true
	case r.Kind == KindDurationRange:
		return durationRangeLine(b, tr("StrSkill20"), va, vb, tr)
	case r.Kind == KindWrapped:
		return a, a != ""
	case r.Kind == KindWrapped2:
		return a + b, a+b != ""
	case r.Kind == KindCurseDuration:
		if ctx.CurseDiv > 0 {
			va /= ctx.CurseDiv
		}

		return durationLine(a, "", va, tr)
	case r.Kind == KindSynergySigned:
		return synergyLine(a, b, va, false), true
	case r.Kind == KindFormatted:
		if r.TextA == "" || r.TextB == "" {
			return "", false
		}

		return a + ": " + strings.Replace(b, "%d", strconv.Itoa(va), 1), true
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

		return synergyLine(a, b, va, true), true
	case r.Kind == KindSynergyHead:
		if r.TextA == "" {
			return "", false
		}

		return strings.Replace(a, "%s", b, 1), true
	}

	return "", false
}

// synergyLine is the dsc3 kind-63 line (percent) and the kind-67 line (no
// percent): [texta ": "] signed value ["%"] " " textb, 0x4e96b0.
func synergyLine(a, b string, v int, percent bool) string {
	head := ""
	if a != "" {
		head = a + ": "
	}

	pct := ""
	if percent {
		pct = "%"
	}

	return head + signed(v, true) + pct + " " + b
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

// CurrentLevel is the "Current Skill Level: n" label line (StrSkill2, 0x109e,
// built at 0x4ec3dd/0x4ec5d0): the label then the level, no separator.
func CurrentLevel(label string, level int) string {
	return label + strconv.Itoa(level)
}

// NextLevel renders the "Next Level" block: the label (StrSkill1 "Next Level",
// or StrSkill17 "First Level" when the skill has no points yet, chosen by the
// caller), then the main block evaluated at level+1 (damage lines come from
// its kind 8-11 and 13 rows through ctx, at their row position). The second
// (dsc2) block is NOT part of it: the builder at 0x4ec180 evaluates descline
// indexes 0..5 only for this block. ctx supplies the level+1 values. It is
// empty when nothing would be shown or when the skill is at its cap
// (maxLvl > 0).
func NextLevel(label string, d Desc, level, maxLvl int,
	tr func(string) string, evalNext func(string) int, ctx *Ctx) []string {
	if maxLvl > 0 && level >= maxLvl {
		return nil
	}

	body := BlockCtx(d.Lines, tr, evalNext, ctx)

	if len(body) == 0 {
		return nil
	}

	return append([]string{label}, body...)
}
