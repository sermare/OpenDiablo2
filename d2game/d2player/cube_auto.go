package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2cube"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The cube autotest (OD2_AUTOCUBE, see d2gamescreen/autocube.go). A scenario
// puts items into the cube grid, presses the Transmute button, and checks what
// came out. The base items are looked up by item type in the game data, so the
// scenarios need no hard-coded codes beyond the gems and runes that the recipes
// themselves name.

// CubeScenarios are the names AutoCube accepts, in the default order.
var CubeScenarios = []string{"gems", "runes", "reroll", "socket", "craft", "keys", "invalid", "runeword"}

// emptyCube removes every item from the cube grid.
func (g *GameControls) emptyCube() {
	for _, it := range append([]InventoryItem(nil), g.cube.grid.items...) {
		g.cube.grid.Remove(it)
	}
}

func (g *GameControls) autoItem(code string, q d2drop.Quality, ilvl int) (*diablo2item.Item, error) {
	it, err := g.inventory.item.ItemFromCode(code, q, ilvl, uint32(len(code)*7919+ilvl))
	if err != nil {
		return nil, err
	}

	return it.Identify(), nil
}

func (g *GameControls) fillCube(items ...*diablo2item.Item) error {
	for _, it := range items {
		if !g.cube.grid.AutoPlace(it, true) {
			return fmt.Errorf("no room for %s in the cube", it.CommonCode)
		}
	}

	return nil
}

// cubeCodes lists the item codes now in the cube.
func (g *GameControls) cubeCodes() []string {
	var out []string

	for _, it := range g.cube.grid.items {
		out = append(out, it.GetItemCode())
	}

	return out
}

// AutoCube runs one cube scenario and returns whether it passed and a one line
// description for the log.
func (g *GameControls) AutoCube(name string) (ok bool, detail string) {
	data, err := g.cubeTables()
	if err != nil {
		return false, err.Error()
	}

	g.OpenCube()
	g.emptyCube()
	g.cubeLast = nil

	defer g.emptyCube()

	switch name {
	case "gems":
		return g.autoRecipe(nil, []string{"gcv", "gcv", "gcv"}, "gfv")
	case "runes":
		return g.autoRecipe(nil, []string{"r01", "r01", "r01"}, "r02")
	case "reroll":
		return g.autoReroll(data)
	case "socket":
		return g.autoSocketMagic(data)
	case "craft":
		return g.autoCraft(data)
	case "keys":
		return g.autoKeys()
	case "invalid":
		return g.autoInvalid()
	case "runeword":
		return g.autoRuneword(data)
	}

	return false, "unknown scenario " + name
}

// autoRecipe puts plain items (and any extra items) in the cube, presses
// Transmute and expects exactly one item with the code want.
func (g *GameControls) autoRecipe(extra []*diablo2item.Item, codes []string, want string) (bool, string) {
	for _, c := range codes {
		it, err := g.autoItem(c, d2drop.QualityNormal, 1)
		if err != nil {
			return false, err.Error()
		}

		extra = append(extra, it)
	}

	if err := g.fillCube(extra...); err != nil {
		return false, err.Error()
	}

	g.cube.PressTransmute()

	got := g.cubeCodes()
	if g.cubeLast == nil || len(got) != 1 || got[0] != want {
		return false, fmt.Sprintf("%v -> %v, want %s", codes, got, want)
	}

	return true, fmt.Sprintf("%v -> %s (%s)", codes, want, g.cubeLast.Recipe)
}

// baseOfType finds a spawnable base item of a type that is not a quest item
// and fits in the cube.
func (g *GameControls) baseOfType(data *cubeData, types ...string) string {
	for _, c := range []string{"lsd", "hax", "ssd", "kit", "cap", "rin", "amu"} {
		b, ok := data.cat.Base(c)
		if !ok {
			continue
		}

		for _, t := range types {
			if b.HasType(t) {
				return c
			}
		}
	}

	return ""
}

func (g *GameControls) autoReroll(data *cubeData) (bool, string) {
	base := g.baseOfType(data, "weap")
	if base == "" {
		return false, "no weapon base"
	}

	magic, err := g.autoItem(base, d2drop.QualityMagic, 40)
	if err != nil {
		return false, err.Error()
	}

	before := strings.Join(magic.Spec().Prefixes, ",") + "|" + strings.Join(magic.Spec().Suffixes, ",")
	items := []*diablo2item.Item{magic}

	for i := 0; i < 3; i++ {
		gem, gerr := g.autoItem("gpv", d2drop.QualityNormal, 1)
		if gerr != nil {
			return false, gerr.Error()
		}

		items = append(items, gem)
	}

	if err = g.fillCube(items...); err != nil {
		return false, err.Error()
	}

	g.cube.PressTransmute()

	out, ok := g.singleProduct()
	if !ok || out.CommonCode != base {
		return false, fmt.Sprintf("reroll gave %v", g.cubeCodes())
	}

	spec := out.Spec()
	if spec.Quality != int(d2drop.QualityMagic) || len(spec.Prefixes)+len(spec.Suffixes) == 0 {
		return false, fmt.Sprintf("reroll result is not a magic item: %+v", spec)
	}

	after := strings.Join(spec.Prefixes, ",") + "|" + strings.Join(spec.Suffixes, ",")

	return true, fmt.Sprintf("magic %s [%s] -> magic %s [%s]", base, before, base, after)
}

func (g *GameControls) singleProduct() (*diablo2item.Item, bool) {
	if g.cubeLast == nil || len(g.cube.grid.items) != 1 {
		return nil, false
	}

	it, ok := g.cube.grid.items[0].(*diablo2item.Item)

	return it, ok
}

func (g *GameControls) autoSocketMagic(data *cubeData) (bool, string) {
	base := g.baseOfType(data, "weap")
	if base == "" {
		return false, "no weapon base"
	}

	weapon, err := g.autoItem(base, d2drop.QualityMagic, 30)
	if err != nil {
		return false, err.Error()
	}

	items := []*diablo2item.Item{weapon}

	for i := 0; i < 3; i++ {
		gem, gerr := g.autoItem("gcv", d2drop.QualityNormal, 1)
		if gerr != nil {
			return false, gerr.Error()
		}

		items = append(items, gem)
	}

	if err = g.fillCube(items...); err != nil {
		return false, err.Error()
	}

	g.cube.PressTransmute()

	out, ok := g.singleProduct()
	if !ok {
		return false, fmt.Sprintf("socket gave %v", g.cubeCodes())
	}

	spec := out.Spec()
	if spec.Sockets < 1 || spec.Quality != int(d2drop.QualityMagic) {
		return false, fmt.Sprintf("no sockets on the result: %+v", spec)
	}

	return true, fmt.Sprintf("3 chipped gems + magic %s -> magic %s with %d sockets", base, base, spec.Sockets)
}

func (g *GameControls) autoCraft(data *cubeData) (bool, string) {
	ring, err := g.autoItem("rin", d2drop.QualityMagic, 40)
	if err != nil {
		return false, err.Error()
	}

	items := []*diablo2item.Item{ring}

	for _, c := range []string{"jew", "r11", "gpb"} {
		it, ierr := g.autoItem(c, d2drop.QualityNormal, 1)
		if ierr != nil {
			return false, ierr.Error()
		}

		items = append(items, it)
	}

	if err = g.fillCube(items...); err != nil {
		return false, err.Error()
	}

	g.cube.PressTransmute()

	out, ok := g.singleProduct()
	if !ok {
		return false, fmt.Sprintf("craft gave %v", g.cubeCodes())
	}

	spec := out.Spec()
	if spec.Quality != int(d2drop.QualityCrafted) || !spec.Crafted || len(spec.CubeMods) == 0 {
		return false, fmt.Sprintf("not a crafted item: %+v", spec)
	}

	return true, fmt.Sprintf("crafted ring ilvl=%d mods=%d affixes=%d", spec.ILvl, len(spec.CubeMods), len(spec.Prefixes)+len(spec.Suffixes))
}

func (g *GameControls) autoKeys() (bool, string) {
	var portal string

	prev := g.cubePortal
	g.cubePortal = func(kind string) error { portal = kind; return nil }

	defer func() { g.cubePortal = prev }()

	items := make([]*diablo2item.Item, 0, 3)

	for _, c := range []string{"pk1", "pk2", "pk3"} {
		it, err := g.autoItem(c, d2drop.QualityNormal, 1)
		if err != nil {
			return false, err.Error()
		}

		items = append(items, it)
	}

	if err := g.fillCube(items...); err != nil {
		return false, err.Error()
	}

	g.cube.PressTransmute()

	if portal != "Pandemonium Portal" || len(g.cube.grid.items) != 0 {
		return false, fmt.Sprintf("keys gave portal=%q cube=%v", portal, g.cubeCodes())
	}

	return true, "3 keys -> Pandemonium Portal, keys used up"
}

// autoInvalid checks the input validation: items that are no recipe stay in the
// cube and the button says so.
func (g *GameControls) autoInvalid() (bool, string) {
	a, err := g.autoItem("gcv", d2drop.QualityNormal, 1)
	if err != nil {
		return false, err.Error()
	}

	b, err := g.autoItem("gcr", d2drop.QualityNormal, 1)
	if err != nil {
		return false, err.Error()
	}

	if err = g.fillCube(a, b); err != nil {
		return false, err.Error()
	}

	g.cube.SetStatus("")
	g.cube.PressTransmute()

	if g.cubeLast != nil || len(g.cube.grid.items) != 2 || g.cube.Status() == "" {
		return false, fmt.Sprintf("invalid contents changed: %v status=%q", g.cubeCodes(), g.cube.Status())
	}

	return true, fmt.Sprintf("2 mismatched gems kept, status %q", g.cube.Status())
}

// autoRuneword sockets the runes of a complete, non-ladder runeword into an
// item with that many sockets, one by one like dropping them on it.
func (g *GameControls) autoRuneword(data *cubeData) (bool, string) {
	for _, w := range data.rw.List {
		if !w.Complete || w.Ladder || len(w.Runes) < 2 || len(w.Runes) > 4 {
			continue
		}

		base := ""

		for _, c := range []string{"lsd", "hax", "ssd", "kit", "cap", "plt", "crn", "xhn", "ltp", "mac", "clb", "bst", "wnd", "tkf"} {
			probe := d2cube.Item{Code: c}
			if b, ok := data.cat.Base(c); ok && b.GemSockets >= len(w.Runes) && w.Fits(data.cat, &probe) {
				base = c
				break
			}
		}

		if base == "" {
			continue
		}

		it, err := g.autoItem(base, d2drop.QualityNormal, 30)
		if err != nil {
			return false, err.Error()
		}

		spec := it.Spec()
		spec.Sockets = len(w.Runes)

		if it, err = g.inventory.item.ItemFromSpec(spec); err != nil {
			return false, err.Error()
		}

		for i, r := range w.Runes {
			rune, rerr := g.autoItem(r, d2drop.QualityNormal, 1)
			if rerr != nil {
				return false, rerr.Error()
			}

			next, msg, serr := g.SocketItem(it, rune)
			if serr != nil {
				return false, fmt.Sprintf("socket %s: %v", r, serr)
			}

			g.Infof("AUTOCUBE socket step %d: %s", i+1, msg)
			it = next
		}

		if got := runewordOf(it); got == "" || (it.Rolled() == nil && got != w.Name) {
			return false, fmt.Sprintf("runes of %q in %s made %q", w.Name, base, got)
		}

		// an item with the runes in the wrong order must not form it
		wrong := spec
		wrong.SocketCodes = nil

		bad, berr := g.inventory.item.ItemFromSpec(wrong)
		if berr != nil {
			return false, berr.Error()
		}

		for i := len(w.Runes) - 1; i >= 0; i-- {
			rune, _ := g.autoItem(w.Runes[i], d2drop.QualityNormal, 1)
			if next, _, serr := g.SocketItem(bad, rune); serr == nil {
				bad = next
			}
		}

		sameOrder := true

		for i := range w.Runes {
			if w.Runes[i] != w.Runes[len(w.Runes)-1-i] {
				sameOrder = false
			}
		}

		if !sameOrder && runewordOf(bad) != "" && (bad.Rolled() != nil || bad.Runeword == w.Name) {
			return false, "the runes in reverse order also formed " + w.Name
		}

		return true, fmt.Sprintf("runeword %q formed in %s from %v (reverse order formed %q)", w.Name, base, w.Runes, runewordOf(bad))
	}

	return false, "no usable runeword in the table"
}

// runewordOf is the runeword an item became: the cube model's display name, or
// the Runes.txt key of an item made by the item creator.
func runewordOf(it *diablo2item.Item) string {
	if it.Rolled() != nil {
		return it.RolledRuneword()
	}

	return it.Runeword
}
