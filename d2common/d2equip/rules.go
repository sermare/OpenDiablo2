package d2equip

import "fmt"

// Class codes of the ItemTypes.txt "Class" column.
const (
	ClassAmazon      = "ama"
	ClassSorceress   = "sor"
	ClassNecromancer = "nec"
	ClassPaladin     = "pal"
	ClassBarbarian   = "bar"
	ClassDruid       = "dru"
	ClassAssassin    = "ass"
)

// Item is an item as the equip rules see it.
type Item struct {
	Code string
	Type string // item type code (armor.txt/weapons.txt "type")

	Weapon    bool
	TwoHanded bool // weapons.txt 2handed
	OneOrTwo  bool // weapons.txt 1or2handed

	Identified bool
	Ethereal   bool

	// Requirements: the base values (armor.txt/weapons.txt reqstr, reqdex) and the
	// level requirement (the highest of the base's levelreq and the item's affixes').
	ReqStr, ReqDex, ReqLevel int
	// ReqPercent is the item's "requirements %" stat (stat 91, e.g. -20 for "Requirements -20%").
	ReqPercent int

	Durability, MaxDurability int
	NoDurability              bool
	Indestructible            bool // stat 152 (item_indesctructible), VERIFIED in 0x629b00
}

// Hero is what the rules need to know about the hero.
type Hero struct {
	Class string // ama, sor, nec, pal, bar, dru, ass
	Str   int
	Dex   int
	Level int
}

// Reason says why an equip was refused.
type Reason string

// The reasons.
const (
	ReasonOK           Reason = "ok"
	ReasonBodyLoc      Reason = "item does not fit this body location"
	ReasonClass        Reason = "item is restricted to another class"
	ReasonUnidentified Reason = "item is not identified"
	ReasonStrength     Reason = "not enough strength"
	ReasonDexterity    Reason = "not enough dexterity"
	ReasonLevel        Reason = "character level too low"
	ReasonDualWield    Reason = "this class cannot wield a weapon in the off hand"
	ReasonNoAmmoWeapon Reason = "ammunition needs a weapon that shoots it"
	ReasonBroken       Reason = "item is broken"
	ReasonNoRoom       Reason = "no room in the inventory for the item that has to make way"
)

// Decision is the verdict on placing an item in a body location.
type Decision struct {
	OK     bool
	Reason Reason
	// Detail is a human readable explanation (needed and current values).
	Detail string
	// Displaced lists the locations whose items must leave the body for this one to
	// fit (a two-handed weapon pushes out the shield, a shield pushes out a
	// two-handed weapon; the caller puts them in the inventory or refuses when full).
	Displaced []Loc
}

func refuse(r Reason, format string, a ...interface{}) Decision {
	return Decision{Reason: r, Detail: fmt.Sprintf(format, a...)}
}

// Rules bundles the item type table with the rule functions.
type Rules struct {
	Types *Types
}

// FitsLoc reports whether an item of the given type may sit in the body location
// (ItemTypes.txt Body/BodyLoc1/BodyLoc2, with the weapon switch locations 11/12
// accepting what 4/5 accept).
func (r Rules) FitsLoc(itemType string, loc Loc) bool {
	loc = loc.Primary()

	for _, l := range r.Types.Locs(itemType) {
		if l == loc {
			return true
		}
	}

	return false
}

// IsShield reports whether the item type is a shield (Any Shield).
func (r Rules) IsShield(itemType string) bool { return r.Types.IsA(itemType, "shld") }

// IsQuiver reports whether the item type is ammunition for a shooting weapon
// (the type has a Quiver column).
func (r Rules) IsQuiver(itemType string) bool {
	return r.Types.find(itemType, func(t *Type) string { return t.Quiver }, 0) != ""
}

// shoots returns the quiver type a weapon type needs ("" for non shooting weapons).
func (r Rules) shoots(itemType string) string {
	return r.Types.find(itemType, func(t *Type) string { return t.Shoots }, 0)
}

// twoHanded reports whether the weapon occupies both hands for this class. A
// Barbarian may wield the swords marked 1or2handed in one hand (UNVERIFIED in
// the binary: INV_CheckHandItemsCompatible special-cases class 4, the column
// is community knowledge).
func twoHanded(class string, it *Item) bool {
	if !it.Weapon || !it.TwoHanded {
		return false
	}

	return !(class == ClassBarbarian && it.OneOrTwo)
}

// canWieldOffhandWeapon says whether a class may hold a weapon (not a shield or
// ammunition) in the left hand slot. Barbarians (any one handed weapon) and
// Assassins (claws, type h2h) only: INV_CheckHandItemsCompatible special-cases
// classes 4 and 6 (inventory-trade.md, the claw type check is UNVERIFIED).
func (r Rules) canWieldOffhandWeapon(class string, it *Item) bool {
	switch class {
	case ClassBarbarian:
		return true
	case ClassAssassin:
		return r.Types.IsA(it.Type, "h2h")
	}

	return false
}

// CheckHands checks a right/left hand pair (either may be nil) of one weapon
// set for class. It returns nil when the pair can be held together.
func (r Rules) CheckHands(class string, right, left *Item) *Decision {
	if left == nil {
		return nil
	}

	if r.IsQuiver(left.Type) {
		if right == nil || !right.Weapon || !r.Types.IsA(left.Type, r.shoots(right.Type)) || r.shoots(right.Type) == "" {
			d := refuse(ReasonNoAmmoWeapon, "%s needs a weapon that shoots it", left.Code)

			return &d
		}

		return nil
	}

	if left.Weapon && !r.IsShield(left.Type) {
		if !r.canWieldOffhandWeapon(class, left) || twoHanded(class, left) {
			d := refuse(ReasonDualWield, "%s cannot hold %s in the off hand", class, left.Code)

			return &d
		}
	}

	return nil
}

// Place decides whether it may be placed at loc given the body: body maps each
// occupied location to its item (a location's current item is replaced, not
// counted as a conflict). The requirement checks use the hero as given.
func (r Rules) Place(h Hero, body map[Loc]*Item, it *Item, loc Loc) Decision {
	if it == nil || loc == LocNone || int(loc) >= NumLocs {
		return refuse(ReasonBodyLoc, "no item or location")
	}

	if !r.FitsLoc(it.Type, loc) {
		return refuse(ReasonBodyLoc, "%s (%s) does not fit %s", it.Code, it.Type, loc)
	}

	if req := Check(h, it, r.Types); !req.OK() {
		return Decision{Reason: req.Reason(), Detail: req.Detail()}
	}

	// a broken item (0 durability) can still be placed in the body: only its
	// properties are off (see Broken)
	if !loc.IsHand() {
		return Decision{OK: true, Reason: ReasonOK}
	}

	return r.placeHand(h, body, it, loc)
}

func (r Rules) placeHand(h Hero, body map[Loc]*Item, it *Item, loc Loc) Decision {
	other := loc.Other()
	otherItem := body[other]
	d := Decision{OK: true, Reason: ReasonOK}

	right, left := it, otherItem // loc is a right hand slot
	if loc == LocLeftHand || loc == LocSwapLeft {
		right, left = otherItem, it
	}

	// the item itself in the off hand slot
	if left == it {
		if !it.Weapon && !r.IsShield(it.Type) && !r.IsQuiver(it.Type) {
			return Decision{OK: true, Reason: ReasonOK} // not a hand item at all: the loc check decides
		}
	}

	if left == it && r.IsQuiver(it.Type) {
		if dec := r.CheckHands(h.Class, right, left); dec != nil {
			return *dec
		}

		return d
	}

	if left == it && it.Weapon && !r.IsShield(it.Type) {
		if dec := r.CheckHands(h.Class, nil, left); dec != nil {
			return *dec
		}
	}

	if otherItem == nil {
		return d
	}

	// two-handed weapon vs off hand: the other hand's item has to leave
	if right == it && twoHanded(h.Class, it) {
		if r.shoots(it.Type) != "" && left != nil && r.IsQuiver(left.Type) && r.Types.IsA(left.Type, r.shoots(it.Type)) {
			return d // bow + its quiver
		}

		d.Displaced = []Loc{other}

		return d
	}

	if left == it && right != nil && twoHanded(h.Class, right) && !r.IsQuiver(it.Type) {
		d.Displaced = []Loc{other}

		return d
	}

	// a quiver in the other hand while the new right hand weapon does not shoot it
	if right == it && left != nil && r.IsQuiver(left.Type) {
		if dec := r.CheckHands(h.Class, right, left); dec != nil {
			d.Displaced = []Loc{other}
		}
	}

	// an off hand weapon that the new main hand makes illegal cannot happen:
	// the class decides, not the main hand
	return d
}
