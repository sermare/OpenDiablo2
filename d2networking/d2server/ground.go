package d2server

import (
	"fmt"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// Server-owned ground items. The exe's server owns every item unit and looks a pickup up by unit id
// (ITEMMODE_ServerHandlePickupRequest 0x55ace0, VERIFIED): a unit whose mode is no longer 3 ("on the ground")
// is not taken, there is no owner or timer test, and the cursor must be empty. The first request wins.
// Gold goes through ITEMMODE_PickupGoldFromGround 0x55a5f0 -> PARTY_DistributeGoldPickup 0x53e640, which
// shareGold already ports. Which heroes see a drop is the exe's room activation (UNVERIFIED, not traced):
// here every hero in the same level (area) does, a superset of the rooms.

// maxGround bounds the registry (a hero cannot flood the server with piles).
const maxGround = 4096

// groundBook is the registry of items lying on the ground, by item id.
type groundBook struct {
	mu    sync.Mutex
	next  int
	items map[string]d2netpacket.GroundEntry
	byKey map[string]string // drop key -> id
	// claims are the loot sources (monster, chest) a client was granted the roll of: key -> level.
	claims map[string]int
}

func newGroundBook() *groundBook {
	return &groundBook{items: map[string]d2netpacket.GroundEntry{}, byKey: map[string]string{}, claims: map[string]int{}}
}

// add registers a drop; known is true when the key was registered already (then out is the existing entry).
func (b *groundBook) add(e d2netpacket.GroundEntry) (out d2netpacket.GroundEntry, known, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if e.Key != "" {
		if id, found := b.byKey[e.Key]; found {
			return b.items[id], true, true
		}
	}

	if len(b.items) >= maxGround {
		return e, false, false
	}

	b.next++
	e.ID = fmt.Sprintf("g%d", b.next)
	b.items[e.ID] = e

	if e.Key != "" {
		b.byKey[e.Key] = e.ID
	}

	return e, false, true
}

// maxClaims bounds the claim registry.
const maxClaims = 16384

// claim grants the roll of a loot source to the first caller; a later caller (or the same one again) is refused.
func (b *groundBook) claim(key string, level int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if key == "" {
		return false
	}

	if _, taken := b.claims[key]; taken {
		return false
	}

	if len(b.claims) >= maxClaims { // a flood: forget the oldest generation rather than refuse every roll
		b.claims = map[string]int{}
	}

	b.claims[key] = level

	return true
}

// pruneClaims forgets the claims of levels nobody is in any more (their monsters and chests spawn afresh on the
// next visit).
func (b *groundBook) pruneClaims(occupied map[int]bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for k, lvl := range b.claims {
		if !occupied[lvl] {
			delete(b.claims, k)
		}
	}
}

// take removes an item; the first caller gets it, every later one gets false.
func (b *groundBook) take(id string) (d2netpacket.GroundEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e, ok := b.items[id]
	if !ok {
		return e, false
	}

	delete(b.items, id)
	delete(b.byKey, e.Key)

	return e, true
}

func (b *groundBook) get(id string) (d2netpacket.GroundEntry, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.items[id]

	return e, ok
}

// inLevel lists the items of a level.
func (b *groundBook) inLevel(level int) []d2netpacket.GroundEntry {
	b.mu.Lock()
	defer b.mu.Unlock()

	var out []d2netpacket.GroundEntry

	for _, e := range b.items {
		if e.Level == level {
			out = append(out, e)
		}
	}

	return out
}

// areaOf is the level a hero is in, per the roster (0 = unknown).
func (g *GameServer) areaOf(id string) int {
	g.soc.mu.Lock()
	defer g.soc.mu.Unlock()

	if m, ok := g.soc.roster.Member(id); ok {
		return m.Area
	}

	return 0
}

// groundSend sends a ground packet to one hero.
func (g *GameServer) groundSend(id string, p d2netpacket.GroundItemPacket) {
	c := g.connByID(id)
	if c == nil {
		return
	}

	pkt, err := d2netpacket.CreateGroundItemPacket(p)
	if err == nil {
		err = c.SendPacketToClient(pkt)
	}

	if err != nil {
		g.Errorf("GroundItem %s to %s: %v", p.Op, id, err)
	}
}

// groundLevelSend sends a ground packet to every hero in the level, and to the heroes in always (the
// sender's roster area can lag behind its real level until its level change arrives).
func (g *GameServer) groundLevelSend(level int, p d2netpacket.GroundItemPacket, always ...string) {
	for id := range g.connList() {
		if g.areaOf(id) == level || contains(always, id) {
			g.groundSend(id, p)
		}
	}
}

func (g *GameServer) groundDeny(id, item string) {
	g.groundSend(id, d2netpacket.GroundItemPacket{Op: d2netpacket.GroundDeny, Entries: []d2netpacket.GroundEntry{{ID: item}}})
}

// onGroundItem handles a client's ground item request.
func (g *GameServer) onGroundItem(client ClientConnection, packet d2netpacket.NetPacket) error {
	p, err := d2netpacket.UnmarshalGroundItem(packet.PacketData)
	if err != nil {
		return err
	}

	if len(p.Entries) != 1 {
		return fmt.Errorf("ground %s: %d entries", p.Op, len(p.Entries))
	}

	me, e := client.GetUniqueID(), p.Entries[0]

	switch p.Op {
	case d2netpacket.GroundDrop:
		return g.groundDrop(me, p, e)
	case d2netpacket.GroundPickup:
		got, ok := g.soc.ground.take(e.ID)
		if !ok { // lost the race between get and take
			g.groundDeny(me, e.ID)
			return nil
		}

		g.Infof("GROUND pickup id=%s by %q gold=%d", got.ID, g.nameOf(me), got.Gold)
		g.groundLevelSend(got.Level, d2netpacket.GroundItemPacket{Op: d2netpacket.GroundRemove, Picker: me,
			Entries: []d2netpacket.GroundEntry{{ID: got.ID, Gold: got.Gold, Owner: got.Owner}}}, me)

		if got.Gold > 0 {
			return g.shareGold(d2netpacket.PartyGoldPacket{Picker: me, Amount: got.Gold, PlayerDropped: got.Owner != ""})
		}
	case d2netpacket.GroundClaim:
		op := d2netpacket.GroundLost
		if g.soc.ground.claim(e.Key, e.Level) {
			op = d2netpacket.GroundGrant
		}

		g.groundSend(me, d2netpacket.GroundItemPacket{Op: op, Entries: []d2netpacket.GroundEntry{{Key: e.Key, Level: e.Level}}})
	case d2netpacket.GroundExpire:
		if got, ok := g.soc.ground.take(e.ID); ok {
			g.groundLevelSend(got.Level, d2netpacket.GroundItemPacket{Op: d2netpacket.GroundRemove,
				Entries: []d2netpacket.GroundEntry{{ID: got.ID}}})
		}
	default:
		return fmt.Errorf("ground: unknown op %q", p.Op)
	}

	return nil
}

func (g *GameServer) groundDrop(me string, p d2netpacket.GroundItemPacket, e d2netpacket.GroundEntry) error {
	if e.Item == nil && e.Gold <= 0 {
		return nil // nothing to put down (a forged negative pile would subtract at the pickup)
	}

	out, known, ok := g.soc.ground.add(e)
	if !ok {
		g.Warningf("GROUND drop refused: %d items already on the ground", maxGround)
		return nil
	}

	if known {
		g.groundSend(me, d2netpacket.GroundItemPacket{Op: d2netpacket.GroundBind, Ref: p.Ref, Entries: []d2netpacket.GroundEntry{out}})
		return nil
	}

	g.Infof("GROUND drop id=%s level=%d gold=%d by %q", out.ID, out.Level, out.Gold, g.nameOf(me))
	placed := d2netpacket.GroundItemPacket{Op: d2netpacket.GroundPlaced, Ref: p.Ref, Sender: me, Entries: []d2netpacket.GroundEntry{out}}
	g.groundLevelSend(out.Level, placed, me)

	return nil
}

// groundSync tells a hero that entered a level which items lie there.
func (g *GameServer) groundSync(id string, level int) {
	if g.soc == nil || g.soc.ground == nil {
		return
	}

	if list := g.soc.ground.inLevel(level); len(list) > 0 {
		g.groundSend(id, d2netpacket.GroundItemPacket{Op: d2netpacket.GroundSync, Entries: list})
	}
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}

	return false
}

// groundPrune forgets the loot claims of levels that no hero is in.
func (g *GameServer) groundPrune() {
	if g.soc == nil || g.soc.ground == nil {
		return
	}

	occupied := map[int]bool{}
	for id := range g.connList() {
		occupied[g.areaOf(id)] = true
	}

	g.soc.ground.pruneClaims(occupied)
}
