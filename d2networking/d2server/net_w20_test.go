package d2server

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

func TestJoinErrCode(t *testing.T) {
	tests := []struct {
		err  error
		want uint32
	}{
		{errServerFull, d2gs.JoinErrFull},
		{errNameInUse, d2gs.JoinErrNameOrGame},
		{errPlayerAlreadyExists, d2gs.JoinErrNameOrGame},
		{errNoPlayerState, 0},
	}

	for _, tc := range tests {
		if got := joinErrCode(tc.err); got != tc.want {
			t.Errorf("%v: %d want %d", tc.err, got, tc.want)
		}
	}
}

// A refused client receives the 0xB4 packet with the exe's code.
func TestRejectJoinWritesB4(t *testing.T) {
	g, _, _ := testServer()
	a, b := net.Pipe()

	defer a.Close()
	defer b.Close()

	_ = b.SetReadDeadline(time.Now().Add(2 * time.Second))

	go g.rejectJoin(a, d2gs.JoinErrFull)

	plain, err := d2gs.ReadBlob(bufio.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	jr, err := d2gs.ParseJoinReject(plain)
	if err != nil || jr.Code != 0xf {
		t.Fatalf("%+v %v", jr, err)
	}
}

func TestPartyHealRelay(t *testing.T) {
	g, a, b := testServer()
	heal, _ := d2netpacket.CreatePartyHealPacket(d2netpacket.PartyHealPacket{Healer: "spoof", Target: "b", Amount: 20 << 8})

	count := func(c *fakeConn) int {
		n := 0

		for _, p := range c.got {
			if p.PacketType == d2netpackettype.PartyHeal {
				n++
			}
		}

		return n
	}

	// not in one party: refused
	if err := g.onPartyHeal(a, heal); err != nil || count(b) != 0 {
		t.Fatalf("heal across parties was relayed (%v)", err)
	}

	party(t, g, a, d2netpacket.PartyInvite, "b")
	party(t, g, b, d2netpacket.PartyAccept, "")

	if err := g.onPartyHeal(a, heal); err != nil || count(b) != 1 {
		t.Fatalf("party heal not relayed: %v %d", err, count(b))
	}

	p, _ := d2netpacket.UnmarshalPartyHeal(b.got[len(b.got)-1].PacketData)
	if p.Healer != "a" || p.Amount != 20<<8 { // the sender is the healer whatever it says
		t.Fatalf("%+v", p)
	}
}

// A hit from a hero standing in a town is refused (0x461460 tests the
// attacker's room); outside the town the same hit goes through.
func TestPvPTownSafeZone(t *testing.T) {
	g, a, b := testServer()
	g.soc.hitQueueOn = true
	g.socialArea("a", 1)
	g.socialArea("b", 1)

	g.soc.mu.Lock()
	_ = g.setHostileLocked("a", "b", true)
	g.soc.mu.Unlock()

	hit, _ := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{Target: "b", Damage: 9, Raw: 50})

	g.socialArea("a", 1) // Rogue Encampment

	if err := g.onPvPHit(a, hit); err != nil || g.soc.hits.Pending("b") != 0 {
		t.Fatalf("a hit from town was queued (%v)", err)
	}

	// w45: a victim standing in town is safe from a hero outside (COMBAT_ApplyDamageToUnit 0x57a640 tests the victim's room)
	g.socialArea("a", 2)

	if err := g.onPvPHit(a, hit); err != nil || g.soc.hits.Pending("b") != 0 {
		t.Fatalf("a hit on a hero in town was queued (%v)", err)
	}

	g.socialArea("b", 2)

	if err := g.onPvPHit(a, hit); err != nil || g.soc.hits.Pending("b") != 1 {
		t.Fatalf("a hit from the wilderness was refused (%v)", err)
	}

	g.flushHits()

	if pvpCount(b) != 1 {
		t.Fatalf("delivered %d", pvpCount(b))
	}
}
