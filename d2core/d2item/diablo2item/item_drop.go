package diablo2item

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const lodVersion = 100

// classItemTypes are the item types of ItemTypes.txt with a Class set; items of
// these types use the Class Specific rows of ItemRatio.txt. (The loader's
// Class field cannot tell "no class" from the first hero.)
var classItemTypes = map[string]bool{
	"h2h": true, "h2h2": true, "orb": true, "head": true, "ashd": true, "phlm": true,
	"pelt": true, "cloa": true, "abow": true, "aspe": true, "ajav": true,
}

// DropOptions describe the situation of a drop.
type DropOptions struct {
	// Seed seeds the Diablo II generator used for every roll of the drop.
	Seed uint32
	// ILvl is the level of the dropper (monster level, or area level for
	// objects); it becomes the item level of everything dropped.
	ILvl int
	// UpgradeLevel, if > 0, moves the starting treasure class along its
	// level group (monster level, expansion, above normal difficulty).
	UpgradeLevel int
	Players      int
	MagicFind    int
	MaxDrops     int
	// RollExtras makes the generator roll ethereal (5%) and sockets (33%,
	// normal and superior items) for LoD items, VERIFIED in Game.exe (see
	// d2drop/extras.go). Off by default, so existing drops are unchanged; the
	// rolls use a generator derived from the item seed and never touch the
	// drop stream.
	RollExtras bool
	// Difficulty (0 normal, 1 nightmare, 2 hell) caps the rolled sockets.
	Difficulty int
	// Classic is a game without the expansion: classic probabilities and no
	// throwing weapons (see d2drop.Context.Classic).
	Classic bool
	// QualityLevel, if UseQualityLevel, replaces the item level in the quality
	// roll (chests use their tier, see d2drop.Chest).
	QualityLevel    int
	UseQualityLevel bool
	// GoldFind is the gold find of the killer plus that of its owner, applied to
	// the gold amounts of DropAll.
	GoldFind int
}

// dropTables adapts the parsed records to the d2drop interfaces.
type dropTables struct {
	tcs      *d2drop.TreasureTable
	items    map[string]*d2drop.ItemInfo
	ratios   map[[2]bool]*d2drop.Ratio
	prefixes []d2drop.Affix
	suffixes []d2drop.Affix
	uniques  []d2drop.Row
	setItems []d2drop.Row
	spawned  map[string]bool
	dropper  *d2drop.Dropper
	rec      *d2records.RecordManager
}

// Item implements d2drop.ItemSource.
func (t *dropTables) Item(code string) (*d2drop.ItemInfo, bool) {
	i, ok := t.items[code]

	return i, ok
}

// ItemRatio implements d2drop.RatioSource.
func (t *dropTables) ItemRatio(classSpecific, uber bool) (*d2drop.Ratio, bool) {
	r, ok := t.ratios[[2]bool{classSpecific, uber}]

	return r, ok
}

func (f *ItemFactory) dropTables() *dropTables {
	if f.drop != nil {
		return f.drop
	}

	rec := f.asset.Records
	t := &dropTables{
		tcs:     d2drop.NewTreasureTable(),
		items:   make(map[string]*d2drop.ItemInfo),
		ratios:  make(map[[2]bool]*d2drop.Ratio),
		spawned: make(map[string]bool),
		rec:     rec,
	}

	for code, icr := range rec.Item.All {
		info := &d2drop.ItemInfo{
			Code: code, Level: icr.Level, Rarity: icr.Rarity, Spawnable: icr.Spawnable,
			Quest: icr.Quest != 0, Unique: icr.Unique, MagicLevel: icr.MagicLevel,
			Types: rec.FindEquivalentTypesByItemCommonRecord(icr),
		}
		info.Uber = d2drop.UberTier(code, icr.UberCode, icr.UltraCode, icr.Type, info.Types, icr.Quest != 0)

		info.Version = icr.Version

		if tr := rec.Item.Types[icr.Type]; tr != nil {
			info.TypeNormal, info.TypeMagic, info.TypeRare = tr.Normal, tr.Magic, tr.Rare
			info.TypeRarity = tr.Rarity
			info.Throwable = tr.Throwable
		}

		// The class-specific rows are chosen by the item's own type only.
		info.ClassSpecific = classItemTypes[icr.Type]

		t.items[code] = info
	}

	t.loadTreasure()

	for _, r := range rec.Item.Ratios {
		if !r.Version {
			continue
		}

		conv := func(d d2records.DropRatioInfo) d2drop.DropRatio {
			return d2drop.DropRatio{Base: d.Frequency, Divisor: d.Divisor, Min: d.DivisorMin}
		}

		t.ratios[[2]bool{r.ClassSpecific, r.Uber}] = &d2drop.Ratio{
			Unique: conv(r.UniqueDropInfo), Rare: conv(r.RareDropInfo),
			Set: conv(r.SetDropInfo), Magic: conv(r.MagicDropInfo),
			HiQuality: conv(r.HiQualityDropInfo), Normal: conv(r.NormalDropInfo),
		}
	}

	t.prefixes = affixesFromRecords(rec.Item.Magic.Prefix, true)
	t.suffixes = affixesFromRecords(rec.Item.Magic.Suffix, false)

	for name, u := range rec.Item.Unique {
		t.uniques = append(t.uniques, d2drop.Row{
			ID: name, Base: u.Code, Rarity: u.Rarity, Level: u.Level, Version: u.Version,
			Enabled: u.Enabled, Ladder: u.Ladder, NoLimit: u.NoLimit,
		})
	}

	for key, s := range rec.Item.SetItems {
		t.setItems = append(t.setItems, d2drop.Row{
			ID: key, Base: s.ItemCode, Rarity: s.Rarity, Level: s.QualityLevel, Enabled: true,
		})
	}

	// Map iteration is random; sort so seeds give reproducible drops.
	sort.Slice(t.uniques, func(i, j int) bool { return t.uniques[i].ID < t.uniques[j].ID })
	sort.Slice(t.setItems, func(i, j int) bool { return t.setItems[i].ID < t.setItems[j].ID })

	t.dropper = &d2drop.Dropper{TCs: t.tcs, Items: t, Ratios: t}
	f.drop = t

	return t
}

// loadTreasure fills the treasure class table from the expansion records
// (falling back to the classic ones) and appends the generated item-type
// classes ("armo3" ... "armo96").
func (t *dropTables) loadTreasure() {
	src := t.rec.Item.Treasure.Expansion
	if len(src) == 0 {
		src = t.rec.Item.Treasure.Normal
	}

	// The records are a map; the loader keeps the file position in Index,
	// and the game walks level groups in file order.
	recs := make([]*d2records.TreasureClassRecord, 0, len(src))
	for _, r := range src {
		recs = append(recs, r)
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Index != recs[j].Index {
			return recs[i].Index < recs[j].Index
		}

		return recs[i].Name < recs[j].Name
	})

	for _, r := range recs {
		tc := &d2drop.TreasureClass{
			Name: r.Name, Group: r.Group, Level: r.Level, Picks: r.NumPicks, NoDrop: r.FreqNoDrop,
			Mods: d2drop.QualityMods{
				Unique: r.FreqUnique, Set: r.FreqSet, Rare: r.FreqRare, Magic: r.FreqMagic,
			},
		}

		for _, tr := range r.Treasures {
			e := d2drop.ParseEntry(tr.Code, tr.Probability)

			switch {
			case t.items[e.Code] != nil, src[e.Code] != nil:
			case t.rec.Item.Unique[e.Code] != nil:
				e.Kind, e.Base, e.Version = d2drop.EntryUnique, t.rec.Item.Unique[e.Code].Code, t.rec.Item.Unique[e.Code].Version
			case t.rec.Item.SetItems[e.Code] != nil:
				e.Kind, e.Base = d2drop.EntrySet, t.rec.Item.SetItems[e.Code].ItemCode
			}

			tc.Entries = append(tc.Entries, e)
		}

		t.tcs.Add(tc)
	}

	items := make([]*d2drop.ItemInfo, 0, len(t.items))
	for _, i := range t.items {
		items = append(items, i)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })

	typeCodes := make([]string, 0)

	for code, ty := range t.rec.Item.Types {
		if ty.TreasureClass == 1 {
			typeCodes = append(typeCodes, code)
		}
	}

	sort.Strings(typeCodes)

	for _, g := range d2drop.BuildTypeTreasureClasses(items, typeCodes) {
		if _, exists := t.tcs.TreasureClass(g.Name); !exists {
			t.tcs.Add(g)
		}
	}
}

func affixesFromRecords(m map[string]*d2records.ItemAffixCommonRecord, prefix bool) []d2drop.Affix {
	out := make([]d2drop.Affix, 0, len(m))

	for name, a := range m {
		out = append(out, d2drop.Affix{
			ID: name, Prefix: prefix, Version: a.Version, Spawnable: a.Spawnable, Rare: a.Rare,
			Level: a.Level, MaxLevel: a.MaxLevel, Frequency: a.Frequency, Group: a.GroupID,
			Class: a.Class, IType: a.ItemInclude, EType: a.ItemExclude,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// DropResult is what a treasure class produced.
type DropResult struct {
	Items []*Item
	// Gold holds the amounts of the gold drops, in drop order.
	Gold []int
}

// DropItems rolls the treasure class tcName the way the real game does (see
// package d2drop): nested classes, NoDrop, quality from ItemRatio with magic
// find, item level from the dropper, weighted affix selection. Results are
// reproducible from opts.Seed. Gold drops are left out; see DropAll.
func (f *ItemFactory) DropItems(tcName string, opts DropOptions) ([]*Item, error) {
	res, err := f.DropAll(tcName, opts)
	if err != nil {
		return nil, err
	}

	return res.Items, nil
}

// DropAll is DropItems that also returns the gold. The amount of a gold drop is
// rolled on a generator of its own seeded from the dropper's (the way every
// dropped item is created): rand(5*ilvl)+ilvl, scaled by the "mul" of the
// entry, then by the gold find (VERIFIED formulas, see d2drop.GoldAmount).
func (f *ItemFactory) DropAll(tcName string, opts DropOptions) (*DropResult, error) {
	t := f.dropTables()
	rng := d2rand.New(opts.Seed)

	drops, err := t.dropper.Roll(&d2drop.Context{
		RNG: rng, ILvl: opts.ILvl, UpgradeLevel: opts.UpgradeLevel, Players: opts.Players,
		MagicFind: opts.MagicFind, MaxDrops: opts.MaxDrops, Classic: opts.Classic,
		QualityLevel: opts.QualityLevel, UseQualityLevel: opts.UseQualityLevel,
	}, tcName)
	if err != nil {
		return nil, fmt.Errorf("rolling %q: %w", tcName, err)
	}

	res := &DropResult{}

	for i := range drops {
		if drops[i].Code == d2drop.GoldCode {
			res.Gold = append(res.Gold, goldDrop(rng, &drops[i], opts.GoldFind))

			continue
		}

		if item := f.itemFromDrop(t, rng, &drops[i]); item != nil {
			if opts.RollExtras {
				f.rollExtras(t, item, opts.Difficulty)
			}

			res.Items = append(res.Items, item)
		}
	}

	return res, nil
}

// goldDrop rolls the amount of one gold drop.
func goldDrop(rng *d2rand.Seed, drop *d2drop.Drop, goldFind int) int {
	own := d2rand.New(rng.Step())
	amount := d2drop.GoldAmount(own, drop.ILvl, 0)

	if drop.Mul != 0 {
		amount = d2drop.ScaleGoldMul(amount, drop.Mul)
	}

	if goldFind != 0 {
		amount = d2drop.ApplyGoldFind(amount, goldFind, 0)
	}

	return amount
}

func (f *ItemFactory) itemFromDrop(t *dropTables, rng *d2rand.Seed, drop *d2drop.Drop) *Item {
	icr := f.asset.Records.Item.All[drop.Code]
	if icr == nil {
		return nil // gold and other codes that are not items in this build
	}

	info := t.items[drop.Code]
	item := &Item{
		factory:    f,
		CommonCode: drop.Code,
		TypeCode:   icr.Type,
		itemLevel:  drop.ILvl,
		Seed:       int64(rng.Step()),
	}
	// nolint:gosec // not concerned with crypto-strong randomness
	item.rand = rand.New(rand.NewSource(item.Seed))

	quality := drop.Quality
	aff := &d2drop.AffixItem{
		ILvl: drop.ILvl, QLvl: info.Level, MagicLevel: info.MagicLevel, Types: info.Types,
		Version: lodVersion, Jewel: icr.Type == jewelItemCode,
	}

	for quality != d2drop.QualityNone {
		aff.Quality = quality

		if f.applyQuality(t, rng, item, drop, aff, quality) {
			break
		}

		next := d2drop.FallbackQuality(quality)
		if next == quality {
			break
		}

		quality = next
	}

	item.genQuality = quality

	return item.init()
}

// applyQuality fills the item for a quality; false means nothing could be
// picked and the caller should fall back to the next quality.
func (f *ItemFactory) applyQuality(
	t *dropTables, rng *d2rand.Seed, item *Item, drop *d2drop.Drop, aff *d2drop.AffixItem, q d2drop.Quality,
) bool {
	rec := f.asset.Records.Item

	switch q {
	case d2drop.QualityUnique:
		id := drop.ForcedID

		if id == "" {
			row := d2drop.PickRow(rng, t.uniques, item.CommonCode, drop.ILvl, lodVersion, true, t.spawned)
			if row == nil {
				return false
			}

			id = row.ID
		}

		if rec.Unique[id] == nil {
			return false
		}

		if !rec.Unique[id].NoLimit {
			t.spawned[id] = true
		}

		item.UniqueCode = id
	case d2drop.QualitySet:
		id := drop.ForcedID

		if id == "" {
			row := d2drop.PickRow(rng, t.setItems, item.CommonCode, drop.ILvl, lodVersion, true, nil)
			if row == nil {
				return false
			}

			id = row.ID
		}

		si := rec.SetItems[id]
		if si == nil {
			return false
		}

		item.SetItemCode, item.SetCode = id, si.SetKey
	case d2drop.QualityRare:
		// The rare names (RarePrefix/RareSuffix) are not modelled by Item yet,
		// only the magic affixes of the rare.
		return setAffixes(item, d2drop.RollRareAffixes(rng, t.prefixes, t.suffixes, aff))
	case d2drop.QualityMagic:
		return setAffixes(item, d2drop.RollMagicAffixes(rng, t.prefixes, t.suffixes, aff))
	}

	return true
}

func setAffixes(item *Item, res d2drop.MagicAffixes) bool {
	for _, a := range res.Prefixes {
		item.PrefixCodes = append(item.PrefixCodes, a.ID)
	}

	for _, a := range res.Suffixes {
		item.SuffixCodes = append(item.SuffixCodes, a.ID)
	}

	return len(res.Prefixes)+len(res.Suffixes) > 0
}
