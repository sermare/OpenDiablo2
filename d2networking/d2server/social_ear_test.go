package d2server

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

func killsTo(c *fakeConn) (out []d2netpacket.PvPHitPacket) {
	for _, p := range c.got {
		if p.PacketType == d2netpackettype.PvPHit {
			if h, err := d2netpacket.UnmarshalPvPHit(p.PacketData); err == nil && h.Kill {
				out = append(out, h)
			}
		}
	}

	return out
}

// The victim's client reports a kill; only a hardcore victim hostile to the
// killer gives an ear, and the ear's player is named from the roster.
func TestServerPvPKillEar(t *testing.T) {
	g, a, b := testServer() // Ann (a, sorceress L30), Bob (b, sorceress L20)
	party(t, g, a, d2netpacket.PartyHostile, "b")

	kill, _ := d2netpacket.CreatePvPHitPacket(d2netpacket.PvPHitPacket{
		Kill: true, Target: "a", X: 12, Y: 34, VictimName: "spoofed", VictimLevel: 99, VictimClass: 6})

	// softcore victim: nothing is sent
	if err := g.onPvPHit(b, kill); err != nil || len(killsTo(a)) != 0 {
		t.Fatalf("a softcore kill gave an ear: %v %v", err, killsTo(a))
	}

	b.state.Hardcore = true
	g.socialAddPlayer(b)
	party(t, g, a, d2netpacket.PartyHostile, "b")

	if err := g.onPvPHit(b, kill); err != nil {
		t.Fatal(err)
	}

	got := killsTo(a)
	if len(got) != 1 || got[0].VictimName != "Bob" || got[0].VictimLevel != 20 || got[0].VictimClass != earClassOrder[d2enum.HeroSorceress] ||
		got[0].X != 12 || got[0].Y != 34 || got[0].Attacker != "b" {
		t.Fatalf("kill relay %+v (name, class and level come from the roster)", got)
	}

	// a kill by a player the victim has no hostility with is refused
	g2, a2, b2 := testServer()
	b2.state.Hardcore = true
	g2.socialAddPlayer(b2)

	if err := g2.onPvPHit(b2, kill); err != nil || len(killsTo(a2)) != 0 {
		t.Fatalf("a kill without hostility gave an ear: %v", killsTo(a2))
	}
}
