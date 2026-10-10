package d2realm

import (
	"sync"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2mp"
)

// PlayerMoved and PlayerCast are the native 0x0f and 0x4d packets the realm
// sends for the other heroes (informational: Player reads the World events).
type (
	PlayerMoved = d2gs.PlayerMove
	PlayerCast  = d2gs.UnitSkillOnLocation
)

// Walk sends a walk or run order (C2S 0x01 / 0x03) to a tile position.
func (c *Client) Walk(run bool, x, y float64) error {
	return c.write(d2gs.MoveToLocation{Run: run, X: d2mp.Sub(x), Y: d2mp.Sub(y)}.MarshalPacket())
}

// Cast uses the selected skill of a hand on a tile position (C2S 0x05 / 0x0c).
func (c *Client) Cast(right bool, x, y float64) error {
	return c.write(d2gs.CastOnLocation{Right: right, X: d2mp.Sub(x), Y: d2mp.Sub(y)}.MarshalPacket())
}

// SelectSkill selects the skill of a hand (C2S 0x3c).
func (c *Client) SelectSkill(skill uint16, right bool) error {
	return c.write(d2gs.SelectSkill{Skill: uint32(skill), Right: right, ItemID: 0xFFFFFFFF}.MarshalPacket())
}

// Interact attacks a monster or uses an object (C2S 0x13).
func (c *Client) Interact(kind d2mp.Kind, id uint32) error {
	return c.write(d2gs.InteractUnit{UnitType: uint32(kind), UnitID: id}.MarshalPacket())
}

// PickUp picks up a ground item (C2S 0x16).
func (c *Client) PickUp(id uint32) error {
	return c.write(d2gs.PickUpUnit{UnitType: uint32(d2mp.KindItem), UnitID: id}.MarshalPacket())
}

// Drop drops an inventory item (C2S 0x17).
func (c *Client) Drop(id uint32) error {
	return c.write(d2gs.DropCursorItem{ItemID: id}.MarshalPacket())
}

// GameCommand sends a command that has no d2gs packet (party, trade, ...).
func (c *Client) GameCommand(cmd d2mp.Command) error { return c.Send(Command{Cmd: cmd}) }

// Drain removes and returns every queued event for which match is true.
func (c *Client) Drain(match func(interface{}) bool) []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	var out, keep []interface{}

	for _, e := range c.events {
		if match(e) {
			out = append(out, e)
		} else {
			keep = append(keep, e)
		}
	}

	c.events = keep

	return out
}

// Player is a client in a game: the realm connection plus the world replica.
type Player struct {
	*Client
	Joined GameJoined

	mu  sync.Mutex
	rep *d2mp.Replica
}

// NewPlayer wraps a connected client that has joined a game. clock returns the
// local time in ms (nil = wall clock).
func NewPlayer(c *Client, j GameJoined, rules d2mp.Rules, clock func() float64) *Player {
	if clock == nil {
		start := time.Now()
		clock = func() float64 { return float64(time.Since(start)) / float64(time.Millisecond) }
	}

	return &Player{Client: c, Joined: j, rep: d2mp.NewReplica(d2mp.ReplicaConfig{
		Self: j.UnitID, Seed: j.Seed, Rules: rules, Clock: clock})}
}

// Pump applies the world events received so far.
func (p *Player) Pump() {
	evs := p.Drain(func(e interface{}) bool { _, ok := e.(World); return ok })

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, e := range evs {
		for _, ev := range e.(World).Events {
			p.rep.Apply(ev)
		}
	}
}

// View runs f on the replica after applying pending events.
func (p *Player) View(f func(r *d2mp.Replica)) {
	p.Pump()
	p.mu.Lock()
	defer p.mu.Unlock()
	f(p.rep)
}

// WalkTo walks (or runs) with local prediction.
func (p *Player) WalkTo(run bool, x, y float64) error {
	p.mu.Lock()
	p.rep.PredictWalk(run, x, y)
	p.mu.Unlock()

	return p.Walk(run, x, y)
}

// WaitFor pumps until cond holds on the replica or the timeout passes.
func (p *Player) WaitFor(timeout time.Duration, cond func(r *d2mp.Replica) bool) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ok := false
		p.View(func(r *d2mp.Replica) { ok = cond(r) })

		if ok {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return false
}
