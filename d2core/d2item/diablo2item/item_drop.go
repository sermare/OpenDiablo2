package diablo2item

import (
	"errors"
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
	// Difficulty is 0 normal, 1 nightmare, 2 hell; it limits sockets and
	// scales item creation.
	Difficulty int
}

// dropTables adapts the parsed records to the d2drop interfaces.
type dropTables struct {
	tcs     *d2drop.TreasureTable
	items   map[string]*d2drop.ItemInfo
	ratios  map[[2]bool]*d2drop.Ratio
	dropper *d2drop.Dropper
	rec     *d2records.RecordManager
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
		tcs:    d2drop.NewTreasureTable(),
		items:  make(map[string]*d2drop.ItemInfo),
		ratios: make(map[[2]bool]*d2drop.Ratio),
		rec:    rec,
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
	entries, err := f.rollEntries(tcName, opts)
	if err != nil {
		return nil, err
	}

	res := &DropResult{}

	for _, e := range entries {
		if e.Item != nil {
			res.Items = append(res.Items, e.Item)
		} else {
			res.Gold = append(res.Gold, e.Gold)
		}
	}

	return res, nil
}

// rollEntries is DropAll keeping the drop order of items and gold.
func (f *ItemFactory) rollEntries(tcName string, opts DropOptions) ([]LootEntry, error) {
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

	var out []LootEntry

	for i := range drops {
		if drops[i].Code == d2drop.GoldCode {
			out = append(out, LootEntry{Gold: goldDrop(rng, &drops[i], opts.GoldFind)})

			continue
		}

		if item := f.itemFromDrop(t, rng, &drops[i], dropGame{classic: opts.Classic, difficulty: opts.Difficulty}); item != nil {
			out = append(out, LootEntry{Item: item})
		}
	}

	return out, nil
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

// dropGame is the game the drop happens in: the creation reads its version and
// difficulty.
type dropGame struct {
	classic    bool
	difficulty int
}

// itemFromDrop creates one dropped item with the item creator: the quality,
// unique or set row and the ethereal / socket modifiers come from the
// treasure class roll, everything else is rolled by the creator from the
// drop's generator.
func (f *ItemFactory) itemFromDrop(_ *dropTables, rng *d2rand.Seed, drop *d2drop.Drop, g dropGame) *Item {
	if f.asset.Records.Item.All[drop.Code] == nil {
		return nil // gold and other codes that are not items in this build
	}

	p := CreateParams{
		Code: drop.Code, ILvl: drop.ILvl, Quality: drop.Quality, Name: drop.ForcedID,
		Difficulty: g.difficulty, Classic: g.classic, Rng: rng,
	}

	if drop.EFlag {
		p.Flags |= d2drop.FlagForceEthereal
	}

	if drop.GFlag {
		p.Flags |= d2drop.FlagForceSockets
	}

	item, err := f.Create(p)
	if err != nil {
		// a classic game does not create expansion items; without creator
		// tables (a bare test asset) the old item model is used
		if errors.Is(err, d2drop.ErrNotCreated) {
			return nil
		}

		return f.legacyItemFromDrop(rng, drop)
	}

	return item
}

// legacyItemFromDrop builds an item without the creator (no tables to load
// it from): the base item with its quality, properties rolled by the old
// model.
func (f *ItemFactory) legacyItemFromDrop(rng *d2rand.Seed, drop *d2drop.Drop) *Item {
	icr := f.asset.Records.Item.All[drop.Code]
	item := &Item{
		factory:    f,
		CommonCode: drop.Code,
		TypeCode:   icr.Type,
		itemLevel:  drop.ILvl,
		quality:    drop.Quality,
		Seed:       int64(rng.Step()),
	}
	// nolint:gosec // not concerned with crypto-strong randomness
	item.rand = rand.New(rand.NewSource(item.Seed))

	return item.init()
}
