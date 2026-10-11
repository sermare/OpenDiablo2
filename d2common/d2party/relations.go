package d2party

import "errors"

// Per-direction relation nodes. In the original every server player owns a
// list with one node per OTHER player it has a relationship with (party,
// hostile or invitation); the node A->B and the node B->A are separate
// records (PLAYER_AddRelationEntry 0x558f30). A node carries a bit saying the
// other player is in a party (party id != 0xFFFF at creation), which the server
// pushes to clients with packet 0x8B (PLAYER_SendPartyRelationUpdate
// 0x559210, which skips self). No relation is created when owner and other are
// the same unit or when EITHER has state 7. Row 7 of states.txt is
// STATE_PLAYERBODY, i.e. the player is dead and only a body remains (the link
// between the exe's check and that state name is the table row: VERIFIED row,
// UNVERIFIED intent).

// ErrPlayerBody is returned when a relation is refused because a player has
// state 7 (is a body).
var ErrPlayerBody = errors.New("a dead player cannot take part in a relation")

// Relation is one direction of a relationship, as the owner holds it.
type Relation struct {
	// OtherInParty is the "other is in a party" bit (the byte of packet 0x8B).
	OtherInParty bool
}

// SetPlayerBody marks (or clears) state 7 of a player.
func (r *Roster) SetPlayerBody(id string, body bool) {
	if e, ok := r.players[id]; ok {
		e.body = body
	}
}

// PlayerBody reports state 7 of a player.
func (r *Roster) PlayerBody(id string) bool {
	e, ok := r.players[id]

	return ok && e.body
}

// AddRelation creates the owner's node for other if it has none. It reports
// whether a node exists afterwards; owner==other and a state-7 player on
// either side make it a no-op that returns ErrSelf/ErrPlayerBody. An existing
// node is kept as it is (its bit is refreshed only by RefreshPartyBits).
func (r *Roster) AddRelation(owner, other string) error {
	o, ok1 := r.players[owner]
	t, ok2 := r.players[other]

	switch {
	case !ok1 || !ok2:
		return ErrUnknownPlayer
	case owner == other:
		return ErrSelf
	case o.body || t.body:
		return ErrPlayerBody
	}

	if _, has := o.rel[other]; has {
		return nil
	}

	if o.rel == nil {
		o.rel = map[string]*Relation{}
	}

	o.rel[other] = &Relation{OtherInParty: t.party != 0}

	return nil
}

// RelationOf returns the owner's node for other.
func (r *Roster) RelationOf(owner, other string) (Relation, bool) {
	if o, ok := r.players[owner]; ok {
		if n := o.rel[other]; n != nil {
			return *n, true
		}
	}

	return Relation{}, false
}

// RefreshPartyBits recomputes the "other is in a party" bit of every node
// (PLAYERLIST_RefreshPartyRelationFlags 0x559410; body not read in detail,
// UNVERIFIED that it is a plain recompute).
func (r *Roster) RefreshPartyBits() {
	for _, o := range r.players {
		for id, n := range o.rel {
			n.OtherInParty = r.players[id] != nil && r.players[id].party != 0
		}
	}
}

// dropRelations removes every node that involves id (PLAYERLIST_FreeRelationList
// 0x559540: the leaving player's own list and its entries in the others').
func (r *Roster) dropRelations(id string) {
	delete(r.players[id].rel, id)

	for _, o := range r.players {
		delete(o.rel, id)
	}

	r.players[id].rel = nil
}
