package d2server

import (
	"strings"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2portal"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client/d2clientconnectiontype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

type fakeConn struct {
	id    string
	state *d2hero.HeroState
	got   []d2netpacket.NetPacket
}

func (f *fakeConn) GetUniqueID() string { return f.id }
func (f *fakeConn) GetConnectionType() d2clientconnectiontype.ClientConnectionType {
	return d2clientconnectiontype.LANClient
}
func (f *fakeConn) GetPlayerState() *d2hero.HeroState  { return f.state }
func (f *fakeConn) SetPlayerState(s *d2hero.HeroState) { f.state = s }
func (f *fakeConn) SendPacketToClient(p d2netpacket.NetPacket) error {
	f.got = append(f.got, p)
	return nil
}

func hero(name string, level, gold int, items ...d2hero.StoredItem) *d2hero.HeroState {
	return &d2hero.HeroState{HeroName: name, HeroType: d2enum.HeroSorceress, Gold: gold,
		Stats: &d2hero.HeroStatsState{Level: level}, Containers: &d2hero.HeroContainers{Items: items}}
}

func item(code string, x, y int) d2hero.StoredItem {
	return d2hero.StoredItem{Code: code, Page: d2hero.PageInventory, X: x, Y: y}
}

func testServer() (*GameServer, *fakeConn, *fakeConn) {
	g := &GameServer{connections: map[string]ClientConnection{}, soc: newSocial(), Logger: d2util.NewLogger()}
	g.soc.size = func(code string) (int, int, bool) { return 1, 1, true }

	a := &fakeConn{id: "a", state: hero("Ann", 30, 500, item("rin", 0, 0))}
	b := &fakeConn{id: "b", state: hero("Bob", 20, 40, item("amu", 4, 2))}
	g.connections["a"], g.connections["b"] = a, b
	g.socialAddPlayer(a)
	g.socialAddPlayer(b)

	return g, a, b
}

func party(t *testing.T, g *GameServer, c *fakeConn, op, target string) {
	t.Helper()

	p, _ := d2netpacket.CreatePartyCommandPacket(op, target)
	if err := g.onPartyCommand(c, p); err != nil {
		t.Fatal(err)
	}
}

func lastRoster(t *testing.T, c *fakeConn) d2netpacket.RosterUpdatePacket {
	t.Helper()

	for i := len(c.got) - 1; i >= 0; i-- {
		if c.got[i].PacketType == d2netpackettype.RosterUpdate {
			u, err := d2netpacket.UnmarshalRosterUpdate(c.got[i].PacketData)
			if err != nil {
				t.Fatal(err)
			}

			return u
		}
	}

	t.Fatal("no roster update")

	return d2netpacket.RosterUpdatePacket{}
}

func TestServerPartyAndPvP(t *testing.T) {
	g, a, b := testServer()

	// by name, accept, the roster reaches both
	party(t, g, a, d2netpacket.PartyInvite, "Bob")

	if u := lastRoster(t, b); !strings.Contains(u.Notice, "invites you") || u.Player != "b" {
		t.Fatalf("invitation notice %+v", u)
	}

	party(t, g, b, d2netpacket.PartyAccept, "")

	if !g.soc.roster.SameParty("a", "b") {
		t.Fatal("not in a party after accept")
	}

	// hostility against a party mate is allowed and takes the declarer out of
	// the party (VERIFIED Game.exe 0x5a3870); an open town portal of the
	// declarer is destroyed
	g.soc.portals.Open(d2portal.Pair{Owner: "a", OwnerName: "Ann"})
	party(t, g, a, d2netpacket.PartyHostile, "b")

	if u := lastRoster(t, a); !strings.Contains(u.Notice, "hostile") || u.Player != "" && u.Player != "a" || !g.soc.roster.Hostile("a", "b") {
		t.Fatalf("hostile declaration: %+v", u)
	}

	if g.soc.roster.SameParty("a", "b") {
		t.Fatal("the declarer must have left the party")
	}

	if _, ok := g.soc.portals.Get("a"); ok {
		t.Fatal("the declarer's portal must be destroyed")
	}

	// a second declaration inside a minute is refused (after peace)
	party(t, g, a, d2netpacket.PartyPeace, "b")
	party(t, g, a, d2netpacket.PartyHostile, "b")

	if g.soc.roster.Hostile("a", "b") {
		t.Fatal("cooldown must refuse the second declaration")
	}

	g.soc.roster.Now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	party(t, g, a, d2netpacket.PartyHostile, "b")

	if !g.soc.roster.Hostile("a", "b") {
		t.Fatal("allowed again after the cooldown")
	}

	hit, _ := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Attacker: "spoofed", Target: "b", Damage: 17, Raw: 100})
	before := len(a.got)

	if err := g.onPvPHit(a, hit); err != nil {
		t.Fatal(err)
	}

	var relayed d2netpacket.PvPHitPacket

	for _, p := range b.got {
		if p.PacketType == d2netpackettype.PvPHit {
			relayed, _ = d2netpacket.UnmarshalPvPHit(p.PacketData)
		}
	}

	if relayed.Attacker != "a" || relayed.Damage != 17 || len(a.got) != before {
		t.Fatalf("relay %+v (the attacker id comes from the connection)", relayed)
	}

	// the hostility is one-way: b cannot hit a
	back, _ := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Target: "a", Damage: 5})
	aBefore := len(a.got)
	_ = g.onPvPHit(b, back)

	for _, p := range a.got[aBefore:] {
		if p.PacketType == d2netpackettype.PvPHit {
			t.Fatal("b is not hostile to a: the hit must be blocked")
		}
	}
}

func TestServerPartyXP(t *testing.T) {
	g, a, b := testServer()
	party(t, g, a, d2netpacket.PartyInvite, "b")
	party(t, g, b, d2netpacket.PartyAccept, "")

	p, _ := d2netpacket.CreatePartyXPPacket(d2netpacket.PartyXPPacket{Killer: "x", XP: 100, Monster: "zombie"})
	if err := g.onPartyXP(a, p); err != nil {
		t.Fatal(err)
	}

	total := 0

	for _, c := range []*fakeConn{a, b} {
		for _, pk := range c.got {
			if pk.PacketType != d2netpackettype.PartyXP {
				continue
			}

			x, _ := d2netpacket.UnmarshalPartyXP(pk.PacketData)
			if x.Player != c.id || x.Killer != "a" {
				t.Fatalf("award %+v to %s", x, c.id)
			}

			total += x.Amount
		}
	}

	if total != 100 {
		t.Fatalf("shares add up to %d, want 100", total)
	}
}

func tradeCmd(t *testing.T, g *GameServer, c *fakeConn, p d2netpacket.TradeCommandPacket) {
	t.Helper()

	np, _ := d2netpacket.CreateTradeCommandPacket(p)
	if err := g.onTradeCommand(c, np); err != nil {
		t.Fatal(err)
	}
}

func lastTrade(t *testing.T, c *fakeConn) d2netpacket.TradeUpdatePacket {
	t.Helper()

	for i := len(c.got) - 1; i >= 0; i-- {
		if c.got[i].PacketType == d2netpackettype.TradeUpdate {
			u, _ := d2netpacket.UnmarshalTradeUpdate(c.got[i].PacketData)

			return u
		}
	}

	t.Fatal("no trade update")

	return d2netpacket.TradeUpdatePacket{}
}

func TestServerTrade(t *testing.T) {
	old := d2playertrade.AcceptLock
	d2playertrade.AcceptLock = 0

	defer func() { d2playertrade.AcceptLock = old }()

	g, a, b := testServer()

	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "Bob"})

	if u := lastTrade(t, b); u.State != "requested" || u.Requester || u.PartnerName != "Ann" {
		t.Fatalf("request as b sees it: %+v", u)
	}

	// offers before the answer are refused
	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: d2playertrade.Offer{Gold: 1}})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: true})

	if u := lastTrade(t, a); u.State != "open" {
		t.Fatalf("open: %+v", u)
	}

	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer,
		Offer: d2playertrade.Offer{Items: []d2hero.StoredItem{item("rin", 0, 0)}, Gold: 100}})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer,
		Offer: d2playertrade.Offer{Items: []d2hero.StoredItem{item("amu", 4, 2)}, Gold: 40}})
	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeAccept})

	if u := lastTrade(t, b); !u.TheyAccepted || u.YouAccepted || len(u.Theirs.Items) != 1 {
		t.Fatalf("after a accepted, b sees %+v", u)
	}

	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeAccept})

	ua, ub := lastTrade(t, a), lastTrade(t, b)
	if ua.State != "done" || ub.State != "done" || ua.Moved == nil || ub.Moved == nil {
		t.Fatalf("done: %+v %+v", ua, ub)
	}

	if ua.Moved.GoldBefore != 500 || ua.Moved.GoldAfter != 440 || ub.Moved.GoldBefore != 40 || ub.Moved.GoldAfter != 100 {
		t.Fatalf("gold a %+v b %+v", ua.Moved, ub.Moved)
	}

	if a.state.Gold != 440 || b.state.Gold != 100 {
		t.Fatalf("server state gold a=%d b=%d", a.state.Gold, b.state.Gold)
	}

	if len(a.state.Containers.Items) != 1 || a.state.Containers.Items[0].Code != "amu" ||
		len(b.state.Containers.Items) != 1 || b.state.Containers.Items[0].Code != "rin" {
		t.Fatalf("items a=%v b=%v", a.state.Containers.Items, b.state.Containers.Items)
	}

	if len(g.soc.trades) != 0 {
		t.Fatal("a finished trade must be forgotten")
	}
}

func TestServerTradeCancelAndLeave(t *testing.T) {
	g, a, b := testServer()

	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "b"})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: false})

	if u := lastTrade(t, a); u.State != "cancelled" || len(g.soc.trades) != 0 {
		t.Fatalf("declined: %+v", u)
	}

	// a trade open when one trader leaves ends for the other
	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "b"})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: true})
	delete(g.connections, "a")
	g.socialRemovePlayer("a")

	if u := lastTrade(t, b); u.State != "cancelled" || len(g.soc.trades) != 0 || g.soc.roster.Has("a") {
		t.Fatalf("leave: %+v", u)
	}

	// a failed commit (gold gone) cancels and changes nothing
	g2, a2, b2 := testServer()
	d2playertrade.AcceptLock = 0

	defer func() { d2playertrade.AcceptLock = 2 * time.Second }()

	tradeCmd(t, g2, a2, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "b"})
	tradeCmd(t, g2, b2, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: true})
	tradeCmd(t, g2, a2, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: d2playertrade.Offer{Gold: 100}})
	a2.state.Gold = 10 // spent elsewhere meanwhile
	tradeCmd(t, g2, a2, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeAccept})
	tradeCmd(t, g2, b2, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeAccept})

	if u := lastTrade(t, a2); u.State != "cancelled" || !strings.Contains(u.Reason, "gold") || a2.state.Gold != 10 || b2.state.Gold != 40 {
		t.Fatalf("failed commit: %+v", u)
	}
}

// The server echoes the sequence number of each side's last offer in every
// update, so a client can tell a stale copy of its offer from a current one;
// the other side's offers do not move it.
func TestServerTradeEchoesOfferSeq(t *testing.T) {
	g, a, b := testServer()

	tradeCmd(t, g, a, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRequest, Target: "Bob"})
	tradeCmd(t, g, b, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeRespond, Accept: true})

	if u := lastTrade(t, a); u.YourSeq != 0 {
		t.Fatalf("seq before any offer: %d", u.YourSeq)
	}

	steps := []struct {
		by         *fakeConn
		offer      d2playertrade.Offer
		seq        uint32
		aSeq, bSeq uint32
	}{
		{a, d2playertrade.Offer{Items: []d2hero.StoredItem{item("rin", 0, 0)}}, 1, 1, 0},
		{b, d2playertrade.Offer{Gold: 5}, 1, 1, 1},
		{a, d2playertrade.Offer{Items: []d2hero.StoredItem{item("rin", 0, 0)}, Gold: 100}, 2, 2, 1},
	}

	for i, st := range steps {
		tradeCmd(t, g, st.by, d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer, Offer: st.offer, Seq: st.seq})

		ua, ub := lastTrade(t, a), lastTrade(t, b)
		if ua.YourSeq != st.aSeq || ub.YourSeq != st.bSeq {
			t.Errorf("step %d: seq a=%d b=%d want %d %d", i, ua.YourSeq, ub.YourSeq, st.aSeq, st.bSeq)
		}
	}

	if u := lastTrade(t, a); len(u.Yours.Items) != 1 || u.Yours.Gold != 100 {
		t.Fatalf("a's final offer %+v", u.Yours)
	}
}
