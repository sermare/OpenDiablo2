package d2gsnet

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// The social packets have no native counterpart: they must cross the tunnel in
// both directions, big ones (a trade with items) in several chunks.
func TestSocialPacketsCrossTheTunnel(t *testing.T) {
	c, s := newPair()

	roster := d2party.New()
	roster.Add(d2party.Member{ID: "a", Name: "Ann", Level: 20})
	roster.Add(d2party.Member{ID: "b", Name: "Bob", Level: 12})
	_ = roster.Invite("a", "b")
	_, _ = roster.Accept("b")

	bigItem := d2hero.StoredItem{Code: "rin", Page: d2hero.PageInventory, X: 3, Y: 2, Prefixes: []string{strings.Repeat("p", 3000)}}

	client := []d2netpacket.NetPacket{}

	add := func(np d2netpacket.NetPacket, err error) {
		if err != nil {
			t.Fatal(err)
		}

		client = append(client, np)
	}

	add(d2netpacket.CreatePartyCommandPacket(d2netpacket.PartyInvite, "b"))
	add(d2netpacket.CreateTradeCommandPacket(d2netpacket.TradeCommandPacket{Op: d2netpacket.TradeOffer,
		Offer: d2playertrade.Offer{Items: []d2hero.StoredItem{bigItem}, Gold: 77}}))
	add(d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Attacker: "a", Target: "b", Damage: 17, Raw: 100}))
	add(d2netpacket.CreatePartyXPPacket(d2netpacket.PartyXPPacket{Killer: "a", XP: 99, Monster: "zombie"}))

	for _, np := range client {
		pkts, err := c.Encode(np)
		if err != nil {
			t.Fatal(err)
		}

		got, ignored, err := s.Decode(wire(t, pkts))
		if err != nil || ignored != 0 || len(got) != 1 || got[0].PacketType != np.PacketType {
			t.Fatalf("%s to the server: %v ignored=%d %v", np.PacketType, err, ignored, got)
		}

		if string(got[0].PacketData) != string(np.PacketData) {
			t.Fatalf("%s changed on the way", np.PacketType)
		}
	}

	server := []d2netpacket.NetPacket{}

	ru, _ := d2netpacket.CreateRosterUpdatePacket(d2netpacket.RosterUpdatePacket{Roster: roster.Snapshot(), Notice: "Bob joined"})
	tu, _ := d2netpacket.CreateTradeUpdatePacket(d2netpacket.TradeUpdatePacket{State: "done", PartnerName: "Ann",
		Moved: &d2playertrade.Moved{Gave: []d2hero.StoredItem{bigItem}, GoldBefore: 5, GoldAfter: 82}})
	server = append(server, ru, tu)

	for _, np := range server {
		pkts, err := s.Encode(np)
		if err != nil {
			t.Fatal(err)
		}

		got, ignored, err := c.Decode(wire(t, pkts))
		if err != nil || ignored != 0 || len(got) != 1 || got[0].PacketType != np.PacketType {
			t.Fatalf("%s to the client: %v ignored=%d %v", np.PacketType, err, ignored, got)
		}
	}

	// the roster arrives intact
	pkts, _ := s.Encode(ru)
	got, _, _ := c.Decode(wire(t, pkts))
	up, err := d2netpacket.UnmarshalRosterUpdate(got[0].PacketData)

	if err != nil || got[0].PacketType != d2netpackettype.RosterUpdate {
		t.Fatal(err)
	}

	replica := d2party.New()
	replica.Restore(up.Roster)

	if !replica.SameParty("a", "b") || up.Notice != "Bob joined" {
		t.Fatalf("roster replica: %s", replica.Summary())
	}
}
