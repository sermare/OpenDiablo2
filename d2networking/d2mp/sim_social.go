package d2mp

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRef splits an item reference "id:code".
func ParseRef(ref string) (id uint32, code string, ok bool) {
	a, b, found := strings.Cut(ref, ":")
	n, err := strconv.ParseUint(a, 10, 32)

	return uint32(n), b, found && err == nil
}

// Command executes a tunnelled client command for a hero.
func (s *Sim) Command(id uint32, c Command) {
	p, ok := s.pl[id]
	if !ok {
		return
	}

	switch c.Type {
	case CmdRespawn:
		s.Respawn(id)
	case CmdUseWaypoint:
		s.UseWaypoint(id, uint16(c.A))
	case CmdPortal:
		if !p.u.Dead {
			s.castPortal(p)
		}
	case CmdPartyInvite:
		s.partyInvite(p, c.Target)
	case CmdPartyAccept:
		s.partyAccept(p)
	case CmdPartyLeave:
		s.roster.Leave(sid(id))
		s.syncParties()
	case CmdTradeRequest:
		s.tradeRequest(p, c.Target)
	case CmdTradeRespond:
		s.tradeRespond(p, c.A != 0)
	case CmdTradeOffer:
		s.tradeOffer(p, c.Items, uint32(c.A))
	case CmdTradeAccept:
		s.tradeAccept(p)
	case CmdTradeCancel:
		s.cancelTrade(id, "cancelled")
	}
}

// ---- party (package d2party) ----

func (s *Sim) syncParties() {
	for _, p := range s.sortedPlayers() {
		party := uint16(s.roster.PartyID(sid(p.u.ID)))
		if party != p.u.Party {
			p.u.Party = party
			s.toAll(Event{Type: EvParty, ID: p.u.ID, A: int32(party)})
		}
	}
}

func (s *Sim) partyInvite(p *pstate, target uint32) {
	t, ok := s.pl[target]
	if !ok {
		return
	}

	if err := s.roster.Invite(sid(p.u.ID), sid(target)); err != nil {
		s.msg(p.u.ID, "cannot invite: %v", err)

		return
	}

	s.msg(target, "%s invites you to a party", p.u.Name)
	s.msg(p.u.ID, "you invited %s", t.u.Name)
}

func (s *Sim) partyAccept(p *pstate) {
	if _, err := s.roster.Accept(sid(p.u.ID)); err != nil {
		s.msg(p.u.ID, "cannot accept: %v", err)

		return
	}

	s.syncParties()
}

// ---- trade ----
//
// The trade window follows the rules of core/d2playertrade.Session (request,
// answer, both offers, both accept, an offer change locks accepting for a
// moment, a walk away or a death cancels). That package pulls in the whole
// hero/inventory stack, so the headless server carries this small copy of the
// state machine; inventories are item lists here.

type tradeOffer struct {
	items []uint32
	gold  uint32
}

type trade struct {
	ids       [2]uint32
	open      bool
	offer     [2]tradeOffer
	acc       [2]bool
	lockUntil uint32
}

func (t *trade) side(id uint32) int {
	if t.ids[0] == id {
		return 0
	}

	return 1
}

func (s *Sim) tradeView(t *trade, viewer uint32, state uint8, reason string) Event {
	me := t.side(viewer)
	them := 1 - me
	refs := func(o tradeOffer) []string {
		var out []string

		for _, id := range o.items {
			if u, ok := s.units[id]; ok {
				out = append(out, fmt.Sprintf("%d:%s", id, u.Name))
			}
		}

		return out
	}

	return Event{Type: EvTrade, ID: viewer, Trade: TradeView{
		State: state, Partner: t.ids[them], MyItems: refs(t.offer[me]), TheirItems: refs(t.offer[them]),
		MyGold: t.offer[me].gold, TheirGold: t.offer[them].gold, MyAccept: t.acc[me], TheirAccept: t.acc[them],
		Incoming: state == TradeRequested && me == 1, Reason: reason,
	}}
}

func (s *Sim) tradeBroadcast(t *trade, state uint8, reason string) {
	for _, id := range t.ids {
		s.send(id, s.tradeView(t, id, state, reason))
	}
}

func (s *Sim) tradeRequest(p *pstate, target uint32) {
	o, ok := s.pl[target]
	if !ok || target == p.u.ID || p.u.Dead || o.u.Dead || o.u.Level != p.u.Level {
		return
	}

	if s.trades[p.u.ID] != nil || s.trades[target] != nil {
		s.msg(p.u.ID, "a trade is already going on")

		return
	}

	t := &trade{ids: [2]uint32{p.u.ID, target}}
	s.trades[p.u.ID], s.trades[target] = t, t
	s.tradeBroadcast(t, TradeRequested, "")
}

func (s *Sim) tradeRespond(p *pstate, accept bool) {
	t := s.trades[p.u.ID]
	if t == nil || t.open || t.ids[1] != p.u.ID {
		return
	}

	if !accept {
		s.cancelTrade(p.u.ID, "declined")

		return
	}

	t.open = true
	s.tradeBroadcast(t, TradeOpen, "")
}

func (s *Sim) tradeOffer(p *pstate, refs []string, gold uint32) {
	t := s.trades[p.u.ID]
	if t == nil || !t.open {
		return
	}

	var ids []uint32

	seen := map[uint32]bool{}

	for _, r := range refs {
		id, _, _ := ParseRef(r)
		owned := false

		for _, it := range p.inv {
			owned = owned || it.id == id
		}

		if !owned || seen[id] {
			s.msg(p.u.ID, "offer refused: item %q is not yours or offered twice", r)

			return
		}

		seen[id] = true
		ids = append(ids, id)
	}

	if gold > p.u.Gold {
		s.msg(p.u.ID, "offer refused: not enough gold")

		return
	}

	me := t.side(p.u.ID)
	t.offer[me] = tradeOffer{items: ids, gold: gold}
	t.acc = [2]bool{}
	t.lockUntil = s.now + s.cfg.TradeLock
	s.tradeBroadcast(t, TradeOpen, "")
}

func (s *Sim) tradeAccept(p *pstate) {
	t := s.trades[p.u.ID]
	if t == nil || !t.open {
		return
	}

	if s.now < t.lockUntil {
		s.msg(p.u.ID, "the offer changed a moment ago; wait before accepting")

		return
	}

	t.acc[t.side(p.u.ID)] = true

	if !(t.acc[0] && t.acc[1]) {
		s.tradeBroadcast(t, TradeOpen, "")

		return
	}

	if len(t.offer[0].items)+int(t.offer[0].gold)+len(t.offer[1].items)+int(t.offer[1].gold) == 0 {
		t.acc = [2]bool{}
		s.tradeBroadcast(t, TradeOpen, "")

		return
	}

	a, b := s.pl[t.ids[0]], s.pl[t.ids[1]]
	if t.offer[0].gold > a.u.Gold || t.offer[1].gold > b.u.Gold {
		s.cancelTrade(p.u.ID, "not enough gold")

		return
	}

	move := func(from, to *pstate, ids []uint32) {
		for _, id := range ids {
			for i, it := range from.inv {
				if it.id == id {
					from.inv = append(from.inv[:i], from.inv[i+1:]...)
					to.inv = append(to.inv, it)
					s.units[id].Owner = to.u.ID

					break
				}
			}
		}
	}

	move(a, b, t.offer[0].items)
	move(b, a, t.offer[1].items)
	a.u.Gold = a.u.Gold - t.offer[0].gold + t.offer[1].gold
	b.u.Gold = b.u.Gold - t.offer[1].gold + t.offer[0].gold

	s.tradeBroadcast(t, TradeDone, "")
	delete(s.trades, t.ids[0])
	delete(s.trades, t.ids[1])
	s.sendInv(a)
	s.sendInv(b)
}

func (s *Sim) cancelTrade(id uint32, reason string) {
	t := s.trades[id]
	if t == nil {
		return
	}

	delete(s.trades, t.ids[0])
	delete(s.trades, t.ids[1])

	for _, tid := range t.ids {
		s.send(tid, s.tradeView(t, tid, TradeCancelled, reason))
	}
}
