package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2itemdesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const excelPath = "/data/global/excel/"

// DescriptionTables returns the tables the exact item descriptions are built
// from (package d2itemdesc), loaded from the game archives on first use. It
// returns nil when the game data does not have them.
func (f *ItemFactory) DescriptionTables() *d2itemdesc.Tables {
	f.descOnce.Do(func() {
		src := func(name string) ([]byte, bool) {
			data, err := f.asset.LoadFile(excelPath + name)
			return data, err == nil
		}

		// TranslateString answers with the key when it has no string
		tr := func(key string) string {
			if s := f.asset.TranslateString(key); s != key {
				return s
			}

			return ""
		}

		tables, err := d2itemdesc.Load(src, tr)
		if err != nil {
			return
		}

		f.descTables = tables
	})

	return f.descTables
}

// SetOrigin attaches the item exactly as a .d2s save held it. Such an item is
// described from the save (all properties, affixes, runeword, sockets) instead
// of from the generated approximation.
func (i *Item) SetOrigin(o *d2s.Item) { i.origin = o }

// Origin returns the saved item attached with SetOrigin, or nil.
func (i *Item) Origin() *d2s.Item { return i.origin }

var tokenOfColor = map[d2itemdesc.Color]d2ui.ColorToken{
	d2itemdesc.White:  d2ui.ColorTokenWhite,
	d2itemdesc.Blue:   d2ui.ColorTokenBlue,
	d2itemdesc.Yellow: d2ui.ColorTokenYellow,
	d2itemdesc.Gold:   d2ui.ColorTokenGold,
	d2itemdesc.Green:  d2ui.ColorTokenGreen,
	d2itemdesc.Orange: d2ui.ColorTokenOrange,
	d2itemdesc.Gray:   d2ui.ColorTokenGrey,
	d2itemdesc.Red:    d2ui.ColorTokenRed,
}

// Tokenize turns description lines into tooltip strings with color tokens.
func Tokenize(lines []d2itemdesc.Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = d2ui.ColorTokenize(l.Text, tokenOfColor[l.Color])
	}

	return out
}

// describeOrigin renders the tooltip of an item that carries its saved form.
func (i *Item) describeOrigin() ([]string, bool) {
	if i.origin == nil || i.factory == nil {
		return nil, false
	}

	tables := i.factory.DescriptionTables()
	if tables == nil {
		return nil, false
	}

	// what the player changed since the save was read
	cp := *i.origin
	cp.Identified = cp.Identified || i.IsIdentified()

	if q := i.Quantity(); q > 0 {
		cp.Quantity = uint16(q)
	}

	if cur, maxDur := i.Durability(); maxDur > 0 && maxDur == int(cp.MaxDurability) {
		cp.Durability = uint16(cur)
	}

	var ctx *d2itemdesc.Context
	if i.factory.DescribeContext != nil {
		ctx = i.factory.DescribeContext()
	}

	return Tokenize(tables.Describe(&cp, ctx)), true
}
