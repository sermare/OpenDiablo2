// Package d2cube matches the contents of the Horadric Cube against the recipes
// that quests and the Pandemonium event rely on. It is pure: the engine hands
// over the item codes found in the cube and gets back what to do with them.
//
// Evidence: the input lists are the rows of CubeMain.txt (Khalim Flail + Heart
// + Eye + Brain, Staff of Kings + Viper amulet, Pandemonium key pk1+pk2+pk3,
// Pandemonium Finale key dhn+bey+mbr; the "op 28" column of these rows is not
// decoded, so any gating it does is UNVERIFIED and not modelled). The results
// of the two Pandemonium rows ("Pandemonium Portal", "Pandemonium Finale
// Portal") are handled by the executable, not by the table. The output of the
// Khalim row says "Super Khalim Flail": the quest item qf2 (Khalim's Will) is
// the only such item the quest system knows, so it is used (UNVERIFIED).
package d2cube

import "sort"

// ResultKind says what a recipe produces.
type ResultKind int

// The result kinds.
const (
	// ResultItem: the inputs are consumed and the item Code appears.
	ResultItem ResultKind = iota
	// ResultPortal: the inputs are consumed and a portal opens (Portal says which).
	ResultPortal
)

// Portal names the portals the cube can open.
type Portal int

// The portals.
const (
	// PortalNone is the zero value.
	PortalNone Portal = iota
	// PortalPandemonium leads to one of the three uber areas (keys).
	PortalPandemonium
	// PortalFinale leads to the uber Tristram (organs).
	PortalFinale
)

// Recipe is one cube recipe.
type Recipe struct {
	Name   string
	Inputs []string // item codes, a code twice means two items
	Kind   ResultKind
	Code   string // ResultItem
	Portal Portal // ResultPortal
	// Expansion marks recipes of Lord of Destruction (version 100 rows).
	Expansion bool
}

// Item codes.
const (
	CodeStaffShaft = "msf"
	CodeViper      = "vip"
	CodeStaff      = "hst"
	CodeFlail      = "qf1"
	CodeHeart      = "qhr"
	CodeEye        = "qey"
	CodeBrain      = "qbr"
	CodeWill       = "qf2"
	CodeKeyTerror  = "pk1"
	CodeKeyHate    = "pk2"
	CodeKeyDestr   = "pk3"
	CodeHorn       = "dhn"
	CodeBaalEye    = "bey"
	CodeMephBrain  = "mbr"
)

// Recipes lists the recipes in CubeMain.txt order.
//
//nolint:gochecknoglobals // static lookup data
var Recipes = []Recipe{
	{Name: "Staff of Kings + Viper amulet -> Horadric Staff", Inputs: []string{CodeStaffShaft, CodeViper}, Kind: ResultItem, Code: CodeStaff},
	{Name: "Khalim Flail + Heart + Eye + Brain -> Khalim's Will", Inputs: []string{CodeFlail, CodeHeart, CodeEye, CodeBrain}, Kind: ResultItem, Code: CodeWill},
	{Name: "Pandemonium key", Inputs: []string{CodeKeyTerror, CodeKeyHate, CodeKeyDestr}, Kind: ResultPortal, Portal: PortalPandemonium, Expansion: true},
	{Name: "Pandemonium Finale key", Inputs: []string{CodeHorn, CodeBaalEye, CodeMephBrain}, Kind: ResultPortal, Portal: PortalFinale, Expansion: true},
}

func key(codes []string) string {
	c := append([]string{}, codes...)
	sort.Strings(c)

	s := ""
	for _, x := range c {
		s += x + ","
	}

	return s
}

// Match finds the recipe whose inputs are exactly the codes in the cube (the
// original refuses a cube with extra items). expansion says whether the hero
// plays Lord of Destruction.
func Match(codes []string, expansion bool) (Recipe, bool) {
	k := key(codes)

	for _, r := range Recipes {
		if key(r.Inputs) == k && (expansion || !r.Expansion) {
			return r, true
		}
	}

	return Recipe{}, false
}
