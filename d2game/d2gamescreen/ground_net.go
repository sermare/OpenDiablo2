package d2gamescreen

import (
	"fmt"
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// Server-owned ground items (client side; the registry is d2server/ground.go). In a networked game every
// item or pile that lands on the ground is reported to the server, which gives it a shared id; a pickup is a
// request the server answers: the first hero gets the item (remove with its id as picker), a later one is
// denied (ITEMMODE_ServerHandlePickupRequest 0x55ace0 finds the unit no longer on the ground). The clients still
// roll monster and chest loot alike; the drop key lets the server keep one copy and bind the others.

// networked reports whether ground items go through the server.
func (v *Game) networked() bool {
	return v.gameClient != nil && !v.gameClient.IsSinglePlayer()
}

// groundKey identifies a drop independent of the reporting client.
func (v *Game) groundKey(it *d2mapentity.Item, x, y int) string {
	if it.IsGold() {
		return fmt.Sprintf("g|%d|%d|%d|%d", v.currentLevel(), x, y, it.Gold)
	}

	return fmt.Sprintf("i|%d|%d|%d|%s|%d", v.currentLevel(), x, y, it.Item.GetItemCode(), d2player.GroundItemData(it.Item).Seed)
}

func entitySubtile(it *d2mapentity.Item) (x, y int) {
	fx, fy := it.GetPositionF()

	return int(math.Round(fx * subtilesInTile)), int(math.Round(fy * subtilesInTile))
}

// announceGround reports a new ground item to the server.
func (v *Game) announceGround(it *d2mapentity.Item, owner string) {
	if !v.networked() || it == nil || it.NetID != "" {
		return
	}

	x, y := entitySubtile(it)
	it.NetKey = v.groundKey(it, x, y)
	e := d2netpacket.GroundEntry{Key: it.NetKey, Level: v.currentLevel(), X: x, Y: y, Gold: it.Gold, Owner: owner}

	if it.Item != nil {
		data := d2player.GroundItemData(it.Item)
		e.Item = &data
	}

	v.sendGround(d2netpacket.GroundDrop, it.ID(), e)
}

func (v *Game) sendGround(op, ref string, e d2netpacket.GroundEntry) {
	pkt, err := d2netpacket.CreateGroundItemPacket(d2netpacket.GroundItemPacket{Op: op, Ref: ref, Entries: []d2netpacket.GroundEntry{e}})
	if err == nil {
		err = v.gameClient.SendPacketToServer(pkt)
	}

	if err != nil {
		v.Warningf("GROUND %s: %v", op, err)
	}
}

// expireGround tells the server an item timed out on the ground.
func (v *Game) expireGround(it *d2mapentity.Item) {
	if v.networked() && it.NetID != "" {
		v.sendGround(d2netpacket.GroundExpire, "", d2netpacket.GroundEntry{ID: it.NetID})
	}
}

// requestGroundPickup asks the server for a networked item; it reports true when the request took over the
// pickup (the item is taken when the server approves, in onGroundItem).
func (v *Game) requestGroundPickup(it *d2mapentity.Item) bool {
	if !v.networked() || it.NetID == "" {
		return false
	}

	if it.NetPending {
		return true
	}

	if !it.IsGold() && v.gameControls.CursorItem() != nil {
		v.Infof("cannot pick up %q: the cursor already holds an item", plainLabel(it.Label()))
		return true
	}

	it.NetPending = true

	v.Infof("GROUND pickup request id=%s", it.NetID)
	v.sendGround(d2netpacket.GroundPickup, "", d2netpacket.GroundEntry{ID: it.NetID})

	return true
}

// groundEntities lists the ground items of the level this client holds.
func (v *Game) groundEntities() []*d2mapentity.Item {
	var out []*d2mapentity.Item

	for _, e := range v.gameClient.MapEngine.Entities() {
		if it, ok := e.(*d2mapentity.Item); ok {
			out = append(out, it)
		}
	}

	return out
}

func (v *Game) groundByNetID(id string) *d2mapentity.Item {
	for _, it := range v.groundEntities() {
		if it.NetID == id {
			return it
		}
	}

	return nil
}

// onGroundItem applies a server ground item operation (on the game loop).
func (v *Game) onGroundItem(p d2netpacket.GroundItemPacket) {
	if v.gameClient == nil || v.gameClient.MapEngine == nil || len(p.Entries) == 0 {
		return
	}

	switch p.Op {
	case d2netpacket.GroundPlaced, d2netpacket.GroundBind:
		e := p.Entries[0]

		if p.Op == d2netpacket.GroundBind {
			v.Infof("GROUND bound id=%s", e.ID)
		}

		if it, ok := v.gameClient.MapEngine.Entities()[p.Ref].(*d2mapentity.Item); ok && (p.Op == d2netpacket.GroundBind || p.Sender == v.me()) {
			it.NetID = e.ID

			return
		}

		v.adoptGround(e)
	case d2netpacket.GroundSync:
		for _, e := range p.Entries {
			v.adoptGround(e)
		}
	case d2netpacket.GroundRemove:
		v.groundRemoved(p)
	case d2netpacket.GroundDeny:
		if it := v.groundByNetID(p.Entries[0].ID); it != nil {
			// somebody else took it first (or it is gone): the item vanishes here too
			it.NetPending = false

			v.Infof("GROUND pickup denied id=%s", it.NetID)
			v.gameClient.MapEngine.RemoveEntity(it)
		}
	}
}

// adoptGround shows a ground item the server announced: it binds the id to the local copy this client rolled
// itself (same key), or creates the item.
func (v *Game) adoptGround(e d2netpacket.GroundEntry) {
	if e.Level != v.currentLevel() || v.groundByNetID(e.ID) != nil {
		return
	}

	for _, it := range v.groundEntities() {
		if it.NetID == "" && it.NetKey != "" && it.NetKey == e.Key {
			it.NetID = e.ID

			return
		}
	}

	var (
		ent *d2mapentity.Item
		err error
	)

	engine := v.gameClient.MapEngine

	if e.Item != nil {
		item, ierr := d2player.GroundItemFromData(v.itemFactory(), e.Item)
		if ierr != nil {
			v.Warningf("GROUND item %s: %v", e.ID, ierr)
			return
		}

		ent, err = engine.NewGroundItem(item, e.X, e.Y)
	} else {
		ent, err = engine.NewGoldPile(e.Gold, v.asset.TranslateString("gld"), e.X, e.Y)
	}

	if err != nil {
		v.Warningf("GROUND item %s: %v", e.ID, err)
		return
	}

	ent.NetID, ent.NetKey, ent.DropOwner = e.ID, e.Key, e.Owner
	engine.AddEntity(ent)
	v.playSoundAt(ent.DropSound, ent.GetPosition(), "drop")
	kind := "item"
	if ent.IsGold() {
		kind = "gold"
	}

	v.Infof("GROUND shown id=%s kind=%s at (%d,%d)", e.ID, kind, e.X, e.Y)
}

// groundRemoved handles an item that left the ground: for the hero that got it, the pickup completes.
func (v *Game) groundRemoved(p d2netpacket.GroundItemPacket) {
	it := v.groundByNetID(p.Entries[0].ID)
	if it == nil {
		return
	}

	v.gameClient.MapEngine.RemoveEntity(it)

	if p.Picker != v.me() {
		return
	}

	it.NetPending = false
	v.Infof("GROUND won id=%s", it.NetID)

	if it.IsGold() { // the purse is paid by the PartyGold answer
		v.playSoundAt("item_gold", it.GetPosition(), "pickup")

		if v.ground.onPickup != nil {
			v.ground.onPickup(it)
		}

		return
	}

	if v.gameControls.CursorItem() != nil {
		// the cursor filled while the request travelled: the item goes back down at the hero's feet
		px, py := v.localPlayer.GetPositionF()
		if cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), 1, false); len(cells) > 0 {
			_, _ = v.spawnGroundItem(it.Item, cells[0])
		}

		return
	}

	if !v.takeGroundItem(it) {
		// quick pickup found no room: the item goes back down at the hero's feet (a new drop)
		px, py := v.localPlayer.GetPositionF()
		if cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), 1, false); len(cells) > 0 {
			_, _ = v.spawnGroundItem(it.Item, cells[0])
		}
	}
}

// commandGroundRace is the test console command "groundrace item|gold": the pickup request for the nearest
// networked ground item of that kind goes out at once (no walk), so two processes can race for one item.
func (v *Game) commandGroundRace(args []string) error {
	if !v.networked() || v.localPlayer == nil || len(args) != 1 {
		return fmt.Errorf("groundrace item|gold: needs a networked game")
	}

	px, py := v.localPlayer.GetPositionF()

	var best *d2mapentity.Item

	bestD := math.MaxFloat64

	for _, it := range v.groundEntities() {
		x, y := it.GetPositionF()
		if d := math.Hypot(px-x, py-y); it.NetID != "" && it.IsGold() == (args[0] == "gold") && d < bestD {
			best, bestD = it, d
		}
	}

	if best == nil {
		v.Infof("GROUNDRACE none")
		return nil
	}

	v.Infof("GROUNDRACE request id=%s", best.NetID)
	v.requestGroundPickup(best)

	return nil
}
