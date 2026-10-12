package diablo2item

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// groupLeaders are the stats a save stores together with followers (min/max
// pairs); the followers have no id of their own in the property list.
var groupLeaders = map[int][]int{
	17: {18}, 48: {49}, 50: {51}, 52: {53}, 54: {55, 56}, 57: {58, 59},
}

type propKey struct{ id, param int }

// d2sProperties turns the stat writes of a list into a save property list: the
// writes of one stat and parameter are summed (the stat list does the same),
// zero values and stats without a save width are left out like the game does,
// and the followers of group stats are put behind their leader. dropped counts
// the stats whose value or parameter does not fit its save field.
func d2sProperties(ws []SaveWrite, tables *d2s.ItemTables) (props []d2s.Property, dropped int) {
	sum := map[propKey]int64{}

	var order []propKey

	for _, w := range ws {
		k := propKey{w.ID, w.Param}
		if _, seen := sum[k]; !seen {
			order = append(order, k)
		}

		sum[k] += int64(w.Value)
	}

	follower := map[int]int{} // follower stat -> leader

	for l, fs := range groupLeaders {
		for _, f := range fs {
			follower[f] = l
		}
	}

	// a follower whose leader is missing gets the leader with value 0
	for _, k := range order {
		if l, ok := follower[k.id]; ok {
			if _, has := sum[propKey{l, 0}]; !has {
				sum[propKey{l, 0}] = 0
				order = append(order, propKey{l, 0})
			}
		}
	}

	emit := func(k propKey, v int64) bool {
		bitsN, paramBits, add, _, ok := tables.StatSaveInfo(k.id)
		if !ok {
			return true // the game does not save it either
		}

		if v+int64(add) < 0 || (bitsN < 31 && v+int64(add) >= int64(1)<<uint(bitsN)) ||
			(paramBits > 0 && (k.param < 0 || (paramBits < 31 && k.param >= 1<<uint(paramBits)))) {
			dropped++
			return false
		}

		props = append(props, d2s.Property{ID: k.id, Param: uint32(k.param), Value: v})

		return true
	}

	for _, k := range order {
		if _, isFollower := follower[k.id]; isFollower {
			continue
		}

		v := sum[k]
		fs := groupLeaders[k.id]

		if v == 0 && len(fs) == 0 {
			continue
		}

		if v == 0 {
			allZero := true

			for _, f := range fs {
				if sum[propKey{f, 0}] != 0 {
					allZero = false
				}
			}

			if allZero {
				continue
			}
		}

		start := len(props)

		if !emit(k, v) {
			continue
		}

		ok := true

		for _, f := range fs {
			if !emit(propKey{f, 0}, sum[propKey{f, 0}]) {
				ok = false
				break
			}
		}

		if !ok {
			props = props[:start]
		}
	}

	return props, dropped
}

// D2SItemFromFacts completes it, which the caller has filled with the item's
// place, code, flags, level, id and base numbers (defense, durability,
// quantity, sockets), with the identity and property lists the item creator
// rolled (ItemFacts.Save), then builds the writable item (d2s.NewItem). The
// items in its sockets are written too. dropped counts stats that do not fit
// their save field; the error says why an item cannot be written (a quality
// the save format has no room for, or no creator data).
func D2SItemFromFacts(it d2s.Item, fc *ItemFacts, tables *d2s.ItemTables) (out d2s.Item, dropped int, err error) {
	sf := fc.Save
	if sf == nil {
		return it, 0, fmt.Errorf("%s was not made with the item creator", it.Code)
	}

	switch it.Quality {
	case 0, d2s.QualityNormal:
	case d2s.QualityLow:
		it.LowQualityID = uint8(sf.FileIndex & 7)
	case d2s.QualityHigh:
		it.HighQuality = uint8(sf.FileIndex & 7)
	case d2s.QualityMagic:
		it.MagicPrefix, it.MagicSuffix = uint16(sf.MagicPrefix), uint16(sf.MagicSuffix)
	case d2s.QualitySet:
		it.SetID = uint16(sf.FileIndex)
	case d2s.QualityUnique:
		it.UniqueID = uint16(sf.FileIndex)
	case d2s.QualityRare, d2s.QualityCrafted:
		it.RareName1, it.RareName2 = uint8(sf.RareNames[0]), uint8(sf.RareNames[1])

		for k, id := range sf.RareAffixes {
			if id != 0 {
				it.RareMask |= 1 << uint(k)
				it.RareAffixes[k] = uint16(id)
			}
		}
	default:
		return it, 0, fmt.Errorf("quality %d of %s cannot be written (tempered items have no save form)", it.Quality, it.Code)
	}

	if sf.Auto != 0 {
		it.HasClassData, it.ClassData = true, uint16(sf.Auto)
	}

	if sf.HasPicture {
		it.HasPicture, it.Picture = true, uint8(sf.Picture)
	}

	if sf.RunewordID != 0 {
		it.RunewordID = uint16(sf.RunewordID)
	}

	var d int

	it.Properties, d = d2sProperties(sf.Props, tables)
	dropped += d

	if it.RunewordID != 0 {
		it.RunewordProperties, d = d2sProperties(sf.RunewordProps, tables)
		dropped += d
	}

	if it.Quality == d2s.QualitySet {
		it.SetProperties, it.SetListMask = nil, 0

		for tier, l := range sf.SetLists {
			if len(l) == 0 {
				continue
			}

			list, d := d2sProperties(l, tables)
			dropped += d

			it.SetListMask |= 1 << uint(tier)
			it.SetProperties = append(it.SetProperties, list)
		}
	}

	it.Children = nil

	for k := range sf.Children {
		ch, d, cerr := childFromFacts(&sf.Children[k], k, tables)
		if cerr != nil {
			return it, 0, fmt.Errorf("socketed %s: %w", sf.Children[k].Code, cerr)
		}

		dropped += d

		it.Children = append(it.Children, ch)
	}

	it.SocketCount = uint8(len(it.Children))
	if int(it.TotalSockets) < len(it.Children) {
		it.TotalSockets = uint8(len(it.Children))
	}

	built, err := d2s.NewItem(it, tables)
	if err != nil {
		return it, 0, fmt.Errorf("%s: %w", it.Code, err)
	}

	return built, dropped, nil
}

// childFromFacts is the item in socket number slot of another item.
func childFromFacts(fc *ItemFacts, slot int, tables *d2s.ItemTables) (d2s.Item, int, error) {
	ch := d2s.Item{
		Code: fc.Code, Identified: true, Ethereal: fc.Ethereal, Location: d2s.LocationSocketed, X: uint8(slot),
		ID: uint32(fc.Seed) | 1, Level: uint8(clampLevel(fc.ILvl)), Quality: uint8(fc.Quality),
	}

	if tables.ItemKindOf(fc.Code) == 0 {
		return ch, 0, fmt.Errorf("unknown item %q", fc.Code)
	}

	if ch.Quality == 0 {
		ch.Quality = d2s.QualityNormal
	}

	if tables.IsStackable(fc.Code) && fc.Quantity > 0 {
		ch.Quantity = uint16(fc.Quantity)
	}

	if fc.Save == nil {
		if tables.IsCompact(fc.Code) {
			b, err := d2s.NewItem(ch, tables)

			return b, 0, err
		}

		return ch, 0, fmt.Errorf("%s was not made with the item creator", fc.Code)
	}

	built, dropped, err := D2SItemFromFacts(ch, fc, tables)

	return built, dropped, err
}

func clampLevel(l int) int {
	switch {
	case l < 1:
		return 1
	case l > 99:
		return 99
	}

	return l
}
