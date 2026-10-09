package d2drop

// RNG is the random source used for every roll. *d2rand.Seed implements it.
type RNG interface {
	// Roll returns a number in [0, n); it returns 0 for n < 1.
	Roll(n int32) uint32
	// Chance is a fair coin flip (one generator step, lowest bit).
	Chance() bool
}

// Quality is an item quality as numbered by the game.
type Quality int

// Item qualities (verified from the switch in ITEMGEN_ApplyQualityToItem).
const (
	QualityNone     Quality = 0
	QualityLow      Quality = 1
	QualityNormal   Quality = 2
	QualitySuperior Quality = 3
	QualityMagic    Quality = 4
	QualitySet      Quality = 5
	QualityRare     Quality = 6
	QualityUnique   Quality = 7
	QualityCrafted  Quality = 8
)

// QualityMods are the six quality modifiers of a treasure class, in the order
// the game stores them. The first four come from the Magic, Rare, Set and
// Unique columns and are in units of 1/1024: they reduce the denominator of
// the matching quality roll. E and G are the "ce"/"cg" modifiers whose
// meaning is unverified (the game rolls rand(1024) against them).
type QualityMods struct {
	Magic, Rare, Set, Unique, E, G int
}

// Max merges two sets of modifiers slot by slot, the way a nested treasure
// class inherits from its parent (the larger value wins, a zero never wins).
func (m QualityMods) Max(o QualityMods) QualityMods {
	return QualityMods{
		Magic:  maxInt(m.Magic, o.Magic),
		Rare:   maxInt(m.Rare, o.Rare),
		Set:    maxInt(m.Set, o.Set),
		Unique: maxInt(m.Unique, o.Unique),
		E:      maxInt(m.E, o.E),
		G:      maxInt(m.G, o.G),
	}
}

// EntryKind says what a treasure class entry refers to.
type EntryKind int

const (
	// EntryAuto means: a treasure class if one with that name exists,
	// otherwise a base item code.
	EntryAuto EntryKind = iota
	// EntryUnique is a named row of UniqueItems (always quality unique).
	EntryUnique
	// EntrySet is a named row of SetItems (always quality set).
	EntrySet
)

// Entry is one ItemN/ProbN pair of a treasure class.
type Entry struct {
	Code string // item code, treasure class name, or unique/set item name
	Prob int
	Kind EntryKind
	// Base is the base item code of a unique or set row (Kind != EntryAuto):
	// the item the game creates for that row. The loader fills it from
	// UniqueItems/SetItems.
	Base string
	// Mods are the per-entry modifiers (cm, cr, cs, cu, ce, cg).
	Mods QualityMods
	// Mul is the gold multiplier of "gld,mul=N" entries (0 if absent).
	Mul int
}

// TreasureClass is one row of TreasureClassEx.txt.
type TreasureClass struct {
	Name   string
	Group  int
	Level  int
	Picks  int
	NoDrop int
	Mods   QualityMods
	// Entries in file order.
	Entries []Entry
}

// TotalProb is the sum of the entry probabilities.
func (tc *TreasureClass) TotalProb() int {
	total := 0

	for i := range tc.Entries {
		total += tc.Entries[i].Prob
	}

	return total
}

// TreasureSource resolves treasure classes.
type TreasureSource interface {
	// TreasureClass looks a treasure class up by name.
	TreasureClass(name string) (*TreasureClass, bool)
	// Upgrade implements ITEMGEN_GetTreasureClassByLevel: for grouped
	// classes ("Act 1 Good" -> "Act 1 (N) Good") it returns the class of the
	// group that applies at the given level.
	Upgrade(tc *TreasureClass, level int) *TreasureClass
}

// ItemInfo is what the dropper needs to know about a base item.
type ItemInfo struct {
	Code          string
	Level         int // qlvl, the base item's level
	Rarity        int
	TypeRarity    int // ItemTypes.Rarity of the item's type: the weight in generated "armo3"-style classes
	Types         []string // item type and all of its ancestors
	Spawnable     bool
	Quest         bool
	Unique        bool // base item that is always unique
	ClassSpecific bool
	Uber          bool // exceptional/elite tier (selects the Uber ItemRatio row)
	MagicLevel    int
	TypeNormal    bool // ItemTypes.Normal: always normal quality
	TypeMagic     bool // ItemTypes.Magic: always magic quality
	TypeRare      bool // ItemTypes.Rare: may be rare
}

// HasType reports whether the item is of the type code (or descends from it).
func (i *ItemInfo) HasType(code string) bool {
	for _, t := range i.Types {
		if t == code {
			return true
		}
	}

	return false
}

// ItemSource looks base items up by code.
type ItemSource interface {
	Item(code string) (*ItemInfo, bool)
}

// DropRatio is one Unique/Rare/Set/Magic column triple of ItemRatio.txt.
type DropRatio struct {
	Base, Divisor, Min int
}

// Ratio is a row of ItemRatio.txt.
type Ratio struct {
	Unique, Rare, Set, Magic DropRatio
	HiQuality, Normal        DropRatio // Min unused
}

// RatioSource selects the ItemRatio row for a kind of item.
type RatioSource interface {
	ItemRatio(classSpecific, uber bool) (*Ratio, bool)
}

// UberTier reports whether an item selects the "Uber" rows of ItemRatio.txt
// (VERIFIED against the game's tier test, 62b650): an armor or weapon (it has
// "armo" or "weap" among its types) that is the exceptional or elite version
// of its base (its code equals its ubercode or ultracode), that is not a
// missile potion ("tpot") and not a quest item. Items of the normal tier and
// everything that is neither armor nor weapon use the plain rows.
func UberTier(code, uberCode, ultraCode, primaryType string, types []string, quest bool) bool {
	isGear := false

	for _, t := range types {
		if t == "armo" || t == "weap" {
			isGear = true
		}
	}

	switch {
	case !isGear, code == "":
		return false
	case code != uberCode && code != ultraCode:
		return false
	}

	return primaryType != "tpot" && !quest
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
