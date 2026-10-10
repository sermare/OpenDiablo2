package d2player

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2cube"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The Horadric Cube: the Transmute button of the cube panel and the glue
// between the cube grid and package d2cube. The recipes are read from the
// CubeMain.txt of the game data (the compiled table of the install is the
// authority); nothing about them is coded here.

// cubeData is the lazily loaded recipe table.
type cubeData struct {
	cat *d2cube.Catalog
	tab *d2cube.Table
	rw  *d2cube.Runewords
}

var errCubeEmpty = errors.New("the cube is empty")

// errNoRecipe is the answer of a transmute that matches no recipe: the game
// does nothing and keeps the items.
var errNoRecipe = errors.New("those items do not make anything")

func (g *GameControls) cubeTables() (*cubeData, error) {
	if g.cubeData != nil {
		return g.cubeData, nil
	}

	load := func(path string) ([]byte, error) {
		b, err := g.asset.LoadFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}

		return b, nil
	}

	var raw [8][]byte

	for i, p := range []string{
		d2resource.ItemTypes, d2resource.Weapons, d2resource.Armor, d2resource.Misc,
		d2resource.MagicPrefix, d2resource.MagicSuffix, d2resource.CubeRecipes, d2resource.Runes,
	} {
		b, err := load(p)
		if err != nil {
			return nil, err
		}

		raw[i] = b
	}

	cat := d2cube.NewCatalog(raw[0], raw[1], raw[2], raw[3])
	cat.LoadAffixes(raw[4], raw[5])
	tab := d2cube.ParseTable(raw[6], cat)

	g.cubeData = &cubeData{cat: cat, tab: tab, rw: d2cube.ParseRunewords(raw[7])}
	g.Infof("CUBE recipes loaded: %d usable, %d skipped, %d runewords", len(tab.Recipes), len(tab.Skipped), len(g.cubeData.rw.List))

	for _, s := range tab.Skipped {
		g.Warningf("CUBE recipe skipped: %s", s)
	}

	return g.cubeData, nil
}

// heroClassCode is the CubeMain class code of the hero.
func (g *GameControls) heroClassCode() string {
	switch g.hero.Class {
	case d2enum.HeroAmazon:
		return "ama"
	case d2enum.HeroSorceress:
		return "sor"
	case d2enum.HeroNecromancer:
		return "nec"
	case d2enum.HeroPaladin:
		return "pal"
	case d2enum.HeroBarbarian:
		return "bar"
	case d2enum.HeroDruid:
		return "dru"
	case d2enum.HeroAssassin:
		return "ass"
	}

	return ""
}

func (g *GameControls) cubeContext() *d2cube.Context {
	return &d2cube.Context{
		PlayerLevel: g.hero.Stats.Level,
		Difficulty:  g.hero.QuestDifficulty,
		Ladder:      g.cubeLadder || os.Getenv("OD2_AUTOCUBE_LADDER") == "1",
		Expansion:   !g.cubeClassic,
		Class:       g.heroClassCode(),
	}
}

// SetCubeRules tells the cube whether the game is a Lord of Destruction game
// (version 100 recipes) and a ladder game (ladder-only recipes). The default is
// an expansion game that is not a ladder game.
func (g *GameControls) SetCubeRules(expansion, ladder bool) {
	g.cubeClassic, g.cubeLadder = !expansion, ladder
}

// inferQuality is the quality of an item that was not made by the item
// generator with a quality (items from NewItem).
func inferQuality(it *diablo2item.Item, spec diablo2item.Spec) d2drop.Quality {
	switch {
	case spec.Quality != 0:
		return d2drop.Quality(spec.Quality)
	case spec.Unique != "":
		return d2drop.QualityUnique
	case spec.SetItem != "":
		return d2drop.QualitySet
	}

	switch n := len(spec.Prefixes) + len(spec.Suffixes); {
	case n > 2:
		return d2drop.QualityRare
	case n > 0:
		return d2drop.QualityMagic
	}

	return d2drop.QualityNormal
}

// toCubeItem describes a cube grid item for the recipe matcher.
func toCubeItem(it *diablo2item.Item) d2cube.Item {
	spec := it.Spec()

	// an item made by the item creator keeps its socketed items as items; the
	// cube sees their codes
	if len(spec.SocketCodes) == 0 {
		for _, c := range spec.Socketed {
			spec.SocketCodes = append(spec.SocketCodes, c.Code)
		}
	}

	ci := d2cube.Item{
		Code: spec.Code, Quality: inferQuality(it, spec), ILvl: it.ItemLevel(), Seed: spec.Seed,
		Unique: spec.Unique, SetItem: spec.SetItem, Set: spec.Set,
		Prefixes: spec.Prefixes, Suffixes: spec.Suffixes,
		Ethereal: spec.Ethereal, Quantity: spec.Quantity,
		Sockets: spec.Sockets, Socketed: spec.SocketCodes, Runeword: spec.Runeword,
		Identified: spec.Identified, Crafted: spec.Crafted,
	}

	for _, m := range spec.CubeMods {
		ci.Mods = append(ci.Mods, d2cube.RolledMod{Code: m.Code, Param: m.Param, Min: m.Min, Max: m.Max, Value: m.Value})
	}

	return ci
}

// cubeItems lists the diablo2item items in the cube grid (in grid order).
func (g *GameControls) cubeItems() (items []*diablo2item.Item, cube []d2cube.Item) {
	for _, it := range g.cube.grid.items {
		if item, ok := it.(*diablo2item.Item); ok {
			items = append(items, item)
			cube = append(cube, toCubeItem(item))
		}
	}

	return items, cube
}

// CubeCanTransmute reports whether the cube holds something that a recipe
// accepts (the Transmute button state); it does not change anything.
func (g *GameControls) CubeCanTransmute() bool {
	data, err := g.cubeTables()
	if err != nil {
		return false
	}

	_, items := g.cubeItems()

	return data.cat.Find(data.tab, g.cubeContext(), items) != nil
}

// TransmuteResult says what a transmute did, for the log and the autotests.
type TransmuteResult struct {
	Recipe   string
	Row      int
	Consumed []string
	Products []string
	// ProductCodes are the base codes of the items made, in the order of Products.
	ProductCodes []string
	Portals      []string
}

// SetCubePortalHandler sets the function that opens the portal of a portal
// recipe ("Cow Portal", "Pandemonium Portal", "Pandemonium Finale Portal").
func (g *GameControls) SetCubePortalHandler(f func(kind string) error) { g.cubePortal = f }

// SetCubeSeed makes the transmute rolls reproducible (autotests).
func (g *GameControls) SetCubeSeed(seed uint32) { g.cubeRNG = d2rand.New(seed) }

func (g *GameControls) cubeRand() *d2rand.Seed {
	if g.cubeRNG == nil {
		seed := uint32(time.Now().UnixNano())

		if s, err := strconv.ParseUint(os.Getenv("OD2_AUTOCUBE_SEED"), 10, 32); err == nil {
			seed = uint32(s)
		}

		g.cubeRNG = d2rand.New(seed)
	}

	return g.cubeRNG
}

// Transmute is the Transmute button: it finds the recipe for the items in the
// cube, replaces them with the result and saves the hero. With nothing to make
// it changes nothing and returns an error.
func (g *GameControls) Transmute() (*TransmuteResult, error) {
	data, err := g.cubeTables()
	if err != nil {
		return nil, err
	}

	items, cubeItems := g.cubeItems()
	if len(items) == 0 {
		return nil, errCubeEmpty
	}

	// the cursor must be free of the cube's business, but an item held on the
	// cursor is not in the cube and is never touched
	ctx := g.cubeContext()

	m := data.cat.Find(data.tab, ctx, cubeItems)
	if m == nil {
		return nil, errNoRecipe
	}

	rng := g.cubeRand()

	res, err := data.cat.Execute(m, ctx, cubeItems, rng)
	if err != nil {
		return nil, err
	}

	// build every product before touching the grid, so a failure keeps the items
	built := make([]*diablo2item.Item, len(res.Products))

	for i := range res.Products {
		p := &res.Products[i]
		if p.Portal != "" {
			continue
		}

		var src *diablo2item.Item
		if p.Source >= 0 {
			src = items[p.Source]
		}

		if built[i], err = g.buildCubeProduct(p, src, rng); err != nil {
			return nil, fmt.Errorf("recipe %q: %w", res.Recipe.Description, err)
		}
	}

	out := &TransmuteResult{Recipe: res.Recipe.Description, Row: res.Recipe.Row}

	for _, idx := range res.Consumed {
		out.Consumed = append(out.Consumed, itemName(items[idx]))
		g.cube.grid.Remove(items[idx])
		delete(g.itemOrigin, items[idx])
	}

	for i, item := range built {
		if item == nil {
			kind := res.Products[i].Portal
			out.Portals = append(out.Portals, kind)

			if g.cubePortal == nil {
				g.Warningf("CUBE %s: no portal handler", kind)
			} else if perr := g.cubePortal(kind); perr != nil {
				g.Warningf("CUBE %s failed: %v", kind, perr)
			}

			continue
		}

		out.Products = append(out.Products, oneLine(itemName(item)))
		out.ProductCodes = append(out.ProductCodes, strings.TrimSpace(item.CommonCode))

		if !g.cube.grid.AutoPlace(item, true) && !g.inventory.grid.AutoPlace(item, true) {
			g.Warningf("CUBE no room for %s; it is lost", item.CommonCode)
		}
	}

	g.Infof("CUBE transmute row=%d recipe=%q consumed=%v products=%v portals=%v",
		out.Row, out.Recipe, out.Consumed, out.Products, out.Portals)
	g.cubeLast = out
	g.saveHero()

	return out, nil
}

// buildCubeProduct makes the game item a recipe result describes. src is the
// cube item a "useitem" result was made from (nil for new items).
func (g *GameControls) buildCubeProduct(p *d2cube.Product, src *diablo2item.Item, rng *d2rand.Seed) (*diablo2item.Item, error) {
	f := g.inventory.item
	ci := &p.Item

	var spec diablo2item.Spec

	switch {
	case src != nil:
		spec = src.Spec()
		spec.Code, spec.Quality = ci.Code, int(ci.Quality)
		spec.ILvl, spec.Ethereal = ci.ILvl, ci.Ethereal
	case p.Roll:
		q := ci.Quality
		if q == d2cube.QualityTempered || q == d2drop.QualityCrafted {
			q = d2drop.QualityRare // crafted and tempered items roll as rares, then keep fewer affixes
		}

		rolled, err := f.ItemFromCode(ci.Code, q, ci.ILvl, uint32(ci.Seed))
		if err != nil {
			return nil, err
		}

		spec = rolled.Spec()
		spec.Quality = int(ci.Quality)
	default:
		it, err := f.ItemFromCode(ci.Code, d2drop.QualityNormal, ci.ILvl, uint32(ci.Seed))
		if err != nil {
			return nil, err
		}

		spec = it.Spec()
		spec.Quality = int(d2drop.QualityNormal)
	}

	if len(ci.Prefixes) > 0 && p.Item.Prefixes != nil && src == nil {
		spec.Prefixes = g.knownAffixes(ci.Prefixes, true)
	}

	if len(ci.Suffixes) > 0 && src == nil {
		spec.Suffixes = g.knownAffixes(ci.Suffixes, false)
	}

	if p.CraftAffixes > 0 {
		spec.Prefixes, spec.Suffixes = trimAffixes(spec.Prefixes, spec.Suffixes, p.CraftAffixes)
	}

	if src == nil {
		spec.Ethereal, spec.Quantity = ci.Ethereal, ci.Quantity
	}

	spec.Sockets, spec.SocketCodes, spec.Runeword, spec.Crafted = ci.Sockets, ci.Socketed, ci.Runeword, ci.Crafted
	// a cube product is rebuilt from the cube's fields (the legacy item model),
	// not from the creator's roll, which no longer describes it
	spec.Rolled, spec.Socketed = nil, nil
	spec.Identified = true // UNVERIFIED: cube results are shown identified
	spec.CubeMods = spec.CubeMods[:0:0]

	for _, m := range ci.Mods {
		spec.CubeMods = append(spec.CubeMods, diablo2item.ExtraMod{Code: m.Code, Param: m.Param, Min: m.Min, Max: m.Max, Value: m.Value})
	}

	if ci.Repaired || ci.Recharged {
		spec.Durability = -1 // back to the base item's full durability
	}

	if src == nil {
		spec.Seed = int64(rng.Step())
		if spec.Seed == 0 {
			spec.Seed = 1
		}
	}

	return f.ItemFromSpec(spec)
}

// knownAffixes keeps the affix names that exist in the game's records (a forced
// pre=/suf= may name an affix OpenDiablo2 did not load).
func (g *GameControls) knownAffixes(names []string, prefix bool) []string {
	var out []string

	for _, n := range names {
		var ok bool

		if prefix {
			ok = g.asset.Records.Item.Magic.Prefix[n] != nil
		} else {
			ok = g.asset.Records.Item.Magic.Suffix[n] != nil
		}

		if ok {
			out = append(out, n)
		}
	}

	return out
}

// trimAffixes keeps n affixes of a rolled item, taking prefixes and suffixes in
// turn (crafted items carry few random affixes besides their recipe mods).
func trimAffixes(pre, suf []string, n int) (outPre, outSuf []string) {
	for len(outPre)+len(outSuf) < n && (len(outPre) < len(pre) || len(outSuf) < len(suf)) {
		if len(outPre) < len(pre) && (len(outPre) <= len(outSuf) || len(outSuf) >= len(suf)) {
			outPre = append(outPre, pre[len(outPre)])
		} else if len(outSuf) < len(suf) {
			outSuf = append(outSuf, suf[len(outSuf)])
		}
	}

	return outPre, outSuf
}

// SocketItem puts a gem, rune or jewel (the held item) into a socketed item
// (dropping a gem on an item). A runeword forms when the last socket is filled
// with runes in the order of a Runes.txt row. It returns the log text and the
// new item, which replaces the target; the filler is used up.
func (g *GameControls) SocketItem(target, filler *diablo2item.Item) (*diablo2item.Item, string, error) {
	data, err := g.cubeTables()
	if err != nil {
		return nil, "", err
	}

	if target.Rolled() != nil {
		// an item made by the item creator takes the socket through the
		// creator's runeword walk (diablo2item.Item.Socket)
		if err := target.Socket(filler); err != nil {
			return nil, "", err
		}

		msg := fmt.Sprintf("socketed %s into %s (%d/%d)", filler.CommonCode, target.CommonCode, len(target.Socketed()), target.NumSockets())
		if rw := target.RolledRuneword(); rw != "" {
			msg += " runeword=" + strings.ReplaceAll(rw, " ", "_")
		}

		return target, msg, nil
	}

	ci := toCubeItem(target)

	rw, err := data.cat.Socket(&ci, filler.CommonCode, data.rw, g.cubeContext().Ladder)
	if err != nil {
		return nil, "", err
	}

	spec := target.Spec()
	spec.SocketCodes, spec.Runeword = ci.Socketed, ci.Runeword

	item, err := g.inventory.item.ItemFromSpec(spec)
	if err != nil {
		return nil, "", err
	}

	msg := fmt.Sprintf("socketed %s into %s (%d/%d)", filler.CommonCode, target.CommonCode, len(ci.Socketed), ci.Sockets)
	if rw != nil {
		msg += " runeword=" + strings.ReplaceAll(rw.Name, " ", "_")
	}

	return item, msg, nil
}

// onTransmuteButton is the Transmute button: it runs the transmute and shows
// the outcome (or why nothing happened) under the button. Nothing is changed
// when the cube's content is not a recipe.
func (g *GameControls) onTransmuteButton() {
	res, err := g.Transmute()
	if err != nil {
		g.cube.SetStatus(err.Error())
		g.Infof("CUBE transmute refused: %v", err)

		return
	}

	switch {
	case len(res.Products) > 0:
		g.cube.SetStatus("Made " + res.Products[0])
	case len(res.Portals) > 0:
		g.cube.SetStatus("A portal opens")
	default:
		g.cube.SetStatus("Done")
	}
}
