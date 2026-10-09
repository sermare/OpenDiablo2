package diablo2item

// ExportStat is one rolled stat of an item in the form a save file keeps:
// the ItemStatCost row id and the main value. Stats with extra parameters
// (skill ids, charges, procs) are reported with HasParams set; the exporter
// decides whether it can store them.
type ExportStat struct {
	ID        int
	Name      string
	Value     int
	HasParams bool
}

// ItemFacts are the item's numbers a save needs besides its identity (see
// Spec): base defense, durability, sockets and the rolled stats per source.
type ItemFacts struct {
	Defense       int
	Durability    int
	MaxDurability int
	Sockets       int
	// Affix is the stats of the magic prefixes and suffixes, Unique those of a
	// unique item, SetItem those of a set item's own properties.
	Affix, Unique, SetItem []ExportStat
}

// Facts returns the numbers a save file keeps for the item.
func (i *Item) Facts() ItemFacts {
	f := ItemFacts{}

	if i.attributes != nil {
		f.Defense = i.attributes.defense
		f.Sockets = i.attributes.numSockets
	}

	f.Durability, f.MaxDurability = i.Durability()

	collect := func(pools ...PropertyPool) []ExportStat {
		var out []ExportStat

		for _, pool := range pools {
			for _, prop := range i.properties[pool] {
				for _, st := range prop.stats {
					rec := i.factory.asset.Records.Item.Stats[st.Name()]
					vals := st.Values()

					if rec == nil || len(vals) == 0 {
						continue
					}

					out = append(out, ExportStat{
						ID: rec.Index, Name: st.Name(), Value: vals[0].Int(), HasParams: len(vals) > 1,
					})
				}
			}
		}

		return out
	}

	f.Affix = collect(PropertyPoolPrefix, PropertyPoolSuffix)
	f.Unique = collect(PropertyPoolUnique)
	f.SetItem = collect(PropertyPoolSetItem)

	return f
}
