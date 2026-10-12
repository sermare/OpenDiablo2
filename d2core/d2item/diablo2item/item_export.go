package diablo2item

import (
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

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

	// Code, Quality, ILvl, Seed, Identified, Ethereal and Quantity describe the
	// item itself, so that an item in a socket (which has no StoredItem of its
	// own) can be written from its facts alone.
	Code       string
	Quality    int
	ILvl       int
	Seed       int64
	Identified bool
	Ethereal   bool
	Quantity   int

	// Save is the identity and property lists of an item made with the item
	// creator, in the numbers a .d2s keeps (see SaveForm); nil for an item that
	// was not made with it.
	Save *SaveForm
}

// SaveWrite is one stat of a property list as the game wrote it: the
// ItemStatCost row, the parameter and the value in stat units (the fixed point
// shift of the property engine already undone).
type SaveWrite struct {
	ID    int
	Param int
	Value int
}

// SaveForm is everything a .d2s keeps of an item made with the item creator
// besides the base numbers in ItemFacts: the ids are the ones the game's save
// routine writes (ITEM_WriteSavedItemProperties 0x630170 VERIFIED: the magic
// prefix ids are the position in the combined affix table minus the number of
// suffix rows, suffix ids and rare names are written as they are, the
// automagic id is the position minus suffix and prefix rows, unique, set,
// low and superior quality rows are written as they are).
type SaveForm struct {
	// FileIndex is the row of the unique or set item, the low quality subtype
	// or the superior subtype, according to the quality.
	FileIndex int
	// MagicPrefix and MagicSuffix are the ids of a magic item.
	MagicPrefix, MagicSuffix int
	// RareNames are the two name words and RareAffixes the prefix and suffix
	// ids of a rare or crafted item, alternating (prefix 1, suffix 1, ...);
	// 0 is an empty slot.
	RareNames   [2]int
	RareAffixes [6]int
	// Auto is the automagic id as a save keeps it, 0 for none.
	Auto int
	// HasPicture and Picture are the graphic variant of charms and jewels.
	HasPicture bool
	Picture    int
	// RunewordID is the save id of the runeword, 0 for none.
	RunewordID int

	// Props are the item's own properties, RunewordProps those of the runeword,
	// SetLists the set bonus lists by tier (an empty tier has no list).
	Props, RunewordProps []SaveWrite
	SetLists             [][]SaveWrite

	// Children are the items in the sockets.
	Children []ItemFacts
}

// runewordSaveBase is the save id of Runeword1 (VERIFIED by the d2itemdesc
// audit against Runes.txt); ids follow the order of the "RunewordN" names.
const runewordSaveBase = 27

// runewordNameOf is the Runes.txt name of the runeword with a save id, "".
func runewordNameOf(c *d2drop.Creator, id int) string {
	if c.Runes == nil {
		return ""
	}

	for _, w := range c.Runes.Words {
		if strings.HasPrefix(w.Name, "Runeword") && runewordSaveID(c, w.Name) == id {
			return w.Name
		}
	}

	return ""
}

// runewordSaveID returns the id a save keeps for the runeword with the Runes.txt
// name, 0 when the table has none.
func runewordSaveID(c *d2drop.Creator, name string) int {
	if c.Runes == nil {
		return 0
	}

	type rw struct{ n, order int }

	var list []rw

	for k, w := range c.Runes.Words {
		if n, err := strconv.Atoi(strings.TrimPrefix(w.Name, "Runeword")); err == nil && strings.HasPrefix(w.Name, "Runeword") {
			list = append(list, rw{n, k})
		}
	}

	sort.SliceStable(list, func(a, b int) bool { return list[a].n < list[b].n })

	for rank, e := range list {
		if c.Runes.Words[e.order].Name == name {
			return runewordSaveBase + rank
		}
	}

	return 0
}

// saveForm builds the SaveForm of an item made with the creator.
func (i *Item) saveForm() *SaveForm {
	r := i.rolled
	if r == nil || i.factory == nil {
		return nil
	}

	c, err := i.factory.Creator()
	if err != nil || c.Affixes == nil {
		return nil
	}

	ns, np := c.Affixes.NSuffix, c.Affixes.NPrefix

	prefix := func(id int) int {
		if id > ns {
			return id - ns
		}

		return id
	}

	sf := &SaveForm{FileIndex: r.Unique, RareNames: r.RareNames}
	if sf.FileIndex < 0 {
		sf.FileIndex = 0
	}

	sf.MagicPrefix, sf.MagicSuffix = prefix(r.Prefix[0]), r.Suffix[0]

	for k := 0; k < 3; k++ {
		sf.RareAffixes[2*k], sf.RareAffixes[2*k+1] = prefix(r.Prefix[k]), r.Suffix[k]
	}

	if r.Auto > 0 {
		sf.Auto = r.Auto
		if r.Auto > ns+np {
			sf.Auto = r.Auto - ns - np
		}
	}

	if b := c.Items.ByCode[i.CommonCode]; b != nil {
		if ty := c.Items.Types[b.Type]; ty != nil && ty.VarInvGfx != 0 {
			sf.HasPicture, sf.Picture = true, r.Gfx[1]&7
		}
	}

	if i.runeword != "" {
		sf.RunewordID = runewordSaveID(c, i.runeword)
	}

	tiers := make([][]SaveWrite, SetBonusTiers)

	for _, w := range r.Writes {
		if w.Kind == 'S' || w.Stat == statIDSockets {
			continue
		}

		sw := SaveWrite{ID: w.Stat, Param: w.Param, Value: w.Value >> uint(i.factory.valShift(c, w.Stat))}

		switch {
		case w.List >= setListFirst && w.List <= setListLast:
			tiers[w.List-setListFirst] = append(tiers[w.List-setListFirst], sw)
		case w.List == d2drop.ListRuneword:
			sf.RunewordProps = append(sf.RunewordProps, sw)
		default:
			sf.Props = append(sf.Props, sw)
		}
	}

	used := 0

	for t, l := range tiers {
		if len(l) > 0 {
			used = t + 1
		}
	}

	sf.SetLists = tiers[:used]

	for _, ch := range i.socketed {
		sf.Children = append(sf.Children, ch.Facts())
	}

	return sf
}

// Facts returns the numbers a save file keeps for the item.
func (i *Item) Facts() ItemFacts {
	f := ItemFacts{
		Code: i.CommonCode, Quality: int(i.quality), ILvl: i.itemLevel, Seed: i.Seed,
	}

	if i.attributes != nil {
		f.Identified, f.Ethereal, f.Quantity = i.attributes.identitified, i.attributes.ethereal, i.attributes.currentStackSize
	}

	f.Save = i.saveForm()

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
