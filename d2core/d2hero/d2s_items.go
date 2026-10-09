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
	d2sSlotAltRight  = 11 // weapon set II
	d2sSlotAltLeft   = 12
)

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

// resolveNames maps the affix ids of a save to the names in the game tables:
// the unique or set item name, or the magic prefix and suffix names.
func (f *HeroStateFactory) resolveNames(it *d2s.Item, names *affixNames) (unique, setItem string, prefixes, suffixes []string) {
	switch it.Quality {
	case d2s.QualityMagic:
		if n := nameAt(names.prefix, int(it.MagicPrefix)); n != "" {
			prefixes = append(prefixes, n)
		}

		if n := nameAt(names.suffix, int(it.MagicSuffix)); n != "" {
			suffixes = append(suffixes, n)
		}
	case d2s.QualityUnique:
		if n := nameAt(names.unique, int(it.UniqueID)); n != "" && f.asset.Records.Item.Unique[n] != nil {
			unique = n
		}
	case d2s.QualitySet:
		if n := nameAt(names.set, int(it.SetID)); n != "" && f.asset.Records.Item.SetItems[n] != nil {
			setItem = n
		}
	}

	return unique, setItem, prefixes, suffixes
}

// applyNames replaces the random affixes of an imported item with the unique,
// set or magic affix names of the save.
func (f *HeroStateFactory) applyNames(s *StoredItem, it *d2s.Item, names *affixNames) {
	if u, si, pre, suf := f.resolveNames(it, names); u != "" || si != "" || len(pre)+len(suf) > 0 {
		s.Unique, s.SetItem, s.Prefixes, s.Suffixes = u, si, pre, suf
		s.Origin = false
	}
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

// applyStarterItems gives a new character the items of the item1..item10 columns
// of charstats.txt: a location naming a d2s equipment slot (1..12) puts the item
// on the body, anything else in the inventory (placed first-fit in the 10x4 grid);
// count repeats a non-stacking item (the starting potions).
func (f *HeroStateFactory) applyStarterItems(state *HeroState, hero d2enum.Hero) {
	rec := f.asset.Records.Character.Stats[hero]
	if rec == nil {
		return
	}

	const cols, rows = 10, 4

	var used [rows][cols]bool

	place := func(w, h int) (int, int, bool) {
		for y := 0; y+h <= rows; y++ {
			for x := 0; x+w <= cols; x++ {
				free := true

				for dy := 0; dy < h && free; dy++ {
					for dx := 0; dx < w; dx++ {
						if used[y+dy][x+dx] {
							free = false
							break
						}
					}
				}

				if free {
					for dy := 0; dy < h; dy++ {
						for dx := 0; dx < w; dx++ {
							used[y+dy][x+dx] = true
						}
					}

					return x, y, true
				}
			}
		}

		return 0, 0, false
	}

	containers := &HeroContainers{Items: []StoredItem{}, Equipped: []StoredItem{}}

	for i, code := range rec.StartItem {
		code = strings.TrimSpace(code)

		common := f.asset.Records.Item.All[code]
		if code == "" || common == nil {
			continue
		}

		slot := starterSlot(rec.StartItemLocation[i])
		count := rec.StartItemCount[i]

		if slot > 0 {
			containers.Equipped = append(containers.Equipped, StoredItem{
				Code: code, Page: PageEquipped, X: slot, Quality: int(d2s.QualityNormal), Identified: true, Origin: true,
			})

			continue
		}

		for n := 0; n < count; n++ {
			x, y, ok := place(common.InventoryWidth, common.InventoryHeight)
			if !ok {
				break
			}

			containers.Items = append(containers.Items, StoredItem{
				Code: code, Page: PageInventory, X: x, Y: y, Quality: int(d2s.QualityNormal), Identified: true, Origin: true,
			})
		}
	}

	containers.EquippedSet = len(containers.Equipped) > 0
	state.Containers = containers
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
