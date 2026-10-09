package d2drop

// BaseItem is a row of weapons.txt, armor.txt or misc.txt reduced to what item
// creation reads. The game concatenates the three tables in that order and
// uses the index as the item's class id; Class is that index.
type BaseItem struct {
	Class int
	Kind  BaseKind

	Code, NormCode, UberCode, UltraCode string
	Type, Type2                         string
	Version                             int // 0 classic, 100 Lord of Destruction
	Level, LevelReq                     int
	Rarity                              int
	Spawnable                           bool
	Quest, QuestDiffCheck               int
	Unique                              bool
	MagicLevel                          int
	AutoPrefix                          int // AutoMagic group (0 = none)

	MinAC, MaxAC                 int // armor
	Block                        int
	Absorbs                      int
	Speed                        int
	Durability                   int
	NoDurability                 bool
	MinDam, MaxDam               int // one-handed melee (misc.txt: none)
	TwoHandMinDam, TwoHandMaxDam int
	MinMisDam, MaxMisDam         int // thrown / missile
	StrBonus, DexBonus           int
	ReqStr, ReqDex               int
	Stackable                    bool
	MinStack, MaxStack           int
	SpawnStack                   int
	HasInv                       bool
	GemSockets                   int
	Throwable                    bool
	Useable                      bool
	TwoHanded                    bool
	OneOrTwoHanded               bool
	Cost                         int
	InvWidth, InvHeight          int
}

// BaseKind says which table a base item comes from.
type BaseKind int

// The three base item tables, in the order the game concatenates them.
const (
	KindWeapon BaseKind = iota
	KindArmor
	KindMisc
)

// ItemType is a row of ItemTypes.txt. Index is the row index, which the game
// uses as the type id (row 0 is the empty "None" type).
type ItemType struct {
	Index                int
	Code                 string
	Equiv1, Equiv2       string
	Normal, Magic, Rare  bool
	Charm, Gem, Beltable bool
	MaxSock1, MaxSock25  int
	MaxSock40            int
	TreasureClass        bool
	Rarity               int
	Class                int // hero class index (amazon 0 ... assassin 6), -1 for none
	VarInvGfx            int
	Throwable            bool
	Body                 bool
	Quiver               bool
	AutoStack            bool
	StaffMods            string
	CostFormula          int
	Ancestors            []string // this type and every type it descends from (via Equiv1/Equiv2)
}

// IsA reports whether the type is, or descends from, the type with that code.
func (t *ItemType) IsA(code string) bool {
	for _, a := range t.Ancestors {
		if a == code {
			return true
		}
	}

	return false
}

// ItemTables is everything item creation needs to know about base items and
// item types.
type ItemTables struct {
	Items  []*BaseItem // indexed by class id
	ByCode map[string]*BaseItem
	Types  map[string]*ItemType
}

// Type returns the ItemTypes row of a base item (its primary type).
func (t *ItemTables) Type(b *BaseItem) *ItemType {
	return t.Types[b.Type]
}

// IsA reports whether a base item is of the given type code, directly or by
// descent (type or type2).
func (t *ItemTables) IsA(b *BaseItem, code string) bool {
	for _, ty := range []string{b.Type, b.Type2} {
		if tt := t.Types[ty]; tt != nil && tt.IsA(code) {
			return true
		}
	}

	return false
}
