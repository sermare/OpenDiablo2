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
// ethereal item (INV_CheckItemRequirements 0x62ebf0, VERIFIED structure; that
// the percent is applied as base*percent/100 with integer division is
// UNVERIFIED, the helper's arguments are not visible in the decompiler output).
func RequiredStat(base, percent int, ethereal bool) int {
	need := base + base*percent/100
	if ethereal {
		need -= 10
	}

	return need
}

// RequiredLevel is the level an item needs: the highest of the base item's
// levelreq and the levelreq of each affix / unique / set row on it. The
// "highest of" shape is community knowledge; the game's code path
// (INV_CheckItemRequirements 0x62ebf0 calling FUN_0062b720) is UNVERIFIED, the
// engine does not feed affix levels into Item.ReqLevel yet (EquipItemOf uses
// the base only).
func RequiredLevel(base int, affixes ...int) int {
	for _, a := range affixes {
		if a > base {
			base = a
		}
	}

	return base
}

// Check evaluates the requirements of an item for a hero. A requirement of a
// stat only counts when the hero's stat is at least 1 (the game fails the
// check for stat < 1). types may be nil: then the class check is skipped.
func Check(h Hero, it *Item, types *Types) Requirement {
	r := Requirement{
		HaveStr: h.Str, HaveDex: h.Dex, HaveLevel: h.Level,
		NeedStr: RequiredStat(it.ReqStr, it.ReqPercent, it.Ethereal),
		// UNVERIFIED: the decompiler shows the same percent shift for both stats
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
