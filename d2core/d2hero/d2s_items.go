package d2hero

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
)

// Slots of Item.Equipped in a .d2s that the inventory panel shows.
const (
	d2sSlotNeck      = 2
	d2sSlotRightRing = 6
	d2sSlotLeftRing  = 7
	d2sSlotBelt      = 8
	d2sSlotAltRight  = 11 // weapon set II
	d2sSlotAltLeft   = 12
)

// ImportedItem is an item of a .d2s save in the form the inventory UI needs: the
// arguments for diablo2item.ItemFactory.NewItem (base code first, then the unique,
// set or magic affix names) and where the save keeps it. The values of an affix
// are not copied from the save: the UI rolls them from the affix ranges, so tooltips
// show the right item names and base stats but only approximate magic values.
type ImportedItem struct {
	Codes      []string `json:"codes"`
	Location   uint8    `json:"location"`           // d2s.LocationStored, LocationEquipped, LocationBelt...
	Equipped   uint8    `json:"equipped,omitempty"` // d2s equipment slot (1 head ... 12 weapon set II left)
	Page       uint8    `json:"page,omitempty"`     // 1 inventory, 4 cube, 5 stash
	X          uint8    `json:"x"`
	Y          uint8    `json:"y"`
	Quality    uint8    `json:"quality"`
	Identified bool     `json:"identified,omitempty"`
	Ethereal   bool     `json:"ethereal,omitempty"`
	Sockets    uint8    `json:"sockets,omitempty"`
	Durability uint16   `json:"durability,omitempty"`
	MaxDur     uint16   `json:"maxDurability,omitempty"`
	Quantity   uint16   `json:"quantity,omitempty"`
	// AutoPlace asks the inventory to put the item in the first free cell (X and Y are unset).
	AutoPlace bool `json:"autoPlace,omitempty"`
}

// affixNames maps the ids a .d2s stores to the names in the game tables.
type affixNames struct {
	unique, set, prefix, suffix []string
}

// loadAffixNames reads the id-ordered name columns of the item tables. The ids of
// a save are row numbers; the "Expansion" separator row of UniqueItems and
// SetItems is not counted (the d2txt reader skips it), blank rows of the magic
// affix tables are (verified against the sample save for unique, set and magic).
func (f *HeroStateFactory) loadAffixNames() *affixNames {
	read := func(path, column string) []string {
		data, err := f.asset.LoadFile(path)
		if err != nil {
			return nil
		}

		d := d2txt.LoadDataDictionary(data)

		var names []string

		for d.Next() {
			names = append(names, d.String(column))
		}

		return names
	}

	return &affixNames{
		unique: read(d2resource.UniqueItems, "index"),
		set:    read(d2resource.SetItems, "index"),
		prefix: read(d2resource.MagicPrefix, "Name"),
		suffix: read(d2resource.MagicSuffix, "Name"),
	}
}

func nameAt(list []string, id int) string {
	if id <= 0 || id >= len(list) {
		return ""
	}

	return list[id]
}

// itemCodes returns the NewItem arguments for an item, or nil when its base
// code is unknown to the game tables (ears, modded items).
func (f *HeroStateFactory) itemCodes(it *d2s.Item, names *affixNames) []string {
	code := strings.TrimSpace(it.Code)
	if f.asset.Records.Item.All[code] == nil {
		return nil
	}

	codes := []string{code}

	switch it.Quality {
	case d2s.QualityMagic:
		if n := nameAt(names.prefix, int(it.MagicPrefix)); n != "" {
			codes = append(codes, n)
		}

		if n := nameAt(names.suffix, int(it.MagicSuffix)); n != "" {
			codes = append(codes, n)
		}
	case d2s.QualityUnique:
		if n := nameAt(names.unique, int(it.UniqueID)); n != "" && f.asset.Records.Item.Unique[n] != nil {
			codes = append(codes, n)
		}
	case d2s.QualitySet:
		if n := nameAt(names.set, int(it.SetID)); n != "" && f.asset.Records.Item.SetItems[n] != nil {
			codes = append(codes, n)
		}
	}

	return codes
}

// importedItems converts every top-level item of a save.
func (f *HeroStateFactory) importedItems(items []d2s.Item) []ImportedItem {
	names := f.loadAffixNames()
	out := make([]ImportedItem, 0, len(items))

	for i := range items {
		it := &items[i]

		codes := f.itemCodes(it, names)
		if codes == nil {
			continue
		}

		out = append(out, ImportedItem{
			Codes:      codes,
			Location:   it.Location,
			Equipped:   it.Equipped,
			Page:       it.Page,
			X:          it.X,
			Y:          it.Y,
			Quality:    it.Quality,
			Identified: it.Identified,
			Ethereal:   it.Ethereal,
			Sockets:    it.TotalSockets,
			Durability: it.Durability,
			MaxDur:     it.MaxDurability,
			Quantity:   it.Quantity,
		})
	}

	return out
}

// starterSlots names the location column of charstats.txt: LoD writes tokens,
// the classic file the slot numbers of a save.
var starterSlots = map[string]int{
	"head": 1, "neck": 2, "tors": 3, "rarm": 4, "larm": 5, "rrin": 6, "lrin": 7, "belt": 8, "feet": 9, "glov": 10,
}

// starterSlot returns the d2s equipment slot of a starting item, 0 for the inventory.
func starterSlot(loc string) int {
	loc = strings.ToLower(strings.TrimSpace(loc))
	if n, ok := starterSlots[loc]; ok {
		return n
	}

	n, err := strconv.Atoi(loc)
	if err != nil || n < 0 || n > 12 {
		return 0
	}

	return n
}

// starterItems lists the items a new character of the class starts with, from the
// item1..item10 columns of charstats.txt: a location naming a d2s equipment slot
// (1..12) puts the item on the body, anything else in the inventory; count repeats a non-stacking item (the starting potions).
func (f *HeroStateFactory) starterItems(hero d2enum.Hero) []ImportedItem {
	rec := f.asset.Records.Character.Stats[hero]
	if rec == nil {
		return nil
	}

	var out []ImportedItem

	for i, code := range rec.StartItem {
		code = strings.TrimSpace(code)
		if code == "" || f.asset.Records.Item.All[code] == nil {
			continue
		}

		slot := starterSlot(rec.StartItemLocation[i])
		count := rec.StartItemCount[i]

		if slot > 0 {
			count = 1
		}

		for n := 0; n < count; n++ {
			it := ImportedItem{Codes: []string{code}, Quality: d2s.QualityNormal, Identified: true}

			if slot > 0 {
				it.Location, it.Equipped = d2s.LocationEquipped, uint8(slot)
			} else {
				it.Location, it.Page, it.AutoPlace = d2s.LocationStored, 1, true
			}

			out = append(out, it)
		}
	}

	return out
}

// ActiveWeaponSetSlot maps a weapon slot of a save to the slot shown in the
// inventory, taking the active weapon set into account: with set II active its
// weapons (slots 11 and 12) take the places of 4 and 5 and set I's are hidden.
// It returns 0 for an item that is not shown.
func ActiveWeaponSetSlot(slot uint8, weaponSetII bool) uint8 {
	switch slot {
	case 4, 5:
		if weaponSetII {
			return 0
		}

		return slot
	case d2sSlotAltRight, d2sSlotAltLeft:
		if weaponSetII {
			return slot - 7
		}

		return 0
	}

	return slot
}
