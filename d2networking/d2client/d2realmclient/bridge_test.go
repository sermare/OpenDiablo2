package d2realmclient

import (
	"sync"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

// engineSink is a stand-in for the game client: it records the packets.
type engineSink struct {
	mu   sync.Mutex
	pkts []d2netpacket.NetPacket
}

func (s *engineSink) put(np d2netpacket.NetPacket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pkts = append(s.pkts, np)

	return nil
}

func (s *engineSink) find(t d2netpackettype.NetPacketType) []d2netpacket.NetPacket {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []d2netpacket.NetPacket

	for _, p := range s.pkts {
		if p.PacketType == t {
			out = append(out, p)
		}
	}

	return out
}

func (s *engineSink) wait(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for i := 0; i < 400; i++ {
		if cond() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for %s", what)
}

func newPlayer(a PlayerAdd) (d2netpacket.NetPacket, error) {
	return d2netpacket.CreateAddPlayerPacket(a.ID, a.Name, int(a.X*5), int(a.Y*5), d2enum.HeroSorceress, nil, nil,
		d2inventory.CharacterEquipment{}, 0, 0, 0, nil, d2enum.DifficultyNormal)
}

func testBridge(rules d2mp.Rules, name string) (*Bridge, *engineSink) {
	sink := &engineSink{}

	return New(Config{Rules: rules, Hero: Hero{Name: name, Class: d2s.Sorceress, Level: 1}, Sink: sink.put, NewPlayer: newPlayer}), sink
}

func TestNames(t *testing.T) {
	for in, want := range map[string]string{"Alice": "Alice", "Bob the Great": "BobtheGreat", "x": "Herox", "": "Hero",
		"a_b": "a_b", "-lead": "lead", "trail_": "trail", "Averyveryverylongname": "Averyveryverylo", "Zoë": "Zo"} {
		if got := CharName(in); got != want {
			t.Errorf("CharName(%q) = %q, want %q", in, got, want)
		}

		if !d2realm.ValidAccount(AccountName(in)) {
			t.Errorf("AccountName(%q) = %q is not valid", in, AccountName(in))
		}
	}
}

// Two bridges on one realm: they see each other walk, chat, and a kill of a
// dummy by one shows at the other; the replicas agree at the end.
func TestTwoBridgesShareAWorld(t *testing.T) {
	rules := d2mp.NewEngineRules(126.9, 117.5, 2)

	srv := d2realm.New(d2realm.Config{Store: d2realm.NewMemStore(), Rules: rules})

	addr, err := srv.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer srv.Close()

	host, hs := testBridge(rules, "Alice")
	join, js := testBridge(rules, "Bobby")

	if err = host.Connect(addr.String(), time.Second); err != nil {
		t.Fatal(err)
	}

	hj, err := host.Create(DefaultGame)
	if err != nil {
		t.Fatal(err)
	}

	host.SetSelf("host-id")

	if err = host.Announce(hj.Seed, d2enum.RegionAct1Town); err != nil {
		t.Fatal(err)
	}

	if err = host.Begin("host-id", PlayerAdd{Name: "Alice", Class: d2s.Sorceress, X: -1, Y: -1}); err != nil {
		t.Fatal(err)
	}

	host.Resume()

	defer host.Close()

	if err = join.Connect(addr.String(), time.Second); err != nil {
		t.Fatal(err)
	}

	jj, err := join.Join(DefaultGame, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	if jj.Seed != hj.Seed {
		t.Fatalf("seeds differ: %#x %#x", jj.Seed, hj.Seed)
	}

	join.SetSelf("join-id")

	if err = join.Announce(jj.Seed, d2enum.RegionAct1Town); err != nil {
		t.Fatal(err)
	}

	if err = join.Begin("join-id", PlayerAdd{Name: "Bobby", Class: d2s.Sorceress, X: -1, Y: -1}); err != nil {
		t.Fatal(err)
	}

	join.Resume()

	defer join.Close()

	// each side learns of the other hero and of the two dummies
	hs.wait(t, "host sees the joiner", func() bool { return len(hs.find(d2netpackettype.AddPlayer)) == 2 })
	js.wait(t, "joiner sees the host", func() bool { return len(js.find(d2netpackettype.AddPlayer)) == 2 })
	hs.wait(t, "host sees dummies", func() bool { return len(hs.find(d2netpackettype.RealmUnit)) == 2 })
	js.wait(t, "joiner sees dummies", func() bool { return len(js.find(d2netpackettype.RealmUnit)) == 2 })

	peer, _ := d2netpacket.UnmarshalAddPlayer(hs.find(d2netpackettype.AddPlayer)[1].PacketData)
	if peer.Name != "Bobby" || peer.ID != PeerID(jj.UnitID) {
		t.Fatalf("host got peer %+v", peer)
	}

	// the joiner walks; the host is told where to (after the interpolation delay)
	mv, _ := d2netpacket.CreateMovePlayerPacket("join-id", 127, 117.5, 135, 117.5)
	if err = join.Send(mv); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host sees the walk", func() bool { return len(hs.find(d2netpackettype.MovePlayer)) > 0 })

	m, _ := d2netpacket.UnmarshalMovePlayer(hs.find(d2netpackettype.MovePlayer)[0].PacketData)
	if m.PlayerID != PeerID(jj.UnitID) || m.DestX != 135 {
		t.Fatalf("walk seen as %+v", m)
	}

	// chat and a cast reach the other side with the sender's name
	chat, _ := d2netpacket.CreateChatPacket("", "", "hello there")
	if err = join.Send(chat); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host gets chat", func() bool {
		for _, p := range hs.find(d2netpackettype.Chat) {
			if c, _ := d2netpacket.UnmarshalChat(p.PacketData); c.Name == "Bobby" && c.Text == "hello there" {
				return true
			}
		}

		return false
	})

	cast, _ := d2netpacket.CreateCastPacket("join-id", int(d2mp.SkillFireBolt), 140, 117.5)
	if err = join.Send(cast); err != nil {
		t.Fatal(err)
	}

	hs.wait(t, "host sees the cast", func() bool { return len(hs.find(d2netpackettype.CastSkill)) > 0 })

	// the joiner kills a dummy; both see the death
	dummy, _ := d2netpacket.UnmarshalRealmUnit(js.find(d2netpackettype.RealmUnit)[0].PacketData)
	if dummy.Op != d2netpacket.RealmUnitSpawn || dummy.MaxHP == 0 {
		t.Fatalf("dummy %+v", dummy)
	}

	if err = join.Attack(dummy.UnitID); err != nil {
		t.Fatal(err)
	}

	deaths := func(s *engineSink) (n int) {
		for _, p := range s.find(d2netpackettype.RealmUnit) {
			if u, _ := d2netpacket.UnmarshalRealmUnit(p.PacketData); u.Op == d2netpacket.RealmUnitDeath && u.UnitID == dummy.UnitID {
				n++
			}
		}

		return n
	}

	hs.wait(t, "host sees the kill", func() bool { return deaths(hs) == 1 })
	js.wait(t, "joiner sees the kill", func() bool { return deaths(js) == 1 })

	// equal worlds
	time.Sleep(300 * time.Millisecond)

	if hd, jd := host.StableDigest(), join.StableDigest(); hd != jd || hd == 0 {
		t.Fatalf("digests differ: %x %x\nhost   %s\njoiner %s", hd, jd, host.Summary(), join.Summary())
	}

	// leaving is announced
	join.Close()
	hs.wait(t, "host sees the leave", func() bool { return len(hs.find(d2netpackettype.PlayerDisconnectionNotification)) == 1 })
}
