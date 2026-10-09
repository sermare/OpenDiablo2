package d2player

import (
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// AutoEquip is the OD2_AUTOEQUIP scenario: it takes worn items off and puts them
// back, tries placements the rules forbid and logs every decision with its reason
// and the recalculated totals, as "EQUIP auto ..." lines. The hero's body is
// restored at the end. It uses the same code paths a click does.
func (g *GameControls) AutoEquip() {
	log := func(format string, a ...interface{}) { g.Infof("EQUIP auto "+format, a...) }

	if err := g.AutoPanel("inventory"); err != nil {
		log("inventory did not open: %v", err)
	}

	if _, _, ok := g.heroState.EquipRules(); !ok {
		log("equip rules unavailable (tables missing)")
		return
	}

	g.equipNoSave = true
	defer func() {
		g.equipNoSave = false
		g.saveHero()
	}()

	before := g.EquipSummary()
	start := g.totalsKey()

	log("begin %s", before)
	log("stats %s", g.statsLine())

	g.autoRoundTrip(log)
	g.autoWrongSlot(log)
	g.autoRequirements(log)
	g.autoHands(log)
	g.autoRings(log)
	g.autoSwap(log)
	g.autoDurability(log)

	after := g.EquipSummary()
	log("end %s", after)
	log("restored=%v totals_equal=%v", before == after, start == g.totalsKey())
	log("stats %s", g.statsLine())
}

func (g *GameControls) statsLine() string {
	t := g.hero.Stats.Totals
	if t == nil {
		return "no totals"
	}

	return fmt.Sprintf("def=%d ar=%d dmg=%d-%d life=%d mana=%d str=%d dex=%d", t.Defense, t.AttackRating, t.DamageMin,
		t.DamageMax, t.MaxLife, t.MaxMana, t.Str, t.Dex)
}

// totalsKey identifies the totals the body decides.
func (g *GameControls) totalsKey() string { return g.statsLine() }

func (g *GameControls) newItem(code string) *diablo2item.Item {
	it, err := g.inventory.item.NewItem(code)
	if err != nil {
		g.Infof("EQUIP auto cannot create %q: %v", code, err)

		return nil
	}

	it.Identify()

	return it
}

// autoRoundTrip takes up to three worn items off and puts them back.
func (g *GameControls) autoRoundTrip(log func(string, ...interface{})) {
	n := 0

	for _, loc := range drawnLocs {
		if n >= 3 {
			return
		}

		it := g.inventory.WornAt(loc)
		if it == nil || loc == d2equip.LocBelt {
			continue
		}

		n++

		key := g.totalsKey()

		if !g.TryUnequipToCursor(loc) {
			log("roundtrip slot=%s: could not take %s off", loc, it.GetItemCode())
			continue
		}

		log("roundtrip slot=%s took off %s -> %s", loc, it.GetItemCode(), g.statsLine())

		res := g.TryEquipCursor(loc)
		log("roundtrip slot=%s put back %s ok=%v restored=%v", loc, it.GetItemCode(), res.OK, key == g.totalsKey())
	}
}

// autoWrongSlot offers a worn item to a slot it does not fit.
func (g *GameControls) autoWrongSlot(log func(string, ...interface{})) {
	from := d2equip.LocHead
	to := d2equip.LocTorso

	it := g.inventory.WornAt(from)
	if it == nil {
		log("wrongslot skipped: nothing worn on the head")
		return
	}

	g.TryUnequipToCursor(from)

	res := g.TryEquipCursor(to)
	log("wrongslot item=%s slot=%s ok=%v reason=%q", it.GetItemCode(), to, res.OK, res.Reason)

	g.TryEquipCursor(from) // back where it was
}

// autoRequirements tries an item the hero is too weak for and one of another class.
func (g *GameControls) autoRequirements(log func(string, ...interface{})) {
	rules, bases, ok := g.heroState.EquipRules()
	if !ok || g.hero.Stats.Totals == nil {
		return
	}

	hero := g.hero.Stats.Totals

	var strong, other string

	codes := make([]string, 0, len(g.asset.Records.Item.All))
	for code := range g.asset.Records.Item.All {
		codes = append(codes, code)
	}

	sort.Strings(codes)

	best := 0

	for _, code := range codes {
		b := bases[code]
		if b.Type == "" {
			continue
		}

		if rules.FitsLoc(b.Type, d2equip.LocTorso) && rules.Types.ClassOf(b.Type) == "" && b.ReqStr > hero.Str && b.ReqStr > best {
			strong, best = code, b.ReqStr
		}

		if other == "" && rules.FitsLoc(b.Type, d2equip.LocHead) {
			if c := rules.Types.ClassOf(b.Type); c != "" && c != g.equipClass() {
				other = code
			}
		}
	}

	if strong == "" {
		log("strength skipped: no torso armor needs more than the hero's %d strength", hero.Str)
	} else if it := g.newItem(strong); it != nil {
		g.inventory.SetCursorItem(it)
		res := g.TryEquipCursor(d2equip.LocTorso)
		log("strength item=%s needs=%d have=%d ok=%v reason=%q detail=%q", strong, best, hero.Str, res.OK, res.Reason, res.Detail)
		g.inventory.SetCursorItem(nil)
	}

	if other != "" {
		if it := g.newItem(other); it != nil {
			g.inventory.SetCursorItem(it)
			res := g.TryEquipCursor(d2equip.LocHead)
			log("class item=%s class=%s hero=%s ok=%v reason=%q", other, rules.Types.ClassOf(bases[other].Type), g.equipClass(), res.OK, res.Reason)
			g.inventory.SetCursorItem(nil)
		}
	}

	// an unidentified item
	if it := g.newItem("cap"); it != nil {
		it.Unidentify()
		g.inventory.SetCursorItem(it)
		res := g.TryEquipCursor(d2equip.LocHead)
		log("unidentified item=cap ok=%v reason=%q", res.OK, res.Reason)
		g.inventory.SetCursorItem(nil)
	}
}

// autoHands exercises one-handed, two-handed and shield combinations.
func (g *GameControls) autoHands(log func(string, ...interface{})) {
	// the inventory of the sample hero is full: lift some items out for the moment
	orig := append([]InventoryItem{}, g.inventory.grid.items...)
	freed := g.freeInventorySpace(8)

	savedR, savedL := g.inventory.WornAt(d2equip.LocRightHand), g.inventory.WornAt(d2equip.LocLeftHand)
	g.inventory.SetWorn(d2equip.LocRightHand, nil)
	g.inventory.SetWorn(d2equip.LocLeftHand, nil)

	put := func(code string, loc d2equip.Loc) {
		it := g.newItem(code)
		if it == nil {
			return
		}

		g.inventory.SetCursorItem(it)
		res := g.TryEquipCursor(loc)
		log("hands item=%s slot=%s ok=%v reason=%q displaced=%v rhand=%s lhand=%s", code, res.Loc, res.OK, res.Reason,
			res.Displaced, codeOf(g.inventory.WornAt(d2equip.LocRightHand)), codeOf(g.inventory.WornAt(d2equip.LocLeftHand)))

		if g.inventory.CursorItem() != nil { // refused or swapped out: put it in the inventory
			g.inventory.AutoPlaceCursor()
		}
	}

	put("buc", d2equip.LocLeftHand)  // a shield
	put("wnd", d2equip.LocRightHand) // a one-handed weapon next to it
	put("sst", d2equip.LocRightHand) // a two-handed staff pushes the shield out
	put("buc", d2equip.LocLeftHand)  // the shield pushes the staff out
	put("wnd", d2equip.LocLeftHand)  // a weapon in the off hand
	put("sbw", d2equip.LocRightHand) // a bow (two-handed)

	// leave the body as it was
	g.inventory.SetWorn(d2equip.LocRightHand, savedR)
	g.inventory.SetWorn(d2equip.LocLeftHand, savedL)
	g.inventory.SetCursorItem(nil)

	// the test items that were put into the inventory go again, then the lifted ones come back
	for _, it := range append([]InventoryItem{}, g.inventory.grid.items...) {
		if !containsItem(orig, it) {
			g.inventory.grid.Remove(it)
		}
	}

	g.restoreInventory(freed) // the charms are among them: they count for the totals
	g.recalcWorn()
}

// autoRings puts rings on both ring slots and into a wrong one.
func (g *GameControls) autoRings(log func(string, ...interface{})) {
	savedA, savedB := g.inventory.WornAt(d2equip.LocRightRing), g.inventory.WornAt(d2equip.LocLeftRing)
	savedN := g.inventory.WornAt(d2equip.LocNeck)

	g.inventory.SetWorn(d2equip.LocRightRing, nil)
	g.inventory.SetWorn(d2equip.LocLeftRing, nil)
	g.inventory.SetWorn(d2equip.LocNeck, nil)

	for _, step := range []struct {
		code string
		loc  d2equip.Loc
	}{
		{"rin", d2equip.LocRightRing}, {"rin", d2equip.LocLeftRing}, {"rin", d2equip.LocNeck}, {"amu", d2equip.LocNeck},
		{"amu", d2equip.LocRightRing},
	} {
		it := g.newItem(step.code)
		if it == nil {
			continue
		}

		g.inventory.SetCursorItem(it)
		res := g.TryEquipCursor(step.loc)
		log("rings item=%s slot=%s ok=%v reason=%q", step.code, step.loc, res.OK, res.Reason)
		g.inventory.SetCursorItem(nil)
	}

	g.inventory.SetWorn(d2equip.LocRightRing, savedA)
	g.inventory.SetWorn(d2equip.LocLeftRing, savedB)
	g.inventory.SetWorn(d2equip.LocNeck, savedN)
	g.recalcWorn()
}

// autoSwap switches the weapon sets with W and back.
func (g *GameControls) autoSwap(log func(string, ...interface{})) {
	// give set II a weapon so the switch shows something
	oldII := g.inventory.WornAt(d2equip.LocSwapRight)
	if w := g.newItem("wnd"); oldII == nil && w != nil {
		g.inventory.SetWorn(d2equip.LocSwapRight, w)
	}
	g.recalcWorn()

	a := g.statsLine()
	g.SwapWeapons()
	log("swap to set II active_arms=%d rhand=%s stats=%s", g.inventory.ActiveArms(), codeOf(g.inventory.WornAt(d2equip.LocRightHand)), g.statsLine())

	c := g.snapshotContainers()
	log("swap persisted active_arms=%d worn=%d", c.ActiveArms, len(c.Equipped))

	g.SwapWeapons()
	log("swap back active_arms=%d same_stats=%v", g.inventory.ActiveArms(), a == g.statsLine())

	g.inventory.SetWorn(d2equip.LocSwapRight, oldII)
	g.recalcWorn()
}

type liftedItem struct {
	item InventoryItem
	x, y int
}

// freeInventorySpace takes n items out of the inventory grid (returning them for restoreInventory).
func (g *GameControls) freeInventorySpace(n int) []liftedItem {
	var out []liftedItem

	items := append([]InventoryItem{}, g.inventory.grid.items...)
	for i := len(items) - 1; i >= 0 && len(out) < n; i-- {
		x, y := items[i].InventoryGridSlot()
		out = append(out, liftedItem{items[i], x, y})
		g.inventory.grid.Remove(items[i])
	}

	return out
}

func (g *GameControls) restoreInventory(items []liftedItem) {
	for _, l := range items {
		if g.inventory.grid.Set(l.x, l.y, l.item) != nil {
			g.inventory.grid.AutoPlaceForPickup(l.item)
		}
	}
}

// autoDurability simulates monster hits on the hero (OnHeroHit) and breaks a piece
// of armor on purpose to show that its properties go off; durabilities and totals
// are restored afterwards. The chance is the real one unless OD2_DURABILITY_CHANCE is set.
func (g *GameControls) autoDurability(log func(string, ...interface{})) {
	type saved struct {
		item *diablo2item.Item
		cur  int
	}

	var keep []saved

	counts := map[d2equip.Loc]int{}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it, ok := g.inventory.WornAt(l).(*diablo2item.Item); ok {
			if cur, maxDur := it.Durability(); maxDur > 0 {
				keep = append(keep, saved{it, cur})
			}
		}
	}

	const hits = 60

	for i := 0; i < hits; i++ {
		before := map[d2equip.Loc]int{}

		for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
			if it, ok := g.inventory.WornAt(l).(*diablo2item.Item); ok {
				before[l], _ = it.Durability()
			}
		}

		g.OnHeroHit()

		for l, b := range before {
			if cur, _ := g.inventory.WornAt(l).(*diablo2item.Item).Durability(); cur != b {
				counts[l]++
			}
		}
	}

	log("durability simulated_hits=%d losses_by_slot=%v", hits, counts)

	// break the torso piece
	torso, ok := g.inventory.WornAt(d2equip.LocTorso).(*diablo2item.Item)
	if !ok {
		return
	}

	_, maxDur := torso.Durability()
	torso.SetDurability(1)
	g.recalcWorn()

	withDef := g.statsLine()

	for i := 0; i < 500; i++ {
		if cur, _ := torso.Durability(); cur == 0 {
			break
		}

		g.OnHeroHit()
	}

	cur, _ := torso.Durability()
	log("durability break item=%s dur=%d/%d broken=%v before=[%s] after=[%s]", torso.GetItemCode(), cur, maxDur, cur == 0,
		withDef, g.statsLine())

	for _, k := range keep {
		k.item.SetDurability(k.cur)
	}

	g.recalcWorn()
}

func containsItem(list []InventoryItem, it InventoryItem) bool {
	for _, x := range list {
		if x == it {
			return true
		}
	}

	return false
}
