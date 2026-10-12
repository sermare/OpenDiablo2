package d2player

// The messages around hiring (w44, Game.exe 1.14b). All ids were resolved against string.tbl; the keys are the string.tbl keys.
//
// Click on an entry of the hire list (the callback of the list box at 0x4afe80, which has no function in the database): it plays the
// click sound (id 2), stores the slot of the entry (0x7b8ce8) and the NPC id (0x7b8cec); when the hero already owns a mercenary (the
// pet list has an entry of type 7, CLIENT_PetList_FindUnitIdByTypeForOwner 0x474c60) UI_CloseNpcMessageDialog 0x4afe30 closes the
// list and opens the confirm dialog of mode 5 (UI_CreateNpcItemConfirmDialog 0x4af750), titled string 0xd18 with the two answers
// Yes (0xd10) and No (0xd11); without a mercenary UI_CancelNpcDialogChoice 0x4ae620 sends the hire request (a packet with the offer
// id of the slot) at once and shows "Waiting for confirmation of transaction..." (0xd19) until the reply arrives. UNVERIFIED: the
// first tests of the callback (it also reads a panel mask and a word at 0x7245a8) were decoded by hand from the bytes.
const (
	hireKeyReplace = "VerifyTransaction9"  // 0xd18 "This Mercenary will replace your current one."
	hireKeyYes     = "VerifyTransaction1"  // 0xd10
	hireKeyNo      = "VerifyTransaction2"  // 0xd11
	hireKeyWaiting = "TransactionResults1" // 0xd19
	hireKeyCancel  = "Back"                // 0xd48 "cancel": the last entry of the list dialog
	hireKeyOK      = "Continue"            // 0xd4b: the button under an error message
)

// Exported names of the keys for the game screen.
const (
	HireKeyReplace = hireKeyReplace
	HireKeyYes     = hireKeyYes
	HireKeyNo      = hireKeyNo
	HireKeyWaiting = hireKeyWaiting
	HireKeyCancel  = hireKeyCancel
	HireKeyOK      = hireKeyOK
)

// HireClickAction is what a click on an entry of the hire list does.
type HireClickAction int

// The two outcomes of a click.
const (
	HireSendRequest    HireClickAction = iota // the hero has no mercenary: ask the server at once
	HireConfirmReplace                        // ask "This Mercenary will replace your current one." first
)

// HireClick decides what a click on an entry does (the pet list test of 0x4afe80).
func HireClick(hasMerc bool) HireClickAction {
	if hasMerc {
		return HireConfirmReplace
	}

	return HireSendRequest
}

// The reply codes of the transaction packet (UI_ApplyNpcTransactionResultA/B 0x4b1c50, 0x4b1ea0; the code is the byte at +2 of the
// reply, stored at 0x7b8e11). 5 is the success of a hire (the first time the hireling popup of the registry flag PopupHireling
// opens), 3 and 6 reopen the NPC menu quietly; every other code shows a message with a Continue button (0xd4b).
const (
	HireReplyOK        = 5
	HireReplyNoGold    = 12 // 0xc
	HireReplyMaxMercs  = 16 // 0x10
	HireReplyCannotNow = 15 // 0xf
	HireReplyNoRoom    = 10 // 0xa
	HireReplyNoItem    = 11 // 0xb
	hireReplySold1     = 7
	hireReplySold2     = 8
)

// HireReplyKey is the string.tbl key of the message of a reply code ("" for the codes that show none).
func HireReplyKey(code int) string {
	switch code {
	case 0, 1, 3, 5, 6:
		return ""
	case hireReplySold1, hireReplySold2:
		return "TransactionResults6" // 0xd1e "That item has just been traded."
	case HireReplyNoRoom:
		return "TransactionResults4" // 0xd1c
	case HireReplyNoItem:
		return "TransactionResults8" // 0xd20
	case HireReplyNoGold:
		return "TransactionResults10" // 0xd22 "You do not have enough gold for that."
	case HireReplyCannotNow:
		return "TransactionResults11" // 0xd23 "She cannot come right now.  Try again later."
	case HireReplyMaxMercs:
		return "TransactionResults5" // 0xd1d "You already have the maximum number of Mercenaries."
	}

	return "TransactionResults9" // 0xd21 "I cannot complete that request."
}

// The layout of the list dialog (UI_OpenMercHireDialog 0x4b24c0): a dialog 0x1ea x 0x15e, a list box 0x1ea x 0x118 with six rows
// (CTRL_ListBoxWrapperConstruct 0x4bc1d0) at x = (screen width - 0x1ea) / 2 and y = screen height / 2 - 0xa0, added to the dialog
// at offset (0x23, 0x1e); the title is the dialog's own non-selectable line (menu entry 0x15), the Cancel entry a menu entry at
// 0x13b. UNVERIFIED: the dialog's own top left (taken as the list's minus the 0x1e offset) and the row height (the list height
// divided by its six rows).
const (
	HireListW       = 0x1ea
	HireListDialogH = 0x15e
	HireListBoxH    = 0x118
	HireListRows    = 6
	HireListOffsetY = 0x1e
	HireListCancelY = 0x13b
	hireListRowH    = HireListBoxH / HireListRows
)

// HireListRect is the list box (x, y, w, h) on a screen of the given size.
func HireListRect(screenW, screenH int) (x, y, w, h int) {
	return (screenW - HireListW) / 2, screenH/2 - 0xa0, HireListW, HireListBoxH
}

// HireListDialogRect is the dialog around the list.
func HireListDialogRect(screenW, screenH int) (x, y, w, h int) {
	lx, ly, _, _ := HireListRect(screenW, screenH)

	return lx, ly - HireListOffsetY, HireListW, HireListDialogH
}

// HireListRowAt gives the entry under a point of a list scrolled to entry first (n entries), -1 for none.
func HireListRowAt(screenW, screenH, first, n, px, py int) int {
	x, y, w, h := HireListRect(screenW, screenH)
	if px < x || px >= x+w || py < y || py >= y+h {
		return -1
	}

	i := first + (py-y)/hireListRowH
	if i < 0 || i >= n || i >= first+HireListRows {
		return -1
	}

	return i
}

// HireListClampFirst keeps the first visible entry in range.
func HireListClampFirst(first, n int) int {
	if m := n - HireListRows; first > m {
		first = m
	}

	if first < 0 {
		first = 0
	}

	return first
}
