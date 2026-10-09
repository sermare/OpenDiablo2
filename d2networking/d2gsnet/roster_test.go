package d2gsnet

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2party"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// Level and party id reach the client through the real 0x5b / 0x75 packets;
// when the tunnelled copy disagrees the real value wins.
func TestRealRosterPacketsWinOverTunnel(t *testing.T) {
	c, s := newPair()

	ap, err := mkAdd(31)
	if err != nil {
		t.Fatal(err)
	}

	pkts, err := s.Encode(ap)
	if err != nil {
		t.Fatal(err)
	}

	// the real roster entry is on the wire and decodes with the verified fields
	var entry *d2gs.RosterEntry

	for _, p := range pkts {
		if p[0] == d2gs.S2CRosterEntry {
			re, perr := d2gs.ParseRosterEntry(p)
			if perr != nil {
				t.Fatal(perr)
			}

			entry = &re
		}
	}

	if entry == nil || entry.Level != 31 || entry.Name != "Ann" || entry.PartyID != d2gs.NoParty || entry.Class != uint8(d2enum.HeroSorceress) {
		t.Fatalf("entry %+v", entry)
	}

	// corrupt the tunnelled level: the client must still report the real one
	pkts = replaceTunnelLevel(t, pkts, 99)

	got, ignored, err := c.Decode(wire(t, pkts))
	if err != nil || ignored != 0 {
		t.Fatalf("%v ignored=%d", err, ignored)
	}

	var seen d2netpacket.AddPlayerPacket

	for _, np := range got {
		if np.PacketType == d2netpackettype.AddPlayer {
			seen, _ = d2netpacket.UnmarshalAddPlayer(np.PacketData)
		}
	}

	if seen.Stats == nil || seen.Stats.Level != 31 {
		t.Fatalf("level %+v", seen.Stats)
	}

	unit, _ := c.IDs.Unit("a")
	if p, ok := c.Peer(unit); !ok || p.Level != 31 || p.Name != "Ann" || p.PartyID != d2gs.NoParty {
		t.Fatalf("peer %+v %v", p, ok)
	}

	// party: server roster -> real 0x75 -> client roster patched
	roster := d2party.New()
	roster.Add(d2party.Member{ID: "a", Name: "Ann", Level: 32})
	roster.Add(d2party.Member{ID: "b", Name: "Bob", Level: 12})
	_ = roster.Invite("a", "b")
	_, _ = roster.Accept("b")

	snap := roster.Snapshot()
	ru, _ := d2netpacket.CreateRosterUpdatePacket(d2netpacket.RosterUpdatePacket{Roster: snap})

	pkts, err = s.Encode(ru)
	if err != nil {
		t.Fatal(err)
	}

	real75 := 0

	for _, p := range pkts {
		if p[0] == d2gs.S2CRosterUpdate {
			real75++
		}
	}

	if real75 != 2 {
		t.Fatalf("%d real 0x75 packets", real75)
	}

	// unit id of b on the server side must be what the client was told
	bUnit, _ := s.IDs.Unit("b")
	c.IDs.Set("b", bUnit)

	got, ignored, err = c.Decode(wire(t, pkts))
	if err != nil || ignored != 0 || len(got) != 1 {
		t.Fatalf("%v ignored=%d %d", err, ignored, len(got))
	}

	up, _ := d2netpacket.UnmarshalRosterUpdate(got[0].PacketData)
	if len(up.Roster.Players) != 2 {
		t.Fatalf("%+v", up.Roster)
	}

	for _, in := range up.Roster.Players {
		want := snap.Players[0]
		if in.ID == "b" {
			want = snap.Players[1]
		}

		if in.Level != want.Level || in.Party != want.Party || in.Party == 0 {
			t.Errorf("%s: %+v want level %d party %d", in.ID, in, want.Level, want.Party)
		}
	}
}

func mkAdd(level int) (d2netpacket.NetPacket, error) {
	return d2netpacket.CreateAddPlayerPacket("a", "Ann", 5, 6, d2enum.HeroSorceress, &d2hero.HeroStatsState{Level: level},
		nil, d2inventory.CharacterEquipment{}, 0, 0, 0, nil, 0)
}

// replaceTunnelLevel rewrites the tunnelled AddPlayer with another level.
func replaceTunnelLevel(t *testing.T, pkts [][]byte, level int) [][]byte {
	t.Helper()

	var (
		out  [][]byte
		asm  d2gs.TunnelAssembler
		done bool
	)

	for _, p := range pkts {
		if p[0] != d2gs.S2CMetaAE {
			out = append(out, p)

			continue
		}

		typ, _, fin, err := asm.Add(p)
		if err != nil {
			t.Fatal(err)
		}

		if !fin {
			continue
		}

		done = true
		np, _ := mkAdd(level)
		out = append(out, d2gs.Tunnel(d2gs.S2CMetaAE, typ, np.PacketData)...)
	}

	if !done {
		t.Fatal("no tunnelled AddPlayer")
	}

	return out
}
