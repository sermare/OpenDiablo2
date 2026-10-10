package d2player

// Layout of the party screen of the original (Game.exe 1.14b), pure numbers so they can be tested.
// Sources (see d2-re-notes ui-panels-next.md): UI_InitPartyRowWidgets 0x496540 (the x of every
// button of a row, as panel-relative numbers) and UI_DrawPartyScreenCloseButton 0x496dd0.
// The row y is a register argument the decompiler lost (UNVERIFIED), so only the x values
// and the close button are taken from the original here; row y keeps the earlier tuned numbers.

// Panel-relative x of the row buttons in the original table (add the panel offset, 80 at 800x600).
const (
	partyRowInviteX   = 0xbe  // the invite/accept/leave button (widget 0)
	partyRowRelationX = 0x0e  // hostile toggle (roster relation bit 08)
	partyRowBit01X    = 0xf5  // roster relation bit 01 toggle; the Go panel has no widget for it yet
	partyRowListenX   = 0x10a // roster relation bit 02 toggle
	partyRowSeeX      = 0x11f // roster relation bit 04 toggle
)

// Close button of the party screen: 0x116 from the panel left, bottom at 0x1a5 + panel offset y.
const (
	partyCloseDX     = 0x116
	partyCloseBottom = 0x1a5
	partyCloseSize   = 32
)

// PartyRowX returns the screen x of the row buttons for the mode.
func (m Mode) PartyRowX() (invite, relation, bit01, listen, see int) {
	ox := m.LeftPanelX()

	return ox + partyRowInviteX, ox + partyRowRelationX, ox + partyRowBit01X, ox + partyRowListenX, ox + partyRowSeeX
}

// PartyCloseRect returns the close button rectangle. The original draws the 32x32
// picture standing on y = 0x1a5 + offset y.
func (m Mode) PartyCloseRect() UIRect {
	ox, oy := m.PanelOffset()

	return UIRect{"party", "close", ox + partyCloseDX, partyCloseBottom + oy - partyCloseSize, partyCloseSize, partyCloseSize}
}
