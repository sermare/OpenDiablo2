package d2player

// MiniPanelSlot is where the mini panel strip stands for a given set of open panels.
type MiniPanelSlot int

const (
	// MiniPanelDefault is the centred strip (no panel group open).
	MiniPanelDefault MiniPanelSlot = iota
	// MiniPanelShiftRight is the strip pushed right (a left-side panel is open).
	MiniPanelShiftRight
	// MiniPanelShiftLeft is the strip pushed left (a right-side panel is open).
	MiniPanelShiftLeft
	// MiniPanelHidden is the strip not drawn at all (both groups open).
	MiniPanelHidden
)

// MiniPanelPlacement ports UI_DrawMiniPanel (0x47b510): the first predicate (0x47a860, "left group") moves the
// strip to the right, the second (0x47a950, "right group") to the left, and both true hides it.
//
// The exe groups panels by panel id, and the ids 0xc and 0x17-0x1c (stash, cube, trade windows) are members of
// BOTH groups, together with 3 (skill select). Only 1 (inventory) and 4 (skill tree) are right-group only; 2
// (character), 0xf (quest log), 0x10, 0x14, 0x16 (party) and 0x24 (mercenary) are left-group only. So opening the
// stash, the cube or a trade window hides the strip whether or not the inventory is open next to it.
// (Which id is which, VERIFIED w44 from the open / close handlers UI_OnPanelOpened 0x450de0 and UI_OnPanelClosed 0x4511a0:
// 0xc the NPC shop, 0x17 the player to player trade, 0x18 the NPC menu dialog, 0x19 the stash (bank sprites), 0x1a the
// cube (transmogrifier sprite); 0x1b and 0x1c have no handler action. The skill select panel is not wired.)
func MiniPanelPlacement(leftGroup, rightGroup bool) MiniPanelSlot {
	switch {
	case leftGroup && rightGroup:
		return MiniPanelHidden
	case leftGroup:
		return MiniPanelShiftRight
	case rightGroup:
		return MiniPanelShiftLeft
	}

	return MiniPanelDefault
}

// MiniPanelStripX is the strip's left edge as an offset from the screen's horizontal centre (W/2) for a slot.
func MiniPanelStripX(slot MiniPanelSlot, singlePlayer bool) int {
	switch slot {
	case MiniPanelShiftRight:
		if singlePlayer {
			return 0x38 - 2
		}

		return 0x23 - 2
	case MiniPanelShiftLeft:
		return -0xcd
	}

	if singlePlayer {
		return -0x4a - 3
	}

	return -0x54 - 3
}

// panelGroups returns the exe's two mini panel predicates for the current panels.
func (g *GameControls) panelGroups() (leftGroup, rightGroup bool) {
	both := g.stash.IsOpen() || g.cube.IsOpen() || g.Trade.IsOpen()

	return g.isLeftPanelOpen(), g.isRightPanelOpen() || both
}
