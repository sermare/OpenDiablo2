package diablo2item

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

// ReplenishQuantityID is item_replenish_quantity (ItemStatCost id 253), the "replenishes quantity" stat of
// javelins such as Titan's Revenge.
const ReplenishQuantityID = 253

// StatTotal is the sum of the stat with this ItemStatCost id over everything the item carries: the rolled
// properties of StatItem (items made by the drop creator) and the stats of the legacy property pools (items
// the hero loaded from a save are made that way and have no rolled data).
func (i *Item) StatTotal(id int) int {
	total := 0

	for _, p := range i.StatItem().Props {
		if int(p.ID) == id {
			total += int(p.Value)
		}
	}

	name := statNameByID(i.factory.asset.Records.Item.Stats, id)
	if name == "" {
		return total
	}

	for _, pool := range i.properties {
		for _, p := range pool {
			if p == nil {
				continue
			}

			for _, s := range p.stats {
				if s.Name() != name {
					continue
				}

				if vals := s.Values(); len(vals) > 0 {
					total += vals[0].Int()
				}
			}
		}
	}

	return total
}

func statNameByID(stats d2records.ItemStatCosts, id int) string {
	for name, rec := range stats {
		if rec != nil && rec.Index == id {
			return name
		}
	}

	return ""
}
