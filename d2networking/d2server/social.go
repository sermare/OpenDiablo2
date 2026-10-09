package d2server

import (
	"fmt"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// social is the server's party, trade and player-versus-player state. The
// server is authoritative for the roster (who is in which party, who is
// hostile to whom) and for the trades (it moves the items between the two
// heroes' saved containers); the clients resolve their own combat like they do
// against monsters.
type social struct {
	mu     sync.Mutex
	roster *d2party.Roster
	trades map[string]*d2playertrade.Session // by player id; both traders map to the session
	// size replaces the item size lookup of the game data (tests).
	size d2playertrade.Sizer
}

func newSocial() *social {
	return &social{roster: d2party.New(), trades: map[string]*d2playertrade.Session{}}
}

// socialAddPlayer enters a player into the roster.
func (g *GameServer) socialAddPlayer(client ClientConnection) {
	st := client.GetPlayerState()
	if st == nil {
		return
	}

	g.soc.mu.Lock()
	g.soc.roster.Add(d2party.Member{ID: client.GetUniqueID(), Name: st.HeroName, Class: st.HeroType,
		Level: heroLevel(st), Area: d2level.RogueEncampment, Hardcore: st.Hardcore})
	g.soc.mu.Unlock()

	g.broadcastRoster("")
}

// socialRemovePlayer takes a player out of the roster and ends its trade.
func (g *GameServer) socialRemovePlayer(id string) {
	g.soc.mu.Lock()
	name := g.nameOf(id)

	if s := g.soc.trades[id]; s != nil {
		_ = s.Cancel(id)
		g.endTrade(s, "partner left")
	}

	g.soc.roster.Remove(id)
	g.soc.mu.Unlock()

	g.broadcastRoster(fmt.Sprintf("%s left the game", name))
}

// connByID looks a connection up. Like the rest of the server it reads the
// connection map without the server lock (OnClientConnected runs under it).
func (g *GameServer) connByID(id string) ClientConnection {
	return g.connections[id]
}

// broadcastRoster sends the roster to every client. Called without soc.mu.
func (g *GameServer) broadcastRoster(notice string) { g.sendRoster(notice, "") }

func (g *GameServer) sendRoster(notice, player string) {
	g.soc.mu.Lock()
	snap := g.soc.roster.Snapshot()
	g.soc.mu.Unlock()

	pkt, err := d2netpacket.CreateRosterUpdatePacket(d2netpacket.RosterUpdatePacket{Roster: snap, Notice: notice, Player: player})
	if err != nil {
		g.Errorf("RosterUpdate: %v", err)
		return
	}

	conns := make([]ClientConnection, 0, len(g.connections))

	for _, c := range g.connections {
		conns = append(conns, c)
	}

	for _, c := range conns {
		if err := c.SendPacketToClient(pkt); err != nil {
			g.Errorf("RosterUpdate to %s: %v", c.GetUniqueID(), err)
		}
	}
}

// resolve returns the id for a player id or a hero name.
func (g *GameServer) resolveLocked(target string) (string, bool) {
	if g.soc.roster.Has(target) {
		return target, true
	}

	return g.soc.roster.FindByName(target)
}

func (g *GameServer) nameOf(id string) string {
	if m, ok := g.soc.roster.Member(id); ok {
		return m.Name
	}

	return id
}

// onPartyCommand handles invitations, leaving and hostility.
func (g *GameServer) onPartyCommand(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPartyCommand(packet.PacketData)
	if err != nil {
		return err
	}

	me := client.GetUniqueID()

	g.soc.mu.Lock()
	r := g.soc.roster
	target, found := g.resolveLocked(p.Target)
	myName := g.nameOf(me)

	var (
		opErr  error
		notice string
		forWho string
	)

	switch p.Op {
	case d2netpacket.PartyInvite:
		if !found {
			opErr = d2party.ErrUnknownPlayer
		} else if opErr = r.Invite(me, target); opErr == nil {
			notice, forWho = fmt.Sprintf("%s invites you to a party", myName), target
			g.Infof("PARTY invite from=%q to=%q", myName, g.nameOf(target))
		}
	case d2netpacket.PartyAccept:
		from := r.InvitedBy(me)

		if id, aerr := r.Accept(me); aerr != nil {
			opErr = aerr
		} else {
			notice = fmt.Sprintf("%s joined the party of %s", myName, g.nameOf(from))
			g.Infof("PARTY join name=%q party=%d with=%q members=%d", myName, id, g.nameOf(from), len(r.PartyMembers(me)))
		}
	case d2netpacket.PartyDecline:
		from := r.InvitedBy(me)
		if opErr = r.Decline(me); opErr == nil {
			notice = fmt.Sprintf("%s declined the invitation of %s", myName, g.nameOf(from))
			g.Infof("PARTY decline name=%q", myName)
		}
	case d2netpacket.PartyLeave:
		if r.Leave(me) {
			notice = fmt.Sprintf("%s left the party", myName)
			g.Infof("PARTY leave name=%q", myName)
		} else {
			opErr = fmt.Errorf("not in a party")
		}
	case d2netpacket.PartyHostile, d2netpacket.PartyPeace:
		if !found {
			opErr = d2party.ErrUnknownPlayer
		} else if opErr = r.SetHostile(me, target, p.Op == d2netpacket.PartyHostile); opErr == nil {
			notice = fmt.Sprintf("%s is now %s toward %s", myName, map[string]string{
				d2netpacket.PartyHostile: "hostile", d2netpacket.PartyPeace: "at peace"}[p.Op], g.nameOf(target))
			g.Infof("PARTY %s name=%q toward=%q", p.Op, myName, g.nameOf(target))
		}
	default:
		opErr = fmt.Errorf("unknown party command %q", p.Op)
	}

	g.soc.mu.Unlock()

	if opErr != nil {
		g.Infof("PARTY refused op=%s name=%q: %v", p.Op, myName, opErr)
		g.sendRoster(fmt.Sprintf("%s: %v", p.Op, opErr), me)

		return nil
	}

	g.sendRoster(notice, forWho)

	return nil
}

// pvpBlocked is why a hit is refused, "" when it is allowed. Call with soc.mu held.
func (g *GameServer) pvpBlockedLocked(attacker, target string) string {
	r := g.soc.roster

	switch {
	case !r.Has(attacker) || !r.Has(target):
		return "unknown player"
	case attacker == target:
		return "self"
	case r.SameParty(attacker, target):
		return "party members do not hurt each other"
	case !r.Hostile(attacker, target):
		return "not hostile"
	}

	return ""
}

// onPvPHit relays a hit of a hostile player to the defender's client.
func (g *GameServer) onPvPHit(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPvPHit(packet.PacketData)
	if err != nil {
		return err
	}

	p.Attacker = client.GetUniqueID() // the sender is the attacker, whatever it says

	g.soc.mu.Lock()
	reason := g.pvpBlockedLocked(p.Attacker, p.Target)
	an, tn := g.nameOf(p.Attacker), g.nameOf(p.Target)
	g.soc.mu.Unlock()

	if reason != "" {
		g.Infof("PVP BLOCKED attacker=%q target=%q reason=%q", an, tn, reason)
		return nil
	}

	target := g.connByID(p.Target)
	if target == nil {
		return nil
	}

	g.Infof("PVP HIT attacker=%q target=%q damage=%d raw=%d", an, tn, p.Damage, p.Raw)

	pkt, err := d2netpacket.CreatePvPHitPacket(p)
	if err != nil {
		return err
	}

	return target.SendPacketToClient(pkt)
}

// onPartyXP splits the experience of a kill among the killer's party.
func (g *GameServer) onPartyXP(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalPartyXP(packet.PacketData)
	if err != nil {
		return err
	}

	p.Killer = client.GetUniqueID()

	g.soc.mu.Lock()
	shares := g.soc.roster.ShareXP(p.Killer, p.XP)
	names := make([]string, len(shares))

	for i, s := range shares {
		names[i] = fmt.Sprintf("%s=%d", g.nameOf(s.ID), s.XP)
	}
	g.soc.mu.Unlock()

	g.Infof("PARTYXP kill monster=%q killer=%q xp=%d shares=%v", p.Monster, g.nameOf(p.Killer), p.XP, names)

	for _, s := range shares {
		c := g.connByID(s.ID)
		if c == nil {
			continue
		}

		out := p
		out.Player, out.Amount = s.ID, s.XP

		pkt, err := d2netpacket.CreatePartyXPPacket(out)
		if err != nil {
			return err
		}

		if err := c.SendPacketToClient(pkt); err != nil {
			g.Errorf("PartyXP to %s: %v", s.ID, err)
		}
	}

	return nil
}

// tradeUpdateFor builds what one trader sees. Call with soc.mu held.
func (g *GameServer) tradeUpdateFor(s *d2playertrade.Session, id, reason string) d2netpacket.TradeUpdatePacket {
	other := s.Other(id)
	first, _ := s.IDs()

	return d2netpacket.TradeUpdatePacket{
		State: s.State().String(), Partner: other, PartnerName: g.nameOf(other), Requester: first == id,
		Yours: s.Offer(id), Theirs: s.Offer(other), YouAccepted: s.Accepted(id), TheyAccepted: s.Accepted(other),
		Reason: reason,
	}
}

// sendTrade sends both traders their view. Call with soc.mu held.
func (g *GameServer) sendTrade(s *d2playertrade.Session, reason string, moved *[2]*d2playertrade.Moved) {
	a, b := s.IDs()

	for i, id := range []string{a, b} {
		c := g.connByID(id)
		if c == nil {
			continue
		}

		u := g.tradeUpdateFor(s, id, reason)
		if moved != nil {
			u.Moved = moved[i]
		}

		pkt, err := d2netpacket.CreateTradeUpdatePacket(u)
		if err != nil {
			g.Errorf("TradeUpdate: %v", err)
			continue
		}

		if err := c.SendPacketToClient(pkt); err != nil {
			g.Errorf("TradeUpdate to %s: %v", id, err)
		}
	}
}

// endTrade tells both traders the trade is over and forgets it. soc.mu held.
func (g *GameServer) endTrade(s *d2playertrade.Session, reason string) {
	a, b := s.IDs()

	g.Infof("TRADE end state=%s between=%q,%q reason=%q", s.State(), g.nameOf(a), g.nameOf(b), reason)
	g.sendTrade(s, reason, nil)
	delete(g.soc.trades, a)
	delete(g.soc.trades, b)
}

// itemSize is the inventory size of an item code.
func (g *GameServer) itemSize(code string) (w, h int, ok bool) {
	if g.soc.size != nil {
		return g.soc.size(code)
	}

	rec := g.asset.Records.Item.All[code]
	if rec == nil {
		return 0, 0, false
	}

	return rec.InventoryWidth, rec.InventoryHeight, true
}

// onTradeCommand drives the trade sessions.
func (g *GameServer) onTradeCommand(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalTradeCommand(packet.PacketData)
	if err != nil {
		return err
	}

	me := client.GetUniqueID()

	g.soc.mu.Lock()
	defer g.soc.mu.Unlock()

	myName := g.nameOf(me)
	s := g.soc.trades[me]

	fail := func(e error) error {
		g.Infof("TRADE refused op=%s name=%q: %v", p.Op, myName, e)

		if s != nil {
			g.sendTrade(s, e.Error(), nil)
		} else {
			pkt, _ := d2netpacket.CreateTradeUpdatePacket(d2netpacket.TradeUpdatePacket{State: d2playertrade.Cancelled.String(), Reason: e.Error()})
			_ = client.SendPacketToClient(pkt)
		}

		return nil
	}

	switch p.Op {
	case d2netpacket.TradeRequest:
		target, ok := g.resolveLocked(p.Target)

		switch {
		case !ok:
			return fail(d2party.ErrUnknownPlayer)
		case target == me:
			return fail(d2party.ErrSelf)
		case s != nil || g.soc.trades[target] != nil:
			return fail(fmt.Errorf("a trade is already open"))
		}

		ns := d2playertrade.NewSession(me, target)
		g.soc.trades[me], g.soc.trades[target] = ns, ns
		g.Infof("TRADE request from=%q to=%q", myName, g.nameOf(target))
		g.sendTrade(ns, "", nil)
	case d2netpacket.TradeRespond:
		if s == nil {
			return fail(d2playertrade.ErrState)
		}

		if err := s.Respond(me, p.Accept); err != nil {
			return fail(err)
		}

		if s.State() == d2playertrade.Cancelled {
			g.endTrade(s, "declined")

			return nil
		}

		g.Infof("TRADE open between=%q,%q", g.nameOf(s.Other(me)), myName)
		g.sendTrade(s, "", nil)
	case d2netpacket.TradeOffer:
		if s == nil {
			return fail(d2playertrade.ErrState)
		}

		if err := s.SetOffer(me, p.Offer); err != nil {
			return fail(err)
		}

		g.Infof("TRADE offer name=%q items=%d gold=%d", myName, len(p.Offer.Items), p.Offer.Gold)
		g.sendTrade(s, "", nil)
	case d2netpacket.TradeAccept:
		if s == nil {
			return fail(d2playertrade.ErrState)
		}

		ready, err := s.Accept(me)
		if err != nil {
			return fail(err)
		}

		g.Infof("TRADE accept name=%q both=%v", myName, ready)

		if !ready {
			g.sendTrade(s, "", nil)
			return nil
		}

		g.commitTrade(s)
	case d2netpacket.TradeCancel:
		if s == nil {
			return nil
		}

		if err := s.Cancel(me); err != nil {
			return fail(err)
		}

		g.endTrade(s, myName+" cancelled")
	default:
		return fail(fmt.Errorf("unknown trade command %q", p.Op))
	}

	return nil
}

// commitTrade moves the offers between the two heroes and saves both. soc.mu held.
func (g *GameServer) commitTrade(s *d2playertrade.Session) {
	ida, idb := s.IDs()
	ca, cb := g.connByID(ida), g.connByID(idb)

	if ca == nil || cb == nil {
		_ = s.Cancel(ida)
		g.endTrade(s, "a trader is gone")

		return
	}

	sa, sb := ca.GetPlayerState(), cb.GetPlayerState()
	na, nb := sa.HeroName, sb.HeroName
	offerA, offerB := s.Offer(ida), s.Offer(idb)

	res, err := d2playertrade.Commit(
		d2playertrade.Hand{Containers: sa.Containers, Gold: &sa.Gold},
		d2playertrade.Hand{Containers: sb.Containers, Gold: &sb.Gold},
		offerA, offerB, g.itemSize)
	if err != nil {
		_ = s.Cancel(ida)
		g.Infof("TRADE failed between=%q,%q: %v", na, nb, err)
		g.endTrade(s, "trade failed: "+err.Error())

		return
	}

	s.Finish()

	codes := func(items []d2hero.StoredItem) []string {
		out := make([]string, len(items))
		for i, it := range items {
			out[i] = it.Code
		}

		return out
	}

	g.Infof("TRADE done a=%q gave=%v got=%v gold %d->%d | b=%q gave=%v got=%v gold %d->%d", na,
		codes(res[0].Gave), codes(res[0].Got), res[0].GoldBefore, res[0].GoldAfter,
		nb, codes(res[1].Gave), codes(res[1].Got), res[1].GoldBefore, res[1].GoldAfter)

	for _, st := range []*d2hero.HeroState{sa, sb} {
		if g.heroStateFactory == nil {
			continue // tests
		}

		g.heroStateFactory.RecalcStats(st)

		if err := g.heroStateFactory.Save(st); err != nil {
			g.Errorf("TRADE: saving %s: %v", st.HeroName, err)
		}

		g.saveD2S(st)
	}

	g.sendTrade(s, "", &[2]*d2playertrade.Moved{&res[0], &res[1]})
	delete(g.soc.trades, ida)
	delete(g.soc.trades, idb)
}

// socialPacket dispatches the social packet types; handled is false for others.
func (g *GameServer) socialPacket(client ClientConnection, packet d2netpacket.NetPacket) (handled bool, err error) {
	switch packet.PacketType {
	case d2netpackettype.PartyCommand:
		return true, g.onPartyCommand(client, packet)
	case d2netpackettype.TradeCommand:
		return true, g.onTradeCommand(client, packet)
	case d2netpackettype.PvPHit:
		return true, g.onPvPHit(client, packet)
	case d2netpackettype.PartyXP:
		return true, g.onPartyXP(client, packet)
	}

	return false, nil
}

// socialLevel records a hero's level (a level up reaches the roster panels).
func (g *GameServer) socialLevel(id string, level int) {
	g.soc.mu.Lock()
	m, ok := g.soc.roster.Member(id)
	changed := ok && m.Level != level

	if changed {
		g.soc.roster.SetLevel(id, level)
	}
	g.soc.mu.Unlock()

	if changed {
		g.broadcastRoster("")
	}
}

// socialArea records the level (area) a hero is in: kills share experience
// among the party members of the same area.
func (g *GameServer) socialArea(id string, area int) {
	g.soc.mu.Lock()
	g.soc.roster.SetArea(id, area)
	g.soc.mu.Unlock()
}
