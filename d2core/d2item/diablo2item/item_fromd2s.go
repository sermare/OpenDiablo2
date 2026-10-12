package diablo2item

import (
	"fmt"
	"math/bits"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// SpecFromD2S describes an item of a .d2s exactly: the affix, unique and set
// rows, the rare names and every property are mapped back to the numbers the
// item creator rolls (the inverse of D2SItemFromFacts), so that ItemFromSpec
// rebuilds the item with its real tooltip instead of a random lookalike. The
// error says why the item cannot be described (no creator data, an unknown
// base item, a quality the creator has no form for).
func (f *ItemFactory) SpecFromD2S(it *d2s.Item) (Spec, error) {
	c, err := f.Creator()
	if err != nil || c.Affixes == nil {
		return Spec{}, fmt.Errorf("no item creator data: %v", err)
	}

	code := strings.TrimSpace(it.Code)
	if c.Items.ByCode[code] == nil || f.asset.Records.Item.All[code] == nil {
		return Spec{}, fmt.Errorf("unknown base item %q", code)
	}

	if it.Quality > d2s.QualityCrafted {
		return Spec{}, fmt.Errorf("quality %d", it.Quality)
	}

	ns, np := c.Affixes.NSuffix, c.Affixes.NPrefix

	prefix := func(id uint16) int {
		if id == 0 {
			return 0
		}

		return int(id) + ns
	}

	r := &d2drop.Rolled{Quality: d2drop.Quality(it.Quality), ILvl: int(it.Level)}
	if it.Quality == 0 {
		r.Quality = d2drop.QualityNormal
	}

	if it.Identified {
		r.Flags |= flagIdentifiedBit
	}

	if it.Ethereal {
		r.Flags |= flagEtherealBit
	}

	if it.Socketed || it.TotalSockets > 0 {
		r.Flags |= flagSocketedBit
	}

	if it.Runeword {
		r.Flags |= d2drop.FlagRuneword
	}

	switch it.Quality {
	case d2s.QualityLow:
		r.Unique = int(it.LowQualityID)
	case d2s.QualityHigh:
		r.Unique = int(it.HighQuality)
	case d2s.QualityMagic:
		r.Prefix[0], r.Suffix[0] = prefix(it.MagicPrefix), int(it.MagicSuffix)
	case d2s.QualitySet:
		r.Unique = int(it.SetID)
	case d2s.QualityUnique:
		r.Unique = int(it.UniqueID)
	case d2s.QualityRare, d2s.QualityCrafted:
		r.RareNames = [2]int{int(it.RareName1), int(it.RareName2)}

		for k := 0; k < 3; k++ {
			r.Prefix[k], r.Suffix[k] = prefix(it.RareAffixes[2*k]), int(it.RareAffixes[2*k+1])
		}
	}

	if it.HasClassData && it.ClassData != 0 {
		r.Auto = int(it.ClassData) + ns + np
	}

	if it.HasPicture {
		r.Gfx[1] = int(it.Picture)
	}

	write := func(list int, props []d2s.Property) {
		for _, p := range props {
			if p.Value == 0 {
				continue
			}

			r.Writes = append(r.Writes, d2drop.StatWrite{
				Kind: 'L', Stat: p.ID, Value: int(p.Value) << uint(f.valShift(c, p.ID)), Param: int(p.Param), List: list,
			})
		}
	}

	write(0, it.Properties)

	tier, listNo := 0, 0

	for ; tier < SetBonusTiers && listNo < len(it.SetProperties); tier++ {
		if it.SetListMask>>uint(tier)&1 == 1 {
			write(setListFirst+tier, it.SetProperties[listNo])
			listNo++
		}
	}

	if bits.OnesCount8(it.SetListMask) != len(it.SetProperties) {
		return Spec{}, fmt.Errorf("set list mask %b with %d lists", it.SetListMask, len(it.SetProperties))
	}

	spec := Spec{
		Code: code, Quality: int(r.Quality), ILvl: int(it.Level), Seed: int64(it.ID),
		Identified: it.Identified, Ethereal: it.Ethereal, Durability: -1, Rolled: r,
		Quantity: int(it.Quantity), Sockets: int(it.TotalSockets), Personal: it.PersonalName,
	}

	if it.Runeword {
		spec.Runeword = runewordNameOf(c, int(it.RunewordID))
		if spec.Runeword == "" {
			return Spec{}, fmt.Errorf("runeword id %d has no row in Runes.txt", it.RunewordID)
		}

		write(d2drop.ListRuneword, it.RunewordProperties)
	}

	if rec := f.asset.Records.Item.All[code]; rec != nil {
		if rec.Durability > 0 && it.MaxDurability != 0 {
			spec.Durability = int(it.Durability)

			if int(it.MaxDurability) != rec.Durability {
				spec.MaxDurability = int(it.MaxDurability)
			}
		}
	}

	if t := c.Items.ByCode[code]; t != nil && it.Defense > 0 {
		spec.Defense = it.Defense
	}

	for k := range it.Children {
		ch := &it.Children[k]

		cs, err := f.childSpecFromD2S(ch)
		if err != nil {
			return Spec{}, fmt.Errorf("socketed %s: %w", ch.Code, err)
		}

		spec.Socketed = append(spec.Socketed, cs)
	}

	return spec, nil
}

// childSpecFromD2S is the spec of an item in a socket: gems and runes are
// plain items, a jewel carries its own affixes like any other item.
func (f *ItemFactory) childSpecFromD2S(ch *d2s.Item) (Spec, error) {
	if ch.Simple || ch.Quality <= d2s.QualityNormal {
		return Spec{Code: strings.TrimSpace(ch.Code), Identified: true, Durability: -1}, nil
	}

	return f.SpecFromD2S(ch)
}
