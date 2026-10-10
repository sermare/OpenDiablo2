package d2level

// PortalShrineDest is the level the Portal shrine sends its user to:
// OBJECT_OperateTeleportToNearActStart (0x580950, naming notes
// names-verified-logic.md; the Ghidra check of the body was not possible in
// this pass, so the "nearest free cell, mask 0x1c09" placement is UNVERIFIED
// here) asks DRLG_GetActStartLevel (0x61a8b0) for the start level of the act
// of the shrine's room, and places the unit near it. It is the town of the
// current act, not a town portal object next to the hero. 0 for a level that
// belongs to no act.
func PortalShrineDest(level int) int {
	return ActStartLevel(ActOfLevel(level))
}
