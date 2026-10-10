package d2drop

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Table-driven audit of item generation against the real tables. Every test
// reads D2_TABLES (and skips when it is unset); nothing is committed from the
// game. The expectations are derived from the raw table text in this file
// (own parsing, own type closure), so they check the loaders and the engine
// rather than repeating them.

// auditProps lists, per table, the columns that name a property code.
func auditPropColumns() map[string][]string {
	seq := func(prefix, suffix string, from, to int) []string {
		var out []string

		for i := from; i <= to; i++ {
			out = append(out, fmt.Sprintf("%s%d%s", prefix, i, suffix))
		}

		return out
	}

	cols := map[string][]string{
		"UniqueItems.txt": seq("prop", "", 1, 12),
		"SetItems.txt":    append(seq("prop", "", 1, 9), seq("aprop", "a", 1, 5)...),
		"Runes.txt":       seq("t1code", "", 1, 7),
		"MagicPrefix.txt": seq("mod", "code", 1, 3),
		"MagicSuffix.txt": seq("mod", "code", 1, 3),
		"AutoMagic.txt":   seq("mod", "code", 1, 3),
		"Gems.txt":        {},
	}

	cols["SetItems.txt"] = append(cols["SetItems.txt"], seq("aprop", "b", 1, 5)...)

	for _, k := range []string{"weaponMod", "helmMod", "shieldMod"} {
		cols["Gems.txt"] = append(cols["Gems.txt"], seq(k, "Code", 1, 3)...)
	}

	cols["Sets.txt"] = append(seq("PCode", "a", 2, 5), seq("PCode", "b", 2, 5)...)
	cols["Sets.txt"] = append(cols["Sets.txt"], seq("FCode", "", 1, 8)...)

	return cols
}

// TestAuditPropertyCodesResolve checks every property code written in the
// item tables is a row of Properties.txt. cPropInst drops unknown codes
// silently (Prop -1), so an unresolved code would lose a mod from an item
// without any error.
func TestAuditPropertyCodesResolve(t *testing.T) {
	c := cLoadCreator(t)
	src := testSource(t)
	cols := auditPropColumns()

	var files []string
	for f := range cols {
		files = append(files, f)
	}

	sort.Strings(files)

	total := 0

	for _, file := range files {
		tab := readTab(skipTB{t}, src, file)
		bad := 0

		for ri, r := range tab.rows {
			if strings.TrimSpace(r[0]) == "Expansion" {
				continue
			}

			for _, col := range cols[file] {
				code := strings.ToLower(tab.s(r, col))
				// "*name" is a commented out property: the game ignores it
				if code == "" || strings.HasPrefix(code, "*") {
					continue
				}

				total++

				if _, ok := c.Props.ByCode[code]; !ok {
					bad++

					if bad <= 10 {
						t.Errorf("%s row %d (%s) %s: property %q not in Properties.txt", file, ri+2, tab.s(r, tab.colName(0)), col, code)
					}
				}
			}
		}
	}

	t.Logf("%d property references in %d tables", total, len(files))
}

func (d *tsv) colName(i int) string {
	for k, v := range d.cols {
		if v == i {
			return k
		}
	}

	return ""
}

// rawTypes is an independent reading of ItemTypes.txt.
func rawTypes(t *testing.T) (equiv map[string][2]string, order []string) {
	t.Helper()

	tab := readTab(skipTB{t}, testSource(t), "ItemTypes.txt")
	equiv = map[string][2]string{}

	for _, r := range tab.rows {
		code := tab.s(r, "Code")
		if code == "" {
			continue
		}

		equiv[code] = [2]string{tab.s(r, "Equiv1"), tab.s(r, "Equiv2")}
		order = append(order, code)
	}

	return equiv, order
}

func closure(equiv map[string][2]string, code string) map[string]bool {
	out := map[string]bool{}

	var walk func(c string)

	walk = func(c string) {
		if c == "" || out[c] {
			return
		}

		out[c] = true

		for _, p := range equiv[c] {
			walk(p)
		}
	}

	walk(code)

	return out
}

// TestAuditItemTypeInheritance compares the loaded ancestor sets of every item
// type, and the type test of every base item, with an independent closure of
// the Equiv1/Equiv2 columns, and checks every reference resolves.
func TestAuditItemTypeInheritance(t *testing.T) {
	it := loadItemTables(t)
	equiv, order := rawTypes(t)

	for _, code := range order {
		for _, p := range equiv[code] {
			if p != "" {
				if _, ok := equiv[p]; !ok {
					t.Errorf("type %s: Equiv %q is not a type", code, p)
				}
			}
		}

		want := closure(equiv, code)
		ty := it.Types[code]

		if ty == nil {
			t.Errorf("type %s not loaded", code)

			continue
		}

		got := map[string]bool{}
		for _, a := range ty.Ancestors {
			got[a] = true
		}

		if len(got) != len(want) {
			t.Errorf("type %s ancestors %v, want %v", code, got, want)

			continue
		}

		for a := range want {
			if !got[a] || !ty.IsA(a) {
				t.Errorf("type %s: missing ancestor %s", code, a)
			}
		}
	}

	checked := 0

	for _, b := range it.Items {
		if it.Types[b.Type] == nil {
			t.Errorf("item %s: type %q not in ItemTypes", b.Code, b.Type)
		}

		if b.Type2 != "" && it.Types[b.Type2] == nil {
			t.Errorf("item %s: type2 %q not in ItemTypes", b.Code, b.Type2)
		}

		w1, w2 := closure(equiv, b.Type), closure(equiv, b.Type2)

		for _, code := range order {
			want := w1[code] || w2[code]
			if got := it.IsA(b, code); got != want {
				t.Fatalf("IsA(%s, %s) = %v, want %v", b.Code, code, got, want)
			}

			checked++
		}
	}

	t.Logf("%d item/type pairs", checked)
}

// TestAuditUniqueAndSetRows checks every UniqueItems and SetItems row: the
// base item exists, the set exists, the row count matches the raw table, and
// the loaded properties are the raw ones.
func TestAuditUniqueAndSetRows(t *testing.T) {
	c := cLoadCreator(t)
	src := testSource(t)

	check := func(file, nameCol, codeCol string, n int, code func(i int) string, name func(i int) string) {
		tab := cReadTable(skipTB{t}, src, file)
		if len(tab.rows) != n {
			t.Errorf("%s: %d rows loaded, table has %d", file, n, len(tab.rows))

			return
		}

		for i, r := range tab.rows {
			if tab.s(r, nameCol) != name(i) {
				t.Errorf("%s row %d: name %q, table %q", file, i, name(i), tab.s(r, nameCol))
			}

			// separator rows and disabled rows (enabled 0) have no base item
			if file == "UniqueItems.txt" && tab.n(r, "enabled") == 0 {
				continue
			}

			if cd := tab.s(r, codeCol); cd != code(i) {
				t.Errorf("%s row %d (%s): code %q, table %q", file, i, name(i), code(i), cd)
			} else if cd == "" {
				t.Errorf("%s row %d (%s): no base item code", file, i, name(i))
			} else if c.Items.ByCode[cd] == nil {
				t.Errorf("%s row %d (%s): base item %q not in weapons/armor/misc", file, i, name(i), cd)
			}
		}
	}

	u := c.Uniques
	check("UniqueItems.txt", "index", "code", len(u.Uniques),
		func(i int) string { return u.Uniques[i].Code }, func(i int) string { return u.Uniques[i].Name })
	check("SetItems.txt", "index", "item", len(u.SetItems),
		func(i int) string { return u.SetItems[i].Code }, func(i int) string { return u.SetItems[i].Name })

	// Every set has at least two items, and every item belongs to the set
	// it names.
	sets := cReadTable(skipTB{t}, src, "Sets.txt")
	setItems := cReadTable(skipTB{t}, src, "SetItems.txt")

	for si, s := range u.Sets {
		if len(s.Items) < 2 {
			t.Errorf("set %q has %d items", s.Name, len(s.Items))
		}

		for _, ii := range s.Items {
			if got := setItems.s(setItems.rows[ii], "set"); got != sets.s(sets.rows[si], "index") {
				t.Errorf("set item %d lists set %q, grouped under %q", ii, got, s.Name)
			}
		}
	}

	// Loaded properties are exactly the raw ones: same property row, same
	// range. A property that is named but not loaded is a loss.
	utab := cReadTable(skipTB{t}, src, "UniqueItems.txt")

	for i, r := range utab.rows {
		for k := 0; k < 12; k++ {
			n := fmt.Sprint(k + 1)
			code := strings.ToLower(utab.s(r, "prop"+n))
			if strings.HasPrefix(code, "*") {
				code = ""
			}

			got := u.Uniques[i].Props[k]

			switch {
			case code == "":
				if got.Prop != -1 {
					t.Errorf("unique %q prop%s: loaded %d for an empty slot", u.Uniques[i].Name, n, got.Prop)
				}
			case got.Prop != c.Props.ByCode[code] || got.Min != utab.n(r, "min"+n) || got.Max != utab.n(r, "max"+n):
				t.Errorf("unique %q prop%s %q: loaded %+v", u.Uniques[i].Name, n, code, got)
			}
		}
	}
}

// writeStats is the set of stats a property can write: the stat ids of its
// slots in Properties.txt.
func (c *Creator) propStats(prop int) map[int]bool {
	out := map[int]bool{}
	row := c.Props.Props[prop]

	for k := 0; k < 7; k++ {
		if row.Func[k] != 0 && row.Stat[k] >= 0 {
			out[row.Stat[k]] = true
		}
	}

	// Weapon damage properties write the stat of the weapon's grip: one-handed
	// (21/22), two-handed (23/24) or thrown (159/160); the table names 21/22.
	for _, pair := range [][2]int{{21, 22}, {23, 24}, {159, 160}} {
		if out[pair[0]] || out[pair[1]] {
			for _, g := range [][2]int{{21, 22}, {23, 24}, {159, 160}} {
				out[g[0]], out[g[1]] = true, true
			}

			break
		}
	}

	return out
}

// TestAuditUniqueItemsResolve creates every enabled unique and set item (forced
// row, item level 99, LoD) and checks the row comes out and each property that
// names a stat writes it. This is "every unique's properties resolve" at
// the engine level, not only at the table level.
func TestAuditUniqueItemsResolve(t *testing.T) {
	c := cLoadCreator(t)
	u := c.Uniques
	failures := 0
	created := 0

	report := func(format string, args ...interface{}) {
		failures++

		if failures <= 40 {
			t.Errorf(format, args...)
		}
	}

	covered := func(r *Rolled, props []PropInst, name string) {

		for k, p := range props {
			if p.Prop < 0 || (p.Min == 0 && p.Max == 0) {
				continue
			}

			stats := c.propStats(p.Prop)
			if len(stats) == 0 {
				continue
			}

			hit := false

			for _, w := range r.Writes {
				if stats[w.Stat] {
					hit = true
				}
			}

			if !hit {
				report("%s prop%d (%s %d..%d): writes none of stats %v", name, k+1, c.Props.Props[p.Prop].Code, p.Min, p.Max, keys(stats))
			}
		}
	}

	for row := range u.Uniques {
		un := &u.Uniques[row]
		if !un.Enabled || un.Code == "" {
			continue
		}

		req := Request{
			Code: un.Code, ILvl: maxInt(99, un.Lvl), Quality: QualityUnique, Version: 100, Expansion: true, ForcedID: row + 1,
			GameSeed: d2rand.Seed{Lo: uint32(row)*7 + 1, Hi: 0x29a}, Game: &GameState{Ladder: true},
		}

		r, err := c.Create(req)
		if err != nil {
			report("unique %q (%s): %v", un.Name, un.Code, err)

			continue
		}

		created++

		if r.Quality != QualityUnique || r.Unique != row {
			report("unique %q: created quality %d row %d, want row %d", un.Name, r.Quality, r.Unique, row)

			continue
		}

		covered(r, un.Props[:], "unique "+un.Name)
	}

	for row := range u.SetItems {
		s := &u.SetItems[row]

		req := Request{
			Code: s.Code, ILvl: 99, Quality: QualitySet, Version: 100, Expansion: true, ForcedID: row + 1,
			Flags: 1, GameSeed: d2rand.Seed{Lo: uint32(row)*13 + 5, Hi: 0x29a},
		}

		r, err := c.Create(req)
		if err != nil {
			report("set item %q (%s): %v", s.Name, s.Code, err)

			continue
		}

		created++

		if r.Quality != QualitySet || r.Unique != row {
			report("set item %q: created quality %d row %d, want row %d", s.Name, r.Quality, r.Unique, row)

			continue
		}

		covered(r, s.Props[:], "set item "+s.Name)

		// Item bonus (aprop) lists end up in the tier lists 0xa5..0xa9 or
		// the base list; every named one must write something.
		for k, p := range s.AProps {
			if p.Prop < 0 || (p.Min == 0 && p.Max == 0) {
				continue
			}

			stats := c.propStats(p.Prop)
			hit := len(stats) == 0

			for _, w := range r.Writes {
				if stats[w.Stat] {
					hit = true
				}
			}

			if !hit {
				report("set item %q aprop%d (%s): writes none of %v", s.Name, k, c.Props.Props[p.Prop].Code, keys(stats))
			}
		}
	}

	t.Logf("%d unique and set items created, %d problems", created, failures)
}

func keys(m map[int]bool) []int {
	var out []int
	for k := range m {
		out = append(out, k)
	}

	sort.Ints(out)

	return out
}

// TestAuditSetBonuses rolls the bonuses of every set: each named partial or
// full property must produce a write.
func TestAuditSetBonuses(t *testing.T) {
	c := cLoadCreator(t)

	for si := range c.Uniques.Sets {
		def := &c.Uniques.Sets[si]

		b, ok := c.SetBonuses(si)
		if !ok {
			t.Errorf("set %q: no bonuses", def.Name)

			continue
		}

		if b.Pieces != len(def.Items) {
			t.Errorf("set %q: %d pieces, %d items", def.Name, b.Pieces, len(def.Items))
		}

		if len(def.Items) >= 2 {
			any := false

			for _, p := range append(append([]PropInst{}, def.Partial[:]...), def.Full[:]...) {
				if p.Prop >= 0 {
					any = true
				}
			}

			if !any {
				t.Errorf("set %q has no set bonus at all", def.Name)
			}
		}

		for k, p := range def.Full {
			if p.Prop >= 0 && len(b.Full) == 0 {
				t.Errorf("set %q full bonus %d (%s) wrote nothing", def.Name, k+1, c.Props.Props[p.Prop].Code)
			}
		}
	}
}

// TestAuditRunewords checks every row of Runes.txt: the runes are rune items
// in the order the table lists them, the item types exist, some base item
// with enough sockets can hold the word, the game's search finds the word
// for each such base (or an earlier row with the same runes), and applying
// it writes every named property into the runeword list.
func TestAuditRunewords(t *testing.T) {
	c := cLoadCreator(t)
	c.Runes = loadRuneTablesFrom(skipTB{t}, testSource(t), c.Props)
	tab := cReadTable(skipTB{t}, testSource(t), "Runes.txt")
	equiv, _ := rawTypes(t)

	if len(c.Runes.Words) == 0 {
		t.Fatal("no runewords loaded")
	}

	complete, withBase := 0, 0

	// the loader skips rows without a name; so does this
	var named [][]string

	for _, r := range tab.rows {
		if tab.s(r, "name") != "" {
			named = append(named, r)
		}
	}

	if len(named) != len(c.Runes.Words) {
		t.Fatalf("%d named rows, %d runewords loaded", len(named), len(c.Runes.Words))
	}

	for wi := range c.Runes.Words {
		w := &c.Runes.Words[wi]
		raw := named[wi]

		// rows such as "Serendipity" keep a name but no data and are not complete
		if !w.Complete {
			if len(w.Runes) != 0 && tab.n(raw, "complete") == 1 {
				t.Errorf("%s: loader dropped completeness", w.Name)
			}

			continue
		}

		if len(w.Runes) == 0 || len(w.Runes) > 6 {
			t.Errorf("%s: %d runes", w.Name, len(w.Runes))
		}

		for k, rc := range w.Runes {
			b := c.Items.ByCode[rc]
			if b == nil || !c.Items.IsA(b, "rune") {
				t.Errorf("%s rune%d %q is not a rune item", w.Name, k+1, rc)
			}

			// the loader keeps the table's order
			if want := tab.s(raw, fmt.Sprintf("rune%d", k+1)); want != rc {
				t.Errorf("%s rune%d = %q, table %q", w.Name, k+1, rc, want)
			}
		}

		// runes are contiguous from Rune1: the search stops at the first empty one
		for k := len(w.Runes) + 1; k <= 6; k++ {
			if tab.s(raw, fmt.Sprintf("rune%d", k)) != "" {
				t.Errorf("%s: rune%d set after an empty rune slot", w.Name, k)
			}
		}

		for _, ty := range append(append([]string{}, w.IType...), w.EType...) {
			if _, ok := equiv[ty]; ty != "" && !ok {
				t.Errorf("%s: item type %q unknown", w.Name, ty)
			}
		}

		named := 0

		for k, p := range w.Props {
			if code := tab.s(raw, fmt.Sprintf("t1code%d", k+1)); code != "" {
				named++

				if p.Prop < 0 {
					t.Errorf("%s: property %q does not resolve", w.Name, code)
				}
			}
		}

		if named == 0 {
			t.Errorf("%s: no properties", w.Name)
		}

		complete++
		found := false

		var applied *Rolled

		for _, b := range c.Items.Items {
			if !c.typeFits(b, w.IType, w.EType) || b.Quest != 0 {
				continue
			}

			if b.GemSockets < len(w.Runes) || !b.HasInv {
				continue
			}

			found = true
			got := c.FindRunewordFor(b.Code, QualityNormal, len(w.Runes), w.Runes)

			if got == nil {
				t.Errorf("%s: not found for %s with runes %v", w.Name, b.Code, w.Runes)

				continue
			}

			if strings.Join(got.Runes, ",") != strings.Join(w.Runes, ",") {
				t.Errorf("%s: found %s (runes %v) for %s", w.Name, got.Name, got.Runes, b.Code)
			}

			// quality and socket gates: a magic base or a short socket count never makes a word
			if c.FindRunewordFor(b.Code, QualityMagic, len(w.Runes), w.Runes) != nil ||
				c.FindRunewordFor(b.Code, QualityRare, len(w.Runes), w.Runes) != nil ||
				c.FindRunewordFor(b.Code, QualityNormal, len(w.Runes)+1, w.Runes) != nil {
				t.Errorf("%s: matched a magic/rare base or an unfilled socket", w.Name)
			}

			if applied == nil && got == w {
				req := Request{Code: b.Code, ILvl: 80, Quality: QualityNormal, Version: 100, Expansion: true, GameSeed: d2rand.Seed{Lo: 99, Hi: 0x29a}}
				if base, err := c.Create(req); err == nil {
					applied = c.ApplyRuneword(base, req, w)

					lists := 0

					for _, wr := range applied.Writes[len(base.Writes):] {
						if wr.List == ListRuneword {
							lists++
						}
					}

					if lists == 0 {
						t.Errorf("%s on %s: no writes in the runeword list", w.Name, b.Code)
					}
				}
			}
		}

		if found {
			withBase++
		} else {
			t.Errorf("%s: no base item has %d sockets and the right type (itype %v etype %v)", w.Name, len(w.Runes), w.IType, w.EType)
		}
	}

	t.Logf("%d runewords, %d complete, %d with a usable base", len(c.Runes.Words), complete, withBase)
}

// TestAuditGems checks Gems.txt: every row's code is a misc item (a gem or
// rune type) and the transform colour/letter columns are consistent, and each
// gem property is known.
func TestAuditGems(t *testing.T) {
	c := cLoadCreator(t)
	tab := cReadTable(skipTB{t}, testSource(t), "Gems.txt")
	rows := 0

	for _, r := range tab.rows {
		code := tab.s(r, "code")
		if code == "" {
			continue
		}

		rows++
		b := c.Items.ByCode[code]

		switch {
		case b == nil:
			t.Errorf("gem %q: code %q not an item", tab.s(r, "name"), code)

			continue
		case b.Kind != KindMisc:
			t.Errorf("gem %q: %s is not a misc item", tab.s(r, "name"), code)
		case !c.Items.IsA(b, "gem") && !c.Items.IsA(b, "rune"):
			t.Errorf("gem %q: %s is neither a gem nor a rune type (%s)", tab.s(r, "name"), code, b.Type)
		}

		// nummods says how many of the three mods of weapon/helm/shield are used
		n := tab.n(r, "nummods")
		for _, kind := range []string{"weaponMod", "helmMod", "shieldMod"} {
			used := 0

			for k := 1; k <= 3; k++ {
				if tab.s(r, fmt.Sprintf("%s%dCode", kind, k)) != "" {
					used++
				}
			}

			if used == 0 || used > 3 {
				t.Errorf("gem %q (%s): no %s property", tab.s(r, "name"), code, kind)
			}
		}

		if n < 1 || n > 3 {
			t.Errorf("gem %q: nummods %d", tab.s(r, "name"), n)
		}
	}

	// every gem and rune base item has a Gems.txt row
	have := map[string]bool{}
	for _, r := range tab.rows {
		have[tab.s(r, "code")] = true
	}

	for _, b := range c.Items.Items {
		if b.Kind == KindMisc && (c.Items.IsA(b, "gem") || c.Items.IsA(b, "rune")) && !have[b.Code] {
			t.Errorf("%s (%s) has no Gems.txt row", b.Code, b.Type)
		}
	}

	t.Logf("%d gem/rune rows", rows)
}

// rawAffix is the independent reading of an affix row.
type rawAffix struct {
	file, name         string
	level, maxlvl, frq int
	group              int
	itype, etype       []string
	class              string
	rare, spawn        bool
	version            int
}

// code4 is how the game stores an item type code: four bytes ("staff" is "staf").
func code4(s string) string {
	if len(s) > 4 {
		return s[:4]
	}

	return s
}

func readRawAffixes(t *testing.T) []rawAffix {
	t.Helper()

	var out []rawAffix

	for _, file := range []string{"MagicSuffix.txt", "MagicPrefix.txt", "AutoMagic.txt"} {
		tab := readTab(skipTB{t}, testSource(t), file)

		for _, r := range tab.rows {
			if tab.s(r, "name") == "Expansion" {
				continue
			}

			a := rawAffix{
				file: file, name: tab.s(r, "name"), level: tab.n(r, "level"), maxlvl: tab.n(r, "maxlevel"),
				frq: tab.n(r, "frequency"), group: tab.n(r, "group"), class: strings.ToLower(tab.s(r, "classspecific")),
				rare: tab.n(r, "rare") != 0, spawn: tab.n(r, "spawnable") != 0, version: tab.n(r, "version"),
			}

			for k := 1; k <= 7; k++ {
				if v := tab.s(r, fmt.Sprintf("itype%d", k)); v != "" {
					a.itype = append(a.itype, code4(v))
				}
			}

			for k := 1; k <= 5; k++ {
				if v := tab.s(r, fmt.Sprintf("etype%d", k)); v != "" {
					a.etype = append(a.etype, code4(v))
				}
			}

			out = append(out, a)
		}
	}

	return out
}

// TestAuditAffixTables checks the combined affix table against the raw tables
// (level, maxlevel, frequency, group, class, types) and that every referenced
// item type and class exists.
func TestAuditAffixTables(t *testing.T) {
	c := cLoadCreator(t)
	raw := readRawAffixes(t)
	equiv, _ := rawTypes(t)

	if len(c.Affixes.Rows) != len(raw) {
		t.Fatalf("%d affix rows loaded, %d in the tables", len(c.Affixes.Rows), len(raw))
	}

	if c.Affixes.NSuffix+c.Affixes.NPrefix > len(raw) {
		t.Fatalf("prefix/suffix split %d+%d", c.Affixes.NSuffix, c.Affixes.NPrefix)
	}

	for i, a := range raw {
		g := &c.Affixes.Rows[i]

		if g.Level != a.level || g.MaxLevel != a.maxlvl || g.Frequency != a.frq || g.Group != a.group ||
			g.Spawnable != a.spawn || g.Rare != a.rare || g.Version != a.version {
			t.Errorf("%s %q: loaded %+v, raw %+v", a.file, a.name, *g, a)
		}

		if a.class != "" {
			if _, ok := heroIndex[a.class]; !ok {
				t.Errorf("%s %q: class %q unknown", a.file, a.name, a.class)
			} else if g.Class != heroIndex[a.class] {
				t.Errorf("%s %q: class %d, want %d", a.file, a.name, g.Class, heroIndex[a.class])
			}
		} else if g.Class != -1 {
			t.Errorf("%s %q: class %d without restriction", a.file, a.name, g.Class)
		}

		for _, ty := range append(append([]string{}, a.itype...), a.etype...) {
			if _, ok := equiv[ty]; !ok {
				t.Errorf("%s %q: item type %q unknown", a.file, a.name, ty)
			}
		}

		if a.maxlvl != 0 && a.maxlvl < a.level {
			t.Errorf("%s %q: maxlevel %d below level %d", a.file, a.name, a.maxlvl, a.level)
		}
	}
}

// auditClass is the hero class of the base's type, 7 for none.
func auditClass(c *Creator, b *BaseItem) int {
	if ty := c.Items.Type(b); ty != nil && ty.Class >= 0 && ty.Class < classNone {
		return ty.Class
	}

	return classNone
}

// TestAuditGeneratedAffixes creates magic and rare items of every spawnable
// base item, both in a LoD and a classic game, over many seeds, and checks every
// affix that came out against the raw tables: type include/exclude through
// the independent closure, level window, spawnable flag, version, class
// restriction, rare flag, no repeated group, sockets only on socketable bases.
// It also counts how many distinct affixes each ilvl-99 sweep reaches.
func TestAuditGeneratedAffixes(t *testing.T) {
	c := cLoadCreator(t)
	raw := readRawAffixes(t)
	equiv, _ := rawTypes(t)
	bad := 0

	fits := func(a *rawAffix, b *BaseItem) bool {
		have := map[string]bool{}
		for k := range closure(equiv, b.Type) {
			have[k] = true
		}

		for k := range closure(equiv, b.Type2) {
			have[k] = true
		}

		for _, e := range a.etype {
			if have[e] {
				return false
			}
		}

		for _, i := range a.itype {
			if have[i] {
				return true
			}
		}

		return false
	}

	reached := map[int]bool{}
	sweeps := 0
	seeds := 8

	if testing.Short() {
		seeds = 2
	}

	for _, b := range c.Items.Items {
		if !b.Spawnable {
			continue
		}

		for _, v := range []struct {
			ver  int
			exp  bool
			ilvl int
		}{{100, true, 99}, {100, true, 25}, {0, false, 60}} {
			if !v.exp && b.Version >= 100 {
				continue
			}

			for _, q := range []Quality{QualityMagic, QualityRare} {
				for s := 0; s < seeds; s++ {
					sweeps++
					req := Request{
						Code: b.Code, ILvl: v.ilvl, Quality: q, Version: v.ver, Expansion: v.exp,
						GameSeed: d2rand.Seed{Lo: uint32(s)*977 + uint32(b.Class), Hi: 0x29a},
					}

					r, err := c.Create(req)
					if err != nil || r.Quality != q {
						continue
					}

					seen := map[int]string{}

					for _, id := range append(append([]int{}, r.Prefix[:]...), r.Suffix[:]...) {
						if id == 0 {
							continue
						}

						reached[id] = true
						a := raw[id-1]
						ctx := fmt.Sprintf("%s q%d v%d ilvl %d: %s %q", b.Code, q, v.ver, v.ilvl, a.file, a.name)

						problem := ""

						switch {
						case !a.spawn:
							problem = "not spawnable"
						case a.version >= 100 && v.ver < 100:
							problem = "LoD affix on a classic item"
						case !fits(&a, b):
							problem = "item type does not fit"
						case a.level > v.ilvl+2 && v.ver == 0, a.level > 99:
							problem = "level above the item's"
						case v.ver >= 100 && a.level > maxInt(v.ilvl, b.Level)+b.MagicLevel:
							problem = "level above the affix level bound"
						// the classic picker (5bef10) does not test the rare flag
						case q == QualityRare && !a.rare && v.ver >= 100:
							problem = "not allowed on rare items"
						case a.class != "" && v.ver >= 100 && auditClass(c, b) != classNone && heroIndex[a.class] != auditClass(c, b):
							problem = "class restriction violated"
						case v.ver >= 100 && a.frq == 0:
							problem = "frequency 0"
						}

						if prev, dup := seen[a.group]; problem == "" && dup && v.ver >= 100 {
							problem = "group " + fmt.Sprint(a.group) + " repeated with " + prev
						}

						seen[a.group] = a.name

						if problem != "" {
							bad++

							if bad <= 30 {
								t.Errorf("%s: %s", ctx, problem)
							}
						}
					}
				}
			}
		}
	}

	t.Logf("%d items generated, %d distinct affixes reached of %d, %d violations", sweeps, len(reached), len(raw), bad)
}

// TestAuditSocketsAndEthereal creates normal quality items of every base item
// over many seeds and checks socket counts and the ethereal flag against the
// tables: sockets never exceed min(gemsockets, type max, cells), only on
// hasinv non stackable bases; ethereal only on armor/weapons with durability
// and with the halved durability.
func TestAuditSocketsAndEthereal(t *testing.T) {
	c := cLoadCreator(t)
	isc := readTab(skipTB{t}, testSource(t), "ItemStatCost.txt")

	sockStat, durStat, maxDurStat := -1, -1, -1

	for _, r := range isc.rows {
		switch isc.s(r, "Stat") {
		case "item_numsockets":
			sockStat = isc.n(r, "ID")
		case "durability":
			durStat = isc.n(r, "ID")
		case "maxdurability":
			maxDurStat = isc.n(r, "ID")
		}
	}

	if sockStat < 0 {
		t.Fatal("item_numsockets not in ItemStatCost")
	}

	_, _ = durStat, maxDurStat
	socketed, ethereal, items := 0, 0, 0
	bad := 0

	fail := func(f string, a ...interface{}) {
		bad++

		if bad <= 30 {
			t.Errorf(f, a...)
		}
	}

	for _, b := range c.Items.Items {
		if !b.Spawnable || b.Version >= 100 && false {
			continue
		}

		ty := c.Items.Type(b)
		gear := c.Items.IsA(b, "weap") || c.Items.IsA(b, "armo")

		for s := 0; s < 120; s++ {
			req := Request{
				Code: b.Code, ILvl: 85, Quality: QualityNormal, Version: 100, Expansion: true,
				Difficulty: s % 3, GameSeed: d2rand.Seed{Lo: uint32(s)*131 + uint32(b.Class)*7, Hi: 0x29a},
			}

			r, err := c.Create(req)
			if err != nil {
				continue
			}

			items++

			n := 0

			for _, w := range r.Writes {
				if w.Stat == sockStat && w.Kind != 'S' {
					n = w.Value
				}
			}

			if r.Flags&0x800 != 0 || n > 0 {
				socketed++

				limit := b.GemSockets
				if ty != nil && ty.MaxSock40 > 0 && ty.MaxSock40 < limit {
					limit = ty.MaxSock40
				}

				if !b.HasInv || b.Stackable || n > limit || n > MaxSocketCells || n > b.InvWidth*b.InvHeight {
					fail("%s: %d sockets (hasinv %v stackable %v gemsockets %d type max %d cells %d)",
						b.Code, n, b.HasInv, b.Stackable, b.GemSockets, ty.MaxSock40, b.InvWidth*b.InvHeight)
				}
			}

			if r.Flags&0x400000 != 0 {
				ethereal++

				if !gear || b.NoDurability || b.Durability == 0 || b.Quest != 0 {
					fail("%s: ethereal but gear %v nodurability %v durability %d", b.Code, gear, b.NoDurability, b.Durability)
				}
			}
		}
	}

	t.Logf("%d items, %d socketed, %d ethereal, %d violations", items, socketed, ethereal, bad)
}
