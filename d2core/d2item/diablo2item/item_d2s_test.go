package diablo2item

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2tbl"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2itemdesc"
)

func saveTables(t *testing.T) *d2s.ItemTables {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		t.Skipf("none of %v in %s", names, dir)

		return nil
	}

	tb, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"),
		read("armor.txt"), read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func descTables(t *testing.T) (*d2itemdesc.Tables, func(string) string) {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	layers := []string{"patch_d2", "d2exp", "d2data"}

	src := func(name string) ([]byte, bool) {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return b, true
		}

		for _, l := range layers {
			if b, err := os.ReadFile(filepath.Join(dir, "itemdesc", l, strings.ToLower(name))); err == nil {
				return b, true
			}
		}

		return nil, false
	}

	strs := d2tbl.TextDictionary{}

	for _, f := range []string{"d2data/string.tbl", "d2exp/expansionstring.tbl", "patch_d2/patchstring.tbl"} {
		b, err := os.ReadFile(filepath.Join(dir, "itemdesc", f))
		if err != nil {
			t.Skipf("string table %s missing", f)
		}

		d, err := d2tbl.LoadTextDictionary(b)
		if err != nil {
			t.Fatal(err)
		}

		for k, v := range d {
			strs[k] = v
		}
	}

	tb, err := d2itemdesc.Load(src, func(k string) string { return strs[k] })
	if err != nil {
		t.Skipf("itemdesc tables: %v", err)
	}

	return tb, func(k string) string { return strs[k] }
}

// asSaved writes the facts of an item the way the hero export does for an item
// made in the game, encodes it, and parses the bits again.
func asSaved(t *testing.T, it *Item, tables *d2s.ItemTables) (before, after d2s.Item) {
	t.Helper()

	fc := it.Facts()

	base := d2s.Item{
		Code: it.CommonCode, Identified: it.IsIdentified(), Ethereal: fc.Ethereal, Location: d2s.LocationStored, Page: 1,
		ID: uint32(fc.Seed) | 1, Quality: uint8(fc.Quality), Level: uint8(clampLevel(fc.ILvl)), Defense: fc.Defense,
		MaxDurability: uint16(fc.MaxDurability), Durability: uint16(fc.Durability), TotalSockets: uint8(fc.Sockets),
	}

	if base.Quality == 0 {
		base.Quality = d2s.QualityNormal
	}

	built, dropped, err := D2SItemFromFacts(base, &fc, tables)
	if err != nil {
		t.Fatalf("%s: %v", it.CommonCode, err)
	}

	if dropped != 0 {
		t.Errorf("%s: %d stats dropped", it.CommonCode, dropped)
	}

	bits, err := d2s.EncodeItem(&built, tables)
	if err != nil {
		t.Fatal(err)
	}

	list := append([]byte{'J', 'M', 1, 0}, bits...)

	back, n, err := d2s.ParseItemList(list, tables)
	if err != nil || n != len(list) || len(back) != 1 {
		t.Fatalf("parse back: %v (%d of %d bytes, %d items)", err, n, len(list), len(back))
	}

	return built, back[0]
}

type propKeyT struct{ id, param int }

func sumProps(ps []d2s.Property) map[propKeyT]int64 {
	m := map[propKeyT]int64{}

	for _, p := range ps {
		if p.Value != 0 {
			m[propKeyT{p.ID, int(p.Param)}] += p.Value
		}
	}

	return m
}

// wantProps are the item's own properties as the game's stat list holds them:
// the stats of the tooltip, which the save must give back one for one.
func wantProps(t *testing.T, it *Item, tables *d2s.ItemTables) map[propKeyT]int64 {
	t.Helper()

	m := map[propKeyT]int64{}

	for _, p := range it.StatItem().Props {
		if _, _, _, _, ok := tables.StatSaveInfo(p.ID); !ok || p.ID == statIDSockets || p.Value == 0 {
			continue
		}

		m[propKeyT{p.ID, p.Param}] += p.Value
	}

	return m
}

func sameProps(t *testing.T, where string, got, want map[propKeyT]int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Errorf("%s: %d properties saved, game has %d\nsaved %v\ngame  %v", where, len(got), len(want), got, want)
		return
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: stat %v saved %d, game has %d", where, k, got[k], v)
		}
	}
}

func TestSaveRoundTripEveryQuality(t *testing.T) {
	f := realCreatorFactory(t)
	tables := saveTables(t)
	desc, tr := descTables(t)

	type tc struct {
		name    string
		code    string
		ilvl    int
		quality d2drop.Quality
		unique  string
		seed    uint32
		flags   d2drop.RequestFlags
	}

	cases := []tc{
		{name: "normal", code: "cap", ilvl: 20, quality: d2drop.QualityNormal, seed: 3},
		{name: "low", code: "cap", ilvl: 20, quality: d2drop.QualityLow, seed: 4},
		{name: "superior", code: "cap", ilvl: 20, quality: d2drop.QualitySuperior, seed: 5},
		{name: "magic", code: "cap", ilvl: 40, quality: d2drop.QualityMagic, seed: 7},
		{name: "magic2", code: "lsd", ilvl: 85, quality: d2drop.QualityMagic, seed: 9},
		{name: "rare", code: "cap", ilvl: 80, quality: d2drop.QualityRare, seed: 11},
		{name: "rare2", code: "lsd", ilvl: 90, quality: d2drop.QualityRare, seed: 12},
		{name: "rare3", code: "uap", ilvl: 90, quality: d2drop.QualityRare, seed: 13},
		{name: "crafted", code: "lsd", ilvl: 70, quality: d2drop.QualityCrafted, seed: 14},
		{name: "unique", code: "uap", ilvl: 99, quality: d2drop.QualityUnique, unique: "Harlequin Crest", seed: 5},
		{name: "set", code: "bhm", ilvl: 99, quality: d2drop.QualitySet, unique: "Tancred's Skull", seed: 5},
		{name: "ethereal-sockets", code: "lrg", ilvl: 50, quality: d2drop.QualityNormal, seed: 21, flags: d2drop.FlagForceSockets},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			it, err := f.Create(CreateParams{
				Code: c.code, ILvl: c.ilvl, Quality: c.quality, Name: c.unique, Seed: c.seed, Flags: c.flags, FreshGame: true,
			})
			if err != nil {
				t.Fatal(err)
			}

			it.Identify()

			before, after := asSaved(t, it, tables)

			if after.Code != strings.TrimSpace(c.code) || after.Level != uint8(it.ItemLevel()) {
				t.Errorf("identity %q level %d", after.Code, after.Level)
			}

			if after.Quality != uint8(it.Quality()) && it.Quality() != d2drop.QualityNone {
				t.Errorf("quality %d, game %d", after.Quality, it.Quality())
			}

			sameProps(t, c.name, sumProps(after.Properties), wantProps(t, it, tables))

			// reloading the save gives the same item back
			spec, err := f.SpecFromD2S(&after)
			if err != nil {
				t.Fatal(err)
			}

			again, err := f.ItemFromSpec(spec)
			if err != nil {
				t.Fatal(err)
			}

			sameItem(t, c.name, it, again, tables)

			if after.Defense != before.Defense || after.MaxDurability != before.MaxDurability {
				t.Errorf("base numbers changed: %+v", after)
			}

			if int(after.TotalSockets) != it.NumSockets() {
				t.Errorf("sockets %d, game %d", after.TotalSockets, it.NumSockets())
			}

			lines := desc.Describe(&after, nil)
			if len(lines) == 0 {
				t.Fatal("no tooltip")
			}

			head := lines[0].Text
			t.Logf("%s tooltip: %q", c.name, head)

			switch it.Quality() {
			case d2drop.QualityRare, d2drop.QualityCrafted:
				if after.RareName1 != uint8(it.Rolled().RareNames[0]) || after.RareName2 != uint8(it.Rolled().RareNames[1]) {
					t.Errorf("rare names %d %d, game %v", after.RareName1, after.RareName2, it.Rolled().RareNames)
				}

				// the tooltip's first line is the rare name the game shows
				c, _ := f.Creator()
				want := strings.TrimSpace(tr(rareName(c, it.Rolled().RareNames[0])) + " " + tr(rareName(c, it.Rolled().RareNames[1])))

				if !strings.EqualFold(strings.TrimSpace(head), want) {
					t.Errorf("rare name %q, game %q", head, want)
				}

				nPre, nSuf := 0, 0
				for _, id := range it.Rolled().Prefix {
					if id != 0 {
						nPre++
					}
				}

				for _, id := range it.Rolled().Suffix {
					if id != 0 {
						nSuf++
					}
				}

				got := 0
				for k := range after.RareAffixes {
					if after.RareMask>>uint(k)&1 == 1 {
						got++
					}
				}

				if got != nPre+nSuf {
					t.Errorf("%d affixes saved, game has %d", got, nPre+nSuf)
				}
			case d2drop.QualityMagic:
				if len(it.PrefixCodes)+len(it.SuffixCodes) == 0 {
					break
				}

				for _, pre := range it.PrefixCodes {
					if !strings.Contains(head, tr(pre)) {
						t.Errorf("tooltip %q lacks prefix %q", head, tr(pre))
					}
				}

				for _, suf := range it.SuffixCodes {
					if !strings.Contains(head, tr(suf)) {
						t.Errorf("tooltip %q lacks suffix %q", head, tr(suf))
					}
				}
			case d2drop.QualityUnique:
				if after.UniqueID == 0 && c.unique != "" {
					t.Error("unique id lost")
				}
			case d2drop.QualitySet:
				if len(after.SetProperties) == 0 {
					t.Error("set bonus lists lost")
				}
			}
		})
	}
}

// sameItem compares a game item with the one rebuilt from its saved form.
func sameItem(t *testing.T, where string, a, b *Item, tables *d2s.ItemTables) {
	t.Helper()

	ra, rb := a.Rolled(), b.Rolled()
	if ra.Prefix != rb.Prefix || ra.Suffix != rb.Suffix || ra.RareNames != rb.RareNames || ra.Auto != rb.Auto ||
		ra.Unique != rb.Unique || ra.Quality != rb.Quality || ra.ILvl != rb.ILvl {
		t.Errorf("%s: rolled identity differs\n%+v\n%+v", where, ra, rb)
	}

	if a.rareName != b.rareName || a.UniqueCode != b.UniqueCode || a.SetItemCode != b.SetItemCode ||
		a.SetCode != b.SetCode || a.RolledRuneword() != b.RolledRuneword() {
		t.Errorf("%s: names %q/%q/%q/%q/%q vs %q/%q/%q/%q/%q", where, a.rareName, a.UniqueCode, a.SetItemCode, a.SetCode,
			a.RolledRuneword(), b.rareName, b.UniqueCode, b.SetItemCode, b.SetCode, b.RolledRuneword())
	}

	if strings.Join(a.PrefixCodes, ",") != strings.Join(b.PrefixCodes, ",") ||
		strings.Join(a.SuffixCodes, ",") != strings.Join(b.SuffixCodes, ",") {
		t.Errorf("%s: affixes %v %v vs %v %v", where, a.PrefixCodes, a.SuffixCodes, b.PrefixCodes, b.SuffixCodes)
	}

	sum := func(it *Item) map[propKeyT]int64 {
		m := map[propKeyT]int64{}

		for _, p := range it.StatItem().Props {
			m[propKeyT{p.ID, p.Param}] += p.Value
		}

		for _, p := range it.StatItem().RunewordProps {
			m[propKeyT{-p.ID, p.Param}] += p.Value
		}

		for tier, l := range it.StatItem().SetLists {
			for _, p := range l {
				m[propKeyT{1000*(tier+1) + p.ID, p.Param}] += p.Value
			}
		}

		for k, v := range m {
			id := k.id
			if id < 0 {
				id = -id
			}

			if id >= 1000 {
				id %= 1000
			}

			// stats the save does not hold (the poison count is derived) cannot come back
			if _, _, _, _, saved := tables.StatSaveInfo(id); v == 0 || id == statIDSockets || !saved {
				delete(m, k)
			}
		}

		return m
	}

	sameProps(t, where+" (reloaded)", sum(b), sum(a))

	if a.attributes.defense != b.attributes.defense || a.NumSockets() != b.NumSockets() || a.IsEthereal() != b.IsEthereal() ||
		len(a.Socketed()) != len(b.Socketed()) {
		t.Errorf("%s: defense %d/%d sockets %d/%d ethereal %v/%v children %d/%d", where, a.attributes.defense,
			b.attributes.defense, a.NumSockets(), b.NumSockets(), a.IsEthereal(), b.IsEthereal(), len(a.Socketed()), len(b.Socketed()))
	}
}

func TestSaveRoundTripRuneword(t *testing.T) {
	f := realCreatorFactory(t)
	tables := saveTables(t)

	var shield *Item

	for seed := uint32(1); seed < 400 && shield == nil; seed++ {
		it, err := f.Create(CreateParams{
			Code: "lrg", ILvl: 50, Quality: d2drop.QualityNormal, Seed: seed, Flags: d2drop.FlagForceSockets,
		})
		if err == nil && it.NumSockets() == 3 {
			shield = it
		}
	}

	if shield == nil {
		t.Fatal("no shield with three sockets")
	}

	for _, code := range []string{"r08", "r09", "r07"} {
		r, err := f.Create(CreateParams{Code: code, ILvl: 1, Quality: d2drop.QualityNormal, Seed: 1})
		if err != nil {
			t.Fatal(err)
		}

		if err := shield.Socket(r); err != nil {
			t.Fatal(err)
		}
	}

	shield.Identify()

	_, after := asSaved(t, shield, tables)

	if !after.Runeword || after.RunewordID != 27 {
		t.Fatalf("runeword id %d (%v), want Runeword1 = 27", after.RunewordID, after.Runeword)
	}

	if len(after.Children) != 3 || after.SocketCount != 3 || after.Children[0].Code != "r08" || after.Children[2].Code != "r07" {
		t.Fatalf("socketed runes %+v", after.Children)
	}

	for k, ch := range after.Children {
		if ch.Location != d2s.LocationSocketed || int(ch.X) != k {
			t.Errorf("rune %d: location %d slot %d", k, ch.Location, ch.X)
		}
	}

	want := map[propKeyT]int64{}

	for _, p := range shield.StatItem().RunewordProps {
		if _, _, _, _, ok := tables.StatSaveInfo(p.ID); ok && p.Value != 0 {
			want[propKeyT{p.ID, p.Param}] += p.Value
		}
	}

	sameProps(t, "runeword list", sumProps(after.RunewordProperties), want)
	sameProps(t, "item list", sumProps(after.Properties), wantProps(t, shield, tables))

	spec, err := f.SpecFromD2S(&after)
	if err != nil {
		t.Fatal(err)
	}

	again, err := f.ItemFromSpec(spec)
	if err != nil {
		t.Fatal(err)
	}

	sameItem(t, "runeword", shield, again, tables)
}

// TestSaveAffixIdsMatchTables sweeps every affix row: the id a save keeps for
// it must name the same affix in the tables the tooltip reads.
func TestSaveAffixIdsMatchTables(t *testing.T) {
	f := realCreatorFactory(t)
	_, tr := descTables(t)
	desc, _ := descTables(t)
	c, _ := f.Creator()
	ns, np := c.Affixes.NSuffix, c.Affixes.NPrefix

	bad := 0

	for id := 1; id <= ns+np; id++ {
		row := c.Affixes.Rows[id-1]
		if row.Name == "" {
			continue
		}

		saveID := id
		it := d2s.Item{Quality: d2s.QualityMagic, Identified: true}

		if id > ns {
			saveID = id - ns
			it.MagicPrefix = uint16(saveID)
		} else {
			it.MagicSuffix = uint16(saveID)
		}

		it.Code = "cap"

		lines := desc.Describe(&it, nil)
		if len(lines) == 0 || !strings.Contains(lines[0].Text, tr(row.Name)) {
			bad++

			if bad < 8 {
				t.Errorf("creator affix %d %q (save id %d) reads %q", id, row.Name, saveID, lines[0].Text)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d affix ids disagree", bad)
	}
}
