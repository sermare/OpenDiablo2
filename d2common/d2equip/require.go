package d2equip

import "fmt"

// Requirement is the outcome of the requirement check of one item.
type Requirement struct {
	NeedStr, NeedDex, NeedLevel int
	HaveStr, HaveDex, HaveLevel int

	StrOK, DexOK, LevelOK, ClassOK, IdentOK bool
}

// OK reports whether every requirement holds.
func (r Requirement) OK() bool { return r.StrOK && r.DexOK && r.LevelOK && r.ClassOK && r.IdentOK }

// Reason returns the first failing reason, in the order the checks run in
// the game (identification, class, strength, dexterity, level).
func (r Requirement) Reason() Reason {
	switch {
	case !r.IdentOK:
		return ReasonUnidentified
	case !r.ClassOK:
		return ReasonClass
	case !r.StrOK:
		return ReasonStrength
	case !r.DexOK:
		return ReasonDexterity
	case !r.LevelOK:
		return ReasonLevel
	}

	return ReasonOK
}

// Detail explains the verdict in words.
func (r Requirement) Detail() string {
	switch r.Reason() {
	case ReasonStrength:
		return fmt.Sprintf("needs %d strength, has %d", r.NeedStr, r.HaveStr)
	case ReasonDexterity:
		return fmt.Sprintf("needs %d dexterity, has %d", r.NeedDex, r.HaveDex)
	case ReasonLevel:
		return fmt.Sprintf("needs level %d, is %d", r.NeedLevel, r.HaveLevel)
	}

	return string(r.Reason())
}

// RequiredStat is the strength (or dexterity) an item needs: the base
// requirement raised by the item's requirement percent stat, minus 10 for an
// ethereal item (INV_CheckItemRequirements 0x62ebf0, VERIFIED including the
// arithmetic: the bonus is MulDiv(base, percent, 100) = base*percent/100 with
// truncating integer division (0x47f2c0), item stat 0x5b; the ethereal 10 is
// taken off the bonus before the base is added, and the same percent is used
// for strength and dexterity).
func RequiredStat(base, percent int, ethereal bool) int {
	need := base + base*percent/100
	if ethereal {
		need -= 10
	}

	return need
}

// RequiredLevel is the level an item needs: the highest of the base item's
// levelreq and the levelreq of each affix / unique / set row on it. VERIFIED
// (the level function at 0x62b720, called by 0x62ebf0): by quality it takes
// the highest levelreq among the magic prefix/suffix/automagic rows, the
// rare/crafted affixes, the SetItems lvlreq (+0x32) or the UniqueItems lvlreq
// (+0x36); crafted items add 10 plus 3 per affix, capped at the hero max
// level - 1 only when that is lower; then the base levelreq (+0x13f) is
// applied as a floor, socketed items count (their own level), granted skills
// (stat 0x6b: the skill's reqlevel; stat 0x61: reqlevel + 6) and the
// item_levelreq stat (0x5c) is added. The result is never below 0. The engine
// does not feed affix levels into Item.ReqLevel yet (EquipItemOf uses the base only).
func RequiredLevel(base int, affixes ...int) int {
	for _, a := range affixes {
		if a > base {
			base = a
		}
	}

	return base
}

// RequiredLevelCrafted is the affix part of the level of a crafted item
// (VERIFIED, 0x62b720): the highest affix levelreq plus 10 plus 3 for each
// affix, but at most maxLevel-1 (the hero's maximum level minus one).
func RequiredLevelCrafted(maxLevel int, affixes ...int) int {
	hi := 0

	for _, a := range affixes {
		if a > hi {
			hi = a
		}
	}

	lvl := hi + 10 + 3*len(affixes)
	if lvl > maxLevel-1 {
		lvl = maxLevel - 1
	}

	return lvl
}

// RequiredLevelTotal is the final level of an item (VERIFIED, 0x62b720): the
// quality part (affix, unique or set level, or RequiredLevelCrafted), floored
// by the base levelreq, raised by the levels of socketed items and of the
// skills it grants (pass stat 0x61 skills already +6), plus the item_levelreq
// stat (0x5c); never below 0.
func RequiredLevelTotal(quality, base int, socketed, skills []int, levelReqStat int) int {
	lvl := RequiredLevel(base, quality)
	lvl = RequiredLevel(lvl, socketed...)
	lvl = RequiredLevel(lvl, skills...)

	lvl += levelReqStat
	if lvl < 1 {
		return 0
	}

	return lvl
}

// Check evaluates the requirements of an item for a hero. A requirement of a
// stat only counts when the hero's stat is at least 1 (the game fails the
// check for stat < 1). types may be nil: then the class check is skipped.
func Check(h Hero, it *Item, types *Types) Requirement {
	r := Requirement{
		HaveStr: h.Str, HaveDex: h.Dex, HaveLevel: h.Level,
		NeedStr:   RequiredStat(it.ReqStr, it.ReqPercent, it.Ethereal),
		NeedDex:   RequiredStat(it.ReqDex, it.ReqPercent, it.Ethereal),
		NeedLevel: it.ReqLevel,
		ClassOK:   true,
		IdentOK:   it.Identified,
	}

	r.StrOK = h.Str >= 1 && h.Str >= r.NeedStr
	r.DexOK = h.Dex >= 1 && h.Dex >= r.NeedDex
	r.LevelOK = it.ReqLevel <= 0 || h.Level >= it.ReqLevel

	if types != nil {
		if c := types.ClassOf(it.Type); c != "" && h.Class != "" && c != h.Class {
			r.ClassOK = false
		}
	}

	return r
}

// ActiveState says which equipped items are switched on.
type ActiveState map[Loc]bool

// HeroFunc returns the hero as it is when exactly the items on in the state
// are active (their strength and dexterity bonuses count).
type HeroFunc func(active func(Loc) bool) Hero

// Resolve is the activation pass of the game (0x55b9c0, VERIFIED shape): first
// every active item that fails its requirements is switched off, then the
// switched off items that pass again are switched on, repeating until nothing
// changes (an item whose bonus lifts another over its requirement turns that
// one on). prev is the state before (nil: everything on). Broken items are
// never switched on. Weapon switch locations (11, 12) are always off.
func Resolve(prev ActiveState, body map[Loc]*Item, broken func(Loc) bool, hero HeroFunc, types *Types) ActiveState {
	cur := ActiveState{}

	for loc, it := range body {
		if it == nil || loc.IsSwap() {
			continue
		}

		on, known := prev[loc]
		cur[loc] = !known || on
	}

	is := func(loc Loc) bool { return cur[loc] }

	// pass 1: switch off what fails
	for _, loc := range ordered(body) {
		it := body[loc]
		if it != nil && cur[loc] && !Check(hero(is), it, types).OK() {
			cur[loc] = false
		}
	}

	// pass 2: switch on what passes again, until stable
	const maxPasses = NumLocs + 1

	for pass := 0; pass < maxPasses; pass++ {
		changed := false

		for _, loc := range ordered(body) {
			it := body[loc]
			if it == nil || loc.IsSwap() || cur[loc] || (broken != nil && broken(loc)) {
				continue
			}

			if Check(hero(is), it, types).OK() {
				cur[loc] = true
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	for loc := range cur {
		if broken != nil && broken(loc) {
			cur[loc] = false
		}
	}

	return cur
}

// ordered returns the occupied locations in slot order (1..12), as the game walks them.
func ordered(body map[Loc]*Item) []Loc {
	var out []Loc

	for l := Loc(1); l < NumLocs; l++ {
		if body[l] != nil {
			out = append(out, l)
		}
	}

	return out
}
