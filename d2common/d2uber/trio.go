package d2uber

// The endgame trio reward (Game.exe 0x5dee90): the game remembers which of
// three special monster classes were killed. When the third one dies, one
// large charm (code cm2) with its skill charges set to the maximum drops, and
// then one item of the standard treasure class per entry of the game's player
// counter (Game +0x8c; whether that counts players is unverified). A kill of
// any other class is a fatal error in the exe; here it is ignored.
//
// The class numbers 0x2c0, 0x2c1 and 0x2c5 are VERIFIED from the exe. Which
// monsters they are is not known here, so the tracker only takes numbers.

// Trio classes.
const (
	TrioClassA = 0x2c0
	TrioClassB = 0x2c1
	TrioClassC = 0x2c5

	// TrioCharm is the charm that drops with maximum charges.
	TrioCharm = "cm2"
)

// TrioDrop is the loot of the completed trio.
type TrioDrop struct {
	Charm       string // TrioCharm; set the skill charges of this item to max
	MaxCharges  bool
	StandardNum int // number of standard-treasure items to drop after it
}

// Trio remembers the kills of one game.
type Trio struct {
	a, b, c bool
}

// Kill records a kill. It returns the drop when this kill completes the trio
// (and on any later kill of a trio class, since the exe tests the three flags
// after every kill; callers should not feed it more than once per monster).
// players is the game's counter (Game +0x8c), not clamped below zero except
// for the loop in the exe, which does nothing for 0 or less.
func (t *Trio) Kill(class, players int) (TrioDrop, bool) {
	switch class {
	case TrioClassA:
		t.a = true
	case TrioClassB:
		t.b = true
	case TrioClassC:
		t.c = true
	default:
		return TrioDrop{}, false
	}

	if !(t.a && t.b && t.c) {
		return TrioDrop{}, false
	}

	n := players
	if n < 0 {
		n = 0
	}

	return TrioDrop{Charm: TrioCharm, MaxCharges: true, StandardNum: n}, true
}
