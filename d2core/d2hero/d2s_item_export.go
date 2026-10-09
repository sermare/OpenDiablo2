package d2hero

import (
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// AffixIDs maps the names OpenDiablo2 keeps for a rolled item to the ids a
// .d2s stores (row numbers of the item tables, the inverse of affixNames).
type AffixIDs struct {
	Unique, Set, Prefix, Suffix map[string]int
}

func invertNames(list []string) map[string]int {
	m := make(map[string]int, len(list))

	for id := len(list) - 1; id > 0; id-- {
		if list[id] != "" {
			m[list[id]] = id // the lowest row wins
		}
	}

	return m
}

// affixIDs returns the reverse of the factory's name tables.
func (f *HeroStateFactory) affixIDs() *AffixIDs {
	n := f.loadAffixNames()

	return &AffixIDs{
		Unique: invertNames(n.unique), Set: invertNames(n.set),
		Prefix: invertNames(n.prefix), Suffix: invertNames(n.suffix),
	}
}

// itemSeedID turns a stored item into a stable non-zero item id (the 32 bit
// id a .d2s keeps per item). Items imported from a save already carry theirs.
func itemSeedID(s *StoredItem) uint32 {
	if s.Seed != 0 {
		return uint32(s.Seed)
	}

	h := uint32(2166136261)
	for _, c := range fmt.Sprintf("%s/%d/%d/%d/%d", s.Code, s.Page, s.X, s.Y, s.ILvl) {
		h = (h ^ uint32(c)) * 16777619
	}

	return h | 1
}

// D2SItemFromStored builds the .d2s item for an item made in the game from its
// stored identity (quality, affix names, item level) and the numbers the game
// rolled (StoredItem.Facts). ok is false, with the reason in why, when the item
// cannot be written faithfully: no facts, an affix without a row in the item
// tables, or a quality the game's model does not generate (rare, crafted).
// Stats the save cannot hold (extra parameters, damage groups, values out of
// range) are left out and counted in dropped.
func D2SItemFromStored(s *StoredItem, tables *d2s.ItemTables, ids *AffixIDs) (it d2s.Item, dropped int, why string) {
	if s.Facts == nil {
		return it, 0, "no rolled values stored for " + s.Code
	}

	if tables == nil || ids == nil || tables.ItemKindOf(s.Code) == 0 {
		return it, 0, "no item tables for " + s.Code
	}

	it = d2s.Item{
		Code: s.Code, Identified: s.Identified, Ethereal: s.Ethereal,
		X: uint8(s.X), Y: uint8(s.Y), ID: itemSeedID(s),
		Quality: d2s.QualityNormal, Level: uint8(clampInt(s.ILvl, 1, 99)),
	}

	if s.Page == PageBelt {
		it.Location = d2s.LocationBelt
	} else {
		it.Location, it.Page = d2s.LocationStored, uint8(s.Page)
	}

	why = applyQuality(&it, s, ids)
	if why != "" {
		return it, 0, why
	}

	f := s.Facts

	if kind := tables.ItemKindOf(s.Code); kind == d2s.KindArmor || kind == d2s.KindWeapon {
		if kind == d2s.KindArmor {
			it.Defense = f.Defense
		}

		it.MaxDurability = uint16(clampInt(f.MaxDurability, 0, 255))
		it.Durability = uint16(clampInt(f.Durability, 0, int(it.MaxDurability)))

		if s.Durability != nil {
			it.Durability = uint16(clampInt(*s.Durability, 0, int(it.MaxDurability)))
		}
	}

	if tables.IsStackable(s.Code) {
		it.Quantity = uint16(clampInt(s.Quantity, 1, 511))
	}

	if f.Sockets > 0 {
		it.TotalSockets = uint8(clampInt(f.Sockets, 0, 6))
	}

	stats := append(append([]diablo2item.ExportStat(nil), f.Affix...), f.Unique...)
	stats = append(stats, f.SetItem...)

	it.Properties, dropped = propertiesFromStats(stats, tables)

	built, err := d2s.NewItem(it, tables)
	if err != nil {
		return it, 0, fmt.Sprintf("%s: %v", s.Code, err)
	}

	return built, dropped, ""
}

// applyQuality fills the quality's identity fields.
func applyQuality(it *d2s.Item, s *StoredItem, ids *AffixIDs) string {
	first := func(l []string) string {
		if len(l) > 0 {
			return l[0]
		}

		return ""
	}

	switch uint8(s.Quality) {
	case 0, d2s.QualityNormal:
	case d2s.QualityLow:
		it.Quality = d2s.QualityLow // the low quality subtype is not modelled, left at 0 (Crude)
	case d2s.QualityHigh:
		it.Quality = d2s.QualityHigh
	case d2s.QualityMagic:
		it.Quality = d2s.QualityMagic

		if n := first(s.Prefixes); n != "" {
			id, ok := ids.Prefix[n]
			if !ok {
				return fmt.Sprintf("magic prefix %q has no row in MagicPrefix", n)
			}

			it.MagicPrefix = uint16(id)
		}

		if n := first(s.Suffixes); n != "" {
			id, ok := ids.Suffix[n]
			if !ok {
				return fmt.Sprintf("magic suffix %q has no row in MagicSuffix", n)
			}

			it.MagicSuffix = uint16(id)
		}
	case d2s.QualityUnique:
		id, ok := ids.Unique[s.Unique]
		if !ok {
			return fmt.Sprintf("unique %q has no row in UniqueItems", s.Unique)
		}

		it.Quality, it.UniqueID = d2s.QualityUnique, uint16(id)
	case d2s.QualitySet:
		id, ok := ids.Set[s.SetItem]
		if !ok {
			return fmt.Sprintf("set item %q has no row in SetItems", s.SetItem)
		}

		it.Quality, it.SetID = d2s.QualitySet, uint16(id)
	default:
		return fmt.Sprintf("quality %d of %s cannot be written (rare/crafted names are not modelled)", s.Quality, s.Code)
	}

	return ""
}

// propertiesFromStats turns rolled stats into a property list: the value of
// every stat that has no parameter field, summed per stat id like the game's
// stat list. dropped counts the stats the save cannot hold.
func propertiesFromStats(stats []diablo2item.ExportStat, tables *d2s.ItemTables) (props []d2s.Property, dropped int) {
	sum := map[int]int64{}

	var order []int

	for _, st := range stats {
		bitsN, paramBits, add, _, ok := tables.StatSaveInfo(st.ID)

		switch {
		case !ok, paramBits > 0, isGroupStat(st.ID):
			dropped++
			continue
		case st.Value+add < 0 || (bitsN < 31 && int64(st.Value+add) >= int64(1)<<uint(bitsN)):
			dropped++
			continue
		}

		if _, seen := sum[st.ID]; !seen {
			order = append(order, st.ID)
		}

		sum[st.ID] += int64(st.Value)
	}

	sort.Ints(order)

	for _, id := range order {
		bitsN, _, add, _, _ := tables.StatSaveInfo(id)
		if v := sum[id]; v+int64(add) >= 0 && (bitsN >= 31 || v+int64(add) < int64(1)<<uint(bitsN)) {
			props = append(props, d2s.Property{ID: id, Value: v})
		} else {
			dropped++
		}
	}

	return props, dropped
}

// isGroupStat reports the stats a save stores together with followers
// (min/max damage pairs); a single rolled value cannot represent them.
func isGroupStat(id int) bool {
	switch id {
	case 17, 18, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59:
		return true
	}

	return false
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// importable reports whether the item would have been imported into the hero's
// containers; only those can be known to be gone. Items the engine never held
// (equipment, unknown base items, ears) always stay.
func importable(it *d2s.Item, known func(code string) bool) bool {
	if known == nil {
		return false
	}

	_, skip := StoredFromD2S(it, known)

	return skip == ""
}

// MergeContainerItems brings the item list of a parsed save in line with the
// hero's containers: an item that is still there is kept bit for bit and only
// moved (page, position, identification), an item that is gone is dropped, and
// an item made in the game is encoded (D2SItemFromStored) and appended.
// Equipped items, items of base types the engine does not know and the socketed
// children of kept items are not touched.
// It returns the number of items added and removed.
func MergeContainerItems(c *d2s.Character, containers *HeroContainers, tables *d2s.ItemTables,
	ids *AffixIDs, known func(code string) bool, warn func(string, ...interface{})) (added, removed int) {
	if containers == nil {
		return 0, 0
	}

	// An item is recognised by its bits without the place: compact items (potions,
	// runes) carry no id, so several originals can look alike; any of them will do.
	identity := func(it *d2s.Item) string {
		c := *it
		c.Location, c.Equipped, c.X, c.Y, c.Page = 0, 0, 0, 0, 0

		b, err := d2s.EncodeItem(&c, tables)
		if err != nil {
			return ""
		}

		return string(b)
	}

	orig := map[string][]int{} // original indexes of every item that can sit in a container

	for i := range c.Items {
		it := &c.Items[i]
		if it.Ear || it.Location == d2s.LocationEquipped || it.Location == d2s.LocationSocketed {
			continue
		}

		if k := identity(it); k != "" {
			orig[k] = append(orig[k], i)
		}
	}

	moved := map[int]*StoredItem{}

	var fresh []d2s.Item

	for i := range containers.Items {
		s := &containers.Items[i]

		if s.D2S != nil {
			if k := identity(s.D2S); k != "" && len(orig[k]) > 0 {
				moved[orig[k][0]] = s
				orig[k] = orig[k][1:]

				continue
			}
		}

		it, dropped, why := D2SItemFromStored(s, tables, ids)
		if why != "" {
			warn("container item %q (page %d) not written: %s", s.Code, s.Page, why)
			continue
		}

		if dropped > 0 {
			warn("container item %q: %d stat(s) the save cannot hold were left out", s.Code, dropped)
		}

		fresh = append(fresh, it)
	}

	kept := make([]d2s.Item, 0, len(c.Items)+len(fresh))

	for i := range c.Items {
		it := c.Items[i]

		s, isMoved := moved[i]

		switch {
		case isMoved:
			it.X, it.Y = uint8(s.X), uint8(s.Y)

			if s.Identified {
				it.Identified = true
			}

			if s.Page == PageBelt {
				it.Location, it.Page = d2s.LocationBelt, 0
			} else {
				it.Location, it.Page = d2s.LocationStored, uint8(s.Page)
			}
		case importable(&it, known):
			// the hero had this item (it was imported) and no longer has it
			removed++
			continue
		}

		kept = append(kept, it)
	}

	c.Items = append(kept, fresh...)

	return len(fresh), removed
}
