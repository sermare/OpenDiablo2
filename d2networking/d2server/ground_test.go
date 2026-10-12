package d2server

import (
	"sync"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

func groundOps(c *fakeConn, op string) []d2netpacket.GroundItemPacket {
	var out []d2netpacket.GroundItemPacket

	for _, pk := range c.got {
		if pk.PacketType != d2netpackettype.GroundItem {
			continue
		}

		if p, _ := d2netpacket.UnmarshalGroundItem(pk.PacketData); p.Op == op {
			out = append(out, p)
		}
	}

	return out
}

func groundSend(t *testing.T, g *GameServer, c *fakeConn, op, ref string, e d2netpacket.GroundEntry) {
	t.Helper()

	pk, _ := d2netpacket.CreateGroundItemPacket(d2netpacket.GroundItemPacket{Op: op, Ref: ref, Entries: []d2netpacket.GroundEntry{e}})
	if err := g.onGroundItem(c, pk); err != nil {
		t.Fatal(err)
	}
}

func dropped(t *testing.T, g *GameServer, c *fakeConn, e d2netpacket.GroundEntry) string {
	t.Helper()

	c.got = nil
	groundSend(t, g, c, d2netpacket.GroundDrop, "ref", e)

	pl := groundOps(c, d2netpacket.GroundPlaced)
	if len(pl) != 1 || pl[0].Entries[0].ID == "" || pl[0].Ref != "ref" {
		t.Fatalf("sender got %+v, want one placed with an id and its ref", pl)
	}

	return pl[0].Entries[0].ID
}

func sword() d2netpacket.GroundEntry {
	return d2netpacket.GroundEntry{Key: "k1", Level: 1, X: 10, Y: 12, Item: &d2hero.StoredItem{Code: "ssd"}}
}

// A drop is broadcast with the item data to the heroes of the level only.
func TestGroundDropBroadcastAndVisibility(t *testing.T) {
	g, a, b := testServer()
	g.socialArea("b", 2) // Bob walked into another level

	a.got, b.got = nil, nil
	id := dropped(t, g, a, sword())

	if len(groundOps(b, d2netpacket.GroundPlaced)) != 0 {
		t.Fatal("a hero in another level must not see the drop")
	}

	g.socialArea("b", 1) // Bob arrives: the server tells him what lies there

	syn := groundOps(b, d2netpacket.GroundSync)
	if len(syn) != 1 || syn[0].Entries[0].ID != id || syn[0].Entries[0].Item.Code != "ssd" {
		t.Fatalf("sync = %+v", syn)
	}
}

// Two heroes pick up the same item: exactly one is approved, the other is denied, and the item is gone.
func TestGroundPickupRace(t *testing.T) {
	g, a, b := testServer()
	g.socialArea("a", 1)
	g.socialArea("b", 1)

	id := dropped(t, g, a, sword())
	a.got, b.got = nil, nil

	groundSend(t, g, a, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})
	groundSend(t, g, b, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})

	for _, c := range []*fakeConn{a, b} {
		rm := groundOps(c, d2netpacket.GroundRemove)
		if len(rm) != 1 || rm[0].Picker != "a" {
			t.Fatalf("%s: remove = %+v, want one with picker a", c.id, rm)
		}
	}

	if len(groundOps(a, d2netpacket.GroundDeny)) != 0 || len(groundOps(b, d2netpacket.GroundDeny)) != 1 {
		t.Fatal("only the second hero is denied")
	}

	groundSend(t, g, a, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})

	if len(groundOps(a, d2netpacket.GroundDeny)) != 1 {
		t.Fatal("a second pickup of the same id must be denied")
	}
}

// Many heroes racing from goroutines still yield a single winner.
func TestGroundPickupRaceConcurrent(t *testing.T) {
	g, a, b := testServer()
	id := dropped(t, g, a, sword())
	a.got, b.got = nil, nil

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		wins int
	)

	for i := 0; i < 16; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, ok := g.soc.ground.take(id); ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if wins != 1 {
		t.Fatalf("%d winners", wins)
	}
}

// Every client reports the loot its own simulation made: the server keeps one item and binds the others.
func TestGroundDropDedupAndBind(t *testing.T) {
	g, a, b := testServer()
	id := dropped(t, g, a, sword())

	b.got = nil
	groundSend(t, g, b, d2netpacket.GroundDrop, "bref", sword())

	bind := groundOps(b, d2netpacket.GroundBind)
	if len(bind) != 1 || bind[0].Ref != "bref" || bind[0].Entries[0].ID != id {
		t.Fatalf("bind = %+v", bind)
	}

	if len(groundOps(a, d2netpacket.GroundPlaced)) != 1 || len(g.soc.ground.items) != 1 {
		t.Fatal("the duplicate drop must not create a second item")
	}
}

// A gold pile goes to the first picker, shared with the party by the server; a second request gets nothing.
func TestGroundGoldSharedOnce(t *testing.T) {
	g, a, b := testServer()
	party(t, g, a, d2netpacket.PartyInvite, "b")
	party(t, g, b, d2netpacket.PartyAccept, "")

	id := dropped(t, g, a, d2netpacket.GroundEntry{Key: "gold", Level: 1, Gold: 100})
	a.got, b.got = nil, nil

	groundSend(t, g, b, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})
	groundSend(t, g, a, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})

	award := func(c *fakeConn) (sum, n int) {
		for _, pk := range c.got {
			if pk.PacketType == d2netpackettype.PartyGold {
				x, _ := d2netpacket.UnmarshalPartyGold(pk.PacketData)
				sum += x.Add
				n++
			}
		}

		return
	}

	sa, na := award(a)
	sb, nb := award(b)

	if na != 1 || nb != 1 || sa+sb != 100 {
		t.Fatalf("awards a=%d/%d b=%d/%d, want one each summing to 100", sa, na, sb, nb)
	}

	if len(groundOps(a, d2netpacket.GroundDeny)) != 1 {
		t.Fatal("the late hero is denied")
	}
}

func TestGroundForgedAndExpire(t *testing.T) {
	g, a, _ := testServer()
	a.got = nil
	groundSend(t, g, a, d2netpacket.GroundDrop, "r", d2netpacket.GroundEntry{Level: 1, Gold: -5})
	groundSend(t, g, a, d2netpacket.GroundDrop, "r", d2netpacket.GroundEntry{Level: 1})

	if len(g.soc.ground.items) != 0 {
		t.Fatal("empty or negative piles must be refused")
	}

	id := dropped(t, g, a, sword())
	groundSend(t, g, a, d2netpacket.GroundExpire, "", d2netpacket.GroundEntry{ID: id})
	groundSend(t, g, a, d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: id})

	if len(groundOps(a, d2netpacket.GroundDeny)) != 1 {
		t.Fatal("an expired item cannot be picked up")
	}

	bad := d2netpacket.NetPacket{PacketType: d2netpackettype.GroundItem, PacketData: []byte("{")}
	if g.onGroundItem(a, bad) == nil {
		t.Fatal("garbage must be an error, not a panic")
	}
}

// Single-roll loot: of two clients claiming the same source only the first is granted the roll; an empty key is
// never granted; claims of an emptied level are forgotten, and the roller's drop reaches the other hero.
func TestGroundLootClaim(t *testing.T) {
	g, a, b := testServer()
	g.socialArea("a", 1)
	g.socialArea("b", 1)

	claim := func(c *fakeConn, key string) string {
		c.got = nil
		groundSend(t, g, c, d2netpacket.GroundClaim, "", d2netpacket.GroundEntry{Key: key, Level: 1})

		for _, op := range []string{d2netpacket.GroundGrant, d2netpacket.GroundLost} {
			if l := groundOps(c, op); len(l) == 1 && l[0].Entries[0].Key == key {
				return op
			}
		}

		return ""
	}

	if got := claim(a, "m|1|fallen|7"); got != d2netpacket.GroundGrant {
		t.Fatalf("first claim = %q", got)
	}

	if got := claim(b, "m|1|fallen|7"); got != d2netpacket.GroundLost {
		t.Fatalf("second claim = %q", got)
	}

	if got := claim(a, "m|1|fallen|7"); got != d2netpacket.GroundLost {
		t.Fatalf("repeat claim = %q", got)
	}

	if got := claim(b, ""); got != d2netpacket.GroundLost {
		t.Fatalf("empty key = %q", got)
	}

	// the granted roller reports the loot: the other hero is shown it
	b.got = nil
	dropped(t, g, a, sword())

	if len(groundOps(b, d2netpacket.GroundPlaced)) != 1 {
		t.Fatal("the other hero must be shown the single roll")
	}

	// both leave level 1: its claims go, the monsters respawn on the next visit
	g.socialArea("a", 2)
	g.socialArea("b", 2)

	if got := claim(b, "m|1|fallen|7"); got != d2netpacket.GroundGrant {
		t.Fatalf("claim after the level emptied = %q", got)
	}
}
