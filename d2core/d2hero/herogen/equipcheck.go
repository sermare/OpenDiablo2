package herogen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// statReqPercent is item_req_percent (ItemStatCost id 91): the "requirements
// +/-N%" property that moves the strength and dexterity an item needs.
const statReqPercent = 91

// WornItem is what the equip rules see of a worn item of a generated hero.
func (t *Tables) wornItem(it *d2s.Item) (*d2equip.Item, error) {
	code := strings.TrimSpace(it.Code)

	base, ok := t.EquipBases[code]
	if !ok {
		return nil, fmt.Errorf("%w: no base item %q", ErrItem, code)
	}

	e := &d2equip.Item{
		Code: code, Type: base.Type, Weapon: base.Weapon, TwoHanded: base.TwoHanded, OneOrTwo: base.OneOrTwo,
		Identified: it.Identified, Ethereal: it.Ethereal, ReqStr: base.ReqStr, ReqDex: base.ReqDex, ReqLevel: base.ReqLevel,
		Durability: int(it.Durability), MaxDurability: int(it.MaxDurability), NoDurability: base.NoDurability,
	}

	// a unique item asks for the level of its UniqueItems row when that is higher than the base item's
	if it.Quality == d2s.QualityUnique && int(it.UniqueID) < len(t.Creator.Uniques.Uniques) {
		if lr := t.Creator.Uniques.Uniques[it.UniqueID].LvlReq; lr > e.ReqLevel {
			e.ReqLevel = lr
		}
	}

	for _, p := range it.Properties {
		if p.ID == statReqPercent {
			e.ReqPercent += int(p.Value)
		}
	}

	return e, nil
}

// CheckWorn proves a hero can wear what it wears, with the game's own equip
// rules (d2equip): each worn item fits its body location, is allowed for the
// class (primal helms for the Barbarian only, orbs for the Sorceress, ...),
// meets the strength, dexterity and level requirement of the base attributes,
// and the two hands go together (a two-hand weapon leaves the other hand
// free, only the Barbarian and the Assassin carry a second weapon, a bow
// takes its quiver). The belt must be big enough for the potions in it.
func (t *Tables) CheckWorn(spec Spec, items []d2s.Item) error {
	hero := d2equip.Hero{Class: classTokens[spec.Class], Str: spec.Strength, Dex: spec.Dexterity, Level: spec.Level}
	body := map[d2equip.Loc]*d2equip.Item{}

	var worn []*d2s.Item

	for i := range items {
		if items[i].Location == d2s.LocationEquipped {
			worn = append(worn, &items[i])
		}
	}

	// right hand before left, so the second hand is judged against the first
	sort.SliceStable(worn, func(a, b int) bool { return worn[a].Equipped < worn[b].Equipped })

	beltCode := ""

	for _, it := range worn {
		e, err := t.wornItem(it)
		if err != nil {
			return err
		}

		loc := d2equip.Loc(it.Equipped)

		dec := t.EquipRules.Place(hero, body, e, loc)
		if !dec.OK {
			return fmt.Errorf("%w: %s cannot be worn at %v by a %s: %s (%s)", ErrSpec, e.Code, loc, hero.Class,
				dec.Reason, dec.Detail)
		}

		if len(dec.Displaced) > 0 {
			return fmt.Errorf("%w: %s at %v pushes out the item at %v", ErrSpec, e.Code, loc, dec.Displaced)
		}

		body[loc] = e

		if loc == d2equip.LocBelt {
			beltCode = e.Code
		}
	}

	return t.checkBelt(items, beltCode)
}

// beltBoxesByType is the numboxes column of Belts.txt by belt type (the armor.txt "belt" column); the same
// numbers as d2inventory.BeltBoxesByType, which this package does not import (it drags in the asset
// loader); a test compares the two.
var beltBoxesByType = [7]int{12, 8, 4, 16, 8, 12, 16}

// BeltBoxesByType returns the table (for the test that compares it with d2inventory's).
func BeltBoxesByType() [7]int { return beltBoxesByType }

// beltDefaultType is the Belts.txt row of a hero without a belt: 4 cells.
const beltDefaultType = 2

// BeltBoxes is the number of potion cells a belt gives (no belt: 4).
func (t *Tables) BeltBoxes(beltCode string) int {
	bt, ok := t.BeltTypes[beltCode]
	if beltCode == "" || !ok || bt < 0 || bt >= len(beltBoxesByType) {
		bt = beltDefaultType
	}

	return beltBoxesByType[bt]
}

func (t *Tables) checkBelt(items []d2s.Item, beltCode string) error {
	boxes := t.BeltBoxes(beltCode)

	for i := range items {
		if it := &items[i]; it.Location == d2s.LocationBelt && int(it.X) >= boxes {
			return fmt.Errorf("%w: belt cell %d does not exist on a belt of %d cells (%q)", ErrSpec, it.X, boxes, beltCode)
		}
	}

	return nil
}
