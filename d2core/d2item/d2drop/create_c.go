package d2drop

// Slice C: unique and set items (PickUniqueItem 5547d0, PickSetItem 5c06e0 /
// 5c0350 / 5c04f0, the one-per-game bitmask) and the property engine
// (props.go for Lord of Destruction items, props_classic.go for classic
// ones). Everything here is proven equal to Game.exe 1.14b running under the
// emulator, see testdata/create_c.json (gen_create_c.py in the oracle).
//
// The tables (UniqueTables, PropTables) are filled by the caller; the tests
// read them from D2_TABLES (create_c_load_test.go: rows named "Expansion"
// are dropped, set items after that row have version 100).
//
// What the engine needs from the other slices: the item state with the unit
// stats written so far (st.writes), st.quality set before pickUnique, and
// Request.Game when the one-per-game limit of unique items matters.

// UniqueItem is a row of UniqueItems.txt reduced to what creation reads.
type UniqueItem struct {
	Name    string
	Version int // 0 classic, 1 and 100 Lord of Destruction
	Enabled bool
	Ladder  bool // ladder-only
	NoLimit bool // may be created again and again in one game
	Rarity  int
	Lvl     int // minimum item level
	LvlReq  int
	Code    string // base item code
	Props   [12]PropInst
}

// SetItem is a row of SetItems.txt.
type SetItem struct {
	Name    string
	Set     int // row of Sets.txt
	Version int
	Lvl     int
	LvlReq  int
	Rarity  int
	Code    string
	// AddFunc is the "add func" column: 0 puts the set bonus properties in
	// the item's base list, other values give each tier its own list.
	AddFunc int
	Props   [9]PropInst
	// AProps are aprop1a, aprop1b, aprop2a ... aprop5b.
	AProps [10]PropInst
}

// SetDef is a row of Sets.txt.
type SetDef struct {
	Name    string
	Version int
	// Items are the SetItems rows belonging to the set, in table order.
	Items []int
	// Partial are PCode2a, PCode2b, PCode3a ... PCode5b (the bonus from 2, 3,
	// 4 and 5 pieces, two properties each); Full are FCode1..8.
	Partial [8]PropInst
	Full    [8]PropInst
}

// cowKingSet is the index of "Cow King's Leathers" in Sets.txt: its items
// only drop when the request has flag 1 (the secret cow level).
const cowKingSet = 0x1d

// UniqueTables holds UniqueItems.txt, SetItems.txt and Sets.txt. Rows with
// the name "Expansion" (separators) are not part of the tables, so row
// indexes are those of the game.
type UniqueTables struct {
	Uniques  []UniqueItem
	SetItems []SetItem
	Sets     []SetDef
}

// GameState is the part of the game object item creation reads and writes.
type GameState struct {
	// Ladder is set for ladder games (ladder-only uniques may drop).
	Ladder bool
	// Spawned is the bitmask at game+0x1b24: one bit per UniqueItems row
	// that has been created in this game (4096 bits).
	Spawned [128]uint32
}

// spawned is the test the game does with the bit of a unique row (5545d0); a
// row beyond the mask counts as spawned.
func (g *GameState) spawned(row int) bool {
	if g == nil {
		return false
	}

	if row > 0x1000 {
		return true
	}

	return g.Spawned[row>>5]&(1<<uint(row&0x1f)) != 0
}

func (g *GameState) markSpawned(row int) {
	g.Spawned[row>>5] |= 1 << uint(row&0x1f)
}

// finishPick completes a unique or set item: its uid is set, it is
// unidentified and its properties are applied (kind 3 / 4).
func (c *Creator) finishPick(st *itemState, kind, row int) {
	st.uniqueRow = row
	st.flags &^= flagIdentified
	c.applyProps(st, kind, nil)
}

// hasDurability is ITEM_HasDurability (629b00).
func (c *Creator) hasDurability(st *itemState) bool {
	return (&propCtx{c: c, st: st}).hasDurability()
}

// pickUnique is ITEMGEN_PickUniqueItem (5547d0): choose a UniqueItems row for
// the item, set it (st.uniqueRow), unidentify the item and apply the
// properties. It reports false when no unique could be created (the caller
// falls back to a rare item); st.uniqueRow is then -1.
//
// Classic items (Request.Version 0) take the first matching row that has not
// been created yet; Lord of Destruction items choose among the matching rows
// weighted by rarity, and fail when the chosen row was created already.
// A classic item with a durability gets five times its durability first.
func (c *Creator) pickUnique(st *itemState) bool {
	if c.Uniques == nil {
		return false
	}

	b := st.base
	ver := st.req.Version
	game := st.req.Game

	if c.hasDurability(st) && ver == 0 {
		st.scaleDurability(5)
	}

	t := c.Uniques
	ladder := game != nil && game.Ladder

	if ver == 0 {
		for row := range t.Uniques {
			u := &t.Uniques[row]

			if u.Version >= 100 || !u.Enabled || (!ladder && u.Ladder) || u.Code != b.Code {
				continue
			}

			if game.spawned(row) && b.Quest == 0 {
				continue
			}

			c.finishPick(st, PropKindUnique, row)

			return true
		}

		st.uniqueRow = -1

		return false
	}

	type cand struct{ row, cum int }

	var cands []cand

	total, forced := 0, -1

	for row := range t.Uniques {
		u := &t.Uniques[row]

		if (u.Version >= 100 && ver < 100) || !u.Enabled || u.Code != b.Code || (!ladder && u.Ladder) || u.Lvl > st.ilvl {
			continue
		}

		if st.req.ForcedID > 0 && st.req.ForcedID-1 == row {
			forced = row
		}

		cands = append(cands, cand{row, total})
		total += maxInt(u.Rarity, 1)
	}

	if len(cands) == 0 {
		return c.pickUniqueNone(st)
	}

	row := forced
	if row < 0 {
		roll := int(st.item.Roll(int32(total)))
		i := 1

		for i < len(cands) && roll >= cands[i].cum {
			i++
		}

		row = cands[i-1].row
	}

	if b.Quest == 0 && game != nil && game.spawned(row) {
		st.uniqueRow = -1

		return false
	}

	st.uniqueRow = row

	if !c.markUnique(st) {
		st.uniqueRow = -1

		return false
	}

	c.finishPick(st, PropKindUnique, row)

	return true
}

// pickUniqueNone is the end of the Lord of Destruction PickUniqueItem when no
// row matched: a base item that is always unique "succeeds" without a row
// (the game leaves it alone), anything else fails. Classic items just fail.
func (c *Creator) pickUniqueNone(st *itemState) bool {
	if st.base.Unique {
		return true
	}

	st.uniqueRow = -1

	return false
}

// markUnique is the check-and-mark function of the one-per-game bitmask
// (554660): rows with "nolimit" are never marked; otherwise the bit is set,
// unless it was set already.
func (c *Creator) markUnique(st *itemState) bool {
	row := st.uniqueRow
	game := st.req.Game

	if row < 0 || row >= len(c.Uniques.Uniques) {
		return false
	}

	if c.Uniques.Uniques[row].NoLimit {
		return true
	}

	if st.base.Quest == 0 && st.quality == QualityUnique && game.spawned(row) {
		return false
	}

	if row > 0x1000 {
		return false
	}

	if game != nil {
		game.markSpawned(row)
	}

	return true
}

// pickSet is ITEMGEN_PickSetItem (5c06e0): choose a SetItems row for the
// item, apply it and its properties. Items of Lord of Destruction version use
// the weighted pick of 5c0350, classic ones 5c04f0. It reports false when no
// set item exists for the base item.
func (c *Creator) pickSet(st *itemState) bool {
	if c.Uniques == nil {
		return false
	}

	if st.req.Version >= 1 {
		return c.pickSetLod(st)
	}

	return c.pickSetClassic(st)
}

func (c *Creator) pickSetLod(st *itemState) bool {
	b, ver := st.base, st.req.Version
	t := c.Uniques

	type cand struct{ row, weight int }

	var cands []cand

	total, forced := 0, -1

	for row := range t.SetItems {
		s := &t.SetItems[row]

		if (s.Version >= 100 && ver < 100) || s.Lvl > st.ilvl || s.Code != b.Code {
			continue
		}

		if s.Set == cowKingSet && st.req.Flags&1 == 0 {
			continue
		}

		if st.req.ForcedID > 0 && st.req.ForcedID-1 == row {
			forced = row
		}

		w := s.Rarity
		if w == 0 {
			w = 1
		}

		cands = append(cands, cand{row, w})
		total += w
	}

	row := forced

	if row < 0 {
		if total == 0 {
			return false
		}

		roll := int(st.item.Roll(int32(total)))

		i := 0

		for ; i < len(cands); i++ {
			if roll < cands[i].weight {
				break
			}

			roll -= cands[i].weight
		}

		if i >= len(cands) {
			return false
		}

		row = cands[i].row
	}

	c.finishPick(st, PropKindSetItem, row)

	return true
}

// pickSetClassic is PickSetItem for classic items (5c04f0): double the
// durability, start at a random classic set and look through all classic
// sets for one that has an item with the base item's code (the last one found
// wins). On failure the game restores the low word of the item generator
// from a saved copy; the high word it writes comes from a caller register (an
// address) and cannot be reproduced: here the saved high word is kept.
func (c *Creator) pickSetClassic(st *itemState) bool {
	b := st.base
	t := c.Uniques
	saved := st.item

	st.scaleDurability(2)

	n := 0
	for n < len(t.Sets) && t.Sets[n].Version < 100 {
		n++
	}

	start := 0
	if n > 0 {
		start = int(st.item.Roll(int32(n)))
	}

	found := -1

	for k := 0; k < n; k++ {
		set := &t.Sets[(k+start)%n]

		for _, it := range set.Items {
			if t.SetItems[it].Code == b.Code {
				found = it

				break
			}
		}
	}

	if found < 0 {
		st.item = saved

		return false
	}

	c.finishPick(st, PropKindSetItem, found)

	return true
}

// makeEthereal is ITEM_MakeEthereal (660a40): set the ethereal flag and raise
// the damage (weapons) or defense (armor) stats by 50%.
func (c *Creator) makeEthereal(st *itemState) {
	pc := &propCtx{c: c, st: st}

	st.flags |= flagEthereal

	half := func(stat int) {
		v, _ := st.unitStat(stat)
		v *= 3
		v = (v - (v >> 31)) >> 1

		st.setStat(stat, v)
	}

	if pc.isWeapon() {
		for _, stat := range []int{statMinDam, statMaxDam, statSecMinDam, statSecMaxDam, statThrowMin, statThrowMax} {
			half(stat)
		}

		return
	}

	half(statDefense)
}
