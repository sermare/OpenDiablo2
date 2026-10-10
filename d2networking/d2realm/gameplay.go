package d2realm

import (
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
)

// Gameplay: every game owns an authoritative simulation (package d2mp). The
// clients send intent with the verified d2gs packets (walk/run 0x01/0x03,
// cast 0x05/0x0c, select skill 0x3c, interact 0x13, pick up 0x16, drop 0x17)
// and the tunnel message Command; the realm steps the simulation in real time
// and sends every member the events of its level (World, in the tunnel) plus
// the native 0x0f PlayerMove / 0x4d UnitSkillOnLocation packets for the other
// heroes' movement and casts (informational for this client, which reads the
// World events).

const defaultTickMs = 40

// startGame creates the simulation of a game and runs its clock until the game
// ends. s.mu is held.
func (s *Server) startGame(g *game) {
	g.sim = d2mp.NewSim(d2mp.Config{Seed: g.seed, Rules: s.cfg.Rules, TickMs: defaultTickMs})
	g.done = make(chan struct{})
	g.start = time.Now()

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		t := time.NewTicker(defaultTickMs * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-g.done:
				return
			case <-t.C:
				s.mu.Lock()
				g.sim.Tick(uint32(time.Since(g.start) / time.Millisecond))
				s.flush(g)
				s.mu.Unlock()
			}
		}
	}()
}

// simJoin enters a hero into the simulation. s.mu is held.
func (s *Server) simJoin(ss *session) {
	g := ss.game
	h := ss.char.Header
	in := d2mp.JoinInfo{ID: ss.unitID, Name: ss.charName, Class: uint8(h.Class), Level: h.Level, Act: int(ss.act) + 1}

	if b := ss.char.Body; b != nil {
		in.Gold = uint32(b.Attributes.Gold)
		in.HP = int32(b.Attributes.MaxHP)
	}

	g.sim.Join(in)
	ss.simLevel = 0
	s.flush(g)
}

// flush sends the queued events of every member. s.mu is held.
func (s *Server) flush(g *game) {
	for _, m := range g.members {
		evs := g.sim.Take(m.unitID)
		if evs == nil {
			continue
		}

		pkts := d2gs.Tunnel(d2gs.S2CMetaAE, MsgWorld, worldBody(evs))

		for _, e := range evs {
			switch e.Type {
			case d2mp.EvSeg:
				if u := g.sim.Unit(e.ID); u != nil && u.Kind == d2mp.KindPlayer && e.ID != m.unitID && e.Seg.Speed > 0 {
					pkts = append(pkts, d2gs.PlayerMove{UnitID: e.ID, Run: e.Seg.Speed > d2mp.WalkSpeed+0.01,
						TargetX: d2mp.Sub(e.Seg.X1), TargetY: d2mp.Sub(e.Seg.Y1),
						CurX: d2mp.Sub(e.Seg.X0), CurY: d2mp.Sub(e.Seg.Y0)}.Marshal())
				}
			case d2mp.EvAttack:
				if u := g.sim.Unit(e.ID); u != nil && u.Kind == d2mp.KindPlayer && e.ID != m.unitID && e.Other == 0 {
					pkts = append(pkts, d2gs.UnitSkillOnLocation{UnitID: e.ID, Skill: uint16(e.A),
						X: uint16(e.B & 0xffff), Y: uint16(uint32(e.B) >> 16)}.Marshal())
				}
			}
		}

		m.send(pkts...)
	}

	// tell the other members where a hero went (lobby level protocol)
	for _, m := range g.members {
		lvl := g.sim.Level(m.unitID)
		if lvl == 0 || lvl == m.simLevel {
			continue
		}

		first := m.simLevel == 0
		m.simLevel = lvl

		if first { // the starting town is not news; only moves are announced
			continue
		}

		act := byte(d2level.ActOfLevel(int(lvl)) - 1)
		m.act, m.level, m.hasLevel = act, lvl, true

		for _, o := range g.members {
			if o != m {
				o.sendMsg(PlayerLevel{UnitID: m.unitID, Act: act, Level: lvl})
			}
		}
	}
}

func worldBody(evs []d2mp.Event) []byte {
	_, body := Encode(World{Events: evs})

	return body
}

// gameplayPacket handles a client packet that drives the simulation. It reports
// whether the packet was one. s.mu is held.
func (s *Server) gameplayPacket(ss *session, p []byte) bool {
	switch p[0] {
	case d2gs.C2SWalkToLocation, d2gs.C2SRunToLocation, d2gs.C2SCastLeftLocation, d2gs.C2SCastRightLocation,
		d2gs.C2SSelectSkill, d2gs.C2SInteractUnit, d2gs.C2SPickUpUnit, d2gs.C2SDropCursorItem:
	default:
		return false
	}

	if ss.game == nil || ss.game.sim == nil {
		return true
	}

	m, err := d2gs.DecodeClient(p)
	if err != nil {
		return true
	}

	sim, id := ss.game.sim, ss.unitID

	switch v := m.(type) {
	case *d2gs.MoveToLocation:
		sim.Walk(id, v.Run, d2mp.FromSub(v.X), d2mp.FromSub(v.Y))
	case *d2gs.CastOnLocation:
		sim.Cast(id, v.Right, d2mp.FromSub(v.X), d2mp.FromSub(v.Y))
	case *d2gs.SelectSkill:
		sim.SelectSkill(id, uint16(v.Skill), v.Right)
	case *d2gs.InteractUnit:
		sim.Interact(id, d2mp.Kind(v.UnitType), v.UnitID)
	case *d2gs.PickUpUnit:
		sim.PickUp(id, v.UnitID)
	case *d2gs.DropCursorItem:
		sim.Drop(id, v.ItemID)
	}

	s.flush(ss.game)

	return true
}

func (s *Server) doCommand(ss *session, m Command) {
	if ss.game == nil || ss.game.sim == nil {
		ss.result(MsgCommand, CodeNotInGame, "not in a game")

		return
	}

	ss.game.sim.Command(ss.unitID, m.Cmd)
	s.flush(ss.game)
}
