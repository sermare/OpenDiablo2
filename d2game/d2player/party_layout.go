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

// Row geometry (second pass, UI_RebuildPartyList 0x496ac0 + UI_InitPartyRowWidgets 0x496540 +
// UI_DrawPartyScreenMemberRows 0x498a40 + UI_DrawPartyScreenButtonTooltips 0x4977f0, all verified).
// The row y that the decompiler lost is EDI in UI_RebuildPartyList: it starts at 0x5a and grows by 0x26 for every
// listed player (the own roster entry is skipped, at most 8 rows are drawn). All y values are the BOTTOM edge of
// the picture, panel relative (add the panel offset y); the picture is 20 high (hit test: bottom-0x14 .. bottom).
const (
	partyRowFirstY    = 0x5a // bottom of the first row, panel relative
	partyRowPitch     = 0x26
	partyRowHostileDY = 7 // the toggle of relation bit 08 sits 7 below the row, the other widgets on the row
	partyRowBtnH      = 0x14
	partyInviteW      = 0x35 // hover width of the invite/accept/leave button (inclusive range 0..0x35)
	partyToggleW      = 0x14 // hover width of the four toggles (inclusive range 0..0x14)
	partyNameHoverX0  = 0x24 // row name hover x range (inclusive) from the panel left
	partyNameHoverX1  = 0x8c
	partyNameHoverTop = 0x1a // the hover rect spans [rowHover-0x1a, rowHover] where rowHover = rowY + 0xd
	partyNameHoverDY  = 0xd
	partyHeaderX      = 0xe3 // "party membership" header picture x (UI_DrawPartyMembershipHeader 0x496990)
	partyHeaderHoverX = 0x118
	partyHeaderTop    = 0x18 // hover y range, panel relative, inclusive (0x18..0x23)
	partyHeaderBottom = 0x23
)

// PartyRowBottom returns the panel-relative bottom y of listed row i (0 based).
func PartyRowBottom(i int) int { return partyRowFirstY + partyRowPitch*i }

// PartyRowRects returns the screen rectangles of the widgets of row i (top left, 20 high; the invite button is
// partyInviteW wide, the toggles partyToggleW). Names: "invite", "hostile" (relation bit 08, frames 4/6), "bit01"
// (frames 12/14, only offered when the game-info flag 4 is set), "listen" (bit 02, frames 0/2), "mute" (bit 04, frames 8/10)
// and "name_hover".
func (m Mode) PartyRowRects(i int) []UIRect {
	ox, oy := m.PanelOffset()
	rowTop := PartyRowBottom(i) + oy - partyRowBtnH
	inv, hostile, bit01, listen, mute := m.PartyRowX()

	return []UIRect{
		{"party", "invite", inv, rowTop, partyInviteW, partyRowBtnH},
		{"party", "hostile", hostile, rowTop + partyRowHostileDY, partyToggleW, partyRowBtnH},
		{"party", "bit01", bit01, rowTop, partyToggleW, partyRowBtnH},
		{"party", "listen", listen, rowTop, partyToggleW, partyRowBtnH},
		{"party", "mute", mute, rowTop, partyToggleW, partyRowBtnH},
		{"party", "name_hover", ox + partyNameHoverX0, PartyRowBottom(i) + partyNameHoverDY - partyNameHoverTop + oy,
			partyNameHoverX1 - partyNameHoverX0, partyNameHoverTop},
	}
}

// PartyHeaderHover is the hover rectangle of the "party membership" header (shown when the hero has a party).
func (m Mode) PartyHeaderHover() UIRect {
	ox, oy := m.PanelOffset()

	return UIRect{"party", "header_hover", ox + partyHeaderX, oy + partyHeaderTop, partyHeaderHoverX - partyHeaderX, partyHeaderBottom - partyHeaderTop}
}
