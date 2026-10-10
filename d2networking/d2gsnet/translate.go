package d2gsnet

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket/d2netpackettype"
)

const subtilesPerTile = 5

// IDs maps the engine's string player ids to the u32 unit ids of the protocol.
type IDs struct {
	mu     sync.Mutex
	next   uint32
	byUUID map[string]uint32
	byUnit map[uint32]string
}

// NewIDs returns an empty map; unit ids start at 1.
func NewIDs() *IDs {
	return &IDs{next: 1, byUUID: map[string]uint32{}, byUnit: map[uint32]string{}}
}

// Assign returns the unit id of a player, allocating one if needed.
func (i *IDs) Assign(uuid string) uint32 {
	i.mu.Lock()
	defer i.mu.Unlock()

	if u, ok := i.byUUID[uuid]; ok {
		return u
	}

	u := i.next
	i.next++
	i.byUUID[uuid], i.byUnit[u] = u, uuid

	return u
}

// Set records a mapping learned from the peer.
func (i *IDs) Set(uuid string, unit uint32) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.byUUID[uuid], i.byUnit[unit] = unit, uuid
}

// Unit returns the unit id of a player id.
func (i *IDs) Unit(uuid string) (uint32, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	u, ok := i.byUUID[uuid]

	return u, ok
}

// UUID returns the player id of a unit id.
func (i *IDs) UUID(unit uint32) (string, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	u, ok := i.byUnit[unit]

	return u, ok
}

// GameInfo is what the host announces in GameFlags + LoadAct: the game seed
// (the .d2s map seed of the host) and the difficulty, so that every client
// builds the same levels.
type GameInfo struct {
	MapSeed    uint32
	Difficulty uint8
	Hardcore   bool
	Expansion  bool
}

func toSub(tile float64) uint16 {
	v := math.Round(tile * subtilesPerTile)
	if v < 0 {
		v = 0
	}

	if v > math.MaxUint16 {
		v = math.MaxUint16
	}

	return uint16(v)
}

func fromSub(s uint16) float64 { return float64(s) / subtilesPerTile }

func rawNP(t d2netpackettype.NetPacketType, data []byte) d2netpacket.NetPacket {
	return d2netpacket.NetPacket{PacketType: t, PacketData: json.RawMessage(data)}
}

func tunnelNP(id byte, np d2netpacket.NetPacket) [][]byte {
	return d2gs.Tunnel(id, byte(np.PacketType), np.PacketData)
}

// ServerSide translates for one client connection of the server.
type ServerSide struct {
	IDs  *IDs
	Info GameInfo
	// PlayerID is the engine id of the player on this connection (set when the
	// tunnelled PlayerConnectionRequest arrives).
	PlayerID string
	// Pos returns the player's current position in tiles (for movement packets).
	Pos func() (x, y float64)

	leftSkill, rightSkill uint32
	tun                   d2gs.TunnelAssembler
	joined                d2gs.JoinGame
}

// Joined returns the announced JoinGame of the connection.
func (s *ServerSide) Joined() d2gs.JoinGame { return s.joined }

// Encode translates an engine packet for the client into protocol packets.
func (s *ServerSide) Encode(np d2netpacket.NetPacket) ([][]byte, error) {
	switch np.PacketType {
	case d2netpackettype.UpdateServerInfo:
		flags := d2gs.GameFlags{Difficulty: s.Info.Difficulty, Hardcore: s.Info.Hardcore, Expansion: s.Info.Expansion}

		return append([][]byte{flags.Marshal()}, tunnelNP(d2gs.S2CMetaAE, np)...), nil
	case d2netpackettype.GenerateMap:
		p, err := d2netpacket.UnmarshalGenerateMap(np.PacketData)
		if err != nil {
			return nil, err
		}

		const rogueEncampment = 1

		return [][]byte{d2gs.LoadAct{Act: 0, Seed: s.Info.MapSeed, StartLevel: rogueEncampment, Aux: uint32(p.RegionType)}.Marshal()}, nil
	case d2netpackettype.AddPlayer:
		p, err := d2netpacket.UnmarshalAddPlayer(np.PacketData)
		if err != nil {
			return nil, err
		}

		// real 0x59 layout (verified): x, y at 22/24. Level and party are not in
		// this packet; they reach other OD2 peers via the tunnelled AddPlayer.
		in := d2gs.AssignPlayer{UnitID: s.IDs.Assign(p.ID), Class: uint8(p.HeroType), Name: p.Name,
			X: toSub(float64(p.X)), Y: toSub(float64(p.Y))}

		// The real 0x5b roster entry carries class, name, level and party id
		// (verified, verify-roster-packets.md); the rest of the hero state is the
		// documented OD2 extension in the tunnelled AddPlayer.
		level := 0
		if p.Stats != nil {
			level = p.Stats.Level
		}

		re := d2gs.RosterEntry{UnitID: in.UnitID, Class: in.Class, Name: p.Name, Level: clampU16(level), PartyID: d2gs.NoParty}

		return append([][]byte{in.Marshal(), re.Marshal()}, tunnelNP(d2gs.S2CMetaAE, np)...), nil
	case d2netpackettype.RosterUpdate:
		p, err := d2netpacket.UnmarshalRosterUpdate(np.PacketData)
		if err != nil {
			return nil, err
		}

		// Level and party id of every player go out as real 0x75 packets. The
		// roster's hostility, invitations, area and notices have no verified real
		// counterpart (0x8c flag bits and 0x8e are UNVERIFIED) and stay in the
		// tunnelled packet.
		var out [][]byte

		for _, in := range p.Roster.Players {
			party := d2gs.NoParty
			if in.Party > 0 && in.Party < int(d2gs.NoParty) {
				party = uint16(in.Party)
			}

			out = append(out, d2gs.RosterUpdate{UnitID: s.IDs.Assign(in.ID), PartyID: party, Level: clampU16(in.Level)}.Marshal())
		}

		return append(out, tunnelNP(d2gs.S2CMetaAE, np)...), nil
	case d2netpackettype.MovePlayer:
		p, err := d2netpacket.UnmarshalMovePlayer(np.PacketData)
		if err != nil {
			return nil, err
		}

		m := d2gs.PlayerMove{UnitID: s.IDs.Assign(p.PlayerID), TargetX: toSub(p.DestX), TargetY: toSub(p.DestY),
			CurX: toSub(p.StartX), CurY: toSub(p.StartY)}

		return [][]byte{m.Marshal()}, nil
	case d2netpackettype.CastSkill:
		p, err := d2netpacket.UnmarshalCast(np.PacketData)
		if err != nil {
			return nil, err
		}

		c := d2gs.UnitSkillOnLocation{UnitID: s.IDs.Assign(p.SourceEntityID), Skill: uint16(p.SkillID), Level: 1,
			X: toSub(p.TargetX), Y: toSub(p.TargetY)}

		return [][]byte{c.Marshal()}, nil
	case d2netpackettype.PlayerDisconnectionNotification:
		p, err := d2netpacket.UnmarshalPlayerDisconnectionRequest(np.PacketData)
		if err != nil {
			return nil, err
		}

		return [][]byte{d2gs.PlayerLeave{UnitID: s.IDs.Assign(p.ID)}.Marshal()}, nil
	case d2netpackettype.ServerClosed:
		return [][]byte{d2gs.GameExit()}, nil
	case d2netpackettype.Chat:
		p, err := d2netpacket.UnmarshalChat(np.PacketData)
		if err != nil {
			return nil, err
		}

		return [][]byte{d2gs.ChatMessage{Type: 1, UnitID: s.IDs.Assign(p.PlayerID), Name: p.Name, Text: p.Text}.Marshal()}, nil
	}

	return tunnelNP(d2gs.S2CMetaAE, np), nil
}

// Decode translates a decompressed client blob into engine packets. ignored is
// the number of bytes skipped because their packet id has no known size.
func (s *ServerSide) Decode(plain []byte) (out []d2netpacket.NetPacket, ignored int, err error) {
	pkts, ignored := d2gs.SplitClient(plain)

	for _, p := range pkts {
		switch p[0] {
		case d2gs.CtlJoinGame:
			if s.joined, err = d2gs.ParseJoinGame(p); err != nil {
				return out, ignored, err
			}
		case d2gs.CtlTunnel:
			typ, data, done, terr := s.tun.Add(p)
			if terr != nil {
				return out, ignored, terr
			}

			if !done {
				continue
			}

			np := rawNP(d2netpackettype.NetPacketType(typ), data)

			if np.PacketType == d2netpackettype.PlayerConnectionRequest {
				req, uerr := d2netpacket.UnmarshalPlayerConnectionRequest(data)
				if uerr != nil {
					return out, ignored, uerr
				}

				s.PlayerID = req.ID
			}

			out = append(out, np)
		case d2gs.C2SWalkToLocation, d2gs.C2SRunToLocation:
			m, derr := decodeMove(p)
			if derr != nil {
				return out, ignored, derr
			}

			var sx, sy float64
			if s.Pos != nil {
				sx, sy = s.Pos()
			}

			mp, merr := d2netpacket.CreateMovePlayerPacket(s.PlayerID, sx, sy, fromSub(m.X), fromSub(m.Y))
			if merr != nil {
				return out, ignored, merr
			}

			out = append(out, mp)
		case d2gs.C2SSelectSkill:
			m, derr := d2gs.DecodeClient(p)
			if derr != nil {
				return out, ignored, derr
			}

			sel := m.(*d2gs.SelectSkill)
			if sel.Right {
				s.rightSkill = sel.Skill
			} else {
				s.leftSkill = sel.Skill
			}
		case d2gs.C2SCastLeftLocation, d2gs.C2SCastRightLocation:
			m, derr := d2gs.DecodeClient(p)
			if derr != nil {
				return out, ignored, derr
			}

			c := m.(*d2gs.CastOnLocation)

			skill := s.leftSkill
			if c.Right {
				skill = s.rightSkill
			}

			cp, cerr := d2netpacket.CreateCastPacket(s.PlayerID, int(skill), fromSub(c.X), fromSub(c.Y))
			if cerr != nil {
				return out, ignored, cerr
			}

			out = append(out, cp)
		case d2gs.C2SChat:
			text, cerr := d2gs.ParseClientChat(p)
			if cerr != nil {
				return out, ignored, cerr
			}

			cp, perr := d2netpacket.CreateChatPacket(s.PlayerID, "", text)
			if perr != nil {
				return out, ignored, perr
			}

			out = append(out, cp)
		case d2gs.CtlLeaveGame:
			dp, derr := d2netpacket.CreatePlayerDisconnectRequestPacket(s.PlayerID)
			if derr != nil {
				return out, ignored, derr
			}

			out = append(out, dp)
		default:
			ignored += len(p) // framed by size, but nothing to do with it
		}
	}

	return out, ignored, nil
}

func decodeMove(p []byte) (d2gs.MoveToLocation, error) {
	m, err := d2gs.DecodeClient(p)
	if err != nil {
		return d2gs.MoveToLocation{}, err
	}

	return *m.(*d2gs.MoveToLocation), nil
}

// ClientSide translates for the client end of a connection.
type ClientSide struct {
	IDs *IDs
	// Info is filled from GameFlags and LoadAct; OnInfo (optional) is called
	// when the LoadAct arrives, before the GenerateMap packet is returned.
	Info   GameInfo
	OnInfo func(GameInfo)
	// PlayerID is the engine id of the local player.
	PlayerID string

	lastSkill  [2]uint32
	haveSkill  [2]bool
	tun        d2gs.TunnelAssembler
	lastInGame uint32

	peerMu sync.Mutex
	peers  map[uint32]PeerInfo
}

// Encode translates an engine packet for the server.
func (c *ClientSide) Encode(np d2netpacket.NetPacket) ([][]byte, error) {
	switch np.PacketType {
	case d2netpackettype.PlayerConnectionRequest:
		req, err := d2netpacket.UnmarshalPlayerConnectionRequest(np.PacketData)
		if err != nil {
			return nil, err
		}

		c.PlayerID = req.ID

		j := d2gs.JoinGame{}
		if st := req.PlayerState; st != nil {
			j.Name, j.Class, j.Difficulty = st.HeroName, uint8(st.HeroType), uint8(st.Difficulty)

			if st.Stats != nil {
				j.Level = uint8(st.Stats.Level)
			}
		}

		return append([][]byte{j.Marshal()}, tunnelNP(d2gs.CtlTunnel, np)...), nil
	case d2netpackettype.MovePlayer:
		p, err := d2netpacket.UnmarshalMovePlayer(np.PacketData)
		if err != nil {
			return nil, err
		}

		return [][]byte{d2gs.MoveToLocation{X: toSub(p.DestX), Y: toSub(p.DestY)}.MarshalPacket()}, nil
	case d2netpackettype.CastSkill:
		p, err := d2netpacket.UnmarshalCast(np.PacketData)
		if err != nil {
			return nil, err
		}

		var out [][]byte

		if !c.haveSkill[0] || c.lastSkill[0] != uint32(p.SkillID) {
			out = append(out, d2gs.SelectSkill{Skill: uint32(p.SkillID), ItemID: 0xffffffff}.MarshalPacket())
			c.lastSkill[0], c.haveSkill[0] = uint32(p.SkillID), true
		}

		return append(out, d2gs.CastOnLocation{X: toSub(p.TargetX), Y: toSub(p.TargetY)}.MarshalPacket()), nil
	case d2netpackettype.PlayerDisconnectionNotification:
		return [][]byte{d2gs.LeaveGame()}, nil
	case d2netpackettype.Chat:
		p, err := d2netpacket.UnmarshalChat(np.PacketData)
		if err != nil {
			return nil, err
		}

		return [][]byte{d2gs.ClientChat(p.Text)}, nil
	}

	return tunnelNP(d2gs.CtlTunnel, np), nil
}

// Decode translates a decompressed server blob into engine packets.
func (c *ClientSide) Decode(plain []byte) (out []d2netpacket.NetPacket, ignored int, err error) {
	pkts, ignored := d2gs.SplitServer(plain)

	for _, p := range pkts {
		switch p[0] {
		case d2gs.S2CGameFlags:
			f, perr := d2gs.ParseGameFlags(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.Info.Difficulty, c.Info.Hardcore, c.Info.Expansion = f.Difficulty, f.Hardcore, f.Expansion
		case d2gs.S2CLoadAct:
			l, perr := d2gs.ParseLoadAct(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.Info.MapSeed = l.Seed

			if c.OnInfo != nil {
				c.OnInfo(c.Info)
			}

			gm, gerr := d2netpacket.CreateGenerateMapPacket(regionFromAux(l.Aux))
			if gerr != nil {
				return out, ignored, gerr
			}

			out = append(out, gm)
		case d2gs.S2CPlayerInGame:
			in, perr := d2gs.ParseAssignPlayer(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.lastInGame = in.UnitID
		case d2gs.S2CRosterEntry:
			re, perr := d2gs.ParseRosterEntry(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.updatePeer(re.UnitID, func(pi *PeerInfo) {
				pi.Class, pi.Name, pi.Level, pi.PartyID, pi.partyKnown = re.Class, re.Name, re.Level, re.PartyID, true
			})
		case d2gs.S2CRosterUpdate:
			ru, perr := d2gs.ParseRosterUpdate(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.updatePeer(ru.UnitID, func(pi *PeerInfo) { pi.Level, pi.PartyID, pi.partyKnown = ru.Level, ru.PartyID, true })
		case d2gs.S2CRosterParty:
			rp, perr := d2gs.ParseRosterParty(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.updatePeer(rp.UnitID, func(pi *PeerInfo) { pi.PartyID, pi.partyKnown = rp.PartyID, true })
		case d2gs.S2CRosterKills:
			rk, perr := d2gs.ParseRosterKills(p)
			if perr != nil {
				return out, ignored, perr
			}

			c.updatePeer(rk.UnitID, func(pi *PeerInfo) { pi.Kills = rk.Kills })
		case d2gs.S2CMetaAE:
			typ, data, done, terr := c.tun.Add(p)
			if terr != nil {
				return out, ignored, terr
			}

			if !done {
				continue
			}

			np := rawNP(d2netpackettype.NetPacketType(typ), data)

			if np.PacketType == d2netpackettype.AddPlayer {
				ap, uerr := d2netpacket.UnmarshalAddPlayer(data)
				if uerr != nil {
					return out, ignored, uerr
				}

				c.IDs.Set(ap.ID, c.lastInGame)
				np.PacketData = c.patchAddPlayer(&ap, np.PacketData)
			}

			if np.PacketType == d2netpackettype.RosterUpdate {
				ru, uerr := d2netpacket.UnmarshalRosterUpdate(data)
				if uerr != nil {
					return out, ignored, uerr
				}

				np.PacketData = c.patchRoster(&ru, np.PacketData)
			}

			out = append(out, np)
		case d2gs.S2CPlayerMove:
			m, perr := d2gs.ParsePlayerMove(p)
			if perr != nil {
				return out, ignored, perr
			}

			id, ok := c.IDs.UUID(m.UnitID)
			if !ok {
				ignored += len(p)
				continue
			}

			mp, merr := d2netpacket.CreateMovePlayerPacket(id, fromSub(m.CurX), fromSub(m.CurY), fromSub(m.TargetX), fromSub(m.TargetY))
			if merr != nil {
				return out, ignored, merr
			}

			out = append(out, mp)
		case d2gs.S2CUnitSkillOnLoc:
			k, perr := d2gs.ParseUnitSkillOnLocation(p)
			if perr != nil {
				return out, ignored, perr
			}

			id, ok := c.IDs.UUID(k.UnitID)
			if !ok {
				ignored += len(p)
				continue
			}

			cp, cerr := d2netpacket.CreateCastPacket(id, int(k.Skill), fromSub(k.X), fromSub(k.Y))
			if cerr != nil {
				return out, ignored, cerr
			}

			out = append(out, cp)
		case d2gs.S2CPlayerLeave:
			l, perr := d2gs.ParsePlayerLeave(p)
			if perr != nil {
				return out, ignored, perr
			}

			id, ok := c.IDs.UUID(l.UnitID)
			if !ok {
				ignored += len(p)
				continue
			}

			dp, derr := d2netpacket.CreatePlayerDisconnectRequestPacket(id)
			if derr != nil {
				return out, ignored, derr
			}

			out = append(out, dp)
		case d2gs.S2CGameExit:
			sc, serr := d2netpacket.CreateServerClosedPacket()
			if serr != nil {
				return out, ignored, serr
			}

			out = append(out, sc)
		case d2gs.S2CChat:
			m, perr := d2gs.ParseChatMessage(p)
			if perr != nil {
				return out, ignored, perr
			}

			id, _ := c.IDs.UUID(m.UnitID)

			cp, cerr := d2netpacket.CreateChatPacket(id, m.Name, m.Text)
			if cerr != nil {
				return out, ignored, cerr
			}

			out = append(out, cp)
		default:
			ignored += len(p) // a packet of the real protocol we do not use: skipped by its size
		}
	}

	return out, ignored, nil
}

func clampU16(v int) uint16 {
	switch {
	case v < 0:
		return 0
	case v > math.MaxUint16:
		return math.MaxUint16
	}

	return uint16(v)
}

// PeerInfo is what the real roster packets taught the client about a unit.
type PeerInfo struct {
	Class   uint8
	Name    string
	Level   uint16
	PartyID uint16 // d2gs.NoParty when none
	Kills   int16

	partyKnown bool // a 0x75 or 0x8d told us the party id
}

// Peer returns the roster data the real packets (0x5b, 0x75, 0x8d, 0x65) gave
// for a unit id.
func (c *ClientSide) Peer(unit uint32) (PeerInfo, bool) {
	c.peerMu.Lock()
	defer c.peerMu.Unlock()

	p, ok := c.peers[unit]

	return p, ok
}

func (c *ClientSide) updatePeer(unit uint32, f func(*PeerInfo)) {
	c.peerMu.Lock()
	defer c.peerMu.Unlock()

	if c.peers == nil {
		c.peers = map[uint32]PeerInfo{}
	}

	p, ok := c.peers[unit]
	if !ok {
		p.PartyID = d2gs.NoParty
	}

	f(&p)
	c.peers[unit] = p
}

// patchAddPlayer makes the verified level from the real 0x5b entry win over
// the one in the tunnelled hero state; everything else in AddPlayer stays the
// OD2 extension.
func (c *ClientSide) patchAddPlayer(ap *d2netpacket.AddPlayerPacket, data []byte) []byte {
	peer, ok := c.Peer(c.lastInGame)
	if !ok || peer.Level == 0 {
		return data
	}

	if ap.Stats == nil {
		ap.Stats = &d2hero.HeroStatsState{}
	}

	ap.Stats.Level = int(peer.Level)

	if b, err := json.Marshal(ap); err == nil {
		return b
	}

	return data
}

// patchRoster applies the verified level and party id of the real packets to a
// tunnelled roster snapshot.
func (c *ClientSide) patchRoster(ru *d2netpacket.RosterUpdatePacket, data []byte) []byte {
	changed := false

	for i := range ru.Roster.Players {
		unit, ok := c.IDs.Unit(ru.Roster.Players[i].ID)
		if !ok {
			continue
		}

		peer, ok := c.Peer(unit)
		if !ok {
			continue
		}

		in := &ru.Roster.Players[i]
		if peer.Level != 0 {
			in.Level = int(peer.Level)
			changed = true
		}

		if peer.partyKnown {
			in.Party = int(peer.PartyID)
			if peer.PartyID == d2gs.NoParty {
				in.Party = 0
			}

			changed = true
		}
	}

	if !changed {
		return data
	}

	if b, err := json.Marshal(ru); err == nil {
		return b
	}

	return data
}

func regionFromAux(aux uint32) d2enum.RegionIdType { return d2enum.RegionIdType(aux) }
