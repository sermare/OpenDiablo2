package d2gsnet

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

// wire sends packets through a real blob (compress + prefix) and back.
func wire(t *testing.T, pkts [][]byte) []byte {
	t.Helper()

	blob, err := d2gs.EncodeStream(pkts...)
	if err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(bytes.NewReader(blob))

	var plain []byte

	for {
		part, err := d2gs.ReadBlob(r)
		if err == io.EOF {
			return plain
		}

		if err != nil {
			t.Fatal(err)
		}

		plain = append(plain, part...)
	}
}

func newPair() (*ClientSide, *ServerSide) {
	return &ClientSide{IDs: NewIDs()}, &ServerSide{IDs: NewIDs(), Info: GameInfo{MapSeed: 0x12345678, Difficulty: 2}}
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.11 }

func TestJoinHandshake(t *testing.T) {
	c, s := newPair()

	state := &d2hero.HeroState{HeroName: "Maricon", HeroType: d2enum.HeroSorceress, Stats: &d2hero.HeroStatsState{Level: 31}}

	req, err := d2netpacket.CreatePlayerConnectionRequestPacket("uuid-a", state)
	if err != nil {
		t.Fatal(err)
	}

	pkts, err := c.Encode(req)
	if err != nil || len(pkts) < 2 || pkts[0][0] != d2gs.CtlJoinGame {
		t.Fatalf("encode: %v %d", err, len(pkts))
	}

	got, ignored, err := s.Decode(wire(t, pkts))
	if err != nil || ignored != 0 || len(got) != 1 || got[0].PacketType != d2netpackettype.PlayerConnectionRequest {
		t.Fatalf("decode: %v ignored=%d %v", err, ignored, got)
	}

	if s.PlayerID != "uuid-a" || s.Joined().Name != "Maricon" || s.Joined().Level != 31 {
		t.Errorf("session %+v %+v", s.PlayerID, s.Joined())
	}

	back, _ := d2netpacket.UnmarshalPlayerConnectionRequest(got[0].PacketData)
	if back.PlayerState.HeroName != "Maricon" {
		t.Error("hero state did not survive the tunnel")
	}

	// server answer: server info, map, player (the sequence of OnClientConnected)
	var seeds []uint32

	c.OnInfo = func(i GameInfo) { seeds = append(seeds, i.MapSeed) }

	usi, _ := d2netpacket.CreateUpdateServerInfoPacket(0x12345678, "uuid-a")
	gm, _ := d2netpacket.CreateGenerateMapPacket(d2enum.RegionAct1Town)
	ap, _ := d2netpacket.CreateAddPlayerPacket("uuid-a", "Maricon", 53, 63, d2enum.HeroSorceress,
		&d2hero.HeroStatsState{Level: 31}, map[int]*d2hero.HeroSkill{}, d2inventory.CharacterEquipment{}, 0, 0, 100, nil, d2enum.DifficultyHell)

	var all [][]byte

	for _, np := range []d2netpacket.NetPacket{usi, gm, ap} {
		p, err := s.Encode(np)
		if err != nil {
			t.Fatal(err)
		}

		for _, b := range p {
			if err := d2gs.Validate(d2gs.ServerToClient, b); err != nil {
				t.Fatalf("id %#x: %v", b[0], err)
			}
		}

		all = append(all, p...)
	}

	in, _, err := c.Decode(wire(t, all))
	if err != nil {
		t.Fatal(err)
	}

	if len(in) != 3 || in[0].PacketType != d2netpackettype.UpdateServerInfo || in[1].PacketType != d2netpackettype.GenerateMap ||
		in[2].PacketType != d2netpackettype.AddPlayer {
		t.Fatalf("packets: %+v", in)
	}

	if len(seeds) != 1 || seeds[0] != 0x12345678 || c.Info.Difficulty != 2 {
		t.Errorf("game info seeds=%v info=%+v", seeds, c.Info)
	}

	if u, ok := c.IDs.Unit("uuid-a"); !ok || u == 0 {
		t.Errorf("unit id not learned: %v %v", u, ok)
	}

	si, _ := d2netpacket.UnmarshalUpdateServerInfo(in[0].PacketData)
	if si.PlayerID != "uuid-a" || si.Seed != 0x12345678 {
		t.Errorf("server info %+v", si)
	}

	ap2, _ := d2netpacket.UnmarshalAddPlayer(in[2].PacketData)
	if ap2.Name != "Maricon" || ap2.X != 53 || ap2.Difficulty != d2enum.DifficultyHell {
		t.Errorf("add player %+v", ap2)
	}
}

func TestMoveCastChatLeave(t *testing.T) {
	c, s := newPair()
	s.PlayerID = "uuid-a"
	s.Pos = func() (float64, float64) { return 10, 20 }

	// walk
	mv, _ := d2netpacket.CreateMovePlayerPacket("uuid-a", 10, 20, 17.4, 21.2)

	p, err := c.Encode(mv)
	if err != nil || len(p) != 1 || p[0][0] != d2gs.C2SWalkToLocation || len(p[0]) != 5 {
		t.Fatalf("walk encode: %v %x", err, p)
	}

	got, _, err := s.Decode(wire(t, p))
	if err != nil || len(got) != 1 {
		t.Fatalf("walk decode: %v %v", err, got)
	}

	m, _ := d2netpacket.UnmarshalMovePlayer(got[0].PacketData)
	if m.PlayerID != "uuid-a" || !near(m.DestX, 17.4) || !near(m.DestY, 21.2) || m.StartX != 10 || m.StartY != 20 {
		t.Errorf("move %+v", m)
	}

	// the server broadcasts it, the other client sees the player move
	other := &ClientSide{IDs: NewIDs()}
	other.IDs.Set("uuid-a", s.IDs.Assign("uuid-a"))

	bp, err := s.Encode(got[0])
	if err != nil || bp[0][0] != d2gs.S2CPlayerMove {
		t.Fatalf("broadcast: %v", err)
	}

	seen, _, err := other.Decode(wire(t, bp))
	if err != nil || len(seen) != 1 {
		t.Fatalf("broadcast decode: %v %v", err, seen)
	}

	sm, _ := d2netpacket.UnmarshalMovePlayer(seen[0].PacketData)
	if sm.PlayerID != "uuid-a" || !near(sm.DestX, 17.4) || !near(sm.StartX, 10) {
		t.Errorf("seen move %+v", sm)
	}

	// cast: first cast selects the skill, the second does not
	cast, _ := d2netpacket.CreateCastPacket("uuid-a", 36, 12, 13)

	p, _ = c.Encode(cast)
	if len(p) != 2 || p[0][0] != d2gs.C2SSelectSkill || p[1][0] != d2gs.C2SCastLeftLocation {
		t.Fatalf("first cast packets %x", p)
	}

	p2, _ := c.Encode(cast)
	if len(p2) != 1 {
		t.Fatalf("second cast should not reselect: %x", p2)
	}

	got, _, err = s.Decode(wire(t, append(p, p2...)))
	if err != nil || len(got) != 2 {
		t.Fatalf("cast decode: %v %d", err, len(got))
	}

	cp, _ := d2netpacket.UnmarshalCast(got[1].PacketData)
	if cp.SkillID != 36 || cp.SourceEntityID != "uuid-a" || !near(cp.TargetX, 12) {
		t.Errorf("cast %+v", cp)
	}

	// chat
	chat, _ := d2netpacket.CreateChatPacket("", "", "hello world")

	p, _ = c.Encode(chat)

	got, _, err = s.Decode(wire(t, p))
	if err != nil || len(got) != 1 || got[0].PacketType != d2netpackettype.Chat {
		t.Fatalf("chat: %v %v", err, got)
	}

	cc, _ := d2netpacket.UnmarshalChat(got[0].PacketData)
	if cc.Text != "hello world" || cc.PlayerID != "uuid-a" {
		t.Errorf("chat %+v", cc)
	}

	cc.Name = "Maricon"
	named, _ := d2netpacket.CreateChatPacket(cc.PlayerID, cc.Name, cc.Text)
	bp, _ = s.Encode(named)

	seen, _, err = other.Decode(wire(t, bp))
	if err != nil || len(seen) != 1 {
		t.Fatalf("chat relay: %v", err)
	}

	rc, _ := d2netpacket.UnmarshalChat(seen[0].PacketData)
	if rc.Name != "Maricon" || rc.Text != "hello world" || rc.PlayerID != "uuid-a" {
		t.Errorf("relayed chat %+v", rc)
	}

	// leave
	dp, _ := d2netpacket.CreatePlayerDisconnectRequestPacket("uuid-a")

	p, _ = c.Encode(dp)

	got, _, err = s.Decode(wire(t, p))
	if err != nil || len(got) != 1 || got[0].PacketType != d2netpackettype.PlayerDisconnectionNotification {
		t.Fatalf("leave: %v %v", err, got)
	}

	bp, _ = s.Encode(got[0])
	if bp[0][0] != d2gs.S2CPlayerLeave {
		t.Fatalf("leave broadcast %x", bp)
	}

	seen, _, _ = other.Decode(wire(t, bp))
	if len(seen) != 1 || seen[0].PacketType != d2netpackettype.PlayerDisconnectionNotification {
		t.Fatalf("leave seen %v", seen)
	}

	if d, _ := d2netpacket.UnmarshalPlayerDisconnectionRequest(seen[0].PacketData); d.ID != "uuid-a" {
		t.Errorf("left %+v", d)
	}
}

func TestUnknownPacketsAreSkippedBySize(t *testing.T) {
	c, s := newPair()

	// server -> client: an item-stat packet (0x20, 10 bytes) and a PlayerStop-like
	// 0x0d (13 bytes) between two packets we use; they are skipped by size
	flags := d2gs.GameFlags{Difficulty: 1}.Marshal()
	unknown1 := make([]byte, 10)
	unknown1[0] = 0x20
	unknown2 := make([]byte, 13)
	unknown2[0] = 0x0d

	out, ignored, err := c.Decode(wire(t, [][]byte{unknown1, flags, unknown2, d2gs.LoadAct{Seed: 5}.Marshal()}))
	if err != nil || ignored != 23 || len(out) != 1 || c.Info.Difficulty != 1 || c.Info.MapSeed != 5 {
		t.Fatalf("server stream: err=%v ignored=%d out=%d info=%+v", err, ignored, len(out), c.Info)
	}

	// client -> server: an id with an unverified size ends the blob: the rest is ignored
	walk := d2gs.MoveToLocation{X: 5, Y: 5}.MarshalPacket()

	got, ignored, err := s.Decode(wire(t, [][]byte{walk, {0x02, 1, 2, 3, 4, 5, 6, 7, 8}, walk}))
	if err != nil || len(got) != 1 || ignored != 14 {
		t.Fatalf("client stream: %v got=%d ignored=%d", err, len(got), ignored)
	}

	// a known packet we do not use (UseItem, 13 bytes) is framed and skipped
	got, ignored, err = s.Decode(wire(t, [][]byte{d2gs.UseItem{ItemID: 1}.MarshalPacket(), walk}))
	if err != nil || len(got) != 1 || ignored != 13 {
		t.Fatalf("known unused: %v got=%d ignored=%d", err, len(got), ignored)
	}
}

func TestBigStateSurvivesTheTunnel(t *testing.T) {
	c, s := newPair()

	// a hero state much larger than one chunk
	skills := map[int]*d2hero.HeroSkill{}
	for i := 0; i < 300; i++ {
		skills[i] = &d2hero.HeroSkill{SkillPoints: i}
	}

	state := &d2hero.HeroState{HeroName: "Big", Stats: &d2hero.HeroStatsState{Level: 2}, Skills: skills}
	req, _ := d2netpacket.CreatePlayerConnectionRequestPacket("big", state)

	p, err := c.Encode(req)
	if err != nil || len(p) < 5 {
		t.Fatalf("expected several chunks, got %d (%v)", len(p), err)
	}

	got, _, err := s.Decode(wire(t, p))
	if err != nil || len(got) != 1 {
		t.Fatalf("decode: %v", err)
	}

	back, err := d2netpacket.UnmarshalPlayerConnectionRequest(got[0].PacketData)
	if err != nil || len(back.PlayerState.Skills) != 300 {
		t.Fatalf("skills lost: %v", err)
	}
}
