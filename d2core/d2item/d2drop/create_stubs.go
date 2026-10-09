package d2drop

// Stubs for the slices implemented elsewhere (B: affixes, C: unique / set
// picks and properties). The lead deletes this file when merging the slices.
// Every roll reports whether it was applied; slice A handles the fallbacks.

// pickUnique is ITEMGEN_PickUniqueItem (5547d0).
func (c *Creator) pickUnique(st *itemState) bool { return false }

// pickSet is the set item pick (5c06e0).
func (c *Creator) pickSet(st *itemState) bool { return false }
