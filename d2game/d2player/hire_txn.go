package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"

// The hire transaction (w48). In the exe a click on an offer sends the request to the game server (HIRE_ServerHandleHireRequest
// 0x575e10 -> HIRE_ProcessHireOffer 0x574f00, revive: HIRE_ServerHandleReviveMercenary 0x577a10), shows "Waiting for confirmation of
// transaction..." (string 0xd19, a 60 s timer) and acts on the reply packet's code byte. This port keeps the same two halves in
// one process: HireDecide is the server half, HireTxn the waiting state of the client half.
//
// Reply codes sent by the server (VERIFIED in the two handlers above):
//
//	5  hired / revived
//	9  the request does not fit: no such NPC, slot not on offer, slot already hired, no dead mercenary to revive
//	11 the quest gate is closed (Qual-Kehk: quest flag 0x24 of Rescue on Mount Arreat; Kashya below level 8: flag 2)
//	12 not enough gold: TRADE_DeductGold takes the purse first, then the stash gold (stats 0xe and 0xf)
//	15 "She cannot come right now": the monster could not be spawned next to the NPC or the hero
//
// Code 16 ("maximum number of Mercenaries", client message 0xd1d) is NOT produced by either handler; the client string exists, so
// HireDecide gives it when the caller says the hero is at the limit without the replace confirmation (never in the port, which
// always asks for the replace confirmation first). UNVERIFIED who sends it, if anyone.
const (
	// HireTxnLatency is the time the port lets the waiting dialog show before the reply is applied (UNVERIFIED: cosmetic; the
	// real latency is the network).
	HireTxnLatency = 0.25
	// HireTxnTimeout is the timer of the waiting dialog (60 s, VERIFIED at UI_CancelNpcDialogChoice 0x4ae620).
	HireTxnTimeout = 60.0
	// MaxMercenaries is the number of mercenaries a hero owns (one).
	MaxMercenaries = 1
	// HireReplyBadRequest is code 9, HireReplyGateClosed code 11.
	HireReplyBadRequest = 9
	HireReplyGateClosed = HireReplyNoItem
)

// HireKind is what a transaction asks for.
type HireKind int

// The two kinds.
const (
	TxnHire HireKind = iota
	TxnRevive
)

// HireRequest is everything the server half looks at.
type HireRequest struct {
	Kind      HireKind
	GateOpen  bool // the quest gate of the seller (hire only)
	OnOffer   bool // hire: the slot is offered and not hired; revive: the hero has a dead mercenary
	CanSpawn  bool // the merc can be placed (hire only)
	Cost      int
	Gold      int // purse (stat 0xe)
	Stash     int // stash gold (stat 0xf)
	Mercs     int // mercenaries the hero owns now
	Replacing bool
}

// HireDecide gives the reply code of a request (the order of the checks follows HIRE_ProcessHireOffer: gate, request, gold, spawn).
func HireDecide(r HireRequest) int {
	if r.Kind == TxnHire && !r.GateOpen {
		return HireReplyGateClosed
	}

	if !r.OnOffer {
		return HireReplyBadRequest
	}

	if r.Kind == TxnHire && r.Mercs >= MaxMercenaries && !r.Replacing {
		return HireReplyMaxMercs
	}

	if r.Cost < 0 || r.Gold+r.Stash < r.Cost {
		return HireReplyNoGold
	}

	if r.Kind == TxnHire && !r.CanSpawn {
		return HireReplyCannotNow
	}

	return HireReplyOK
}

// HirePay deducts a price the way the server does (TRADE_DeductGold: the purse first, the rest from the stash).
func HirePay(gold, stash, cost int) (newGold, newStash int, ok bool) {
	return d2trade.DeductGold(gold, stash, cost)
}

// HireTxn is the waiting state of one request.
type HireTxn struct {
	Pending bool
	Req     HireRequest
	Seller  int
	Slot    int
	// Hold keeps the reply back (a server that never answers: the timer test).
	Hold    bool
	elapsed float64
}

// Start begins waiting for the reply.
func (t *HireTxn) Start(seller, slot int, req HireRequest) {
	*t = HireTxn{Pending: true, Req: req, Seller: seller, Slot: slot}
}

// Cancel forgets the request.
func (t *HireTxn) Cancel() { *t = HireTxn{} }

// Tick advances the waiting time and says whether the reply is due (or the timer ran out: timedOut).
func (t *HireTxn) Tick(dt float64) (reply, timedOut bool) {
	if !t.Pending {
		return false, false
	}

	t.elapsed += dt

	if t.elapsed >= HireTxnTimeout {
		return false, true
	}

	return !t.Hold && t.elapsed >= HireTxnLatency, false
}
