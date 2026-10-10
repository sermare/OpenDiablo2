package d2level

// PortalShrineOffset is the subtile offset (on both axes) from the nearest
// free cell next to the hero where the Portal shrine opens its portal
// (VERIFIED, 0x580950: the cell search around the hero uses radius 3 and
// collision mask 0x1c09, and the portal is placed at that cell plus 5, 5).
const PortalShrineOffset = 5

// PortalShrineDest is the level at the far end of the portal the Portal
// shrine opens. OBJECT_OperateTeleportToNearActStart (0x580950, VERIFIED by
// reading the body) is not a teleport of the hero: it asks
// DRLG_GetActStartLevel (0x61a8b0) for the start level (town) of the act of
// the hero's room and calls SERVER_PlaceUnitAtNearestFreeCell (0x56ae80),
// the same routine the Town Portal skill uses (PLAYER_TeleportToTownOnTownPortalUse
// 0x5bbe10); it makes a linked portal pair, one next to the hero and one at
// the town's arrival point, without recording an owner (the output pointer is
// null) and without destroying the hero's own portals. 0 for a level that
// belongs to no act.
func PortalShrineDest(level int) int {
	return ActStartLevel(ActOfLevel(level))
}

// PortalShrineOpens reports whether the shrine opens a portal in level: the
// exe does nothing in a town room (ROOM_IsTownRoom test inside 0x56ae80).
func PortalShrineOpens(level int) bool {
	return PortalShrineDest(level) != 0 && !IsTown(level)
}
