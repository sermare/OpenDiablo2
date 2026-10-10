package d2playertrade

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// The inventory of a player is 10 columns by 4 rows (inventory.txt).
const (
	InvCols = 10
	InvRows = 4
)

// AcceptLock is how long the accept of both sides is refused after an offer
// changed (UNVERIFIED length; it stops a player from swapping the offer just
// before the other accepts).
var AcceptLock = 2 * time.Second

// Errors.
var (
	ErrState        = errors.New("not possible in this state of the trade")
	ErrNotInTrade   = errors.New("not a party of this trade")
	ErrLocked       = errors.New("the offer changed a moment ago; wait before accepting")
	ErrEmpty        = errors.New("both offers are empty")
	ErrMissingItem  = errors.New("an offered item is not in the inventory")
	ErrDuplicate    = errors.New("an item is offered twice")
	ErrGold         = errors.New("not enough gold")
	ErrNoSpace      = errors.New("not enough room in the inventory")
	ErrUnknownItem  = errors.New("unknown item size")
	ErrNoContainers = errors.New("a hero has no saved inventory yet")
)

// State of a session.
type State int

// The states.
const (
	Requested State = iota // one player asked, the other has not answered
	Open                   // both see the trade window
	Done                   // committed
	Cancelled              // by a player, a disconnect, or a failed commit
)

func (s State) String() string {
	return [...]string{"requested", "open", "done", "cancelled"}[s]
}

// Offer is what one side puts into the trade window: items of its inventory
// page (identified by code and position) and gold.
type Offer struct {
	Items []d2hero.StoredItem `json:"items,omitempty"`
	Gold  int                 `json:"gold,omitempty"`
}

// Empty reports whether nothing is offered.
func (o Offer) Empty() bool { return len(o.Items) == 0 && o.Gold == 0 }

// Session is one trade between two players. The first is who asked.
type Session struct {
	ids      [2]string
	state    State
	offers   [2]Offer
	accepted [2]bool
	lockedTo [2]time.Time
	seqs     [2]uint32 // last client sequence number applied per side
	now      func() time.Time
}

// NewSession starts a request from one player to another.
func NewSession(from, to string) *Session {
	return &Session{ids: [2]string{from, to}, state: Requested, now: time.Now}
}

// SetClock replaces the clock (tests).
func (s *Session) SetClock(now func() time.Time) { s.now = now }

// IDs returns the two players, the requester first.
func (s *Session) IDs() (string, string) { return s.ids[0], s.ids[1] }

// State returns the state.
func (s *Session) State() State { return s.state }

func (s *Session) side(id string) (int, error) {
	switch id {
	case s.ids[0]:
		return 0, nil
	case s.ids[1]:
		return 1, nil
	}

	return 0, ErrNotInTrade
}

// Other returns the other player.
func (s *Session) Other(id string) string {
	if id == s.ids[0] {
		return s.ids[1]
	}

	return s.ids[0]
}

// Involves reports whether the id takes part.
func (s *Session) Involves(id string) bool { _, err := s.side(id); return err == nil }

// Respond answers the request (only the asked player can). Declining cancels.
func (s *Session) Respond(by string, ok bool) error {
	i, err := s.side(by)
	if err != nil {
		return err
	}

	if s.state != Requested || i != 1 {
		return ErrState
	}

	if !ok {
		s.state = Cancelled
		return nil
	}

	s.state = Open

	return nil
}

// Offer returns what a side offers.
func (s *Session) Offer(id string) Offer {
	i, err := s.side(id)
	if err != nil {
		return Offer{}
	}

	return s.offers[i]
}

// OfferSeq is the sequence number of the last offer of a side that was applied
// with SetOfferSeq (0 if none).
func (s *Session) OfferSeq(id string) uint32 {
	i, err := s.side(id)
	if err != nil {
		return 0
	}

	return s.seqs[i]
}

// SetOfferSeq records the client's sequence number of the offer just applied.
// Numbers never go back.
func (s *Session) SetOfferSeq(by string, seq uint32) {
	if i, err := s.side(by); err == nil && seq > s.seqs[i] {
		s.seqs[i] = seq
	}
}

// Accepted reports whether a side has accepted.
func (s *Session) Accepted(id string) bool {
	i, err := s.side(id)

	return err == nil && s.accepted[i]
}

// SetOffer replaces a side's offer. Both acceptances are revoked and both
// sides are locked out of accepting for AcceptLock.
func (s *Session) SetOffer(by string, o Offer) error {
	i, err := s.side(by)
	if err != nil {
		return err
	}

	if s.state != Open {
		return ErrState
	}

	if o.Gold < 0 {
		return ErrGold
	}

	seen := map[[3]interface{}]bool{}

	for _, it := range o.Items {
		k := [3]interface{}{it.Page, it.X, it.Y}
		if seen[k] {
			return ErrDuplicate
		}

		seen[k] = true
	}

	s.offers[i] = o
	s.accepted = [2]bool{}

	until := s.now().Add(AcceptLock)
	s.lockedTo = [2]time.Time{until, until}

	return nil
}

// Accept marks a side as accepting the current offers; ready is true when both
// have. Accepting again does nothing.
func (s *Session) Accept(by string) (ready bool, err error) {
	i, err := s.side(by)
	if err != nil {
		return false, err
	}

	if s.state != Open {
		return false, ErrState
	}

	if s.now().Before(s.lockedTo[i]) {
		return false, ErrLocked
	}

	if s.offers[0].Empty() && s.offers[1].Empty() {
		return false, ErrEmpty
	}

	s.accepted[i] = true

	return s.accepted[0] && s.accepted[1], nil
}

// Unaccept withdraws a side's acceptance.
func (s *Session) Unaccept(by string) error {
	i, err := s.side(by)
	if err != nil {
		return err
	}

	s.accepted[i] = false

	return nil
}

// Cancel ends the trade (either side, any time before it is done).
func (s *Session) Cancel(by string) error {
	if _, err := s.side(by); err != nil {
		return err
	}

	if s.state == Done || s.state == Cancelled {
		return ErrState
	}

	s.state = Cancelled

	return nil
}

// Finish marks a committed trade.
func (s *Session) Finish() { s.state = Done }

// Hand is one hero's tradable possessions: the containers and the gold.
type Hand struct {
	Containers *d2hero.HeroContainers
	Gold       *int
}

// Sizer returns the inventory size of an item code in grid cells.
type Sizer func(code string) (w, h int, ok bool)

// Moved is what one hero gave and got.
type Moved struct {
	Gave       []d2hero.StoredItem // as they were in this hero's inventory
	Got        []d2hero.StoredItem // as they are placed in this hero's inventory now
	GoldBefore int
	GoldAfter  int
}

// Result of a commit, per side (the same order as the arguments).
type Result [2]Moved

func findItem(c *d2hero.HeroContainers, want d2hero.StoredItem) int {
	if c == nil {
		return -1
	}

	for i := range c.Items {
		it := &c.Items[i]
		if it.Page == d2hero.PageInventory && it.X == want.X && it.Y == want.Y && it.Code == want.Code {
			return i
		}
	}

	return -1
}

// plan checks one side's offer against its hand and returns the indexes of the
// offered items.
func plan(h Hand, o Offer) ([]int, error) {
	if h.Gold == nil || o.Gold > *h.Gold {
		return nil, ErrGold
	}

	idx := make([]int, 0, len(o.Items))
	used := map[int]bool{}

	for _, want := range o.Items {
		i := findItem(h.Containers, want)
		if i < 0 {
			return nil, fmt.Errorf("%w: %s at %d,%d", ErrMissingItem, want.Code, want.X, want.Y)
		}

		if used[i] {
			return nil, ErrDuplicate
		}

		used[i] = true
		idx = append(idx, i)
	}

	return idx, nil
}

// place finds slots in the inventory of a hand for the incoming items, after
// the given items left it. The largest items are placed first.
func place(h Hand, leaving map[int]bool, incoming []d2hero.StoredItem, size Sizer) ([]d2hero.StoredItem, error) {
	og := d2inventory.NewOccupancyGrid(InvCols, InvRows)

	if h.Containers != nil {
		for i := range h.Containers.Items {
			it := &h.Containers.Items[i]
			if it.Page != d2hero.PageInventory || leaving[i] {
				continue
			}

			w, hh, ok := size(it.Code)
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrUnknownItem, it.Code)
			}

			og.Fill(it.X, it.Y, w, hh, true)
		}
	}

	order := make([]int, len(incoming))
	for i := range order {
		order[i] = i
	}

	area := func(i int) int {
		w, hh, _ := size(incoming[i].Code)

		return w * hh
	}

	sort.SliceStable(order, func(a, b int) bool { return area(order[a]) > area(order[b]) })

	out := make([]d2hero.StoredItem, len(incoming))

	for _, i := range order {
		w, hh, ok := size(incoming[i].Code)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownItem, incoming[i].Code)
		}

		x, y, ok := og.FindFreeSlot(w, hh, true)
		if !ok {
			return nil, ErrNoSpace
		}

		og.Fill(x, y, w, hh, true)

		it := incoming[i]
		it.Page, it.X, it.Y = d2hero.PageInventory, x, y
		out[i] = it
	}

	return out, nil
}

// Commit moves the offered items and gold between two heroes. Nothing changes
// unless the whole trade can be done.
func Commit(a, b Hand, offerA, offerB Offer, size Sizer) (Result, error) {
	var res Result

	if offerA.Empty() && offerB.Empty() {
		return res, ErrEmpty
	}

	if a.Containers == nil || b.Containers == nil || a.Gold == nil || b.Gold == nil {
		return res, ErrNoContainers // a hero that never saved its containers has nothing to trade yet
	}

	idxA, err := plan(a, offerA)
	if err != nil {
		return res, err
	}

	idxB, err := plan(b, offerB)
	if err != nil {
		return res, err
	}

	itemsOf := func(h Hand, idx []int) []d2hero.StoredItem {
		out := make([]d2hero.StoredItem, len(idx))
		for k, i := range idx {
			out[k] = h.Containers.Items[i]
		}

		return out
	}

	gaveA, gaveB := itemsOf(a, idxA), itemsOf(b, idxB)

	leave := func(idx []int) map[int]bool {
		m := map[int]bool{}
		for _, i := range idx {
			m[i] = true
		}

		return m
	}

	gotA, err := place(a, leave(idxA), gaveB, size)
	if err != nil {
		return res, fmt.Errorf("first player: %w", err)
	}

	gotB, err := place(b, leave(idxB), gaveA, size)
	if err != nil {
		return res, fmt.Errorf("second player: %w", err)
	}

	res[0] = Moved{Gave: gaveA, Got: gotA, GoldBefore: *a.Gold}
	res[1] = Moved{Gave: gaveB, Got: gotB, GoldBefore: *b.Gold}

	apply := func(h Hand, idx []int, got []d2hero.StoredItem) {
		drop := leave(idx)
		kept := make([]d2hero.StoredItem, 0, len(h.Containers.Items)+len(got))

		for i := range h.Containers.Items {
			if !drop[i] {
				kept = append(kept, h.Containers.Items[i])
			}
		}

		h.Containers.Items = append(kept, got...)
	}

	apply(a, idxA, gotA)
	apply(b, idxB, gotB)

	*a.Gold += offerB.Gold - offerA.Gold
	*b.Gold += offerA.Gold - offerB.Gold
	res[0].GoldAfter, res[1].GoldAfter = *a.Gold, *b.Gold

	return res, nil
}
