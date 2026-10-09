package d2herostats

// Faster hit recovery / cast rate / block recovery / attack speed.
//
// Effective percent: the game does not apply the item percent linearly; it
// diminishes it (community-documented, 1.14):
//
//	eff = floor(120*p / (120+p))
//	frames = ceil(256*A / floor(S*(100+eff)/100))
//
// A is the action's base frame count and S the animation speed (256 normally).
// STATUS: the FCR and necromancer FHR parameters below were FITTED so that the
// formula reproduces the community breakpoint tables (see the tests), they were
// not read from the binary; the other FHR/FBR tables are literal community
// values (unverified, no formula fit found). The IAS function is the
// community formula, unverified.

// Kind of speed modifier.
type Kind string

// Speed modifier kinds.
const (
	FHR Kind = "FHR" // faster hit recovery
	FCR Kind = "FCR" // faster cast rate (generic spells)
	FBR Kind = "FBR" // faster block recovery
)

// EffectivePct is the diminished percent: floor(120*p/(120+p)).
func EffectivePct(p int) int {
	if p <= 0 {
		return 0
	}

	return 120 * p / (120 + p)
}

// Frames returns ceil(256*a/floor(s*(100+eff(p))/100)). a is the base frame
// count, s the animation speed (256 normal).
func Frames(a, s, p int) int {
	spd := s * (100 + EffectivePct(p)) / 100
	if spd < 1 {
		spd = 1
	}

	return (256*a + spd - 1) / spd
}

// Breakpoints lists the smallest percent that reaches each successive frame
// count, starting with 0, and stops after `entries` entries (the games tables
// end where the animation hits its minimum).
func Breakpoints(a, s, entries int) []int {
	out := make([]int, 0, entries)
	cur := -1

	for p := 0; p < 100000 && len(out) < entries; p++ {
		if f := Frames(a, s, p); f != cur {
			out = append(out, p)
			cur = f
		}
	}

	return out
}

// Model is a fitted formula: base frames A at animation speed S.
type Model struct {
	A, S, Entries int
}

// Table returns the breakpoints the model produces.
func (m Model) Table() []int { return Breakpoints(m.A, m.S, m.Entries) }

// Entry is one class/kind breakpoint table. Model is non-nil when the formula
// reproduces Table (checked by a test), nil for literal community data.
type Entry struct {
	Class string
	Kind  Kind
	Note  string // animation the table applies to
	Table []int
	Model *Model
}

// Tables are the breakpoint tables per class (1.14 community values; literal
// ones are unverified against the binary).
var Tables = []Entry{
	{Class: "Amazon", Kind: FHR, Table: []int{0, 6, 13, 20, 32, 52, 86, 174, 600}},
	{Class: "Sorceress", Kind: FHR, Table: []int{0, 5, 9, 14, 20, 30, 42, 60, 86, 142, 280}},
	{Class: "Necromancer", Kind: FHR, Table: []int{0, 9, 20, 37, 63, 105, 200}, Model: &Model{14, 256, 7}},
	{Class: "Paladin", Kind: FHR, Table: []int{0, 7, 15, 27, 48, 86, 200}},
	{Class: "Barbarian", Kind: FHR, Table: []int{0, 9, 20, 42, 86, 280}},
	{Class: "Druid", Kind: FHR, Note: "human form", Table: []int{0, 3, 7, 13, 19, 29, 42, 63, 99, 174, 456}},
	{Class: "Assassin", Kind: FHR, Table: []int{0, 7, 15, 27, 48, 86, 200}},

	{Class: "Amazon", Kind: FCR, Table: []int{0, 7, 14, 22, 32, 48, 68, 99, 152}, Model: &Model{20, 256, 9}},
	{Class: "Sorceress", Kind: FCR, Table: []int{0, 9, 20, 37, 63, 105, 200}, Model: &Model{14, 256, 7}},
	{Class: "Necromancer", Kind: FCR, Table: []int{0, 9, 18, 30, 48, 75, 125}, Model: &Model{16, 256, 7}},
	{Class: "Paladin", Kind: FCR, Table: []int{0, 9, 18, 30, 48, 75, 125}, Model: &Model{16, 256, 7}},
	{Class: "Barbarian", Kind: FCR, Table: []int{0, 9, 20, 37, 63, 105, 200}, Model: &Model{14, 256, 7}},
	{Class: "Druid", Kind: FCR, Table: []int{0, 4, 10, 19, 30, 46, 68, 99, 163}, Model: &Model{15, 208, 9}},
	{Class: "Assassin", Kind: FCR, Table: []int{0, 8, 16, 27, 42, 65, 102, 174}, Model: &Model{17, 256, 8}},
	{Class: "Sorceress", Kind: FCR, Note: "lightning/chain lightning", Table: []int{0, 7, 15, 23, 35, 52, 78, 117, 194}, Model: &Model{19, 256, 9}},

	{Class: "Amazon", Kind: FBR, Table: []int{0, 13, 32, 86, 600}},
	{Class: "Sorceress", Kind: FBR, Table: []int{0, 5, 9, 14, 20, 30, 42, 60, 86, 142, 280}},
	{Class: "Necromancer", Kind: FBR, Table: []int{0, 7, 15, 27, 48, 86, 200}},
	{Class: "Paladin", Kind: FBR, Note: "with a shield", Table: []int{0, 5, 9, 14, 20, 30, 42, 60, 86, 142, 280}},
	{Class: "Barbarian", Kind: FBR, Table: []int{0, 9, 20, 42, 86, 280}},
	{Class: "Druid", Kind: FBR, Table: []int{0, 9, 20, 42, 86, 280}},
	{Class: "Assassin", Kind: FBR, Table: []int{0, 13, 32, 86, 600}},
}

// Lookup returns the first table for the class and kind (the default
// animation); ok is false when there is none.
func Lookup(class string, kind Kind) (Entry, bool) {
	for _, e := range Tables {
		if e.Class == class && e.Kind == kind {
			return e, true
		}
	}

	return Entry{}, false
}

// Reached returns the index of the highest breakpoint of the table that the
// given percent reaches (0 for none), i.e. how many frames were saved.
func Reached(table []int, pct int) int {
	idx := 0

	for i, bp := range table {
		if pct >= bp {
			idx = i
		}
	}

	return idx
}

// AttackFrames is the community attack speed formula (UNVERIFIED):
//
//	eias   = floor(120*ias/(120+ias)) + skillIAS - wsm   (clamped to [-85, 75])
//	frames = ceil(256*a / floor(256*(100+eias)/100)) - 1
//
// a is the base frames of the attack animation, wsm the weapon speed modifier
// (negative for fast weapons), skillIAS the skill's own speed bonus. The final
// -1 reflects that the attack action fires its hit on the last frame; this was
// not checked in the binary.
func AttackFrames(a, ias, wsm, skillIAS int) int {
	eias := EffectivePct(ias) + skillIAS - wsm
	if eias > 75 {
		eias = 75
	}

	if eias < -85 {
		eias = -85
	}

	spd := 256 * (100 + eias) / 100

	return (256*a+spd-1)/spd - 1
}
