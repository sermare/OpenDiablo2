package d2hero

import "sort"

// CycleSkill returns the next (dir > 0) or previous (dir < 0) skill the hero
// can use on a button, in skill id order, wrapping around; it is what the
// gamepad bumpers step through. ok is false when no skill is selectable.
// When current is not selectable any more the first (or last) one is returned.
func CycleSkill(skills map[int]*HeroSkill, current int, left bool, dir int) (id int, ok bool) {
	var ids []int

	for sid, s := range skills {
		if Selectable(s, left) {
			ids = append(ids, sid)
		}
	}

	if len(ids) == 0 {
		return 0, false
	}

	sort.Ints(ids)

	pos := sort.SearchInts(ids, current)
	found := pos < len(ids) && ids[pos] == current

	switch {
	case dir >= 0 && found:
		pos = (pos + 1) % len(ids)
	case dir >= 0:
		pos %= len(ids) // the first id above current (or wrap to the first)
	case found:
		pos = (pos - 1 + len(ids)) % len(ids)
	default:
		pos = (pos - 1 + len(ids)) % len(ids) // the last id below current
	}

	return ids[pos], true
}
