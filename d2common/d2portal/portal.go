// Package d2portal holds the town portal rules and the registry of open
// portals, independent of the engine.
//
// A town portal is a pair of linked portal objects: one in the field level the
// hero cast it in and one in the town of that act. Every hero owns at most one
// pair: casting again closes the old pair (both ends) and opens a new one.
// The pair outlives level changes (the objects are only drawn while their
// level is the one on screen) and belongs to its creator: the owner and the
// members of the owner's party may use it, nobody else.
//
// VERIFIED from the notes (session-core.md, SERVER_UsePortalObject): the 5 s
// cooldown, the owner/party rule, the destination in the object data and the
// arrival with start type 3; "both ends of a town portal pair are updated if
// present". UNVERIFIED (public game knowledge, not the binary): one pair per
// owner, the replacement on recast, no cast in town without an open pair, and
// where the town end appears (next to the town waypoint).
package d2portal

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// End is one end of a portal pair: a level and a place in it (tiles).
// Auto marks a town end whose place the client chooses when it builds the
// level (next to the town waypoint).
type End struct {
	Level int     `json:"level"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	Auto  bool    `json:"auto,omitempty"`
}

// Pair is the two linked portals of one owner.
type Pair struct {
	ID        int    `json:"id"`
	Owner     string `json:"owner"`
	OwnerName string `json:"ownerName,omitempty"`
	Field     End    `json:"field"` // the portal in the dungeon or wilderness
	Town      End    `json:"town"`  // the portal in town
}

// Portal is one end of a pair seen from the level it stands in.
type Portal struct {
	PairID    int
	Owner     string
	OwnerName string
	Here      End
	Dest      End // the other end
	InTown    bool
}

// Errors of casting and using portals.
var (
	ErrNoDestination = errors.New("portal: no open portal to go back to (cast it in a field level first)")
	ErrNotYours      = errors.New("portal: it belongs to a player outside your party")
	ErrBadLevel      = errors.New("portal: cannot open a portal in this level")
)

// TownOf is the town of the act a level lies in.
func TownOf(level int) int { return d2level.ActStartLevel(d2level.ActOfLevel(level)) }

// PlanOpen decides the pair a cast makes. level is the level the hero stands
// in at (x, y), existing the hero's open pair or nil. Cast in a field level the
// pair is that spot and the town of the act. Cast in town it needs an open
// pair: the town end moves to the hero and the field end stays where it was.
func PlanOpen(owner, name string, level int, x, y float64, existing *Pair) (Pair, error) {
	if level < 1 || level > 132 {
		return Pair{}, fmt.Errorf("%w: %d", ErrBadLevel, level)
	}

	if d2level.IsTown(level) {
		if existing == nil {
			return Pair{}, ErrNoDestination
		}

		p := *existing
		p.Town = End{Level: level, X: x, Y: y}

		return p, nil
	}

	return Pair{Owner: owner, OwnerName: name, Field: End{Level: level, X: x, Y: y}, Town: End{Level: TownOf(level), Auto: true}}, nil
}

// CanUse applies the owner rule: the owner and the owner's party use a portal.
// With restricted false (a single player game) everybody may.
func CanUse(p Portal, user string, inParty, restricted bool) error {
	if restricted && p.Owner != user && !inParty {
		return ErrNotYours
	}

	return nil
}

// Registry is the set of open pairs, one per owner. It is safe for concurrent
// use (the server shares one among its connections).
type Registry struct {
	mu    sync.Mutex
	pairs map[string]Pair
	next  int
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{pairs: map[string]Pair{}, next: 1} }

// Open stores the pair as its owner's only pair and returns it with its id,
// and the pair it replaced (ok false if there was none).
func (r *Registry) Open(p Pair) (stored, replaced Pair, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	replaced, ok = r.pairs[p.Owner]

	if ok && p.ID == replaced.ID && p.ID != 0 {
		// the same pair with a moved end (cast again in town): keep its id
		r.pairs[p.Owner] = p

		return p, replaced, ok
	}

	p.ID = r.next
	r.next++
	r.pairs[p.Owner] = p

	return p, replaced, ok
}

// Get returns an owner's pair.
func (r *Registry) Get(owner string) (Pair, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pairs[owner]

	return p, ok
}

// Close removes an owner's pair and reports whether there was one.
func (r *Registry) Close(owner string) (Pair, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pairs[owner]
	delete(r.pairs, owner)

	return p, ok
}

// Pairs lists all open pairs, ordered by id.
func (r *Registry) Pairs() []Pair {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Pair, 0, len(r.pairs))
	for _, p := range r.pairs {
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// Replace swaps the whole content (a client taking the server's list).
func (r *Registry) Replace(pairs []Pair) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pairs = make(map[string]Pair, len(pairs))

	for _, p := range pairs {
		r.pairs[p.Owner] = p
		if p.ID >= r.next {
			r.next = p.ID + 1
		}
	}
}

// In lists the portals standing in a level: the field end of the pairs cast
// there and the town end of the pairs whose town is that level.
func (r *Registry) In(level int) []Portal {
	return PortalsIn(r.Pairs(), level)
}

// PortalsIn is In for a list of pairs.
func PortalsIn(pairs []Pair, level int) []Portal {
	var out []Portal

	for _, p := range pairs {
		if p.Field.Level == level {
			out = append(out, Portal{PairID: p.ID, Owner: p.Owner, OwnerName: p.OwnerName, Here: p.Field, Dest: p.Town})
		}

		if p.Town.Level == level {
			out = append(out, Portal{PairID: p.ID, Owner: p.Owner, OwnerName: p.OwnerName, Here: p.Town, Dest: p.Field, InTown: true})
		}
	}

	return out
}
