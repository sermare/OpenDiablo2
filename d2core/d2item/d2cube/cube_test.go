package d2cube

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Real-data tests need D2_TABLES (the extracted 1.14b tables: patch_d2/ItemTypes,
// armor, weapons, misc and itemgen/patch_d2/CubeMain, Runes, MagicPrefix,
// MagicSuffix). They skip without it. The synthetic tests below always run.

type real struct {
	cat *Catalog
	tab *Table
	rw  *Runewords
}

func loadReal(t *testing.T) *real {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(p ...string) []byte {
		b, err := os.ReadFile(filepath.Join(append([]string{dir}, p...)...))
		if err != nil {
			t.Skipf("missing table: %v", err)
		}

		return b
	}

	cat := NewCatalog(read("patch_d2", "ItemTypes.txt"), read("patch_d2", "weapons.txt"),
		read("patch_d2", "armor.txt"), read("patch_d2", "misc.txt"))
	cat.LoadAffixes(read("itemgen", "patch_d2", "MagicPrefix.txt"), read("itemgen", "patch_d2", "MagicSuffix.txt"))

	return &real{cat: cat, tab: ParseTable(read("itemgen", "patch_d2", "CubeMain.txt"), cat),
		rw: ParseRunewords(read("itemgen", "patch_d2", "Runes.txt"))}
}

func (r *real) run(t *testing.T, ctx *Context, items ...Item) *Result {
	t.Helper()

	m := r.cat.Find(r.tab, ctx, items)
	if m == nil {
		t.Fatalf("no recipe for %v", items)
	}

	res, err := r.cat.Execute(m, ctx, items, d2rand.New(7))
	if err != nil {
		t.Fatal(err)
	}

	return res
}

var lodCtx = &Context{PlayerLevel: 80, Difficulty: 2, Ladder: true, Expansion: true, Class: "sor"}

func many(code string, n int, q d2drop.Quality) []Item {
	out := make([]Item, n)
	for i := range out {
		out[i] = Item{Code: code, Quality: q, ILvl: 50}
	}

	return out
}

func TestRealTableParsesCompletely(t *testing.T) {
	r := loadReal(t)

	if len(r.tab.Skipped) != 0 {
		t.Fatalf("rows not understood: %v", r.tab.Skipped)
	}

	if len(r.tab.Recipes) < 140 {
		t.Fatalf("only %d recipes", len(r.tab.Recipes))
	}
}

func TestRealGemUpgrades(t *testing.T) {
	r := loadReal(t)
	for _, c := range []string{"v", "r", "b", "y", "g", "w"} {
		// amethyst's flawless gem is "gzv", every other colour's is "gl?"
		flawless := "l"
		if c == "v" {
			flawless = "z"
		}

		steps := []string{"gc" + c, "gf" + c, "gs" + c, "g" + flawless + c, "gp" + c}

		for i := 0; i < 4; i++ {
			res := r.run(t, lodCtx, many(steps[i], 3, 0)...)

			if len(res.Products) != 1 || res.Products[0].Item.Code != steps[i+1] || len(res.Consumed) != 3 {
				t.Errorf("%s x3 -> %+v, want %s", steps[i], res.Products, steps[i+1])
			}
		}
	}

	for i, c := range []string{"skc", "skf", "sku", "skl"} {
		res := r.run(t, lodCtx, many(c, 3, 0)...)
		if got, want := res.Products[0].Item.Code, []string{"skf", "sku", "skl", "skz"}[i]; got != want {
			t.Errorf("%s: %s want %s", c, got, want)
		}
	}
}

func TestRealRuneUpgradesAndLadder(t *testing.T) {
	r := loadReal(t)

	// r01 x3 -> r02 .. r09 x3 -> r10: classic and LoD
	for n := 1; n <= 9; n++ {
		from := runeCode(n)
		res := r.run(t, lodCtx, many(from, 3, 0)...)

		if res.Products[0].Item.Code != runeCode(n+1) {
			t.Errorf("%s -> %s", from, res.Products[0].Item.Code)
		}
	}

	// r10 x3 + chipped topaz -> r11 (non-ladder)
	items := append(many("r10", 3, 0), Item{Code: "gcy"})
	if res := r.run(t, lodCtx, items...); res.Products[0].Item.Code != "r11" {
		t.Errorf("r10 -> %s", res.Products[0].Item.Code)
	}

	// r14 x3 + chipped emerald -> r15 is ladder-only: refused in a single-player game
	items = append(many("r14", 3, 0), Item{Code: "gcg"})
	single := *lodCtx
	single.Ladder = false

	if m := r.cat.Find(r.tab, &single, items); m != nil {
		t.Errorf("ladder recipe found in a non-ladder game: %s", m.Recipe.Description)
	}

	if res := r.run(t, lodCtx, items...); res.Products[0].Item.Code != "r15" {
		t.Errorf("ladder r14 -> %s", res.Products[0].Item.Code)
	}

	// the 2-rune steps at the top
	items = append(many("r21", 2, 0), Item{Code: "gfw"})
	if res := r.run(t, lodCtx, items...); res.Products[0].Item.Code != "r22" {
		t.Errorf("r21 -> %s", res.Products[0].Item.Code)
	}

	// runes are LoD only
	classic := *lodCtx
	classic.Expansion = false

	if m := r.cat.Find(r.tab, &classic, many("r01", 3, 0)); m != nil {
		t.Error("rune recipe in a classic game")
	}
}

func runeCode(n int) string {
	s := "r"
	if n < 10 {
		s += "0"
	}

	return s + itoa(n)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}

	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestRealRerollMagicAndRare(t *testing.T) {
	r := loadReal(t)

	// 3 perfect gems + a magic item -> a new magic item of the same base
	sword := Item{Code: "lsd", Quality: d2drop.QualityMagic, ILvl: 40, Prefixes: []string{"Jagged"}}
	items := append(many("gpv", 3, 0), sword)
	res := r.run(t, lodCtx, items...)

	if len(res.Products) != 1 {
		t.Fatalf("products %d", len(res.Products))
	}

	p := res.Products[0]
	if p.Item.Code != "lsd" || p.Item.Quality != d2drop.QualityMagic || !p.Roll || len(p.Item.Prefixes) != 0 {
		t.Errorf("reroll magic: %+v", p)
	}

	if len(res.Consumed) != 4 {
		t.Errorf("consumed %v", res.Consumed)
	}

	// 6 perfect skulls + a rare -> a rare of the same base, level from player/item level
	rare := Item{Code: "lsd", Quality: d2drop.QualityRare, ILvl: 60}
	res = r.run(t, lodCtx, append(many("skz", 6, 0), rare)...)

	p = res.Products[0]
	if p.Item.Quality != d2drop.QualityRare || p.Item.Code != "lsd" || !p.Roll {
		t.Errorf("reroll rare: %+v", p)
	}

	if want := 40*80/100 + 40*60/100; p.Item.ILvl != want {
		t.Errorf("reroll rare level %d want %d", p.Item.ILvl, want)
	}

	// skull + rare + Stone of Jordan -> high level rare
	soj := Item{Code: "rin", Quality: d2drop.QualityUnique, Unique: "Stone of Jordan"}
	res = r.run(t, lodCtx, Item{Code: "skz"}, rare, soj)

	if res.Products[0].Item.Quality != d2drop.QualityRare {
		t.Errorf("rare+soj: %+v", res.Products)
	}
}

func TestRealSocketMagic(t *testing.T) {
	r := loadReal(t)

	// chipped and flawless gems take a magic weapon, standard gems a socketed one
	for _, gem := range []string{"gcv", "gsv", "gzv"} {
		weapon := Item{Code: "lsd", Quality: d2drop.QualityMagic, ILvl: 30}
		if gem == "gsv" {
			weapon.Sockets = 1
		}

		items := append(many(gem, 3, 0), weapon)

		m := r.cat.Find(r.tab, lodCtx, items)
		if m == nil {
			t.Fatalf("no recipe for 3x%s + magic weapon", gem)
		}

		res, err := r.cat.Execute(m, lodCtx, items, d2rand.New(3))
		if err != nil {
			t.Fatal(err)
		}

		p := res.Products[0]
		if p.Item.Sockets < 1 || p.Item.Sockets > 2 || p.Item.Quality != d2drop.QualityMagic {
			t.Errorf("%s: sockets=%d quality=%d", gem, p.Item.Sockets, p.Item.Quality)
		}
	}
}

func TestRealPrismaticAmulet(t *testing.T) {
	r := loadReal(t)
	items := []Item{{Code: "amu", Quality: d2drop.QualityMagic, ILvl: 20},
		{Code: "gpv"}, {Code: "gpy"}, {Code: "gpb"}, {Code: "gpg"}, {Code: "gpr"}, {Code: "gpw"}}
	res := r.run(t, lodCtx, items...)
	p := res.Products[0]

	if p.Item.Code != "amu" || len(p.Item.Prefixes) != 1 || p.Item.Prefixes[0] != "Prismatic" || !p.Roll {
		t.Errorf("prismatic amulet: %+v", p)
	}
}

func TestRealCrafting(t *testing.T) {
	r := loadReal(t)
	// magic ring + jewel + rune 11 + perfect sapphire -> crafted ring
	items := []Item{{Code: "rin", Quality: d2drop.QualityMagic, ILvl: 40}, {Code: "jew"}, {Code: "r11"}, {Code: "gpb"}}
	res := r.run(t, lodCtx, items...)
	p := res.Products[0]

	if p.Item.Quality != d2drop.QualityCrafted || !p.Item.Crafted || p.Item.Code != "rin" || !p.Roll {
		t.Fatalf("crafted: %+v", p)
	}

	if want := 50*80/100 + 50*40/100; p.Item.ILvl != want {
		t.Errorf("crafted level %d want %d", p.Item.ILvl, want)
	}

	if p.CraftAffixes < 1 || p.CraftAffixes > 4 {
		t.Errorf("craft affixes %d", p.CraftAffixes)
	}

	if len(p.Item.Mods) != 3 {
		t.Errorf("recipe mods %+v", p.Item.Mods)
	}

	for _, m := range p.Item.Mods {
		if m.Value < m.Min || (m.Max >= m.Min && m.Value > m.Max) {
			t.Errorf("mod out of range: %+v", m)
		}
	}
}

func TestRealQuestAndTokens(t *testing.T) {
	r := loadReal(t)

	res := r.run(t, lodCtx, Item{Code: "msf"}, Item{Code: "vip"})
	if res.Products[0].Item.Code != "hst" {
		t.Errorf("staff: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "leg"}, Item{Code: "tbk"})
	if res.Products[0].Portal != "Cow Portal" {
		t.Errorf("cow: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "pk1"}, Item{Code: "pk2"}, Item{Code: "pk3"})
	if res.Products[0].Portal != "Pandemonium Portal" {
		t.Errorf("keys: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "dhn"}, Item{Code: "bey"}, Item{Code: "mbr"})
	if res.Products[0].Portal != "Pandemonium Finale Portal" {
		t.Errorf("finale: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "tes"}, Item{Code: "ceh"}, Item{Code: "bet"}, Item{Code: "fed"})
	if res.Products[0].Item.Code != "toa" {
		t.Errorf("token: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "qf1"}, Item{Code: "qhr"}, Item{Code: "qey"}, Item{Code: "qbr"})
	if res.Products[0].Item.Code != "qf2" {
		t.Errorf("khalim: %+v", res.Products)
	}
}

func TestRealPotionsAndAmmo(t *testing.T) {
	r := loadReal(t)

	res := r.run(t, lodCtx, many("rvs", 3, 0)...)
	if res.Products[0].Item.Code != "rvl" {
		t.Errorf("3 rejuv: %+v", res.Products)
	}

	items := append(many("hpot", 0, 0), Item{Code: "hp1"}, Item{Code: "hp1"}, Item{Code: "hp1"},
		Item{Code: "mp1"}, Item{Code: "mp1"}, Item{Code: "mp1"}, Item{Code: "gsv"})
	if m := r.cat.Find(r.tab, lodCtx, items); m != nil {
		res, _ = r.cat.Execute(m, lodCtx, items, d2rand.New(1))
		if res.Products[0].Item.Code != "rvl" {
			t.Errorf("potions+standard gem: %+v", res.Products)
		}
	} else {
		t.Error("3 hp + 3 mp + standard gem not found")
	}

	res = r.run(t, lodCtx, Item{Code: "rin", Quality: d2drop.QualityMagic}, Item{Code: "rin", Quality: d2drop.QualityMagic},
		Item{Code: "rin", Quality: d2drop.QualityMagic})
	if res.Products[0].Item.Code != "amu" || res.Products[0].Item.Quality != d2drop.QualityMagic {
		t.Errorf("3 rings: %+v", res.Products)
	}

	res = r.run(t, lodCtx, Item{Code: "aqv", Quantity: 40})
	_ = res
}

func TestRealUpgradeRepairUnsocket(t *testing.T) {
	r := loadReal(t)

	// basic unique weapon -> exceptional, keeps its identity
	uni := Item{Code: "lsd", Quality: d2drop.QualityUnique, Unique: "Rixot's Keen", ILvl: 30, Sockets: 0}
	items := []Item{{Code: "r08"}, {Code: "r12"}, {Code: "gpg"}, uni}
	res := r.run(t, lodCtx, items...)
	p := res.Products[0]

	if p.Source != 3 || p.Item.Unique != "Rixot's Keen" || p.Item.Quality != d2drop.QualityUnique {
		t.Errorf("upgrade identity lost: %+v", p)
	}

	if b, _ := r.cat.Base(p.Item.Code); b == nil || b.Tier != TierExceptional {
		t.Errorf("upgrade did not reach the exceptional tier: %s", p.Item.Code)
	}

	// repair
	res = r.run(t, lodCtx, Item{Code: "lsd", Quality: d2drop.QualityMagic}, Item{Code: "r09"})
	if !res.Products[0].Item.Repaired || res.Products[0].Item.Quality != d2drop.QualityMagic {
		t.Errorf("repair: %+v", res.Products)
	}

	// unsocket destroys the gems but keeps the sockets
	sock := Item{Code: "lsd", Sockets: 2, Socketed: []string{"r01", "gcv"}, Runeword: "Whatever"}
	res = r.run(t, lodCtx, Item{Code: "r15"}, Item{Code: "tsc"}, sock)
	got := res.Products[0].Item

	if len(got.Socketed) != 0 || got.Sockets != 2 || got.Runeword != "" {
		t.Errorf("unsocket: %+v", got)
	}
}

func TestRealSocketNormalItem(t *testing.T) {
	r := loadReal(t)
	items := []Item{{Code: "plt", ILvl: 60}, {Code: "r07"}, {Code: "r10"}, {Code: "gpy"}}
	res := r.run(t, lodCtx, items...)
	p := res.Products[0]

	if p.Item.Code != "plt" || p.Item.Sockets < 1 || p.Item.Sockets > r.cat.MaxSockets("plt", 60) {
		t.Errorf("socketed plate: %+v (max %d)", p.Item, r.cat.MaxSockets("plt", 60))
	}
}

func TestRealWrongContentsDoNotMatch(t *testing.T) {
	r := loadReal(t)

	for name, items := range map[string][]Item{
		"two gems":              many("gcv", 2, 0),
		"extra item":            append(many("gcv", 3, 0), Item{Code: "hp1"}),
		"mixed gems":            {{Code: "gcv"}, {Code: "gcr"}, {Code: "gcb"}},
		"empty":                 nil,
		"magic reroll no magic": append(many("gpv", 3, 0), Item{Code: "lsd", Quality: d2drop.QualityNormal}),
	} {
		if m := r.cat.Find(r.tab, lodCtx, items); m != nil {
			t.Errorf("%s matched %q", name, m.Recipe.Description)
		}
	}

	// disabled (tempered) recipes never match
	tmp := []Item{{Code: "r08"}, {Code: "jew"}, {Code: "rin", Quality: d2drop.QualityMagic}}
	if m := r.cat.Find(r.tab, lodCtx, tmp); m != nil {
		t.Errorf("disabled recipe matched %q", m.Recipe.Description)
	}
}

func TestRealRunewords(t *testing.T) {
	r := loadReal(t)

	var rw *Runeword

	for _, w := range r.rw.List {
		if w.Name == "Ancient's Pledge" {
			rw = w
		}
	}

	if rw == nil {
		t.Skip("Ancient's Pledge not in Runes.txt")
	}

	shield := Item{Code: "kit", Sockets: 3}
	// wrong order first: no runeword, but the sockets still fill
	for _, c := range []string{"r09", "r08", "r07"} {
		if got, err := r.cat.Socket(&shield, c, r.rw, true); err != nil || got != nil {
			t.Fatalf("socket %s: %v %v", c, got, err)
		}
	}

	if shield.Runeword != "" {
		t.Fatal("runeword formed in the wrong order")
	}

	shield = Item{Code: "kit", Sockets: 3}

	var formed *Runeword

	for _, c := range rw.Runes {
		formed, _ = r.cat.Socket(&shield, c, r.rw, true)
	}

	if formed == nil || formed.Name != "Ancient's Pledge" || shield.Runeword != formed.Name {
		t.Fatalf("runeword not formed: %+v", shield)
	}

	// wrong item type, magic item, and a full item are refused
	axe := Item{Code: "hax", Sockets: 3}
	for _, c := range rw.Runes {
		_, _ = r.cat.Socket(&axe, c, r.rw, true)
	}

	if axe.Runeword != "" {
		t.Error("shield runeword formed in an axe")
	}

	if _, err := r.cat.Socket(&shield, "r01", r.rw, true); err != ErrNoFreeSocket {
		t.Errorf("full item: %v", err)
	}

	if _, err := r.cat.Socket(&Item{Code: "kit"}, "r01", r.rw, true); err != ErrNoSockets {
		t.Errorf("no sockets: %v", err)
	}

	if _, err := r.cat.Socket(&Item{Code: "kit", Sockets: 1}, "hp1", r.rw, true); err != ErrNotSocketable {
		t.Errorf("potion: %v", err)
	}

	// ladder-only runewords need a ladder game, incomplete ones never work
	for _, w := range r.rw.List {
		if len(w.Runes) == 0 {
			continue
		}

		it := Item{Code: probeItem(r.cat, w), Sockets: len(w.Runes), Socketed: w.Runes}
		if it.Code == "" {
			continue
		}

		// a runeword with the same runes may shadow this one, so compare the runes
		for _, ladder := range []bool{false, true} {
			got := r.rw.Find(r.cat, &it, ladder)
			allowed := w.Complete && (ladder || !w.Ladder)

			if allowed && (got == nil || !sameRunes(got, w)) {
				t.Errorf("%s ladder=%v: not formed", w.Name, ladder)
			}

			if got == w && !allowed {
				t.Errorf("%s formed although complete=%v ladder-only=%v in ladder=%v", w.Name, w.Complete, w.Ladder, ladder)
			}
		}
	}
}

func sameRunes(a, b *Runeword) bool { return strings.Join(a.Runes, ",") == strings.Join(b.Runes, ",") }

// probeItem finds a base item the runeword fits.
func probeItem(cat *Catalog, w *Runeword) string {
	for _, code := range cat.order {
		it := Item{Code: code}
		if w.Fits(cat, &it) {
			return code
		}
	}

	return ""
}

func TestRealAffixIDs(t *testing.T) {
	r := loadReal(t)

	for _, c := range []struct {
		prefix bool
		id     int
		want   string
	}{{true, 331, "Prismatic"}, {true, 372, "Garnet"}, {true, 191, "Savage"}, {false, 162, "of Spikes"}, {false, 352, "of the Leech"}} {
		if got, ok := r.cat.AffixName(c.prefix, c.id); !ok || got != c.want {
			t.Errorf("affix prefix=%v id=%d = %q want %q", c.prefix, c.id, got, c.want)
		}
	}
}

func TestRealGemSocketing(t *testing.T) {
	r := loadReal(t)
	it := Item{Code: "lsd", Sockets: 2}

	for _, g := range []string{"gpr", "r01"} {
		if _, err := r.cat.Socket(&it, g, r.rw, true); err != nil {
			t.Fatalf("socket %s: %v", g, err)
		}
	}

	if len(it.Socketed) != 2 {
		t.Fatalf("socketed %v", it.Socketed)
	}
}
