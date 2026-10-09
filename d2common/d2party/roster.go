package d2party

import (
	"errors"
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// Errors of the roster operations.
var (
	ErrUnknownPlayer = errors.New("unknown player")
	ErrSelf          = errors.New("cannot do that to yourself")
	ErrPartyFull     = errors.New("the party is full")
	ErrNoInvite      = errors.New("no pending invitation")
	ErrAlreadyParty  = errors.New("already in the same party")
	ErrInParty       = errors.New("party members cannot be hostile; leave the party first")
	ErrLevelTooLow   = errors.New("both players need the hostile level")
	ErrHostile       = errors.New("hostile players cannot party")
)

// Member is one player of the game.
type Member struct {
	ID    string
	Name  string
	Class d2enum.Hero
	Level int
	// Area is the level (area id) the player is in; experience is shared
	// among the party members of the same area.
	Area int
	// Hardcore marks a hardcore hero (a hostile kill gives the killer an ear).
	Hardcore bool
}

type entry struct {
	Member
	party   int             // 0: none
	hostile map[string]bool // ids this player has declared hostility to
}

// Roster holds the players of a game and their relations.
type Roster struct {
	players map[string]*entry
	invites map[string]string // invitee id -> inviter id
	next    int
}

// New returns an empty roster.
func New() *Roster {
	return &Roster{players: map[string]*entry{}, invites: map[string]string{}, next: 1}
}

// Add puts a player into the game (replacing a player with the same id).
func (r *Roster) Add(m Member) {
	if e, ok := r.players[m.ID]; ok {
		e.Member = m

		return
	}

	r.players[m.ID] = &entry{Member: m, hostile: map[string]bool{}}
}

// Remove takes a player out of the game: it leaves its party, loses its
// invitations and every hostility flag that involves it.
func (r *Roster) Remove(id string) {
	if _, ok := r.players[id]; !ok {
		return
	}

	r.Leave(id)
	delete(r.players, id)
	delete(r.invites, id)

	for to, from := range r.invites {
		if from == id {
			delete(r.invites, to)
		}
	}

	for _, e := range r.players {
		delete(e.hostile, id)
	}
}

// Has reports whether the id is in the game.
func (r *Roster) Has(id string) bool { _, ok := r.players[id]; return ok }

// Len is the number of players.
func (r *Roster) Len() int { return len(r.players) }

// Member returns a player.
func (r *Roster) Member(id string) (Member, bool) {
	e, ok := r.players[id]
	if !ok {
		return Member{}, false
	}

	return e.Member, true
}

// FindByName returns the id of the player with that name (case sensitive).
func (r *Roster) FindByName(name string) (string, bool) {
	for id, e := range r.players {
		if e.Name == name {
			return id, true
		}
	}

	return "", false
}

// SetLevel updates a player's character level.
func (r *Roster) SetLevel(id string, level int) {
	if e, ok := r.players[id]; ok {
		e.Level = level
	}
}

// SetArea updates the area a player is in.
func (r *Roster) SetArea(id string, area int) {
	if e, ok := r.players[id]; ok {
		e.Area = area
	}
}

// PartyID returns the party of a player (0: none).
func (r *Roster) PartyID(id string) int {
	if e, ok := r.players[id]; ok {
		return e.party
	}

	return 0
}

// SameParty reports whether two different players are in one party.
func (r *Roster) SameParty(a, b string) bool {
	pa := r.PartyID(a)

	return a != b && pa != 0 && pa == r.PartyID(b)
}

// PartyMembers returns the members of the party of a player (including it), sorted
// by id; just the player alone when it has no party.
func (r *Roster) PartyMembers(id string) []Member {
	e, ok := r.players[id]
	if !ok {
		return nil
	}

	if e.party == 0 {
		return []Member{e.Member}
	}

	var out []Member

	for _, o := range r.players {
		if o.party == e.party {
			out = append(out, o.Member)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

func (r *Roster) partySize(party int) int {
	n := 0

	for _, e := range r.players {
		if e.party == party {
			n++
		}
	}

	return n
}

// Invite records an invitation of to by from. A player can only be invited
// when neither is hostile to the other and the party has room.
func (r *Roster) Invite(from, to string) error {
	f, ok1 := r.players[from]
	t, ok2 := r.players[to]

	switch {
	case !ok1 || !ok2:
		return ErrUnknownPlayer
	case from == to:
		return ErrSelf
	case r.SameParty(from, to):
		return ErrAlreadyParty
	case f.hostile[to] || t.hostile[from]:
		return ErrHostile
	case f.party != 0 && r.partySize(f.party) >= d2enum.MaxPlayersInGame:
		return ErrPartyFull
	}

	r.invites[to] = from

	return nil
}

// InvitedBy returns who invited the player (pending), or "".
func (r *Roster) InvitedBy(id string) string { return r.invites[id] }

// Decline drops the pending invitation of a player.
func (r *Roster) Decline(id string) error {
	if _, ok := r.invites[id]; !ok {
		return ErrNoInvite
	}

	delete(r.invites, id)

	return nil
}

// Accept joins the invited player to the inviter's party (made on demand). A
// player that is in another party leaves it first. It returns the party id.
func (r *Roster) Accept(id string) (int, error) {
	from, ok := r.invites[id]
	if !ok {
		return 0, ErrNoInvite
	}

	f, t := r.players[from], r.players[id]
	if f == nil || t == nil {
		delete(r.invites, id)

		return 0, ErrUnknownPlayer
	}

	if f.party != 0 && r.partySize(f.party) >= d2enum.MaxPlayersInGame {
		return 0, ErrPartyFull
	}

	delete(r.invites, id)

	if t.party != 0 {
		r.Leave(id)
	}

	if f.party == 0 {
		f.party = r.next
		r.next++
	}

	t.party = f.party

	// a party shares no hostility
	for _, o := range r.players {
		if o.party == t.party {
			delete(o.hostile, id)
			delete(t.hostile, o.ID)
		}
	}

	return t.party, nil
}

// Leave takes a player out of its party. A party left with fewer than two
// members is dissolved (verified, 0x53e4e0). It reports whether it was in one.
func (r *Roster) Leave(id string) bool {
	e, ok := r.players[id]
	if !ok || e.party == 0 {
		return false
	}

	party := e.party
	e.party = 0

	if r.partySize(party) < 2 {
		for _, o := range r.players {
			if o.party == party {
				o.party = 0
			}
		}
	}

	return true
}

// CanGoHostile reports why from cannot be hostile to to (nil: it can).
func (r *Roster) CanGoHostile(from, to string) error {
	f, ok1 := r.players[from]
	t, ok2 := r.players[to]

	switch {
	case !ok1 || !ok2:
		return ErrUnknownPlayer
	case from == to:
		return ErrSelf
	case r.SameParty(from, to):
		return ErrInParty
	case f.Level < d2enum.PlayersHostileLevel || t.Level < d2enum.PlayersHostileLevel:
		return ErrLevelTooLow
	}

	return nil
}

// SetHostile declares (or withdraws) the hostility of from to to.
func (r *Roster) SetHostile(from, to string, hostile bool) error {
	if hostile {
		if err := r.CanGoHostile(from, to); err != nil {
			return err
		}

		r.players[from].hostile[to] = true
		delete(r.invites, from) // no invitations with hostile players
		delete(r.invites, to)

		return nil
	}

	f, ok := r.players[from]
	if !ok || r.players[to] == nil {
		return ErrUnknownPlayer
	}

	delete(f.hostile, to)

	return nil
}

// Hostile reports whether from has declared hostility to to.
func (r *Roster) Hostile(from, to string) bool {
	if e, ok := r.players[from]; ok {
		return e.hostile[to]
	}

	return false
}

// Relation is how from sees to: friend inside a party, enemy when hostility is
// declared in either direction (the panel shows red then), else neutral.
func (r *Roster) Relation(from, to string) d2enum.PlayersRelationships {
	switch {
	case r.SameParty(from, to):
		return d2enum.PlayerRelationFriend
	case r.Hostile(from, to) || r.Hostile(to, from):
		return d2enum.PlayerRelationEnemy
	}

	return d2enum.PlayerRelationNeutral
}

// CanAttack reports whether attacker may hurt defender: the attacker has
// declared hostility to the defender (UNVERIFIED direction) and they are not in
// one party (no friendly fire).
func (r *Roster) CanAttack(attacker, defender string) bool {
	return attacker != defender && !r.SameParty(attacker, defender) && r.Hostile(attacker, defender)
}

// XPShare is one party member's part of a kill's experience.
type XPShare struct {
	ID string
	XP int
}

// ShareXP splits the experience of a kill made by killer among the members of
// its party that are in the killer's area. A killer without a party (or alone
// in the area) gets all of it.
//
// The shares are proportional to the members' character levels and add up to
// xp (the rounding remainder goes to the killer). VERIFIED: only members of
// the same level (area) take part. UNVERIFIED: the weights; the original
// computes each member's share with its own level-difference scaling
// (0x57c300, table at 0x6e2960/0x6e298c) which is not modelled.
func (r *Roster) ShareXP(killer string, xp int) []XPShare {
	k, ok := r.players[killer]
	if !ok {
		return nil
	}

	var group []*entry

	if k.party == 0 {
		group = []*entry{k}
	} else {
		for _, o := range r.players {
			if o.party == k.party && o.Area == k.Area {
				group = append(group, o)
			}
		}
	}

	sort.Slice(group, func(i, j int) bool { return group[i].ID < group[j].ID })

	if len(group) == 1 || xp <= 0 {
		return []XPShare{{ID: killer, XP: xp}}
	}

	total := 0
	for _, o := range group {
		total += imax(o.Level, 1)
	}

	out := make([]XPShare, len(group))
	given, ki := 0, 0

	for i, o := range group {
		out[i] = XPShare{ID: o.ID, XP: xp * imax(o.Level, 1) / total}
		given += out[i].XP

		if o.ID == killer {
			ki = i
		}
	}

	out[ki].XP += xp - given

	return out
}

// Info is one player in a Snapshot.
type Info struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Class     d2enum.Hero `json:"class"`
	Level     int         `json:"level"`
	Area      int         `json:"area"`
	Hardcore  bool        `json:"hardcore,omitempty"`
	Party     int         `json:"party,omitempty"`
	Hostile   []string    `json:"hostile,omitempty"`
	InvitedBy string      `json:"invitedBy,omitempty"`
}

// Snapshot is the whole roster as it travels from the server to the clients.
type Snapshot struct {
	Players []Info `json:"players"`
}

// Snapshot captures the roster (players sorted by id).
func (r *Roster) Snapshot() Snapshot {
	ids := make([]string, 0, len(r.players))
	for id := range r.players {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	s := Snapshot{Players: make([]Info, 0, len(ids))}

	for _, id := range ids {
		e := r.players[id]
		in := Info{ID: id, Name: e.Name, Class: e.Class, Level: e.Level, Area: e.Area, Hardcore: e.Hardcore,
			Party: e.party, InvitedBy: r.invites[id]}

		for h := range e.hostile {
			in.Hostile = append(in.Hostile, h)
		}

		sort.Strings(in.Hostile)
		s.Players = append(s.Players, in)
	}

	return s
}

// Restore replaces the roster's content with a snapshot.
func (r *Roster) Restore(s Snapshot) {
	r.players, r.invites = map[string]*entry{}, map[string]string{}

	for _, in := range s.Players {
		e := &entry{Member: Member{ID: in.ID, Name: in.Name, Class: in.Class, Level: in.Level, Area: in.Area,
			Hardcore: in.Hardcore}, party: in.Party, hostile: map[string]bool{}}

		for _, h := range in.Hostile {
			e.hostile[h] = true
		}

		r.players[in.ID] = e

		if in.InvitedBy != "" {
			r.invites[in.ID] = in.InvitedBy
		}

		if in.Party >= r.next {
			r.next = in.Party + 1
		}
	}
}

// Summary is a one-line description for logs: "Name(L12 Sorceress party=1 hostile=[Bob])".
func (r *Roster) Summary() string {
	s := r.Snapshot()
	out := ""

	for i, in := range s.Players {
		if i > 0 {
			out += " "
		}

		names := []string{}

		for _, h := range in.Hostile {
			if m, ok := r.Member(h); ok {
				names = append(names, m.Name)
			}
		}

		out += fmt.Sprintf("%s(L%d %s party=%d hostile=%v)", in.Name, in.Level, in.Class, in.Party, names)
	}

	return fmt.Sprintf("ROSTER n=%d [%s]", len(s.Players), out)
}

func imax(a, b int) int {
	if a > b {
		return a
	}

	return b
}
