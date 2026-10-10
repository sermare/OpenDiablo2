package d2itemdesc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Table-driven audit of the description tables against the raw game tables
// (D2_TABLES; skipped when unset). The raw text is parsed again here so the
// loaders are checked, not repeated.

func rawTable(t *testing.T, name string) *tsv {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
		return parseTSV(b)
	}

	for _, l := range layers {
		if b, err := os.ReadFile(filepath.Join(dir, "itemdesc", l, strings.ToLower(name))); err == nil {
			return parseTSV(b)
		}
	}

	t.Skipf("table %s not extracted", name)

	return nil
}

func TestAuditItemTypeIsA(t *testing.T) {
	tb := realTables(t)
	raw := rawTable(t, "ItemTypes.txt")
	equiv := map[string][2]string{}

	var order []string

	for _, r := range raw.rows {
		if c := raw.str(r, "Code"); c != "" {
			equiv[c] = [2]string{raw.str(r, "Equiv1"), raw.str(r, "Equiv2")}
			order = append(order, c)
		}
	}

	closure := func(code string) map[string]bool {
		out := map[string]bool{}

		var walk func(string)

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

	for _, a := range order {
		want := closure(a)

		for _, b := range order {
			if got := tb.IsA(a, b); got != want[b] {
				t.Fatalf("IsA(%s, %s) = %v, want %v", a, b, got, want[b])
			}
		}
	}

	// every base item's types are known types
	for code, b := range tb.Bases {
		if _, ok := equiv[b.Type]; !ok {
			t.Errorf("item %s: type %q unknown", code, b.Type)
		}
	}
}

func TestAuditUniqueSetRows(t *testing.T) {
	tb := realTables(t)

	raw := rawTable(t, "uniqueitems.txt")
	wantN := 0

	for _, r := range raw.rows {
		if idx := raw.str(r, "index"); idx != "" && !strings.EqualFold(idx, "Expansion") {
			wantN++
		}
	}

	if len(tb.Uniques) != wantN {
		t.Fatalf("%d uniques loaded, %d in the table", len(tb.Uniques), wantN)
	}

	i := 0

	for _, r := range raw.rows {
		idx := raw.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue
		}

		u := tb.Uniques[i]
		if u.Index != idx || u.Code != raw.str(r, "code") || u.LevelReq != raw.num(r, "lvl req") {
			t.Errorf("unique %d: loaded %+v, table %q %q %d", i, u, idx, raw.str(r, "code"), raw.num(r, "lvl req"))
		}

		if u.Code != "" && tb.Bases[u.Code] == nil && raw.num(r, "enabled") != 0 {
			t.Errorf("unique %q: base %q is not an item", idx, u.Code)
		}

		i++
	}

	sets := rawTable(t, "sets.txt")
	items := rawTable(t, "setitems.txt")

	for _, r := range items.rows {
		idx := items.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue
		}

		if tb.Sets[items.str(r, "set")] == nil {
			t.Errorf("set item %q: set %q unknown", idx, items.str(r, "set"))
		}

		if code := items.str(r, "item"); tb.Bases[code] == nil {
			t.Errorf("set item %q: base %q is not an item", idx, code)
		}
	}

	nsets := 0

	for _, r := range sets.rows {
		idx := sets.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue
		}

		nsets++

		s := tb.Sets[idx]
		if s == nil {
			t.Errorf("set %q not loaded", idx)

			continue
		}

		if len(s.Items) < 2 {
			t.Errorf("set %q has %d items", idx, len(s.Items))
		}
	}

	if nsets != len(tb.Sets) {
		t.Errorf("%d sets loaded, %d in the table", len(tb.Sets), nsets)
	}
}

func TestAuditRunewordsAndGems(t *testing.T) {
	tb := realTables(t)
	raw := rawTable(t, "runes.txt")

	type word struct {
		n     int
		runes []string
	}

	var words []word

	for _, r := range raw.rows {
		name := raw.str(r, "Name")
		if !strings.HasPrefix(name, "Runeword") {
			continue
		}

		n, err := strconv.Atoi(strings.TrimPrefix(name, "Runeword"))
		if err != nil {
			continue
		}

		w := word{n: n}

		for i := 1; i <= 6; i++ {
			if c := raw.str(r, fmt.Sprint("Rune", i)); c != "" {
				w.runes = append(w.runes, c)
			}
		}

		words = append(words, w)
	}

	// Runeword95 is listed twice; every row keeps a rank, file order among equals.
	sort.SliceStable(words, func(i, j int) bool { return words[i].n < words[j].n })

	if len(tb.Runewords) != len(words) {
		t.Fatalf("%d runewords loaded, %d rows in the table", len(tb.Runewords), len(words))
	}

	for i, w := range words {
		rw := tb.Runewords[i]
		if rw.Key != fmt.Sprint("Runeword", w.n) {
			t.Errorf("runeword id %d is %s, want Runeword%d", i+runewordIDBase, rw.Key, w.n)
		}

		if strings.Join(rw.Runes, ",") != strings.Join(w.runes, ",") {
			t.Errorf("%s (id %d) runes %v, table %v", rw.Key, i+runewordIDBase, rw.Runes, w.runes)
		}

		for _, r := range rw.Runes {
			if b := tb.Bases[r]; b == nil || !tb.IsA(b.Type, "rune") {
				t.Errorf("%s: rune %q is not a rune item", rw.Key, r)
			}
		}
	}

	gems := rawTable(t, "gems.txt")
	rows := 0

	for _, r := range gems.rows {
		code := gems.str(r, "code")
		if code == "" {
			continue
		}

		rows++

		g := tb.Gems[code]
		if g == nil || tb.Bases[code] == nil {
			t.Errorf("gem %q: loaded %v, base %v", code, g != nil, tb.Bases[code] != nil)

			continue
		}

		for _, list := range [][]PropSpec{g.Weapon, g.Helm, g.Shield} {
			if len(list) == 0 {
				t.Errorf("gem %q: a grip without properties", code)
			}

			for _, p := range list {
				if _, ok := tb.Props[strings.ToLower(p.Code)]; !ok {
					t.Errorf("gem %q: property %q unknown", code, p.Code)
				}
			}
		}
	}

	if rows != len(tb.Gems) {
		t.Errorf("%d gems loaded, %d in the table", len(tb.Gems), rows)
	}
}

// TestAuditAffixAndBonusProps checks that the property codes of the set
// bonuses and the affix names are resolvable by the describer.
func TestAuditAffixAndBonusProps(t *testing.T) {
	tb := realTables(t)

	for _, s := range tb.Sets {
		var specs []PropSpec
		for _, p := range s.Partial {
			specs = append(specs, p...)
		}

		specs = append(specs, s.Full...)

		for _, p := range specs {
			if strings.HasPrefix(p.Code, "*") {
				continue
			}

			if _, ok := tb.Props[strings.ToLower(p.Code)]; !ok {
				t.Errorf("set %q: property %q unknown", s.Index, p.Code)
			}
		}
	}

	for _, si := range tb.SetItems {
		for _, list := range si.Partial {
			for _, p := range list {
				if _, ok := tb.Props[strings.ToLower(p.Code)]; !ok && !strings.HasPrefix(p.Code, "*") {
					t.Errorf("set item %q: property %q unknown", si.Index, p.Code)
				}
			}
		}
	}

	// affix level requirements come straight from the table
	for _, f := range []struct {
		file string
		rows []AffixDef
	}{{"magicprefix.txt", tb.Prefixes}, {"magicsuffix.txt", tb.Suffixes}} {
		raw := rawTable(t, f.file)
		if len(raw.rows) != len(f.rows) {
			t.Errorf("%s: %d rows loaded, %d in the table", f.file, len(f.rows), len(raw.rows))

			continue
		}

		for i, r := range raw.rows {
			if f.rows[i].Name != raw.str(r, "Name") || f.rows[i].LevelReq != raw.num(r, "levelreq") {
				t.Errorf("%s row %d: loaded %+v", f.file, i, f.rows[i])
			}
		}
	}
}
