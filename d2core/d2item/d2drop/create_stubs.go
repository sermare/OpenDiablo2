package d2drop

// Stubs for the slices implemented elsewhere (B: affixes, C: unique / set
// picks and properties). The lead deletes this file when merging the slices.
// Every roll reports whether it was applied; slice A handles the fallbacks.

// rollMagic is ITEMGEN's magic affix roll (554710).
func (c *Creator) rollMagic(st *itemState) bool { return false }

// rollRare is the rare affix roll (5bff00).
func (c *Creator) rollRare(st *itemState) bool { return false }

// rollCrafted is the crafted affix roll (5bff40).
func (c *Creator) rollCrafted(st *itemState) bool { return false }

// rollTempered is the tempered roll (5bf890 twice); it sets st.rareNames.
func (c *Creator) rollTempered(st *itemState) bool { return false }

// pickUnique is ITEMGEN_PickUniqueItem (5547d0).
func (c *Creator) pickUnique(st *itemState) bool { return false }

// pickSet is the set item pick (5c06e0).
func (c *Creator) pickSet(st *itemState) bool { return false }

// pickAutomagic is the automagic affix pick (5bf5f0); it returns the row id
// (0 = none).
func (c *Creator) pickAutomagic(st *itemState) int { return 0 }
