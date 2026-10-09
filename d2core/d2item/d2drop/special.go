package d2drop

// Row is a UniqueItems or SetItems row, reduced to what picking needs.
type Row struct {
	ID      string
	Base    string // base item code
	Rarity  int
	Level   int // lvl: minimum item level
	Version int
	Enabled bool
	Ladder  bool // ladder-only
	NoLimit bool // may spawn more than once per game
}

// PickRow is ITEMGEN_PickUniqueItem / PickSetItem for LoD items: a pick
// weighted by rarity (at least 1) among the enabled rows of the given base
// item whose lvl is not above the item level, that fit the version, are
// allowed in this kind of game, and that have not already spawned unless
// they are nolimit. It returns nil if nothing qualifies, in which case the
// caller falls back with FallbackQuality.
//
// The ladder-only flag position for set items is UNVERIFIED in the notes.
func PickRow(rng RNG, rows []Row, base string, ilvl, version int, ladder bool, spawned map[string]bool) *Row {
	var cands []*Row

	sum := 0

	for i := range rows {
		r := &rows[i]

		switch {
		case !r.Enabled, r.Base != base, r.Level > ilvl, r.Version > version:
			continue
		case r.Ladder && !ladder:
			continue
		case spawned[r.ID] && !r.NoLimit:
			continue
		}

		cands = append(cands, r)
		sum += maxInt(1, r.Rarity)
	}

	if len(cands) == 0 {
		return nil
	}

	roll := int(rng.Roll(int32(sum)))

	for _, r := range cands {
		w := maxInt(1, r.Rarity)
		if roll < w {
			return r
		}

		roll -= w
	}

	return cands[len(cands)-1]
}
