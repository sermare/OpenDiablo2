package d2inventory

// Gold operations of the server, VERIFIED in Game.exe 1.14b (w48). The functions are pure: they take the purses and
// return the new ones, the caller applies them.
//
//	drop     SERVER_HandleC2S50_DropPlayerGold 0x54a620 -> SERVER_DropPlayerGold 0x532f20 (client packet 0x50)
//	deposit  SERVER_HandleStashButtonAction 0x562bb0, action 0x14 (client dialog kind 3, the inventory coin button)
//	withdraw SERVER_HandleStashButtonAction 0x562bb0, action 0x13 (client dialog kind 4, the stash coin button)
//	pickup   ITEMMODE_PickupGoldFromGround 0x55a5f0 / PLAYER_AddGoldClampedOrDropOverflow 0x558e40

// MaxGoldPile is the largest pile a drop makes in one go (SERVER_DropPlayerGold clamps the amount to 2000000000).
const MaxGoldPile = 2000000000

// GoldDialogDigits is the number of digits the edit box of the gold dialog takes (UI_OpenPanelByIndex 0x44f7b0).
const GoldDialogDigits = 10

// DropGold is the drop request: the amount must be 0..carried and not above the carry limit of the level, else the
// server refuses (returns 0 and nothing changes). 0 is accepted and drops nothing. dropped is what lands on the
// ground; the carried gold goes down by the gold of the piles that were made (the same amount).
func DropGold(carried, amount, maxCarry int) (newCarried, dropped int, ok bool) {
	if amount < 0 || amount > carried || amount > maxCarry {
		return carried, 0, false
	}

	if amount > MaxGoldPile {
		amount = MaxGoldPile
	}

	return carried - amount, amount, true
}

// DepositGold moves carried gold into the stash (action 0x14): the amount must be 1..carried; when the whole amount
// fits under the stash limit it moves, otherwise the stash is filled up to the limit and the rest stays in the
// purse; with a full stash nothing moves (ok false).
func DepositGold(carried, stash, amount, stashLimit int) (newCarried, newStash int, ok bool) {
	if amount <= 0 || amount > carried {
		return carried, stash, false
	}

	if stash+amount <= stashLimit {
		return carried - amount, stash + amount, true
	}

	if stash < stashLimit {
		room := stashLimit - stash

		return carried - room, stashLimit, true
	}

	return carried, stash, false
}

// WithdrawGold moves stash gold into the purse (action 0x13): the amount must be 1..stash and the purse must stay
// within its carry limit, else the server only answers with the error sound (ok false, nothing changes). There is no
// partial withdraw.
func WithdrawGold(carried, stash, amount, maxCarry int) (newCarried, newStash int, ok bool) {
	if amount <= 0 || amount > stash || carried+amount > maxCarry {
		return carried, stash, false
	}

	return carried + amount, stash - amount, true
}

// GoldPickup is the pickup of a pile on the ground by one player outside a party (0x55a5f0): the purse takes what
// fits under maxCarry, the rest becomes a new pile at the player's feet. A full purse takes nothing.
func GoldPickup(carried, pile, maxCarry int) (taken, left int) {
	_, left = AddGold(carried, pile, maxCarry)

	return pile - left, left
}

// GoldDialogKind is the kind of the gold dialog (argument of UI_OpenPanelByIndex 0x44f7b0).
type GoldDialogKind int

// Kinds of the gold dialog.
const (
	GoldDialogDrop     GoldDialogKind = iota // kinds 0 and 1: client packet 0x50
	GoldDialogTrade                          // kind 2: the gold offer of a player trade
	GoldDialogDeposit                        // kind 3: inventory coin button, packet 0x4f action 0x14
	GoldDialogWithdraw                       // kind 4: stash coin button, packet 0x4f action 0x13
)

// GoldDialogLimit is the largest amount the dialog accepts: the edit box is made with the limit of the purse it takes
// from (stat 0xe, the carried gold, for every kind except the withdraw, which uses stat 0xf, the stash gold).
func GoldDialogLimit(kind GoldDialogKind, carried, stash int) int {
	if kind == GoldDialogWithdraw {
		return stash
	}

	return carried
}

// GoldDialogPrefill tells whether the dialog opens with the whole limit typed in (kinds 3, 6 and 7 call the edit box
// setter with the limit; the others open with an empty box: U for the withdraw, whose constructor argument may
// prefill it too).
func GoldDialogPrefill(kind GoldDialogKind) bool {
	return kind == GoldDialogDeposit
}

// ClampGoldInput applies the edit box: at most GoldDialogDigits digits and never above limit. The spinner arrows
// change the value by one.
func ClampGoldInput(digits string, limit int) int {
	v := 0

	n := 0

	for _, c := range digits {
		if c < '0' || c > '9' {
			continue
		}

		if n == GoldDialogDigits {
			break
		}

		v = v*10 + int(c-'0')
		n++
	}

	if v > limit {
		v = limit
	}

	if v < 0 {
		v = 0
	}

	return v
}

// GoldDialogRequest decides what pressing OK does for a typed amount (the OK callback 0x44f6e0): 0 only closes the
// dialog (the trade dialog sends its offer anyway), anything else goes to the server as the kind's packet.
func GoldDialogRequest(kind GoldDialogKind, amount int) (send bool) {
	if amount == 0 {
		return kind == GoldDialogTrade
	}

	return true
}
