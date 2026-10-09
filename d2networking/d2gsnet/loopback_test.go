package d2gsnet

import (
	"bufio"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// loopServer is a minimal game server that speaks the d2gs wire (blobs over a
// net.Conn) through ServerSide, relaying like the engine's GameServer does. It
// has no game window, assets or engine: it only exercises the translation
// layer and the packet layouts end to end.
type loopServer struct {
	t     *testing.T
	ids   *IDs
	mu    sync.Mutex
	conns map[string]*loopConn
	order []string
}

type loopConn struct {
	conn net.Conn
	side *ServerSide
	pos  [2]float64
	add  d2netpacket.NetPacket
}

func (s *loopServer) send(c *loopConn, np d2netpacket.NetPacket) {
	pkts, err := c.side.Encode(np)
	if err != nil {
		s.t.Errorf("server encode %s: %v", np.PacketType, err)
		return
	}

	for _, p := range pkts {
		if err := d2gs.Validate(d2gs.ServerToClient, p); err != nil {
			s.t.Errorf("server sent invalid id %#x: %v", p[0], err)
		}
	}

	blob, err := d2gs.EncodeStream(pkts...)
	if err == nil {
		_, err = c.conn.Write(blob)
	}

	if err != nil {
		s.t.Errorf("server write: %v", err)
	}
}

func (s *loopServer) broadcast(np d2netpacket.NetPacket, except string) {
	for _, id := range s.order {
		if id != except {
			s.send(s.conns[id], np)
		}
	}
}

func (s *loopServer) serve(conn net.Conn, done *sync.WaitGroup) {
	defer done.Done()

	c := &loopConn{conn: conn, side: &ServerSide{IDs: s.ids, Info: GameInfo{MapSeed: 77}}}
	c.side.Pos = func() (float64, float64) { return c.pos[0], c.pos[1] }

	r := bufio.NewReader(conn)

	for {
		plain, err := d2gs.ReadBlob(r)
		if err != nil {
			return
		}

		nps, _, err := c.side.Decode(plain)
		if err != nil {
			s.t.Errorf("server decode: %v", err)
			return
		}

		for _, np := range nps {
			s.handle(c, np)
		}
	}
}

func (s *loopServer) handle(c *loopConn, np d2netpacket.NetPacket) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := c.side.PlayerID

	switch np.PacketType { //nolint:exhaustive // only what the scenario sends
	case d2netpackettype.PlayerConnectionRequest:
		req, _ := d2netpacket.UnmarshalPlayerConnectionRequest(np.PacketData)
		c.pos = [2]float64{50, 60}
		s.conns[id] = c
		s.order = append(s.order, id)

		usi, _ := d2netpacket.CreateUpdateServerInfoPacket(77, id)
		gm, _ := d2netpacket.CreateGenerateMapPacket(d2enum.RegionAct1Town)
		s.send(c, usi)
		s.send(c, gm)

		c.add, _ = d2netpacket.CreateAddPlayerPacket(id, req.PlayerState.HeroName, 50, 60, req.PlayerState.HeroType,
			req.PlayerState.Stats, map[int]*d2hero.HeroSkill{}, d2inventory.CharacterEquipment{}, 0, 0, 0, nil, d2enum.DifficultyNormal)

		for _, other := range s.order { // existing players to the newcomer
			if other != id {
				s.send(c, s.conns[other].add)
			}
		}

		s.broadcast(c.add, "") // the newcomer to everybody (itself included)
	case d2netpackettype.MovePlayer:
		mp, _ := d2netpacket.UnmarshalMovePlayer(np.PacketData)
		c.pos = [2]float64{mp.DestX, mp.DestY}

		s.broadcast(np, id)
	case d2netpackettype.CastSkill:
		s.broadcast(np, id)
	case d2netpackettype.Chat:
		ch, _ := d2netpacket.UnmarshalChat(np.PacketData)
		named, _ := d2netpacket.CreateChatPacket(id, id+"-name", ch.Text)

		s.broadcast(named, id)
	case d2netpackettype.PlayerDisconnectionNotification:
		delete(s.conns, id)

		for i, o := range s.order {
			if o == id {
				s.order = append(s.order[:i], s.order[i+1:]...)
				break
			}
		}

		s.broadcast(np, id)
	}
}

type loopClient struct {
	t    *testing.T
	conn net.Conn
	side *ClientSide
	in   chan d2netpacket.NetPacket
}

func newLoopClient(t *testing.T, srv *loopServer, done *sync.WaitGroup) *loopClient {
	t.Helper()

	cc, sc := net.Pipe()

	done.Add(1)

	go srv.serve(sc, done)

	c := &loopClient{t: t, conn: cc, side: &ClientSide{IDs: NewIDs()}, in: make(chan d2netpacket.NetPacket, 64)}

	go func() {
		r := bufio.NewReader(cc)

		for {
			plain, err := d2gs.ReadBlob(r)
			if err != nil {
				close(c.in)
				return
			}

			nps, _, err := c.side.Decode(plain)
			if err != nil {
				t.Errorf("client decode: %v", err)
				return
			}

			for _, np := range nps {
				c.in <- np
			}
		}
	}()

	return c
}

func (c *loopClient) send(np d2netpacket.NetPacket, err error) {
	c.t.Helper()

	if err != nil {
		c.t.Fatal(err)
	}

	pkts, err := c.side.Encode(np)
	if err != nil {
		c.t.Fatal(err)
	}

	blob, err := d2gs.EncodeStream(pkts...)
	if err != nil {
		c.t.Fatal(err)
	}

	if _, err := c.conn.Write(blob); err != nil {
		c.t.Fatal(err)
	}
}

// next returns the next packet of the wanted type, skipping others.
func (c *loopClient) next(want d2netpackettype.NetPacketType) d2netpacket.NetPacket {
	c.t.Helper()

	timeout := time.After(5 * time.Second)

	for {
		select {
		case np, ok := <-c.in:
			if !ok {
				c.t.Fatalf("connection closed waiting for %s", want)
			}

			if np.PacketType == want {
				return np
			}
		case <-timeout:
			c.t.Fatalf("timeout waiting for %s", want)
		}
	}
}

// TestLoopbackJoinSeeMoveCastChatLeave runs two fake clients against a server
// over in-memory connections: join, see each other (with level, from the
// tunnelled AddPlayer, since the real 0x59 carries only a position), walk,
// cast, chat and leave, all through the real d2gs wire format.
func TestLoopbackJoinSeeMoveCastChatLeave(t *testing.T) {
	srv := &loopServer{t: t, ids: NewIDs(), conns: map[string]*loopConn{}}

	var done sync.WaitGroup

	a := newLoopClient(t, srv, &done)
	b := newLoopClient(t, srv, &done)

	join := func(c *loopClient, id, name string, hero d2enum.Hero, level int) {
		st := &d2hero.HeroState{HeroName: name, HeroType: hero, Stats: &d2hero.HeroStatsState{Level: level}}
		c.send(d2netpacket.CreatePlayerConnectionRequestPacket(id, st))
	}

	join(a, "a", "Ann", d2enum.HeroSorceress, 31)
	a.next(d2netpackettype.UpdateServerInfo)
	a.next(d2netpackettype.GenerateMap)

	if got, _ := d2netpacket.UnmarshalAddPlayer(a.next(d2netpackettype.AddPlayer).PacketData); got.Name != "Ann" || got.Stats.Level != 31 {
		t.Fatalf("a does not see itself: %+v", got)
	}

	join(b, "b", "Bob", d2enum.HeroBarbarian, 12)
	b.next(d2netpackettype.UpdateServerInfo)
	b.next(d2netpackettype.GenerateMap)

	seenA, _ := d2netpacket.UnmarshalAddPlayer(b.next(d2netpackettype.AddPlayer).PacketData) // Ann, existing
	seenB, _ := d2netpacket.UnmarshalAddPlayer(b.next(d2netpackettype.AddPlayer).PacketData) // Bob himself
	joinB, _ := d2netpacket.UnmarshalAddPlayer(a.next(d2netpackettype.AddPlayer).PacketData) // Bob announced to Ann

	if seenA.ID != "a" || seenA.Name != "Ann" || seenA.Stats.Level != 31 || seenA.X != 50 || seenA.Y != 60 {
		t.Errorf("b sees a as %+v", seenA)
	}

	if seenB.ID != "b" || joinB.ID != "b" || joinB.Name != "Bob" || joinB.Stats.Level != 12 || joinB.HeroType != d2enum.HeroBarbarian {
		t.Errorf("a sees b as %+v / %+v", seenB, joinB)
	}

	ua, oka := b.side.IDs.Unit("a")
	ub, okb := a.side.IDs.Unit("b")

	if !oka || !okb || ua == ub {
		t.Errorf("unit ids not learned per peer: a=%d/%v b=%d/%v", ua, oka, ub, okb)
	}

	// walk: a -> b sees it
	a.side.PlayerID = "a"
	a.send(d2netpacket.CreateMovePlayerPacket("a", 50, 60, 55.5, 61))

	mv, _ := d2netpacket.UnmarshalMovePlayer(b.next(d2netpackettype.MovePlayer).PacketData)
	if mv.PlayerID != "a" || !near(mv.DestX, 55.5) || !near(mv.DestY, 61) {
		t.Errorf("b sees move %+v", mv)
	}

	// cast
	a.send(d2netpacket.CreateCastPacket("a", 36, 56, 62))

	cp, _ := d2netpacket.UnmarshalCast(b.next(d2netpackettype.CastSkill).PacketData)
	if cp.SourceEntityID != "a" || cp.SkillID != 36 || !near(cp.TargetX, 56) {
		t.Errorf("b sees cast %+v", cp)
	}

	// chat
	a.send(d2netpacket.CreateChatPacket("", "", "hello bob"))

	ch, _ := d2netpacket.UnmarshalChat(b.next(d2netpackettype.Chat).PacketData)
	if ch.Text != "hello bob" || ch.PlayerID != "a" {
		t.Errorf("b sees chat %+v", ch)
	}

	// leave: b is told
	a.send(d2netpacket.CreatePlayerDisconnectRequestPacket("a"))

	gone, _ := d2netpacket.UnmarshalPlayerDisconnectionRequest(b.next(d2netpackettype.PlayerDisconnectionNotification).PacketData)
	if gone.ID != "a" {
		t.Errorf("b sees leave of %+v", gone)
	}

	_ = a.conn.Close()
	_ = b.conn.Close()

	done.Wait()
}
