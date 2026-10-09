package d2hero

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// PageEquipped is the Page of a StoredItem that is worn: its X is the body
// location as the .d2s numbers it (d2equip.Loc, physical: weapon set II in 11
// and 12, see HeroContainers.ActiveArms).
const PageEquipped = 0

const (
	statReqPercent     = 91  // item_req_percent
	statIndestructible = 152 // item_indesctructible
	activeArmsOffset   = 0x10
	classCodeCount     = 7
)

var classCodes = [classCodeCount]string{
	d2equip.ClassAmazon, d2equip.ClassSorceress, d2equip.ClassNecromancer, d2equip.ClassPaladin,
	d2equip.ClassBarbarian, d2equip.ClassDruid, d2equip.ClassAssassin,
}

// ClassCode returns the item class code (ama, sor, ...) of a charstats class id.
func ClassCode(id int) string {
	if id < 0 || id >= classCodeCount {
		return ""
	}

	return classCodes[id]
}

// EquipRules returns the equip rules and base item table (loaded from the game
// data on first use). ok is false when the tables are not available.
func (f *HeroStateFactory) EquipRules() (rules d2equip.Rules, bases d2equip.Bases, ok bool) {
	if f.equipTried {
		return f.equipRules, f.equipBases, f.equipRules.Types != nil
	}

	f.equipTried = true

	types, err := f.asset.LoadFile(itemTypesTxt)
	if err != nil {
		fmt.Printf("equip: ItemTypes unavailable, equip rules off: %v\n", err)
		return rules, nil, false
	}

	armor, aerr := f.asset.LoadFile(armorTxt)
	weapons, werr := f.asset.LoadFile(weaponsTxt)

	if aerr != nil || werr != nil {
		fmt.Printf("equip: armor/weapons unavailable, equip rules off\n")
		return rules, nil, false
	}

	misc, _ := f.asset.LoadFile(miscTxt) // rings, amulets, charms, quivers: a missing file only loses those

	t, err := d2equip.ParseTypes(types)
	if err != nil {
		fmt.Printf("equip: ItemTypes: %v\n", err)
		return rules, nil, false
	}

	b, err := d2equip.ParseBases(armor, weapons, misc)
	if err != nil {
		fmt.Printf("equip: base items: %v\n", err)
		return rules, nil, false
	}

	f.equipRules, f.equipBases = d2equip.Rules{Types: t}, b

	return f.equipRules, f.equipBases, true
}

// EquipItemOf describes a stored (worn) item for the equip rules.
func EquipItemOf(s *StoredItem, bases d2equip.Bases) d2equip.Item {
	b := bases[s.Code]
	it := d2equip.Item{
		Code: s.Code, Type: b.Type, Weapon: b.Weapon, TwoHanded: b.TwoHanded, OneOrTwo: b.OneOrTwo,
		Identified: s.Identified, Ethereal: s.Ethereal,
		ReqStr: b.ReqStr, ReqDex: b.ReqDex, ReqLevel: b.ReqLevel,
		MaxDurability: b.Durability, NoDurability: b.NoDurability,
	}

	it.Durability = it.MaxDurability

	if s.Durability != nil {
		it.Durability = *s.Durability
	}

	if d := s.D2S; d != nil {
		it.Identified = it.Identified || d.Identified
		it.Ethereal = it.Ethereal || d.Ethereal

		if d.MaxDurability > 0 {
			it.MaxDurability = int(d.MaxDurability)
			if s.Durability == nil {
				it.Durability = int(d.Durability)
			}
		}

		for _, list := range [][]d2s.Property{d.Properties, d.RunewordProperties} {
			for _, p := range list {
				switch p.ID {
				case statReqPercent:
					it.ReqPercent += int(p.Value)
				case statIndestructible:
					it.Indestructible = it.Indestructible || p.Value > 0
				}
			}
		}
	}

	return it
}

// LogicalLoc maps the physical body location of a stored item to the one the
// game treats it as given the active weapon set.
func (c *HeroContainers) LogicalLoc(x int) d2equip.Loc {
	return d2equip.EffectiveLoc(d2equip.Loc(x), c.ActiveArms)
}

// EquipStatus is what the activation pass decided about one worn item.
type EquipStatus struct {
	Loc    d2equip.Loc
	Code   string
	Active bool
	// Reason is why the item is off: a requirement, or "broken".
	Reason d2equip.Reason
	Detail string
}

// statItemOfStored converts a worn stored item to a stat list item at a logical location.
func (f *HeroStateFactory) statItemOfStored(s *StoredItem, loc d2equip.Loc, bases d2statlist.Bases, eq d2equip.Item) d2statlist.Item {
	var si d2statlist.Item

	if s.D2S != nil {
		cp := *s.D2S
		cp.Location = d2s.LocationEquipped
		si = statItem(&cp, bases, false)
	} else {
		b := bases[s.Code]
		si = d2statlist.Item{Code: s.Code, Ethereal: s.Ethereal, Weapon: b.Weapon, BaseBlock: b.BaseBlock}
		// UNVERIFIED: a generated item rolls minac..maxac, the middle stands in; the engine's
		// affixes do not flow into the stat list yet (only imported items carry their properties)
		si.Defense = (b.MinAC + b.MaxAC) / 2
	}

	si.Slot = int(loc)
	si.Broken = eq.Broken()

	return si
}

// ResolveEquipped runs the activation pass over the worn items of the
// containers for a hero (class code, level, allocated attributes, difficulty)
// and returns the stat items of the items that are on (broken ones are kept for
// the stat list, which ignores them) together with the verdict on each. The
// charms are the stat items of the inventory charms (their strength/dexterity
// bonuses count for the requirements). A worn item that fails its requirements
// stays on the body but gives nothing (0x55b9c0, VERIFIED).
func (f *HeroStateFactory) ResolveEquipped(c *HeroContainers, base d2statlist.Hero, charms []d2statlist.Item) (
	items []d2statlist.Item, status []EquipStatus) {
	rules, bases, ok := f.EquipRules()
	if !ok || c == nil {
		return nil, nil
	}

	statBases := f.loadStatBases()
	body := map[d2equip.Loc]*d2equip.Item{}
	stored := map[d2equip.Loc]*StoredItem{}
	eqItems := map[d2equip.Loc]d2equip.Item{}

	for i := range c.Equipped {
		s := &c.Equipped[i]
		loc := c.LogicalLoc(s.X)
		eq := EquipItemOf(s, bases)
		eqItems[loc] = eq
		body[loc] = &eq
		stored[loc] = s
	}

	statOf := map[d2equip.Loc]d2statlist.Item{}
	for loc, s := range stored {
		statOf[loc] = f.statItemOfStored(s, loc, statBases, eqItems[loc])
	}

	class := ClassCode(base.Class.ID)

	heroFn := func(active func(d2equip.Loc) bool) d2equip.Hero {
		list := append([]d2statlist.Item{}, charms...)

		for loc, si := range statOf {
			if active(loc) {
				list = append(list, si)
			}
		}

		t := d2statlist.Compute(base, list, nil)

		return d2equip.Hero{Class: class, Str: t.Str, Dex: t.Dex, Level: base.Level}
	}

	broken := func(loc d2equip.Loc) bool { return d2equip.PropertiesOff(body[loc]) }
	active := d2equip.Resolve(nil, body, broken, heroFn, rules.Types)
	hero := heroFn(func(l d2equip.Loc) bool { return active[l] })

	for loc := d2equip.Loc(1); loc < d2equip.NumLocs; loc++ {
		it := body[loc]
		if it == nil {
			continue
		}

		st := EquipStatus{Loc: loc, Code: it.Code, Active: active[loc], Reason: d2equip.ReasonOK}

		switch {
		case loc.IsSwap():
			st.Reason, st.Detail = "inactive weapon set", ""
		case broken(loc):
			st.Reason, st.Detail = d2equip.ReasonBroken, fmt.Sprintf("durability %d/%d", it.Durability, it.MaxDurability)
		case !active[loc]:
			req := d2equip.Check(hero, it, rules.Types)
			st.Reason, st.Detail = req.Reason(), req.Detail()
		}

		status = append(status, st)

		if active[loc] || broken(loc) || loc.IsSwap() {
			items = append(items, statOf[loc])
		}
	}

	return items, status
}

// EquipStatusLine formats the verdicts for logs.
func EquipStatusLine(status []EquipStatus) string {
	parts := make([]string, 0, len(status))

	for _, s := range status {
		state := "on"
		if !s.Active {
			state = "off(" + string(s.Reason) + ")"
		}

		parts = append(parts, fmt.Sprintf("%s=%s:%s", s.Loc, s.Code, state))
	}

	return strings.Join(parts, " ")
}

// activeArmsOf reads the weapon set a save is in (header field 0x10).
func activeArmsOf(data []byte) int {
	if len(data) < activeArmsOffset+4 {
		return 0
	}

	return int(binary.LittleEndian.Uint32(data[activeArmsOffset:]) & 1)
}

// importEquipped records the worn items of a save in the containers.
func importEquipped(c *HeroContainers, data []byte, items []d2s.Item, known func(string) bool) {
	c.EquippedSet = true
	c.ActiveArms = activeArmsOf(data)
	c.Equipped = []StoredItem{}

	for i := range items {
		it := &items[i]
		if it.Location != d2s.LocationEquipped || it.Ear || !known(trimCode(it.Code)) {
			continue
		}

		s, skip := storedFromD2S(it, PageEquipped, known)
		if skip != "" {
			continue
		}

		s.X, s.Y = int(it.Equipped), 0
		c.Equipped = append(c.Equipped, s)
	}
}

// exportEquipment writes the weapon set and the changes of worn items back
// into the item list of a save: the active weapon set (header 0x10) and, for
// every worn item that came from the save, its body location and durability.
// Items worn now that the save did not have in the same state (taken off,
// newly worn) are not written: moving items between the body and containers
// is the container export's job, so these are reported and the save keeps its
// original state for them.
func exportEquipment(c *d2s.Character, state *HeroState, warn func(string, ...interface{})) {
	cont := state.Containers
	if cont == nil || !cont.EquippedSet {
		return
	}

	binary.LittleEndian.PutUint32(c.Header.Raw[activeArmsOffset:], uint32(cont.ActiveArms&1))

	byID := map[uint32]*d2s.Item{}

	for i := range c.Items {
		if c.Items[i].Location == d2s.LocationEquipped {
			byID[c.Items[i].ID] = &c.Items[i]
		}
	}

	seen := map[uint32]bool{}

	for i := range cont.Equipped {
		s := &cont.Equipped[i]
		if s.D2S == nil {
			warn("worn item %q (slot %d) is not from the save, not written", s.Code, s.X)
			continue
		}

		it := byID[s.D2S.ID]
		if it == nil {
			warn("worn item %q (slot %d) was not worn in the save, not written", s.Code, s.X)
			continue
		}

		seen[s.D2S.ID] = true
		it.Equipped = uint8(s.X)

		if s.Durability != nil && it.MaxDurability > 0 {
			d := *s.Durability
			if d < 0 {
				d = 0
			}

			if d > int(it.MaxDurability) {
				d = int(it.MaxDurability)
			}

			it.Durability = uint16(d)
		}
	}

	for id, it := range byID {
		if !seen[id] {
			warn("%q was worn in the save but is not any more, kept worn in the file", strings.TrimSpace(it.Code))
		}
	}
}

// RecalcWorn recomputes the totals of st (a client's copy of the hero stats)
// from the worn items in c and returns the verdict on each item. It is the
// client side twin of the server's RecalcStats on save, and uses the same code.
func (f *HeroStateFactory) RecalcWorn(st *HeroStatsState, hero d2enum.Hero, c *HeroContainers) []EquipStatus {
	tmp := &HeroState{HeroType: hero, Stats: st, Containers: c, Difficulty: d2enum.DifficultyType(st.Difficulty)}

	f.lastEquipStatus = nil
	f.RecalcStats(tmp)

	return f.lastEquipStatus
}

// ClassCodeOfHero returns the item class code (ama, sor, ...) of a hero class.
func ClassCodeOfHero(h d2enum.Hero) string {
	for c, hero := range d2sClassToHero {
		if hero == h {
			return ClassCode(int(c))
		}
	}

	return ""
}
