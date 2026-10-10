package herogen

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Place is where an item sits: the location, the equipment slot or the page and cell.
// X is the equipment slot for worn items and the belt cell (row*4+column) for
// belt potions, like the saves the game writes.
type Place struct {
	Location uint8 // d2s.LocationEquipped, LocationBelt or LocationStored
	Slot     uint8 // equipment slot (1 head ... 10 gloves), for worn items
	Page     uint8 // 1 inventory, 5 stash, for stored items
	X, Y     uint8
}

// Page numbers of stored items.
const (
	PageInventory uint8 = 1
	PageStash     uint8 = 5
)

// Equipment slots of a save.
const (
	SlotHead      uint8 = 1
	SlotAmulet    uint8 = 2
	SlotTorso     uint8 = 3
	SlotRightHand uint8 = 4
	SlotLeftHand  uint8 = 5
	SlotRingRight uint8 = 6
	SlotRingLeft  uint8 = 7
	SlotBelt      uint8 = 8
	SlotFeet      uint8 = 9
	SlotGloves    uint8 = 10
)

// Worn returns the place of a worn item.
func Worn(slot uint8) Place { return Place{Location: d2s.LocationEquipped, Slot: slot, X: slot} }

// InBelt returns the place of a belt potion (cell = row*4+column).
func InBelt(cell uint8) Place { return Place{Location: d2s.LocationBelt, X: cell} }

// InInventory returns a cell of the inventory page.
func InInventory(x, y uint8) Place {
	return Place{Location: d2s.LocationStored, Page: PageInventory, X: x, Y: y}
}

// ItemSpec describes one item of the hero.
type ItemSpec struct {
	// Code is the base item code; for a unique it may be left empty and is
	// taken from the UniqueItems row.
	Code string
	// Unique is the UniqueItems.txt name (the "index" column) to create, or
	// empty for a plain item (potions, tomes).
	Unique string
	// ILvl is the item level; 0 means 99. It is raised to the row's minimum.
	ILvl int
	// Quantity overrides the stack size of a stackable item (0 keeps 1).
	Quantity int
	Place    Place
}

// ErrItem is returned when an item cannot be created.
var ErrItem = errors.New("herogen: item")

// Statistics ids the conversion treats specially (ItemStatCost row ids).
const (
	statDefense       = 0x1f
	statMaxDurability = 0x49
	statQuantity      = 0x46
	statIndestruct    = 152
)

// itemFlags is the flag word every extended item of a 1.14b save carries
// besides the identified bit (the samples all have bit 23 set; meaning unknown).
const itemFlags uint32 = 0x800000

type stackKey struct{ id, param int }

// buildItem creates the save item for a spec. index only feeds the seed and
// the item id, so the file is the same on every run.
func (t *Tables) buildItem(spec ItemSpec, index int) (d2s.Item, error) {
	if spec.Unique == "" {
		return t.plainItem(spec, index)
	}

	cr := t.Creator
	row := -1

	for i := range cr.Uniques.Uniques {
		if cr.Uniques.Uniques[i].Name == spec.Unique {
			row = i
			break
		}
	}

	if row < 0 {
		return d2s.Item{}, fmt.Errorf("%w: unique %q is not in UniqueItems", ErrItem, spec.Unique)
	}

	u := cr.Uniques.Uniques[row]
	if spec.Code != "" && spec.Code != u.Code {
		return d2s.Item{}, fmt.Errorf("%w: unique %q is a %s, not a %s", ErrItem, spec.Unique, u.Code, spec.Code)
	}

	if !u.Enabled || u.Ladder {
		return d2s.Item{}, fmt.Errorf("%w: unique %q is disabled or ladder only", ErrItem, spec.Unique)
	}

	ilvl := spec.ILvl
	if ilvl == 0 {
		ilvl = 99
	}

	if ilvl < u.Lvl {
		ilvl = u.Lvl
	}

	var seed d2rand.Seed
	seed.Init(itemHash("seed", index, u.Code))

	rolled, err := cr.Create(d2drop.Request{
		Code: u.Code, ILvl: ilvl, Quality: d2drop.QualityUnique, Difficulty: 2, Version: 100, Expansion: true,
		ForcedID: row + 1, Flags: d2drop.FlagNoEthereal | d2drop.FlagNoSockets, GameSeed: seed,
	})
	if err != nil {
		return d2s.Item{}, fmt.Errorf("%w: creating %q: %v", ErrItem, spec.Unique, err)
	}

	if rolled.Quality != d2drop.QualityUnique || rolled.Unique != row {
		return d2s.Item{}, fmt.Errorf("%w: %q came out as quality %d row %d, want unique row %d",
			ErrItem, spec.Unique, rolled.Quality, rolled.Unique, row)
	}

	base := cr.Items.ByCode[u.Code]
	it := d2s.Item{
		Flags: itemFlags, Identified: true, Code: u.Code, ID: itemHash("id", index, u.Code),
		Level: uint8(clamp(rolled.ILvl, 1, 127)), Quality: d2s.QualityUnique, UniqueID: uint16(row),
		Version: d2s.DefaultItemVersion,
	}

	t.place(&it, spec.Place)

	if ty := cr.Items.Type(base); ty != nil && ty.VarInvGfx > 0 {
		it.HasPicture, it.Picture = true, uint8(rolled.Gfx[1]&7)
	}

	if rolled.Auto != 0 {
		it.HasClassData, it.ClassData = true, uint16(rolled.Auto)
	}

	props, err := t.properties(rolled.Writes)
	if err != nil {
		return it, fmt.Errorf("%w: %q: %v", ErrItem, spec.Unique, err)
	}

	it.Properties = props

	t.baseStats(&it, rolled.Writes)

	if spec.Quantity > 0 && t.Save.IsStackable(it.Code) {
		it.Quantity = uint16(clamp(spec.Quantity, 1, 511))
	}

	built, err := d2s.NewItem(it, t.Save)
	if err != nil {
		return it, fmt.Errorf("%w: %q: %v", ErrItem, spec.Unique, err)
	}

	return built, nil
}

// plainItem is a potion, tome or other item without magical properties.
func (t *Tables) plainItem(spec ItemSpec, index int) (d2s.Item, error) {
	if t.Save.ItemKindOf(spec.Code) == 0 {
		return d2s.Item{}, fmt.Errorf("%w: unknown item code %q", ErrItem, spec.Code)
	}

	it := d2s.Item{
		Flags: itemFlags, Identified: true, Code: spec.Code, ID: itemHash("id", index, spec.Code),
		Level: uint8(clamp(spec.ILvl, 1, 127)), Quality: d2s.QualityNormal, Version: d2s.DefaultItemVersion,
	}

	if spec.ILvl == 0 {
		it.Level = 12
	}

	t.place(&it, spec.Place)

	if t.Save.IsStackable(spec.Code) {
		it.Quantity = uint16(clamp(spec.Quantity, 1, 511))
	}

	if t.Save.IsCompact(spec.Code) {
		it.ID, it.Level, it.Quality = 0, 0, 0
	}

	built, err := d2s.NewItem(it, t.Save)
	if err != nil {
		return it, fmt.Errorf("%w: %q: %v", ErrItem, spec.Code, err)
	}

	return built, nil
}

func (t *Tables) place(it *d2s.Item, p Place) {
	it.Location, it.Equipped, it.X, it.Y, it.Page = p.Location, p.Slot, p.X, p.Y, p.Page
}

// baseStats fills the defense and the durability from the unit stats the
// generator wrote. The durability is full, plus any "+N durability" property.
// An indestructible item keeps no durability, like the game's saves.
func (t *Tables) baseStats(it *d2s.Item, writes []d2drop.StatWrite) {
	var def, maxDur, qty int

	// the last write of a stat wins; the current durability is rolled between half and full, but the
	// sample hero starts with repaired gear, so only the maximum is read
	for _, w := range writes {
		if w.Kind != 'S' {
			continue
		}

		switch w.Stat {
		case statDefense:
			def = w.Value
		case statMaxDurability:
			maxDur = w.Value
		case statQuantity:
			qty = w.Value
		}
	}

	// a stackable weapon (javelins, throwing knives and axes) keeps the stack the generator rolled
	if t.Save.IsStackable(it.Code) {
		it.Quantity = uint16(clamp(qty, 1, 511))
	}

	if t.Save.ItemKindOf(it.Code) == d2s.KindArmor {
		it.Defense = def
	}

	it.MaxDurability = uint16(clamp(maxDur, 0, 255))
	it.Durability = it.MaxDurability

	for _, p := range it.Properties {
		switch p.ID {
		case statMaxDurability:
			it.Durability = uint16(clamp(int(it.MaxDurability)+int(p.Value), 0, 511))
		case statIndestruct:
			if p.Value != 0 {
				it.MaxDurability, it.Durability = 0, 0
			}
		}
	}
}

// properties turns the stat writes of the item's own list into the property
// list of a save: values are unshifted (ValShift), summed per stat and
// parameter ('L' adds, 'M' sets), stats the save does not hold are skipped
// and the followers of a damage group follow their leader.
func (t *Tables) properties(writes []d2drop.StatWrite) ([]d2s.Property, error) {
	vals := map[stackKey]int64{}

	for _, w := range writes {
		if w.Kind == 'S' {
			continue
		}

		if _, _, _, _, ok := t.Save.StatSaveInfo(w.Stat); !ok {
			continue
		}

		shift := 0
		if sh := t.Creator.Props.ValShift; w.Stat >= 0 && w.Stat < len(sh) {
			shift = sh[w.Stat]
		}

		v := int64(w.Value >> uint(shift))
		k := stackKey{w.Stat, w.Param}

		if w.Kind == 'M' {
			vals[k] = v
		} else {
			vals[k] += v
		}
	}

	follower := map[int]bool{}

	for _, fs := range groupFollowers {
		for _, f := range fs {
			follower[f] = true
		}
	}

	var keys []stackKey

	for k := range vals {
		if !follower[k.id] {
			keys = append(keys, k)
		}
	}

	sort.Slice(keys, func(a, b int) bool {
		if keys[a].id != keys[b].id {
			return keys[a].id < keys[b].id
		}

		return keys[a].param < keys[b].param
	})

	var out []d2s.Property

	for _, k := range keys {
		out = append(out, d2s.Property{ID: k.id, Param: uint32(k.param), Value: vals[k]})

		for _, f := range groupFollowers[k.id] {
			fv, ok := vals[stackKey{f, 0}]
			if !ok {
				return nil, fmt.Errorf("stat %d has no follower %d", k.id, f)
			}

			out = append(out, d2s.Property{ID: f, Value: fv})
		}
	}

	return out, nil
}

// groupFollowers lists the stats a save stores right behind their leader
// (the same table as the d2s package keeps privately).
var groupFollowers = map[int][]int{
	17: {18}, 48: {49}, 50: {51}, 52: {53}, 54: {55, 56}, 57: {58, 59},
}

func itemHash(kind string, index int, code string) uint32 {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "herogen/%s/%d/%s", kind, index, code)

	return h.Sum32() | 1
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
