package d2gamescreen

import (
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Ground item autotest, driven by environment variables so it needs no mouse:
//
//	OD2_AUTOGROUND=<seed>[,count]  drops `count` (default 5) items and gold piles
//	                               around the hero by rolling a treasure class with
//	                               seeds seed, seed+1, ...
//	OD2_AUTOGROUND_TC=<name>       the treasure class (default "Andariel")
//	OD2_AUTOGROUND_ILVL=<n>        item level of the drops (default: hero level)
//	OD2_AUTOCHEST=<seed>[,id...]   spawns chests/barrels (objects.txt ids, default
//	                               7,1,5) and walks the hero to each to open it
//	OD2_AUTOPICKUP=1               walks to every ground item, picks it up to the
//	                               cursor and auto-places it in the inventory
//	OD2_AUTOGROUND_HOLD=<seconds>  keeps the game open that long when done
//
// Every step is logged with an "AUTOGROUND" prefix. OD2_AUTOEXIT quits at the end
// and OD2_AUTOTEST_MUTE silences the sounds, like the other OD2_AUTO* variables.
const (
	autoGroundDelay    = 5.0 // seconds after the hero appears
	autoGroundStepWait = 0.4
	autoGroundTimeout  = 20.0
	autoGroundDefaultN = 5
	autoGroundMaxRolls = 200
)

type autoGroundPhase int

const (
	agWait autoGroundPhase = iota
	agChests
	agPickup
	agPlace
	agFinish
	agDone
)

type autoGround struct {
	phase   autoGroundPhase
	elapsed float64
	timer   float64
	chests  []*d2mapentity.Object
	chestN  int
	target  *d2mapentity.Item
	picked  int
	placed  int
	gold    int
	failed  int
	skip    map[string]bool

	holdUntil float64
}

func autoGroundEnabled() bool {
	return os.Getenv("OD2_AUTOGROUND") != "" || os.Getenv("OD2_AUTOCHEST") != "" ||
		os.Getenv("OD2_AUTOPICKUP") != ""
}

func envInt(name string) (int, error) { return strconv.Atoi(strings.TrimSpace(os.Getenv(name))) }

func envSeed(name string) (uint32, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(os.Getenv(name)), 10, 32)

	return uint32(n), err
}

// advanceAutoGround is the state machine behind the variables above.
func (v *Game) advanceAutoGround(elapsed float64) {
	if !autoGroundEnabled() || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	a := &v.autoGround
	a.elapsed += elapsed
	a.timer += elapsed

	switch a.phase {
	case agWait:
		if a.elapsed >= autoGroundDelay {
			v.autoGroundSetup()
		}
	case agChests:
		v.autoGroundChests()
	case agPickup:
		v.autoGroundPickup()
	case agPlace:
		v.autoGroundPlace()
	case agFinish:
		v.autoGroundFinish()
	case agDone:
		if a.holdUntil > 0 && a.elapsed >= a.holdUntil {
			v.autoTestExit()
		}
	}
}

func (v *Game) autoGroundNext(p autoGroundPhase) {
	v.autoGround.phase = p
	v.autoGround.timer = 0
}

func (v *Game) autoGroundSetup() {
	px, py := v.localPlayer.GetPositionF()
	cx, cy := int(math.Floor(px)), int(math.Floor(py))

	v.Infof("AUTOGROUND start hero=(%d,%d) gold=%d inventory_items=%d", cx, cy, v.localPlayer.Gold,
		v.gameControls.InventoryItemCount())

	if spec := os.Getenv("OD2_AUTOGROUND"); spec != "" {
		v.autoGroundDrop(spec, cx, cy)
	}

	if spec := os.Getenv("OD2_AUTOCHEST"); spec != "" {
		ids := []int{7, 1, 5}

		if parts := strings.Split(spec, ","); len(parts) > 1 {
			ids = ids[:0]

			for _, p := range parts[1:] {
				if id, err := strconv.Atoi(p); err == nil {
					ids = append(ids, id)
				}
			}
		}

		v.autoGround.chests = v.spawnTestChests(cx, cy, ids)
		v.autoGroundNext(agChests)

		return
	}

	v.autoGroundNext(agPickup)
}

// autoGroundDrop rolls the treasure class with consecutive seeds until `count`
// things lie around the hero.
func (v *Game) autoGroundDrop(spec string, cx, cy int) {
	parts := strings.Split(spec, ",")

	seed64, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil {
		v.Warningf("AUTOGROUND: bad seed in OD2_AUTOGROUND=%q", spec)
		return
	}

	count := autoGroundDefaultN

	if len(parts) > 1 {
		if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
			count = n
		}
	}

	tc := os.Getenv("OD2_AUTOGROUND_TC")
	if tc == "" {
		tc = "Andariel"
	}

	loot := &diablo2item.Loot{}
	ilvl := v.areaLevel()

	for s := uint32(seed64); len(loot.Entries) < count && s < uint32(seed64)+autoGroundMaxRolls; s++ {
		l, err := v.itemFactory().DropLoot(tc, diablo2item.DropOptions{Seed: s, ILvl: ilvl, Players: 1}, 0)
		if err != nil {
			v.Warningf("AUTOGROUND: %v", err)
			return
		}

		loot.Entries = append(loot.Entries, l.Entries...)
	}

	if len(loot.Entries) > count {
		loot.Entries = loot.Entries[:count]
	}

	v.Infof("AUTOGROUND drop tc=%q ilvl=%d seed=%d entries=%d", tc, ilvl, seed64, len(loot.Entries))
	v.spawnLoot(loot, cx, cy, "autoground")
}

// autoGroundChests walks to each chest in turn, opens it (the same call a click
// ends in) and waits for the loot to appear.
func (v *Game) autoGroundChests() {
	a := &v.autoGround

	if a.chestN >= len(a.chests) {
		v.autoGroundNext(agPickup)
		return
	}

	ob := a.chests[a.chestN]

	switch {
	case ob.IsOpened():
		if a.timer > 1.5 { // let the opening animation and drop animations play
			a.chestN++
			a.timer = 0
		}
	case v.ground.chest == nil && a.timer > autoGroundStepWait:
		v.walkToObject(ob)
	case a.timer > autoGroundTimeout:
		v.Warningf("AUTOGROUND chest %q was not reached", ob.Label())
		v.ground.chest = nil
		a.chestN++
		a.timer = 0
		a.failed++
	}
}

// nearestGroundItem returns the closest item entity not in skip.
func (v *Game) nearestGroundItem(skip map[string]bool) *d2mapentity.Item {
	px, py := v.localPlayer.GetPositionF()

	var best *d2mapentity.Item

	bestD := math.MaxFloat64

	for _, e := range v.gameClient.MapEngine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok || skip[it.ID()] {
			continue
		}

		x, y := it.GetPositionF()
		if d := math.Hypot(px-x, py-y); d < bestD || (d == bestD && best != nil && it.ID() < best.ID()) {
			best, bestD = it, d
		}
	}

	return best
}

func (v *Game) autoGroundPickup() {
	a := &v.autoGround

	if os.Getenv("OD2_AUTOPICKUP") == "" {
		v.autoGroundNext(agFinish)
		return
	}

	if a.target == nil {
		a.target = v.nearestGroundItem(v.autoSkip())
		if a.target == nil {
			v.autoGroundNext(agFinish)
			return
		}

		a.timer = 0
		v.Infof("AUTOGROUND walk-to name=%q", plainLabel(a.target.Label()))
		v.walkToItem(a.target)

		return
	}

	t := a.target

	switch {
	case v.gameClient.MapEngine.Entities()[t.ID()] == nil: // picked up
		a.picked++
		a.target = nil

		if t.IsGold() {
			a.gold += t.Gold
			a.timer = 0

			return
		}

		a.timer = 0
		v.autoGroundNext(agPlace)
	case a.timer > autoGroundTimeout:
		v.Warningf("AUTOGROUND could not reach %q", plainLabel(t.Label()))
		v.autoSkipAdd(t.ID())

		a.target = nil
		a.failed++
		v.ground.item = nil
	}
}

func (v *Game) autoGroundPlace() {
	a := &v.autoGround

	if a.timer < autoGroundStepWait {
		return
	}

	it := v.gameControls.CursorItem()
	if it == nil {
		v.autoGroundNext(agPickup)
		return
	}

	name, quality := "", ""
	if ci, ok := it.(*diablo2item.Item); ok {
		name, quality = plainLabel(ci.Label()), ci.QualityName()
	}

	x, y, ok := v.gameControls.AutoPlaceCursor()
	if !ok {
		v.Infof("AUTOGROUND inventory-full name=%q quality=%s", name, quality)
		a.failed++
		v.autoGroundNext(agFinish)

		return
	}

	a.placed++
	v.Infof("AUTOGROUND placed name=%q quality=%s slot=(%d,%d) inventory_items=%d",
		name, quality, x, y, v.gameControls.InventoryItemCount())
	v.autoGroundNext(agPickup)
}

func (v *Game) autoGroundFinish() {
	a := &v.autoGround

	v.Infof("AUTOGROUND done picked=%d placed=%d gold_picked=%d gold_total=%d inventory_items=%d failed=%d",
		a.picked, a.placed, a.gold, v.localPlayer.Gold, v.gameControls.InventoryItemCount(), a.failed)

	a.phase = agDone
	a.timer = 0

	if hold, err := strconv.ParseFloat(os.Getenv("OD2_AUTOGROUND_HOLD"), 64); err == nil && hold > 0 {
		v.autoGround.holdUntil = a.elapsed + hold
		return
	}

	v.autoTestExit()
}

func (v *Game) autoSkip() map[string]bool { return v.autoGround.skip }

func (v *Game) autoSkipAdd(id string) {
	if v.autoGround.skip == nil {
		v.autoGround.skip = map[string]bool{}
	}

	v.autoGround.skip[id] = true
}
