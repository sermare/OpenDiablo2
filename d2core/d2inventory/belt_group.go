package d2inventory

// BeltGroup is the belt column group of a misc.txt item code, INV_AreItemsBeltGroupCompatible 0x628c00
// (VERIFIED, tables at 0x740ef0): two items share a column when their base item is the same or both codes lie
// in one of three lists: {rvl, rvs} (full rejuvenation and rejuvenation), {mp1..mp5} and {hp1..hp5}. Every
// other beltable item (stamina vps, antidote yps, thawing wms, the old hpo/mpo codes) only matches itself.
func BeltGroup(code string) string {
	switch code {
	case "rvl", "rvs":
		return "rv"
	case "hp1", "hp2", "hp3", "hp4", "hp5":
		return "hp"
	case "mp1", "mp2", "mp3", "mp4", "mp5":
		return "mp"
	}

	return code
}
