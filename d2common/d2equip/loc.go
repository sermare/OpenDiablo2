package d2equip

// Loc is a body location, numbered like the equipment slot of a .d2s item.
type Loc int

// Body locations.
const (
	LocNone      Loc = 0
	LocHead      Loc = 1
	LocNeck      Loc = 2
	LocTorso     Loc = 3
	LocRightHand Loc = 4 // primary weapon (set I)
	LocLeftHand  Loc = 5 // shield or second weapon (set I)
	LocRightRing Loc = 6
	LocLeftRing  Loc = 7
	LocBelt      Loc = 8
	LocFeet      Loc = 9
	LocGloves    Loc = 10
	LocSwapRight Loc = 11 // set II weapon
	LocSwapLeft  Loc = 12 // set II shield
	// NumLocs is the number of locations including LocNone.
	NumLocs = 13
)

var locNames = [NumLocs]string{"none", "head", "neck", "torso", "rhand", "lhand", "rring", "lring",
	"belt", "feet", "gloves", "swap-rhand", "swap-lhand"}

// String names the location.
func (l Loc) String() string {
	if l < 0 || int(l) >= len(locNames) {
		return "invalid"
	}

	return locNames[l]
}

// locCodes maps the codes of bodylocs.txt (as ItemTypes.txt BodyLoc1/2 use them) to locations.
var locCodes = map[string]Loc{
	"head": LocHead, "neck": LocNeck, "tors": LocTorso, "rarm": LocRightHand, "larm": LocLeftHand,
	"rrin": LocRightRing, "lrin": LocLeftRing, "belt": LocBelt, "feet": LocFeet, "glov": LocGloves,
}

// LocFromCode returns the location of a bodylocs.txt code ("tors", "rarm", ...).
func LocFromCode(code string) Loc { return locCodes[code] }

// Primary maps the weapon switch locations to the hand location they mirror
// (INV_CheckItemFitsBodyLoc accepts 11/12 when 4/5 are accepted).
func (l Loc) Primary() Loc {
	switch l {
	case LocSwapRight:
		return LocRightHand
	case LocSwapLeft:
		return LocLeftHand
	}

	return l
}

// IsSwap reports whether the location belongs to weapon set II.
func (l Loc) IsSwap() bool { return l == LocSwapRight || l == LocSwapLeft }

// Other returns the other hand of the same weapon set (4<->5, 11<->12), or LocNone.
func (l Loc) Other() Loc {
	switch l {
	case LocRightHand:
		return LocLeftHand
	case LocLeftHand:
		return LocRightHand
	case LocSwapRight:
		return LocSwapLeft
	case LocSwapLeft:
		return LocSwapRight
	}

	return LocNone
}

// IsHand reports whether the location is one of the four weapon/shield places.
func (l Loc) IsHand() bool { return l == LocRightHand || l == LocLeftHand || l.IsSwap() }
