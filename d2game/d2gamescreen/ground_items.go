package d2gamescreen

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2ground"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// Items in the world. The client owns the ground items here (single player;
// nothing is sent over the network yet). Item unit modes of the original
// (units.md): 3 on the ground, 4 on the cursor, 0 stored in the inventory;
// this file moves an item through exactly those three states.
const (
	pickupRange     = 0.9  // tiles; the server check in the original is "distance < 0x33" (units unverified)
	chestRange      = 2.5  // tiles; chests block movement so the hero stops next to them
	interactTimeout = 12.0 // seconds before a walk to an item or chest is abandoned
	dropSearchRange = 6    // tiles around the hero (or chest) searched for a free ground cell
	subtilesInTile  = 5
)

// groundState is the pending click interaction with a ground item or chest.
type groundState struct {
	item    *d2mapentity.Item
	chest   *d2mapentity.Object
	elapsed float64
	// onPickup is called after a successful pickup (used by the autotest).
	onPickup func(it *d2mapentity.Item)
	// chestSeq numbers chest openings so each one rolls a different seed.
	chestSeq uint32
}

// lootContainers are the objects.txt ids that open into a treasure class drop.
// The ids and names come from objects.txt (Act 1 and 2 caskets, chests, barrels,
// crates and urns); the sound handles are the Sounds.txt entries named after
// each kind. Which handle the original plays is not in the notes (UNVERIFIED).
var lootContainers = map[int]string{
	1: "object_casket", 3: "object_casket", 50: "object_casket", 51: "object_casket",
	53: "object_casket", 79: "object_casket",
	5: "object_chest_large", 6: "object_chest_large",
	7: "object_barrel_explode", 11: "object_barrel_explode",
	46: "object_chest_small",
}

// OnPlayerDropItem puts an item held on the cursor on the ground next to the
// hero (INV_ServerDropCursorItem: cursor mode 4 becomes ground mode 3).
func (v *Game) OnPlayerDropItem(item d2player.InventoryItem) {
	it, ok := item.(*diablo2item.Item)
	if !ok || v.localPlayer == nil {
		return
	}

	px, py := v.localPlayer.GetPositionF()

	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), 1, true)
	if len(cells) == 0 {
		v.Warningf("no free ground cell to drop %q; keeping it on the cursor", it.Label())
		v.gameControls.SetCursorItem(it)

		return
	}

	ent, err := v.spawnGroundItem(it, cells[0])
	if err != nil {
		v.Errorf("could not drop %q: %v", it.Label(), err)
		return
	}

	v.Infof("AUTOGROUND drop name=%q quality=%s pos=(%d,%d)", plainLabel(ent.Label()), it.QualityName(), cells[0].X, cells[0].Y)
}

// walkToItem starts the walk to a ground item; pickup happens on arrival.
func (v *Game) walkToItem(it *d2mapentity.Item) {
	v.ground.item, v.ground.chest, v.ground.elapsed = it, nil, 0
	v.npcTarget = nil

	x, y := it.GetPositionF()
	v.Infof("walking to pick up %q at (%.1f,%.1f)", plainLabel(it.Label()), x, y)
	v.OnPlayerMove(x, y)
}

// walkToObject starts the walk to a clicked object; chests open on arrival.
func (v *Game) walkToObject(ob *d2mapentity.Object) {
	x, y := ob.GetPositionF()
	v.npcTarget = nil

	switch ob.Kind() {
	case d2level.ObjectDoor, d2level.ObjectWaypoint, d2level.ObjectPortal:
		v.useObject(ob)
		return
	}

	if _, ok := lootContainers[ob.Record().Index]; !ok {
		v.OnPlayerMove(x, y)
		return
	}

	v.ground.item, v.ground.chest, v.ground.elapsed = nil, ob, 0

	v.Infof("walking to open %q (object %d) at (%.1f,%.1f)", ob.Label(), ob.Record().Index, x, y)
	v.OnPlayerMove(x, y)
}

// advanceGroundInteraction completes a pending pickup or chest opening once the
// hero has arrived.
func (v *Game) advanceGroundInteraction(elapsed float64) {
	if v.localPlayer == nil || v.gameControls == nil {
		return
	}

	px, py := v.localPlayer.GetPositionF()

	if it := v.ground.item; it != nil {
		v.ground.elapsed += elapsed
		ix, iy := it.GetPositionF()

		switch {
		case v.gameClient.MapEngine.Entities()[it.ID()] == nil:
			v.ground.item = nil // somebody else picked it up
		case math.Hypot(px-ix, py-iy) <= pickupRange:
			v.ground.item = nil
			v.pickUp(it)
		case v.ground.elapsed > interactTimeout:
			v.Warningf("gave up walking to %q", plainLabel(it.Label()))
			v.ground.item = nil
		}
	}

	if ob := v.ground.chest; ob != nil {
		v.ground.elapsed += elapsed
		ox, oy := ob.GetPositionF()

		switch {
		case math.Hypot(px-ox, py-oy) <= chestRange:
			v.ground.chest = nil
			v.openChest(ob)
		case v.ground.elapsed > interactTimeout:
			v.Warningf("gave up walking to %q", ob.Label())
			v.ground.chest = nil
		}
	}
}

// pickUp moves a ground item to the cursor (packet 0x16, item mode 3 -> 4) or,
// for gold, straight into the hero's purse.
func (v *Game) pickUp(it *d2mapentity.Item) {
	if it.IsGold() {
		v.gameClient.MapEngine.RemoveEntity(it)
		v.gameControls.AddGold(it.Gold)
		v.playSound("item_gold")
		v.Infof("AUTOGROUND pickup gold amount=%d total=%d", it.Gold, v.localPlayer.Gold)

		if v.ground.onPickup != nil {
			v.ground.onPickup(it)
		}

		return
	}

	if v.gameControls.CursorItem() != nil {
		v.Infof("cannot pick up %q: the cursor already holds an item", plainLabel(it.Label()))
		return
	}

	v.gameClient.MapEngine.RemoveEntity(it)
	v.gameControls.SetCursorItem(it.Item)
	v.playSound("cursor_point_drop")
	v.Infof("AUTOGROUND pickup name=%q quality=%s -> cursor", plainLabel(it.Label()), it.Item.QualityName())

	if v.ground.onPickup != nil {
		v.ground.onPickup(it)
	}
}

// openChest plays the chest's animation and sound and drops its treasure class
// around it. The treasure class is "Act N Chest X" for the hero's act and
// difficulty (see d2ground.ChestTreasureClass).
func (v *Game) openChest(ob *d2mapentity.Object) {
	opened, err := ob.Open()
	if err != nil {
		v.Warningf("opening %q: %v", ob.Label(), err)
	}

	if !opened {
		return
	}

	id := ob.Record().Index
	v.playSound(lootContainers[id])

	ilvl := v.areaLevel()
	tc := d2ground.ChestTreasureClass(v.localPlayer.Act, d2ground.Normal, ilvl, v.itemFactory().TreasureClassLevel)

	v.ground.chestSeq++
	seed := v.chestSeed() + v.ground.chestSeq

	cx, cy := ob.GetPositionF()
	loot, err := v.itemFactory().DropLoot(tc, diablo2item.DropOptions{Seed: seed, ILvl: ilvl, Players: 1}, 0)

	if err != nil {
		v.Errorf("chest %d: %v", id, err)
		return
	}

	v.Infof("AUTOGROUND chest id=%d name=%q tc=%q ilvl=%d seed=%d drops=%d", id, ob.Label(), tc, ilvl, seed, len(loot.Entries))
	v.spawnLoot(loot, int(math.Floor(cx)), int(math.Floor(cy)), "chest")
}

// chestSeed is the base of the chest drop seeds: the map seed, so a given
// game always opens the same chests into the same loot.
func (v *Game) chestSeed() uint32 {
	if s, err := envSeed("OD2_AUTOCHEST_SEED"); err == nil {
		return s
	}

	return uint32(v.gameClient.MapEngine.Seed())
}

// areaLevel is the item level of object drops: OD2_AUTOGROUND_ILVL if set,
// else the hero's level (the real game uses the area's level, via 0x61dc00;
// levels.txt monster levels are not wired in yet).
func (v *Game) areaLevel() int {
	if n, err := envInt("OD2_AUTOGROUND_ILVL"); err == nil && n > 0 {
		return n
	}

	if v.localPlayer != nil && v.localPlayer.Stats.Level > 0 {
		return v.localPlayer.Stats.Level
	}

	return 1
}

// itemFactory returns the item factory of the map engine.
func (v *Game) itemFactory() *diablo2item.ItemFactory {
	return v.gameClient.MapEngine.ItemFactory()
}

// spawnLoot puts a roll's items and gold on free ground cells around (cx, cy),
// plays each drop sound and logs what landed where.
func (v *Game) spawnLoot(loot *diablo2item.Loot, cx, cy int, source string) []*d2mapentity.Item {
	cells := v.freeDropCells(cx, cy, len(loot.Entries), false)
	out := make([]*d2mapentity.Item, 0, len(loot.Entries))

	for i, e := range loot.Entries {
		if i >= len(cells) {
			v.Warningf("no free ground cell left for %d drop(s)", len(loot.Entries)-i)
			break
		}

		var (
			ent *d2mapentity.Item
			err error
		)

		if e.Item != nil {
			ent, err = v.spawnGroundItem(e.Item, cells[i])
		} else {
			ent, err = v.spawnGoldPile(e.Gold, cells[i])
		}

		if err != nil {
			v.Warningf("could not create a ground entity: %v", err)
			continue
		}

		quality := "gold"
		if e.Item != nil {
			quality = e.Item.QualityName()
		}

		v.Infof("AUTOGROUND spawn source=%s name=%q quality=%s pos=(%d,%d)",
			source, plainLabel(ent.Label()), quality, cells[i].X, cells[i].Y)

		out = append(out, ent)
	}

	return out
}

func (v *Game) spawnGroundItem(it *diablo2item.Item, c d2ground.Cell) (*d2mapentity.Item, error) {
	ent, err := v.gameClient.MapEngine.NewGroundItem(it, c.X*subtilesInTile+2, c.Y*subtilesInTile+2)
	if err != nil {
		return nil, err
	}

	v.gameClient.MapEngine.AddEntity(ent)
	v.playSound(ent.DropSound)

	return ent, nil
}

func (v *Game) spawnGoldPile(amount int, c d2ground.Cell) (*d2mapentity.Item, error) {
	name := v.asset.TranslateString("gld")

	ent, err := v.gameClient.MapEngine.NewGoldPile(amount, name, c.X*subtilesInTile+2, c.Y*subtilesInTile+2)
	if err != nil {
		return nil, err
	}

	v.gameClient.MapEngine.AddEntity(ent)
	v.playSound(ent.DropSound)

	return ent, nil
}

// freeDropCells finds n usable ground cells around a tile: walkable, not already
// holding a ground item and (when reachable is set, for drops next to the hero)
// reachable by the straight-line walk the engine's pathing does.
func (v *Game) freeDropCells(cx, cy, n int, reachable bool) []d2ground.Cell {
	engine := v.gameClient.MapEngine
	taken := map[d2ground.Cell]bool{}

	for _, e := range engine.Entities() {
		if it, ok := e.(*d2mapentity.Item); ok {
			x, y := it.GetPositionF()
			taken[d2ground.Cell{X: int(math.Floor(x)), Y: int(math.Floor(y))}] = true
		}
	}

	blocked := func(x, y int) bool {
		if taken[d2ground.Cell{X: x, Y: y}] || x < 0 || y < 0 || engine.TileAt(x, y) == nil {
			return true
		}

		if engine.SubTileAt(x*subtilesInTile+2, y*subtilesInTile+2).BlockWalk {
			return true
		}

		if reachable && v.localPlayer != nil {
			start := v.localPlayer.Position
			dest := d2vector.NewPosition(float64(x*subtilesInTile+2), float64(y*subtilesInTile+2))

			if path := engine.PathFind(start, dest); len(path) == 0 || path[len(path)-1].X() != dest.X() ||
				path[len(path)-1].Y() != dest.Y() {
				return true
			}
		}

		return false
	}

	return d2ground.DropCells(cx, cy, n, dropSearchRange, blocked)
}

// playSound plays a Sounds.txt handle (silently skipped for unknown handles and
// when OD2_AUTOTEST_MUTE is set).
func (v *Game) playSound(handle string) {
	if handle == "" || os.Getenv("OD2_AUTOTEST_MUTE") != "" {
		return
	}

	if v.asset.Records.Sound.Details[handle] == nil {
		v.Debugf("unknown sound handle %q", handle)
		return
	}

	v.soundEngine.PlaySoundHandle(handle)
}

// plainLabel strips the colour tokens ("[gold]...") and line breaks of a name.
func plainLabel(s string) string {
	for {
		i := strings.Index(s, "[")
		j := strings.Index(s, "]")

		if i < 0 || j < i {
			break
		}

		s = s[:i] + s[j+1:]
	}

	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}

// spawnTestChests puts chests and barrels from objects.txt around a tile, each
// walkable from the hero. ids default to one of every kind in lootContainers.
func (v *Game) spawnTestChests(cx, cy int, ids []int) []*d2mapentity.Object {
	cells := v.freeDropCells(cx, cy, len(ids)*2, true)
	out := make([]*d2mapentity.Object, 0, len(ids))

	for i, id := range ids {
		rec := v.asset.Records.Object.Details[id]
		if rec == nil || 2*i+1 >= len(cells) {
			v.Warningf("no objects.txt row or free cell for object %d", id)
			continue
		}

		c := cells[2*i+1] // every second cell, so chests do not wall each other in

		ob, err := v.gameClient.MapEngine.NewObject(c.X*subtilesInTile+2, c.Y*subtilesInTile+2, rec, d2resource.PaletteUnits)
		if err != nil {
			v.Warningf("could not create object %d (%s): %v", id, rec.Name, err)
			continue
		}

		v.gameClient.MapEngine.AddEntity(ob)
		v.Infof("AUTOGROUND chest-spawn id=%d name=%q pos=(%d,%d)", id, ob.Label(), c.X, c.Y)

		out = append(out, ob)
	}

	return out
}

// commandSpawnChest is the console command "spawnchest [objectID...]".
func (v *Game) commandSpawnChest(args []string) error {
	ids := []int{}

	for _, a := range args {
		var id int
		if _, err := fmt.Sscanf(a, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		ids = []int{7, 1, 5}
	}

	px, py := v.localPlayer.GetPositionF()
	v.spawnTestChests(int(px), int(py), ids)

	return nil
}

var _ d2interface.MapEntity = (*d2mapentity.Item)(nil)
