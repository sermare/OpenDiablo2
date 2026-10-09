package d2hero

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// StatItemsFromD2S converts the items of a save that act on the hero into
// stat list items: everything equipped (the weapon switch slots are turned
// off by the stat list) and the charms in the inventory page. Socketed
// children are attached to their item.
//
// setOf, if given, maps the SetItems row a save keeps for a set item to the
// 1-based Sets.txt row of its set, which is what groups the pieces of a set
// for the set bonuses (see SetResolver).
func StatItemsFromD2S(items []d2s.Item, bases d2statlist.Bases, setOf ...SetResolver) []d2statlist.Item {
	var resolve SetResolver
	if len(setOf) > 0 {
		resolve = setOf[0]
	}

	var out []d2statlist.Item

	for i := range items {
		it := &items[i]
		if it.Ear || it.Simple && !isCharm(it.Code) {
			continue
		}

		equipped := it.Location == d2s.LocationEquipped
		charm := isCharm(it.Code) && it.Location == d2s.LocationStored && it.Page == PageInventory

		if !equipped && !charm {
			continue
		}

		out = append(out, statItem(it, bases, charm, resolve))
	}

	return out
}

func isCharm(code string) bool {
	code = strings.TrimSpace(code)

	return len(code) == 3 && strings.HasPrefix(code, "cm")
}

// SetResolver maps a SetItems.txt row to the 1-based Sets.txt row of its set,
// 0 if unknown.
type SetResolver func(setItemRow int) int

func statItem(it *d2s.Item, bases d2statlist.Bases, charm bool, resolve SetResolver) d2statlist.Item {
	code := strings.TrimSpace(it.Code)
	si := d2statlist.Item{
		Code: code, Charm: charm, Ethereal: it.Ethereal, Defense: it.Defense,
		Runeword: it.Runeword, SetID: int(it.SetID),
		Props:         props(it.Properties),
		RunewordProps: props(it.RunewordProperties),
		Broken:        it.MaxDurability > 0 && it.Durability == 0,
	}

	if resolve != nil {
		si.SetID = 0
		if it.Quality == d2s.QualitySet {
			si.SetID = resolve(int(it.SetID))
		}
	}

	if !charm {
		si.Slot = int(it.Equipped)
	}

	if b, ok := bases[code]; ok {
		si.Weapon, si.BaseBlock = b.Weapon, b.BaseBlock
	}

	for _, l := range it.SetProperties {
		si.SetLists = append(si.SetLists, props(l))
	}

	for _, c := range it.Children {
		si.Sockets = append(si.Sockets, d2statlist.SocketedItem{Code: strings.TrimSpace(c.Code), Props: props(c.Properties)})
	}

	return si
}

func props(in []d2s.Property) []d2statlist.Prop {
	out := make([]d2statlist.Prop, 0, len(in))

	for _, p := range in {
		out = append(out, d2statlist.Prop{ID: p.ID, Param: int(p.Param), Value: p.Value})
	}

	return out
}
