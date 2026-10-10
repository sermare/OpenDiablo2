package d2player

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// This file enforces the equip rules (d2common/d2equip) on the hero's body:
// clicking an item onto a slot, picking it off, the W weapon switch, the
// recalculation of the hero's totals after each change and the durability loss
// in combat. All of it logs "EQUIP" / "DURABILITY" lines for the autotests.

// Sounds for refusals and breaks (Sounds.txt handles). Which sound the game plays
// for which refusal is UNVERIFIED: the voice lines exist per class, the binary was
// not searched for their use.
const (
	soundBroken     = "ESOUND_CURSOR_DURABILITY_BREAK"
	soundCantUseFmt = "ESOUND_%s_CANTUSEYET"
	soundCantCarry  = "ESOUND_%s_CANTCARRY_1"
	envDurChance    = "OD2_DURABILITY_CHANCE" // test override of the loss chance in percent
	lowDurPct       = 20                      // UNVERIFIED: tooltip turns red at or below this share of the maximum
)

var voiceNames = map[string]string{
	d2equip.ClassAmazon: "AMAZON", d2equip.ClassSorceress: "SORCERESS", d2equip.ClassNecromancer: "NECROMANCER",
	d2equip.ClassPaladin: "PALADIN", d2equip.ClassBarbarian: "BARBARIAN", d2equip.ClassDruid: "DRUID",
	d2equip.ClassAssassin: "ASSASSIN",
}

// EquipResult is the verdict on putting an item at a body location.
type EquipResult struct {
	OK        bool
	Loc       d2equip.Loc // the location it applies to (may differ from the one asked for)
	Reason    d2equip.Reason
	Detail    string
	Displaced []d2equip.Loc
}

func (g *GameControls) equipClass() string { return d2hero.ClassCodeOfHero(g.hero.Class) }

// SetEquipSound sets the function that plays a Sounds.txt handle for the hero.
func (g *GameControls) SetEquipSound(f func(handle string)) { g.equipSound = f }

func (g *GameControls) playEquipSound(handle string) {
	if g.equipSound != nil {
		g.equipSound(handle)
	}
}

// storedOf describes a worn or carried item as a stored item.
func (g *GameControls) storedOf(it InventoryItem, loc d2equip.Loc) (d2hero.StoredItem, bool) {
	item, ok := it.(*diablo2item.Item)
	if !ok {
		return d2hero.StoredItem{}, false
	}

	s := storedFromItem(item, d2hero.PageEquipped, int(d2equip.EffectiveLoc(loc, g.inventory.activeArms)), 0, g.itemOrigin[it])

	return s, true
}

// snapshotEquipped lists everything worn as stored items (file locations).
func (g *GameControls) snapshotEquipped() []d2hero.StoredItem {
	var out []d2hero.StoredItem

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it := g.inventory.WornAt(l); it != nil {
			if s, ok := g.storedOf(it, l); ok {
				out = append(out, s)
			}
		}
	}

	return out
}

// equipOn says whether the worn list is the truth for the hero's stats: the save
// had one, or the player changed something.
func (g *GameControls) equipOn() bool {
	return g.equipTouched || (g.hero.Containers != nil && g.hero.Containers.EquippedSet)
}

// loadEquipment fills the body from the hero's saved worn items.
func (g *GameControls) loadEquipment() {
	c := g.hero.Containers
	if c == nil || !c.EquippedSet {
		return
	}

	g.inventory.ClearBody()
	g.inventory.activeArms = c.ActiveArms

	loaded := 0

	for i := range c.Equipped {
		s := &c.Equipped[i]

		item, err := realiseStored(g.inventory.item, s)
		if err != nil {
			g.Warningf("skipping worn item %q (slot %d): %v", s.Code, s.X, err)
			continue
		}

		if s.D2S != nil {
			g.itemOrigin[item] = s.D2S
		}

		g.inventory.SetWorn(c.LogicalLoc(s.X), item)

		loaded++
	}

	g.Infof("equipment loaded: worn=%d active_arms=%d", loaded, c.ActiveArms)
}

// wornEquipItems returns the worn items as equip rule items by logical location.
func (g *GameControls) wornEquipItems(skip d2equip.Loc, bases d2equip.Bases) map[d2equip.Loc]*d2equip.Item {
	out := map[d2equip.Loc]*d2equip.Item{}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if l == skip {
			continue
		}

		if it := g.inventory.WornAt(l); it != nil {
			if s, ok := g.storedOf(it, l); ok {
				e := d2hero.EquipItemOf(&s, bases)
				out[l] = &e
			}
		}
	}

	return out
}

// evaluateEquip decides whether item may be put at loc. The structural rules
// (body location, class, hands) come from d2equip.Rules.Place with a hero strong
// enough for anything; the requirements are then checked by recalculating the
// hero's totals with the item in place, because the item it replaces (and
// bonuses that other worn items give) change what the hero has.
func (g *GameControls) evaluateEquip(item InventoryItem, loc d2equip.Loc) EquipResult {
	res := EquipResult{Loc: loc}

	rules, bases, ok := g.heroState.EquipRules()
	if !ok {
		res.OK, res.Reason = true, d2equip.ReasonOK

		return res
	}

	s, isItem := g.storedOf(item, loc)
	if !isItem {
		res.Reason, res.Detail = d2equip.ReasonBodyLoc, "not an item that can be worn"

		return res
	}

	eq := d2hero.EquipItemOf(&s, bases)
	body := g.wornEquipItems(loc, bases)

	const plenty = 9999

	d := rules.Place(d2equip.Hero{Class: g.equipClass(), Str: plenty, Dex: plenty, Level: plenty}, body, &eq, loc)
	res.Reason, res.Detail, res.Displaced = d.Reason, d.Detail, d.Displaced

	if !d.OK {
		return res
	}

	// requirements, with the replaced and the displaced items out of the way
	gone := map[d2equip.Loc]bool{loc: true}
	for _, l := range d.Displaced {
		gone[l] = true
	}

	var worn []d2hero.StoredItem

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it := g.inventory.WornAt(l); it != nil && !gone[l] {
			if w, okw := g.storedOf(it, l); okw {
				worn = append(worn, w)
			}
		}
	}

	worn = append(worn, s)

	c := g.snapshotContainers()
	c.Equipped, c.EquippedSet, c.ActiveArms = worn, true, g.inventory.activeArms
	st := *g.hero.Stats

	for _, v := range g.heroState.RecalcWorn(&st, g.hero.Class, c) {
		if v.Loc == loc && !v.Active {
			res.Reason, res.Detail = v.Reason, v.Detail

			return res
		}
	}

	res.OK, res.Reason, res.Detail = true, d2equip.ReasonOK, ""

	return res
}

func isRequirementReason(r d2equip.Reason) bool {
	switch r {
	case d2equip.ReasonStrength, d2equip.ReasonDexterity, d2equip.ReasonLevel, d2equip.ReasonClass, d2equip.ReasonUnidentified:
		return true
	}

	return false
}

func (g *GameControls) refuseEquip(item InventoryItem, res EquipResult) {
	g.Infof("EQUIP decision=refused item=%s slot=%s reason=%q detail=%q", item.GetItemCode(), res.Loc, res.Reason, res.Detail)

	if name, ok := voiceNames[g.equipClass()]; ok && isRequirementReason(res.Reason) {
		g.playEquipSound(fmt.Sprintf(soundCantUseFmt, name))
	}
}

// TryEquipCursor puts the cursor item at loc following the rules. The item it
// replaces goes to the cursor; items a two-handed weapon (or a shield) pushes out
// go to the inventory. A refusal leaves everything as it was.
func (g *GameControls) TryEquipCursor(loc d2equip.Loc) EquipResult {
	item := g.inventory.CursorItem()
	if item == nil {
		return EquipResult{Loc: loc, Reason: d2equip.ReasonBodyLoc, Detail: "nothing on the cursor"}
	}

	res := g.evaluateEquip(item, loc)

	// the game offers the other slot an item fits: a weapon that cannot go to the off hand
	// goes to the weapon hand, ammunition to the off hand (UNVERIFIED, FUN_0055b500 picks
	// between the two body locations of the type)
	if !res.OK {
		alt := d2equip.LocNone

		switch {
		case loc == d2equip.LocLeftHand && res.Reason == d2equip.ReasonDualWield:
			alt = d2equip.LocRightHand
		case loc == d2equip.LocRightHand && res.Reason == d2equip.ReasonBodyLoc:
			alt = d2equip.LocLeftHand // ammunition: BodyLoc1/2 are both hands
		case loc == d2equip.LocLeftRing:
			alt = d2equip.LocRightRing
		case loc == d2equip.LocRightRing:
			alt = d2equip.LocLeftRing
		}

		if alt != d2equip.LocNone {
			if r2 := g.evaluateEquip(item, alt); r2.OK {
				res = r2
			}
		}
	}

	if !res.OK {
		g.refuseEquip(item, res)

		return res
	}

	loc = res.Loc
	old := g.inventory.WornAt(loc)

	var moved []InventoryItem

	for _, d := range res.Displaced {
		it := g.inventory.WornAt(d)
		if it == nil {
			continue
		}

		if !g.inventory.grid.CanAutoPlace(it, true) {
			res.OK, res.Reason, res.Detail = false, d2equip.ReasonNoRoom, "no room in the inventory for "+it.GetItemCode()
			g.Infof("EQUIP decision=refused item=%s slot=%s reason=%q detail=%q", item.GetItemCode(), loc, res.Reason, res.Detail)

			if name, ok := voiceNames[g.equipClass()]; ok {
				g.playEquipSound(fmt.Sprintf(soundCantCarry, name))
			}

			return res
		}

		moved = append(moved, it)
	}

	for i, d := range res.Displaced {
		g.inventory.SetWorn(d, nil)

		if i < len(moved) {
			g.inventory.grid.AutoPlaceForPickup(moved[i])
		}
	}

	g.inventory.SetWorn(loc, item)
	g.inventory.SetCursorItem(old)

	verdict := g.afterEquipChange(fmt.Sprintf("equip item=%s slot=%s replaced=%s displaced=%v", item.GetItemCode(), loc, codeOf(old), res.Displaced))
	g.Infof("EQUIP decision=ok item=%s slot=%s %s", item.GetItemCode(), loc, verdict)

	return res
}

func codeOf(it InventoryItem) string {
	if it == nil {
		return "none"
	}

	return it.GetItemCode()
}

// TryUnequipToCursor takes the item at loc to the cursor (the cursor must be empty).
func (g *GameControls) TryUnequipToCursor(loc d2equip.Loc) bool {
	it := g.inventory.WornAt(loc)
	if it == nil || g.inventory.CursorItem() != nil {
		return false
	}

	g.inventory.SetWorn(loc, nil)
	g.inventory.SetCursorItem(it)

	verdict := g.afterEquipChange(fmt.Sprintf("unequip item=%s slot=%s", it.GetItemCode(), loc))
	g.Infof("EQUIP decision=ok unequip item=%s slot=%s %s", it.GetItemCode(), loc, verdict)

	return true
}

// SwapWeapons switches the weapon set (the W key).
func (g *GameControls) SwapWeapons() {
	g.inventory.SwapWeaponSets()
	g.swapSkillSets()

	verdict := g.afterEquipChange("swap")
	g.Infof("EQUIP decision=ok swap active_arms=%d rhand=%s lhand=%s %s", g.inventory.activeArms,
		codeOf(g.inventory.WornAt(d2equip.LocRightHand)), codeOf(g.inventory.WornAt(d2equip.LocLeftHand)), verdict)
}

// afterEquipChange recalculates the hero's totals from the body and saves.
func (g *GameControls) afterEquipChange(what string) string {
	g.equipTouched = true
	line := g.recalcWorn()

	if !g.equipNoSave {
		g.saveHero()
	}

	return line
}

// recalcWorn recomputes the hero's totals from the worn items (the same code the
// server runs when the hero is saved) and returns the verdicts and the totals.
func (g *GameControls) recalcWorn() string {
	c := g.snapshotContainers()
	g.hero.Containers = c

	status := g.heroState.RecalcWorn(g.hero.Stats, g.hero.Class, c)
	g.equipStatus = map[d2equip.Loc]d2hero.EquipStatus{}

	for _, s := range status {
		g.equipStatus[s.Loc] = s
	}

	return fmt.Sprintf("items=[%s] stats: %s sets=[%s]", d2hero.EquipStatusLine(status), d2hero.StatsSummary(g.hero.Stats),
		g.setSummary())
}

// EquipClick handles a left click on a body slot of the open inventory. It
// reports whether the click was on a slot.
func (g *GameControls) EquipClick(mx, my int) bool {
	if !g.inventory.IsOpen() {
		return false
	}

	loc, ok := g.inventory.LocAt(mx, my)
	if !ok {
		return false
	}

	if g.inventory.CursorItem() == nil {
		g.TryUnequipToCursor(loc)
	} else {
		g.TryEquipCursor(loc)
	}

	return true
}

// ---- durability -------------------------------------------------------------

func (g *GameControls) durabilityRand() *rand.Rand {
	if g.equipRand == nil {
		g.equipRand = rand.New(rand.NewSource(int64(len(g.hero.Name())) * 7919)) //nolint:gosec // not security relevant
	}

	return g.equipRand
}

// chance returns the loss chance in percent, honouring the test override.
func chanceOr(def int) int {
	if v, err := strconv.Atoi(os.Getenv(envDurChance)); err == nil && v >= 0 {
		return v
	}

	return def
}

// OnHeroHit rolls the durability loss of the armor when the hero was hit.
func (g *GameControls) OnHeroHit() {
	_, bases, ok := g.heroState.EquipRules()
	if !ok || g.inventory == nil {
		return
	}

	armor := g.wornEquipItems(d2equip.LocNone, bases)
	candidate := func(l d2equip.Loc) bool {
		it := armor[l]

		return it != nil && !it.Weapon && !l.IsSwap()
	}

	rng := g.durabilityRand()

	total := d2equip.TotalArmorWeight(candidate)
	if total == 0 {
		return
	}

	loc, found := d2equip.PickArmorPiece(rng.Intn(len(d2equip.ArmorWeights)), rng.Intn(total), candidate)
	if !found {
		return
	}

	g.rollItemLoss(loc, "hit", chanceOr(d2equip.ChanceArmor), rng.Intn(100))
}

// OnHeroStrike rolls the durability loss of the weapon after a hit of the hero.
func (g *GameControls) OnHeroStrike() {
	if g.inventory == nil {
		return
	}

	if it := g.inventory.WornAt(d2equip.LocRightHand); it != nil {
		g.rollItemLoss(d2equip.LocRightHand, "strike", chanceOr(d2equip.ChanceWeapon), g.durabilityRand().Intn(100))
	}
}

// WearWeapon is the Impale wear (ITEM_ReduceDurabilityOrConsumeOnSkillUse): chance percent to lose amount
// points of the right-hand weapon's durability.
func (g *GameControls) WearWeapon(chance, amount int) {
	if g.inventory == nil {
		return
	}

	it, ok := g.inventory.WornAt(d2equip.LocRightHand).(*diablo2item.Item)
	if !ok {
		return
	}

	cur, maxDur := it.Durability()
	if maxDur == 0 {
		return
	}

	eq := d2equip.Item{MaxDurability: maxDur, Durability: cur, Indestructible: it.IsIndestructible()}

	dur, broke, lost := d2equip.RollWear(&eq, g.durabilityRand().Intn(100), chance, amount)
	if !lost {
		return
	}

	it.SetDurability(dur)
	g.Infof("DURABILITY skill item=%s %d->%d/%d chance=%d amount=%d", it.GetItemCode(), cur, dur, maxDur, chance, amount)

	if broke {
		g.playEquipSound(soundBroken)
	}

	g.equipTouched = true
	g.Infof("DURABILITY recalc %s", g.recalcWorn())

	if !g.equipNoSave {
		g.saveHero()
	}
}

func (g *GameControls) rollItemLoss(loc d2equip.Loc, why string, chance, roll int) {
	it, ok := g.inventory.WornAt(loc).(*diablo2item.Item)
	if !ok {
		return
	}

	cur, maxDur := it.Durability()
	if maxDur == 0 {
		return
	}

	eq := d2equip.Item{MaxDurability: maxDur, Durability: cur, Indestructible: it.IsIndestructible()}

	dur, lost := d2equip.RollLoss(&eq, roll, chance)
	if !lost {
		return
	}

	it.SetDurability(dur)
	g.Infof("DURABILITY %s item=%s slot=%s %d->%d/%d chance=%d", why, it.GetItemCode(), loc, cur, dur, maxDur, chance)

	if dur == 0 {
		g.Infof("DURABILITY broken item=%s slot=%s", it.GetItemCode(), loc)
		g.playEquipSound(soundBroken)
	}

	g.equipTouched = true
	g.Infof("DURABILITY recalc %s", g.recalcWorn())

	if !g.equipNoSave {
		g.saveHero()
	}
}

// ---- tooltips ---------------------------------------------------------------

// itemTooltipLines adds the durability line (red when low or broken) and a red
// note when the hero cannot use the item.
func (g *GameControls) itemTooltipLines(it InventoryItem) []string {
	var lines []string

	item, ok := it.(*diablo2item.Item)
	if !ok {
		return nil
	}

	if cur, maxDur := item.Durability(); maxDur > 0 {
		label := g.asset.TranslateString("ItemStats1d")
		if label == "" || label == "ItemStats1d" {
			label = "Durability:"
		}

		text := fmt.Sprintf("%s %d of %d", label, cur, maxDur)

		color := d2ui.ColorTokenWhite
		if cur*100 <= maxDur*lowDurPct {
			color = d2ui.ColorTokenRed
		}

		lines = append(lines, d2ui.ColorTokenize(text, color))
	}

	for loc, st := range g.equipStatus {
		if g.inventory.WornAt(loc) == it && !st.Active && st.Reason != "inactive weapon set" {
			lines = append(lines, d2ui.ColorTokenize(fmt.Sprintf("Not active: %s", st.Detail), d2ui.ColorTokenRed))
		}
	}

	if note := g.requirementNote(item); note != "" {
		lines = append(lines, d2ui.ColorTokenize(note, d2ui.ColorTokenRed))
	}

	return append(lines, g.setTooltipLines(item)...)
}

// setTooltipLines lists the set of a set item with the pieces the hero wears
// and the bonuses that are active (nothing for other items).
func (g *GameControls) setTooltipLines(item *diablo2item.Item) []string {
	row := item.SetRow()
	if row == 0 {
		return nil
	}

	info, ok := g.inventory.item.SetInfo(row)
	if !ok {
		return nil
	}

	worn := map[string]bool{}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if w, isItem := g.inventory.WornAt(l).(*diablo2item.Item); isItem && w.SetRow() == row {
			worn[w.SetPieceName()] = true
		}
	}

	return diablo2item.SetTooltipLines(info, worn)
}

// setSummary is the log text of the set bonuses the worn pieces switch on.
func (g *GameControls) setSummary() string {
	t := g.hero.Stats.Totals
	if t == nil || len(t.SetPieces) == 0 {
		return "none"
	}

	var parts []string

	for row, n := range t.SetPieces {
		info, ok := g.inventory.item.SetInfo(row)
		if !ok {
			continue
		}

		partial := 0
		for i := range info.Partial {
			if n >= i+2 && len(info.Partial[i]) > 0 {
				partial++
			}
		}

		parts = append(parts, fmt.Sprintf("%s:%d/%d partial_tiers=%d full=%v", info.Name, n, len(info.Pieces), partial,
			len(info.Pieces) > 0 && n >= len(info.Pieces)))
	}

	sort.Strings(parts)

	return strings.Join(parts, "; ")
}

// requirementNote explains what the hero lacks for an item (empty if nothing).
func (g *GameControls) requirementNote(item *diablo2item.Item) string {
	rules, bases, ok := g.heroState.EquipRules()
	if !ok || g.hero.Stats == nil || g.hero.Stats.Totals == nil {
		return ""
	}

	s, isItem := g.storedOf(item, d2equip.LocNone)
	if !isItem {
		return ""
	}

	eq := d2hero.EquipItemOf(&s, bases)
	if eq.Type == "" {
		return ""
	}

	t := g.hero.Stats.Totals
	req := d2equip.Check(d2equip.Hero{Class: g.equipClass(), Str: t.Str, Dex: t.Dex, Level: g.hero.Stats.Level}, &eq, rules.Types)

	if req.OK() {
		return ""
	}

	return "You cannot use this item: " + req.Detail()
}

// ---- helpers for the autotest ----------------------------------------------

// EquipSummary lists the body for logs.
func (g *GameControls) EquipSummary() string {
	var parts []string

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it := g.inventory.WornAt(l); it != nil {
			part := fmt.Sprintf("%s=%s", l, it.GetItemCode())
			if i, ok := it.(*diablo2item.Item); ok {
				if cur, maxDur := i.Durability(); maxDur > 0 {
					part += fmt.Sprintf("(%d/%d)", cur, maxDur)
				}
			}

			parts = append(parts, part)
		}
	}

	return fmt.Sprintf("active_arms=%d worn=[%s]", g.inventory.activeArms, strings.Join(parts, " "))
}
