package d2drop

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	// DefaultMaxDrops is the cap on items one monster kill produces.
	DefaultMaxDrops = 6
	// maxTCDepth is the nesting limit of the game's explicit stack (0x3f).
	maxTCDepth = 0x3f

	// dynamicTCStep is the level span of generated item-type classes
	// ("armo3", "armo6", ... "armo96").
	dynamicTCStep = 3
	dynamicTCMax  = 96
)

// TreasureTable is a TreasureSource backed by the file order of
// TreasureClassEx.txt (the order matters for level groups).
type TreasureTable struct {
	list   []*TreasureClass
	byName map[string]int
}

// NewTreasureTable returns an empty table.
func NewTreasureTable() *TreasureTable {
	return &TreasureTable{byName: map[string]int{}}
}

// Add appends a class. A later class of the same name shadows the earlier
// one for lookups.
func (t *TreasureTable) Add(tc *TreasureClass) {
	t.byName[tc.Name] = len(t.list)
	t.list = append(t.list, tc)
}

// TreasureClass implements TreasureSource.
func (t *TreasureTable) TreasureClass(name string) (*TreasureClass, bool) {
	i, ok := t.byName[name]
	if !ok {
		return nil, false
	}

	return t.list[i], true
}

// Len is the number of classes.
func (t *TreasureTable) Len() int { return len(t.list) }

// Upgrade implements ITEMGEN_GetTreasureClassByLevel (VERIFIED structure,
// 6567f0): for a class with a non-zero group, walk the following classes of
// the same group and stop at the last one whose successor in the group has a
// level above the given one. level <= 0 (or group 0) returns tc itself.
func (t *TreasureTable) Upgrade(tc *TreasureClass, level int) *TreasureClass {
	if level <= 0 || tc.Group == 0 {
		return tc
	}

	i, ok := t.byName[tc.Name]
	if !ok {
		return tc
	}

	for i+1 < len(t.list) && t.list[i+1].Group == tc.Group && t.list[i+1].Level <= level {
		i++
	}

	return t.list[i]
}

// ParseEntry parses an ItemN string of treasureclassex.txt such as
// `gld,mul=1280` or `weap3,cu=800,cs=0` into an Entry (resolution order of
// the game: item code, treasure class, unique, set, is done by the caller via
// Kind and Auto lookup).
func ParseEntry(s string, prob int) Entry {
	s = strings.Trim(s, `"`)
	parts := strings.Split(s, ",")
	e := Entry{Code: strings.TrimSpace(parts[0]), Prob: prob}

	for _, p := range parts[1:] {
		k, v, ok := strings.Cut(strings.TrimSpace(p), "=")
		if !ok {
			continue
		}

		n, err := strconv.Atoi(v)
		if err != nil {
			continue
		}

		switch k {
		case "cm":
			e.Mods.Magic = n
		case "cr":
			e.Mods.Rare = n
		case "cs":
			e.Mods.Set = n
		case "cu":
			e.Mods.Unique = n
		case "ce":
			e.Mods.E = n
		case "cg":
			e.Mods.G = n
		case "mul":
			e.Mul = n
		}
	}

	return e
}

// BuildTypeTreasureClasses generates the item-type classes of
// ITEMGEN_BuildItemTypeTreasureClasses (655ae0): for every type code and
// N = 3, 6, ... 96 a class "<code><N>" with one pick, level N-3, and one
// entry per spawnable, non-quest item of that type whose level is in the
// range (N-3, N] (VERIFIED; OpenDiablo2 used [N, N+3)). The entry
// probability is the Rarity column of the item's ItemTypes row (VERIFIED
// against the emulated game), at least 1; it is not the item's own rarity.
func BuildTypeTreasureClasses(items []*ItemInfo, typeCodes []string) []*TreasureClass {
	var out []*TreasureClass

	for _, code := range typeCodes {
		for n := dynamicTCStep; n <= dynamicTCMax; n += dynamicTCStep {
			tc := &TreasureClass{
				Name:  code + strconv.Itoa(n),
				Level: n - dynamicTCStep,
				Picks: 1,
			}

			for _, it := range items {
				if it.Quest || !it.Spawnable || !it.HasType(code) {
					continue
				}

				// Missile potions are left out of every class but their own
				// (VERIFIED, 655ae0), although "tpot" descends from "weap".
				if code != "tpot" && it.HasType("tpot") {
					continue
				}

				if it.Level > n-dynamicTCStep && it.Level <= n {
					tc.Entries = append(tc.Entries, Entry{Code: it.Code, Prob: maxInt(1, it.TypeRarity)})
				}
			}

			out = append(out, tc)
		}
	}

	return out
}

// Drop is one item produced by a treasure class roll.
type Drop struct {
	Code    string  // base item code (or the unique/set row name for ForcedID kinds)
	Quality Quality // rolled quality; unique/set rows force 7/5
	// ForcedID is the unique or set item name when the entry named one.
	ForcedID string
	ILvl     int
	// Mul is the gold multiplier of the entry (0 if none).
	Mul int
	// Mods are the merged quality modifiers in effect for this drop.
	Mods QualityMods
	// EFlag, GFlag are the results of rand(1024) < Mods.E / Mods.G
	// (UNVERIFIED semantics).
	EFlag, GFlag bool
}

// Context is the situation of one drop.
type Context struct {
	RNG RNG
	// ILvl is the dropper's level (monster level, or area level for
	// objects). It is the item level of everything dropped.
	ILvl int
	// UpgradeLevel, if > 0, upgrades the starting class through its level
	// group (the game does this with the monster level only in expansion,
	// above normal difficulty, for monsters).
	UpgradeLevel int
	// Players is the number of players in range (>= 1); it scales NoDrop.
	Players   int
	MagicFind int
	// ForcedQuality, if set, replaces the quality roll.
	ForcedQuality Quality
	// MaxDrops caps the number of items (default DefaultMaxDrops).
	MaxDrops int
	// NoNoDrop disables the NoDrop column (the roller's "guaranteed" flag:
	// every pick yields an entry). VERIFIED against the real roller.
	NoNoDrop bool
	// QualityLevel, if UseQualityLevel is set, replaces the item level in the
	// quality roll only (the items keep ILvl). The game hands the roller a
	// level of its own: the monster level for monsters, but the chest tier
	// (0..2) for chests. VERIFIED. It is not clamped (ILvl is, to at least 1).
	QualityLevel    int
	UseQualityLevel bool
	// Classic is a game without the expansion (game+0x70 == 0): classic
	// probabilities (expansion-only entries have no weight), and throwable
	// items are re-picked up to maxClassicRetries times, then replaced by a
	// long sword. VERIFIED against the real roller.
	Classic bool
}

// Dropper rolls treasure classes.
type Dropper struct {
	TCs    TreasureSource
	Items  ItemSource
	Ratios RatioSource

	mu      sync.Mutex
	classic map[*TreasureClass][]int
}

type frame struct {
	tc        *TreasureClass
	remaining int
	mods      QualityMods
	w         []int // entry weights in effect (Prob, or the classic weights)
	total     int
}

func (d *Dropper) newFrame(ctx *Context, tc *TreasureClass, parent *QualityMods) frame {
	f := frame{tc: tc, mods: tc.Mods, remaining: 1}

	if parent != nil {
		f.mods = parent.Max(tc.Mods)
	}

	if tc.Picks != 0 {
		f.remaining = absInt(tc.Picks)
	}

	if ctx.Classic {
		f.w = d.classicWeights(tc, nil)
	} else {
		f.w = make([]int, len(tc.Entries))
		for i := range tc.Entries {
			f.w[i] = tc.Entries[i].Prob
		}
	}

	for _, w := range f.w {
		f.total += w
	}

	return f
}

// classicWeights are the entry weights of a classic (non-expansion) game
// (VERIFIED against the real roller, all classes of TreasureClassEx): an
// entry that exists only in the expansion has weight 0. That is an item or
// a unique/set row of version >= 100, or a reference to a treasure class
// whose own classic total is 0 (derived recursively).
func (d *Dropper) classicWeights(tc *TreasureClass, busy map[*TreasureClass]bool) []int {
	d.mu.Lock()
	cached, ok := d.classic[tc]
	d.mu.Unlock()

	if ok {
		return cached
	}

	if busy == nil {
		busy = map[*TreasureClass]bool{}
	}

	busy[tc] = true
	w := make([]int, len(tc.Entries))

	for i := range tc.Entries {
		e := &tc.Entries[i]
		w[i] = e.Prob

		if d.expansionOnly(e, busy) {
			w[i] = 0
		}
	}

	delete(busy, tc)

	d.mu.Lock()
	if d.classic == nil {
		d.classic = map[*TreasureClass][]int{}
	}

	d.classic[tc] = w
	d.mu.Unlock()

	return w
}

func (d *Dropper) expansionOnly(e *Entry, busy map[*TreasureClass]bool) bool {
	if e.Kind == EntryAuto {
		if sub, isTC := d.TCs.TreasureClass(e.Code); isTC {
			if busy[sub] {
				return false
			}

			total := 0
			for _, w := range d.classicWeights(sub, busy) {
				total += w
			}

			return total == 0
		}

		if d.Items != nil {
			if info, found := d.Items.Item(e.Code); found && info.Version >= expansionVersion {
				return true
			}
		}
	}

	return e.Version >= expansionVersion
}

// expansionVersion is the version column value of expansion-only content.
const expansionVersion = 100

// maxClassicRetries is how many times a classic game re-picks after rolling a
// throwable item before it hands out a long sword instead (VERIFIED).
const maxClassicRetries = 10

// classicReplacement is the item a classic game drops once the retries ran out.
const classicReplacement = "lsd"

func absInt(a int) int {
	if a < 0 {
		return -a
	}

	return a
}

// EffectiveNoDrop is the NoDrop weight with the multi-player correction of
// ITEMGEN_RollTreasureClassDrops: for n > 1 players it becomes
// total*f^n/(1-f^n) with f = nodrop/(nodrop+total), truncated. The structure
// is VERIFIED, the exact rounding is UNVERIFIED (the float math is in x87
// registers). For one player it is the NoDrop column.
func EffectiveNoDrop(noDrop, total, players int) int {
	if noDrop <= 0 || players <= 1 {
		return noDrop
	}

	f := float64(noDrop) / float64(noDrop+total)
	fn := f

	for i := 1; i < players; i++ {
		fn *= f // the game multiplies one factor at a time
	}

	if 1-fn == 0 {
		return 0
	}

	return int(float64(total) * fn / (1 - fn))
}

// NoDropPlayers is the player count n the roller scales NoDrop with
// (VERIFIED, 5585d0): a party of p (capped at 8) counts fully and the other
// players of the game count half, n = p + (total-p)/2. A party of one counts
// as one whatever the game size. For a dropping monster n is capped by the
// monster's "players" stat (stat 100), at least 1.
func NoDropPlayers(party, total, monsterPlayers int, monster bool) int {
	n := party
	if n <= 1 {
		n = 1
	} else {
		if n > 8 {
			n = 8
		}

		n += (total - n) / 2
	}

	if monster {
		n = minInt(n, maxInt(monsterPlayers, 1))
	}

	return n
}

// Roll resolves the treasure class tcName into drops
// (ITEMGEN_RollTreasureClassDrops): nested classes with an explicit stack,
// NoDrop, Prob-weighted picks, negative picks as a deterministic walk, quality
// modifiers merged down the stack, and quality rolled per item with the
// dropper's level as item level.
//
// Simplifications: only the expansion probabilities are used (no classic
// version split), and per-entry modifiers apply to that item only instead of
// being merged back into the frame.
func (d *Dropper) Roll(ctx *Context, tcName string) ([]Drop, error) {
	start, ok := d.TCs.TreasureClass(tcName)
	if !ok {
		return nil, fmt.Errorf("unknown treasure class %q", tcName)
	}

	if ctx.UpgradeLevel > 0 {
		start = d.TCs.Upgrade(start, ctx.UpgradeLevel)
	}

	limit := ctx.MaxDrops
	if limit <= 0 {
		limit = DefaultMaxDrops
	}

	ilvl := maxInt(ctx.ILvl, 1)
	stack := []frame{d.newFrame(ctx, start, nil)}
	retries := 0

	var out []Drop

	for len(stack) > 0 && len(out) < limit {
		f := &stack[len(stack)-1]
		// A class without weight is abandoned before any roll, whatever its
		// NoDrop (VERIFIED, classic games: expansion-only classes).
		if f.remaining <= 0 || f.total == 0 {
			stack = stack[:len(stack)-1]
			continue
		}

		idx := d.pick(ctx, f)
		f.remaining--

		if idx < 0 {
			continue
		}

		e := f.tc.Entries[idx]

		if e.Kind == EntryAuto {
			if sub, isTC := d.TCs.TreasureClass(e.Code); isTC {
				if len(stack) >= maxTCDepth {
					return out, fmt.Errorf("treasure class %q nested too deep", tcName)
				}

				stack = append(stack, d.newFrame(ctx, sub, &f.mods))

				continue
			}
		}

		if ctx.Classic && d.expansionOnly(&e, nil) {
			continue // the game skips such an item without rolling anything
		}

		// The entry's own modifiers are merged into the frame, so they also
		// apply to the later picks of the same class (VERIFIED).
		f.mods = f.mods.Max(e.Mods)

		drop, err := d.makeDrop(ctx, e, f.mods, ilvl)
		if err != nil {
			return out, err
		}

		// A classic game has no throwing weapons: it picks again (the rolls
		// above are spent) and, after maxClassicRetries, drops a long sword.
		if ctx.Classic && d.throwable(drop.Code) {
			retries++

			if retries <= maxClassicRetries {
				f.remaining++

				continue
			}

			drop.Code = classicReplacement
		}

		out = append(out, drop)
	}

	return out, nil
}

// pick chooses the entry index of one pick of a frame, or -1 for no drop.
func (d *Dropper) pick(ctx *Context, f *frame) int {
	tc := f.tc

	if tc.Picks < 0 {
		// Deterministic mode: the n-th pick is the entry whose weight range
		// holds n (VERIFIED: the real code uses n as the roll).
		n := absInt(tc.Picks) - f.remaining
		if n >= f.total {
			return -1
		}

		return entryAt(f.w, n)
	}

	total := f.total
	noDrop := EffectiveNoDrop(tc.NoDrop, total, ctx.Players)

	if ctx.NoNoDrop {
		noDrop = 0
	}

	space := total + noDrop

	if space < 1 {
		return -1
	}

	roll := int(ctx.RNG.Roll(int32(space)))
	if roll < noDrop {
		return -1
	}

	roll -= noDrop

	return entryAt(f.w, roll)
}

// entryAt is the entry whose weight range contains roll, or -1.
func entryAt(w []int, roll int) int {
	for i, x := range w {
		if roll < x {
			return i
		}

		roll -= x
	}

	return -1
}

// throwable reports whether the base item is of a throwable primary type.
func (d *Dropper) throwable(code string) bool {
	if d.Items == nil {
		return false
	}

	info, ok := d.Items.Item(code)

	return ok && info.Throwable
}

func (d *Dropper) makeDrop(ctx *Context, e Entry, mods QualityMods, ilvl int) (Drop, error) {
	drop := Drop{Code: e.Code, ILvl: ilvl, Mul: e.Mul, Mods: mods, Quality: QualityNormal}

	if e.Kind != EntryAuto && e.Base != "" {
		drop.Code = e.Base
	}

	switch {
	case e.Kind == EntryUnique:
		drop.Quality, drop.ForcedID = QualityUnique, e.Code
	case e.Kind == EntrySet:
		drop.Quality, drop.ForcedID = QualitySet, e.Code
	case ctx.ForcedQuality != QualityNone:
		drop.Quality = ctx.ForcedQuality
	default:
		qlvl := ilvl
		if ctx.UseQualityLevel {
			qlvl = ctx.QualityLevel
		}

		q, err := d.quality(ctx, e.Code, qlvl, mods)
		if err != nil {
			return drop, err
		}

		drop.Quality = q
	}

	// Each flag is rolled only when its modifier is set (VERIFIED: rand &
	// 0x3ff compared with the 16-bit modifier).
	if mods.E > 0 {
		drop.EFlag = int(ctx.RNG.Roll(1024)) < mods.E
	}

	if mods.G > 0 {
		drop.GFlag = int(ctx.RNG.Roll(1024)) < mods.G
	}

	return drop, nil
}

func (d *Dropper) quality(ctx *Context, code string, ilvl int, mods QualityMods) (Quality, error) {
	if d.Items == nil {
		return QualityNormal, nil
	}

	info, ok := d.Items.Item(code)
	if !ok {
		return QualityNormal, nil // not a base item we know (e.g. gold)
	}

	if d.Ratios == nil {
		return QualityNormal, fmt.Errorf("no ItemRatio source")
	}

	ratio, ok := d.Ratios.ItemRatio(info.ClassSpecific, info.Uber)
	if !ok {
		return QualityNormal, fmt.Errorf("no ItemRatio row for %q", code)
	}

	return RollQuality(ctx.RNG, ratio, QualityInput{
		ILvl: ilvl, QLvl: info.Level, MagicFind: ctx.MagicFind, Mods: mods,
		TypeNormal: info.TypeNormal, TypeMagic: info.TypeMagic, TypeRare: info.TypeRare,
		Unique: info.Unique, Quest: info.Quest,
	}), nil
}
