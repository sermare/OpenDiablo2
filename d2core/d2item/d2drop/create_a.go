package d2drop

// Slice A: base stats, quality application glue, low/superior quality,
// ethereal, sockets. (ITEMGEN_CreateItemFromRequest 556e00 minus the unit
// creation, InitItemBaseStats 555b10, ApplyQualityToItem 555520, RollSockets
// 554c60, RollEthereal 554d90, ApplySuperiorQuality 5c0710, ApplyLowQuality
// 5c0b00/5c0890, RollExistingItemQuality 555050.)

// QualityTables holds QualityItems.txt and LowQualityItems.txt.
type QualityTables struct{}

// createItem runs the creation after the units exist: base stats, then the
// quality application. NOT IMPLEMENTED YET.
func (c *Creator) createItem(st *itemState) error {
	return ErrNotCreated
}
