package d2cube

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Item is an item in the cube (or the result of a recipe). It mirrors what the
// hero file keeps of an item, so the game converts to and from it directly.
type Item struct {
	Code     string
	Quality  d2drop.Quality // 0 is treated as normal
	ILvl     int
	Seed     int64
	Unique   string
	SetItem  string
	Set      string
	Prefixes []string
	Suffixes []string
	Ethereal bool
	Quantity int // stack size, 0 = default

	Sockets  int      // number of sockets
	Socketed []string // codes of the gems, runes and jewels in them
	Runeword string   // name of the runeword the item became

	// Mods are the properties a recipe attached (crafted mods, extra sockets
	// are not here but in Sockets).
	Mods []RolledMod

	// Damaged is set by recipes that repair: the game sets full durability.
	Repaired   bool
	Recharged  bool
	Identified bool
	Crafted    bool
}

// RolledMod is a recipe property with its value rolled.
type RolledMod struct {
	Code  string
	Param string
	Min   int
	Max   int
	Value int // rolled between Min and Max
}

func (i *Item) quality() d2drop.Quality {
	if i.Quality == 0 {
		return d2drop.QualityNormal
	}

	return i.Quality
}

// Context is the state of the game a recipe is checked against.
type Context struct {
	PlayerLevel int
	Difficulty  int    // 0 normal, 1 nightmare, 2 hell
	Ladder      bool   // ladder / realm game
	Expansion   bool   // Lord of Destruction
	Class       string // class code: ama sor nec pal bar dru ass
}

// Match is a recipe together with the cube items that satisfy its inputs.
type Match struct {
	Recipe *Recipe
	// Assigned[k] lists the cube item indexes that fill Recipe.Inputs[k].
	Assigned [][]int
}

func (c *Context) allows(r *Recipe) bool {
	switch {
	case !r.Enabled:
		return false
	case r.Ladder && !c.Ladder:
		return false
	case c.Difficulty < r.MinDiff:
		return false
	case r.Version >= 100 && !c.Expansion:
		return false
	}

	if len(r.Class) > 0 {
		ok := false

		for _, k := range r.Class {
			if k == c.Class {
				ok = true
			}
		}

		if !ok {
			return false
		}
	}

	return c.condition(r)
}

// condition evaluates op/param/value. UNVERIFIED: the 1.14b table only uses op
// 28 (the quest recipes: staff, Khalim's flail, Cow portal, Pandemonium keys),
// which is always true here; no other operator appears in the table, so any
// other op makes the recipe unusable instead of guessing its meaning.
func (c *Context) condition(r *Recipe) bool {
	return r.Op == 0 || r.Op == 28
}

// IsIngredient reports whether an item is a recipe ingredient rather than the
// item a recipe works on: gems, runes, jewels, scrolls, potions, books.
func (cat *Catalog) IsIngredient(it *Item) bool {
	b, ok := cat.bases[it.Code]
	if !ok {
		return false
	}

	for _, t := range []string{"sock", "scro", "poti", "book", "rpot", "key"} {
		if b.HasType(t) {
			return true
		}
	}

	return false
}

// Matches reports whether one cube item satisfies an input (ignoring qty).
func (cat *Catalog) Matches(in *Input, it *Item) bool {
	b, ok := cat.bases[it.Code]
	if !ok {
		return false
	}

	switch {
	case in.Any:
	case in.Code != "":
		if it.Code != in.Code {
			return false
		}
	case in.Type != "":
		if !b.HasType(in.Type) {
			return false
		}
	case in.Unique != "":
		if it.quality() != d2drop.QualityUnique || !uniqueNameIs(it.Unique, in.Unique) {
			return false
		}
	}

	if in.Quality != 0 && it.quality() != in.Quality {
		return false
	}

	if (in.Ethereal == yes3 && !it.Ethereal) || (in.Ethereal == no3 && it.Ethereal) {
		return false
	}

	if (in.Socketed == yes3 && it.Sockets == 0) || (in.Socketed == no3 && it.Sockets > 0) {
		return false
	}

	if in.Tier != TierNone && b.Tier != in.Tier {
		return false
	}

	if in.Upgrade && !(b.Tier == TierBasic || b.Tier == TierExceptional) {
		return false
	}

	return true
}

func uniqueNameIs(have, want string) bool {
	norm := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		return strings.TrimPrefix(s, "the ")
	}

	return have != "" && norm(have) == norm(want)
}

// Find returns the first recipe of the table (in table order, like the game)
// whose inputs are exactly the items in the cube, or nil.
func (cat *Catalog) Find(tab *Table, ctx *Context, items []Item) *Match {
	if len(items) == 0 {
		return nil
	}

	for _, r := range tab.Recipes {
		if !ctx.allows(r) {
			continue
		}

		if m := cat.assign(r, items); m != nil {
			return m
		}
	}

	return nil
}

// assign searches an assignment of cube items to the inputs such that every
// cube item is used exactly once.
func (cat *Catalog) assign(r *Recipe, items []Item) *Match {
	// quick count check: the number of items must equal the sum of the inputs'
	// item counts (a stackable input of qty N is one item holding N)
	need := 0

	for k := range r.Inputs {
		need += cat.itemCount(&r.Inputs[k])
	}

	if need != len(items) {
		return nil
	}

	used := make([]bool, len(items))
	asg := make([][]int, len(r.Inputs))

	var solve func(k int) bool

	solve = func(k int) bool {
		if k == len(r.Inputs) {
			return true
		}

		in := &r.Inputs[k]
		want := cat.itemCount(in)

		var pick func(start, got int) bool

		pick = func(start, got int) bool {
			if got == want {
				return solve(k + 1)
			}

			for i := start; i < len(items); i++ {
				if used[i] || !cat.Matches(in, &items[i]) || !cat.quantityOK(in, &items[i]) {
					continue
				}

				used[i] = true
				asg[k] = append(asg[k], i)

				if pick(i+1, got+1) {
					return true
				}

				asg[k] = asg[k][:len(asg[k])-1]
				used[i] = false
			}

			return false
		}

		return pick(0, 0)
	}

	if !solve(0) {
		return nil
	}

	return &Match{Recipe: r, Assigned: asg}
}

// itemCount is how many cube items an input takes: qty items, or a single
// stack when the item is stackable and a quantity is asked for.
func (cat *Catalog) itemCount(in *Input) int {
	if in.Qty > 1 && cat.stackableInput(in) {
		return 1
	}

	return in.Qty
}

func (cat *Catalog) stackableInput(in *Input) bool {
	if in.Code == "" {
		return false
	}

	b := cat.bases[in.Code]

	return b != nil && b.Stackable
}

func (cat *Catalog) quantityOK(in *Input, it *Item) bool {
	if in.Qty > 1 && cat.stackableInput(in) {
		return it.Quantity >= in.Qty
	}

	return true
}
