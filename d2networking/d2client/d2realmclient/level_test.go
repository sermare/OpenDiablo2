package d2realmclient

import (
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

// startPair connects a host and a joiner bridge on one realm, both in the town.
func startPair(t *testing.T) (host, join *Bridge, hs, js *engineSink) {
	t.Helper()

	rules := d2mp.NewEngineRules(126.9, 117.5, 2)
	srv := d2realm.New(d2realm.Config{Store: d2realm.NewMemStore(), Rules: rules})

	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { srv.Close() })

	host, hs = testBridge(rules, "Alice")
	join, js = testBridge(rules, "Bobby")

	must := func(err error) {
		t.Helper()

		if err != nil {
			t.Fatal(err)
		}
	}

	must(host.Connect(addr.String(), time.Second))

	hj, err := host.Create(DefaultGame)
	must(err)

	host.SetSelf("host-id")
	must(host.Announce(hj.Seed, d2enum.RegionAct1Town))
	must(host.Begin("host-id", PlayerAdd{Name: "Alice", Class: d2s.Sorceress, X: -1, Y: -1}))
	host.Resume()
	t.Cleanup(host.Close)

	must(join.Connect(addr.String(), time.Second))

	jj, err := join.Join(DefaultGame, time.Second)
	must(err)

	join.SetSelf("join-id")
	must(join.Announce(jj.Seed, d2enum.RegionAct1Town))
	must(join.Begin("join-id", PlayerAdd{Name: "Bobby", Class: d2s.Sorceress, X: -1, Y: -1}))
	join.Resume()
	t.Cleanup(join.Close)

	hs.wait(t, "host sees the joiner", func() bool { return len(hs.find(d2netpackettype.AddPlayer)) == 2 })
	js.wait(t, "joiner sees the host", func() bool { return len(js.find(d2netpackettype.AddPlayer)) == 2 })
	hs.wait(t, "host sees dummies", func() bool { return len(hs.find(d2netpackettype.RealmUnit)) == 2 })

	return host, join, hs, js
}

// Before the local hero's level is known a peer counts as present.
func TestSameLevelUnknownArea(t *testing.T) {
	b := New(Config{})
	b.levels = map[uint32]uint16{7: 2}

	if !b.SameLevel(PeerID(7)) {
		t.Error("area unknown: peer must count as present")
	}

	b.area = 1
	if b.SameLevel(PeerID(7)) {
		t.Error("area 1 vs 2 must differ")
	}

	b.area = 2
	if !b.SameLevel(PeerID(7)) || b.SameLevel("host-id") {
		t.Error("same level / non-peer handling")
	}
}

func lastRoster(t *testing.T, s *engineSink) d2netpacket.RosterUpdatePacket {
	t.Helper()

	all := s.find(d2netpackettype.RosterUpdate)
	if len(all) == 0 {
		t.Fatal("no roster update")
	}

	r, err := d2netpacket.UnmarshalRosterUpdate(all[len(all)-1].PacketData)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

// A hero that walks into another level disappears from the other client's
// engine (its player is removed) and comes back when it returns, while the
// roster keeps listing it with its level.
func TestBridgeHidesHeroInOtherLevel(t *testing.T) {
	host, join, hs, js := startPair(t)

	ch, err := d2netpacket.CreateChangeLevelPacket("join-id", 2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err = join.Send(ch); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host drops the hero that left", func() bool {
		return len(hs.find(d2netpackettype.PlayerDisconnectionNotification)) == 1
	})

	// the joiner sees the host go: it is in the other level now
	js.wait(t, "joiner drops the hero it left behind", func() bool {
		return len(js.find(d2netpackettype.PlayerDisconnectionNotification)) == 1
	})

	// the roster is global and knows both levels
	hs.wait(t, "roster lists the hero in level 2", func() bool {
		if len(hs.find(d2netpackettype.RosterUpdate)) == 0 {
			return false
		}

		for _, p := range lastRoster(t, hs).Roster.Players {
			if p.ID == PeerID(join.joined.UnitID) && p.Area == 2 {
				return true
			}
		}

		return false
	})

	js.wait(t, "joiner roster", func() bool { return len(js.find(d2netpackettype.RosterUpdate)) > 0 })

	if r := lastRoster(t, js).Roster; len(r.Players) != 2 {
		t.Errorf("joiner roster: %+v", r.Players)
	}

	// the two worlds are no longer one: the joiner has no town dummies in view
	if host.StableDigest() == join.StableDigest() {
		t.Error("digests equal although the heroes are in different levels")
	}

	// back to the town: announced again
	back, _ := d2netpacket.CreateChangeLevelPacket("join-id", 1, 0, 0)
	if err = join.Send(back); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host sees the hero come back", func() bool { return len(hs.find(d2netpackettype.AddPlayer)) == 3 })
	js.wait(t, "joiner sees the host again", func() bool { return len(js.find(d2netpackettype.AddPlayer)) == 3 })

	time.Sleep(300 * time.Millisecond)

	if hd, jd := host.StableDigest(), join.StableDigest(); hd != jd {
		t.Errorf("digests differ after the return: %x %x", hd, jd)
	}
}
