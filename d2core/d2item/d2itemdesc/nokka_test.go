package d2itemdesc

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// nokka's example save has an expected parse (names, stats, templates of
// the stat lines). These tests check the descriptions against it. They need
// D2_TABLES, D2S_SAMPLE_BODY (the save) and D2S_SAMPLE_BODY_JSON.

type nokkaAttr struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Values []int  `json:"values"`
}

type nokkaItem struct {
	Type            string `json:"type"`
	Quality         int    `json:"quality"`
	Ethereal        int    `json:"ethereal"`
	Defense         *int   `json:"defense_rating"`
	CurDur          *int   `json:"current_durability"`
	MaxDur          *int   `json:"max_durability"`
	Quantity        *int   `json:"quantity"`
	UniqueName      string `json:"unique_name"`
	SetName         string `json:"set_name"`
	RunewordName    string `json:"runeword_name"`
	RareName        string `json:"rare_name"`
	RareName2       string `json:"rare_name2"`
	MagicPrefixName string `json:"magic_prefix_name"`
	MagicSuffixName string `json:"magic_suffix_name"`
	BaseDamage      *struct {
		Min    int `json:"min"`
		Max    int `json:"max"`
		TwoMin int `json:"twohand_min"`
		TwoMax int `json:"twohand_max"`
	} `json:"base_damage"`
	Magic    []nokkaAttr   `json:"magic_attributes"`
	Runeword []nokkaAttr   `json:"runeword_attributes"`
	Set      [][]nokkaAttr `json:"set_attributes"`
	Children []nokkaItem   `json:"socketed_items"`
}

func loadNokka(t *testing.T) []nokkaItem {
	t.Helper()

	path := os.Getenv("D2S_SAMPLE_BODY_JSON")
	if path == "" {
		t.Skip("D2S_SAMPLE_BODY_JSON not set")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Items []nokkaItem `json:"items"`
	}

	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}

	return doc.Items
}

func texts(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Text
	}

	return out
}

func TestNokkaNamesAndBaseStats(t *testing.T) {
	tb := realTables(t)
	c := sampleChar(t)
	js := loadNokka(t)

	if len(js) != len(c.Items) {
		t.Fatalf("items: parsed %d, expected parse has %d", len(c.Items), len(js))
	}

	for i := range c.Items {
		it, want := &c.Items[i], js[i]
		lines := tb.Describe(it, nil)
		all := strings.Join(texts(lines), "\n")
		head := strings.Join(texts(lines[:minInt(2, len(lines))]), "\n")

		for _, n := range []string{want.UniqueName, want.SetName, want.RunewordName, want.RareName, want.RareName2,
			want.MagicPrefixName, want.MagicSuffixName} {
			if n == "" {
				continue
			}

			// the set name is on the set listing, the others in the name lines
			if n == want.SetName {
				if !strings.Contains(all, n) {
					t.Errorf("item %d (%s): set name %q missing in\n%s", i, want.Type, n, all)
				}

				continue
			}

			if !strings.Contains(head, n) {
				t.Errorf("item %d (%s): name %q missing in head:\n%s", i, want.Type, n, head)
			}
		}

		if want.Defense != nil && it.Defense != *want.Defense {
			t.Errorf("item %d: stored defense %d, expected %d", i, it.Defense, *want.Defense)
		}

		if want.CurDur != nil && !strings.Contains(all, fmt.Sprintf("%d of %d", *want.CurDur, *want.MaxDur)) &&
			!strings.Contains(all, "Indestructible") {
			t.Errorf("item %d: durability %d of %d missing in\n%s", i, *want.CurDur, *want.MaxDur, all)
		}

		if want.Quantity != nil && !strings.Contains(all, fmt.Sprintf("%d", *want.Quantity)) {
			t.Errorf("item %d: quantity %d missing in\n%s", i, *want.Quantity, all)
		}
	}
}

// TestNokkaWeaponDamage compares the base damage the expected parse lists
// with the damage line, for the weapons without damage enhancements.
func TestNokkaWeaponDamage(t *testing.T) {
	tb := realTables(t)
	c := sampleChar(t)
	js := loadNokka(t)

	checked := 0

	for i := range c.Items {
		want := js[i]
		if want.BaseDamage == nil {
			continue
		}

		var wantLine string
		if want.BaseDamage.TwoMax > 0 {
			wantLine = fmt.Sprintf("Two-Hand Damage: %d to %d", want.BaseDamage.TwoMin, want.BaseDamage.TwoMax)
		} else {
			wantLine = fmt.Sprintf("One-Hand Damage: %d to %d", want.BaseDamage.Min, want.BaseDamage.Max)
		}

		got := strings.Join(texts(tb.Describe(&c.Items[i], nil)), "\n")
		if !strings.Contains(got, wantLine) {
			t.Errorf("item %d (%s): %q missing in\n%s", i, want.Type, wantLine, got)
		}

		checked++
	}

	if checked == 0 {
		t.Fatal("no weapon in the sample")
	}
}

// expectedStats sums what the expected parse lists for an item: its own
// attributes, the runeword's and those of the socketed items.
func expectedStats(want nokkaItem) map[int]int {
	sum := make(map[int]int)

	add := func(a []nokkaAttr) {
		for _, x := range a {
			if len(x.Values) == 1 {
				sum[x.ID] += x.Values[0]
			}
		}
	}

	add(want.Magic)
	add(want.Runeword)

	for _, ch := range want.Children {
		add(ch.Magic)
	}

	return sum
}

// TestNokkaStats checks the stats of every item (own + runeword + the
// effects of the socketed runes and gems, which the save does not store and
// come from gems.txt) against the expected parse.
func TestNokkaStats(t *testing.T) {
	tb := realTables(t)
	c := sampleChar(t)
	js := loadNokka(t)

	for i := range c.Items {
		it := &c.Items[i]
		got := make(map[int]int)

		for _, s := range tb.ItemStats(it) {
			if s.Param == 0 {
				got[s.ID] += s.Value
			}
		}

		for id, v := range expectedStats(js[i]) {
			if got[id] != v {
				t.Errorf("item %d (%s): stat %d = %d, expected %d", i, js[i].Type, id, got[id], v)
			}
		}
	}
}

// nokkaWording lists the stats where the template of the expected parse
// words the line differently from the game's own string table (the game
// text is what the descriptions use).
var nokkaWording = map[int]string{
	127: `game: "+2 to All Skills"; template: "+2 to All Skill Levels"`,
	11:  `game: "+180 Maximum Stamina"; template: "+180 to Maximum Stamina"`,
	87:  `game: "Reduces all Vendor Prices 10%"; template: "Reduces Prices 10%"`,
	32:  `game: "+250 Defense vs. Missile"; template: "+250 vs. Missile"`,
}

var nonWord = regexp.MustCompile(`[^a-z0-9\-]+`)

// norm lowers a line and drops signs, percents and spacing differences so
// that nokka's templates and the game strings can be compared.
func norm(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "plus", "")

	return strings.Join(strings.Fields(nonWord.ReplaceAllString(s, " ")), " ")
}

func render(tmpl string, vals []int) string {
	for i, v := range vals {
		tmpl = strings.ReplaceAll(tmpl, fmt.Sprintf("{%d}", i), fmt.Sprint(v))
	}

	return tmpl
}

// TestNokkaTemplates compares the lines of the simple stats (one source,
// not grouped) with nokka's own rendering of the stat. The templates are an
// approximation of the game's text, so the comparison ignores case, signs,
// and the word "plus"; the numbers must be equal.
func TestNokkaTemplates(t *testing.T) {
	tb := realTables(t)
	c := sampleChar(t)
	js := loadNokka(t)

	compared := 0

	for i := range c.Items {
		it, want := &c.Items[i], js[i]
		lines := tb.Describe(it, nil)
		all := norm(strings.Join(texts(lines), "\n"))

		own := make(map[int]int)
		for _, s := range tb.ItemStats(it) {
			own[s.ID]++
		}

		for _, a := range append(append([]nokkaAttr{}, want.Magic...), want.Runeword...) {
			def := tb.Stats[a.ID]
			// grouped, paired, parameterised and multi value stats are
			// rendered by the engine in ways the template cannot express
			if def == nil || def.Grp != 0 || len(a.Values) != 1 || def.Func > 4 && def.Func != 12 ||
				a.ID >= 17 && a.ID <= 18 || a.ID >= 48 && a.ID <= 59 {
				continue
			}

			if _, differs := nokkaWording[a.ID]; differs {
				continue
			}

			// a stat that is listed twice (item + runeword) or that also
			// comes from the sockets is shown as one summed line
			if sumFromChildren(want, a.ID) || countID(want, a.ID) > 1 {
				continue
			}

			compared++

			if exp := norm(render(a.Name, a.Values)); !strings.Contains(all, exp) {
				t.Errorf("item %d (%s): expected line %q (stat %d) not in\n%s", i, want.Type, exp, a.ID, all)
			}
		}
	}

	if compared < 20 {
		t.Fatalf("only %d stat lines compared", compared)
	}

	t.Logf("%d stat lines match the expected parse", compared)
}

func countID(want nokkaItem, id int) int {
	n := 0

	for _, a := range append(append([]nokkaAttr{}, want.Magic...), want.Runeword...) {
		if a.ID == id {
			n++
		}
	}

	return n
}

func sumFromChildren(want nokkaItem, id int) bool {
	for _, ch := range want.Children {
		for _, a := range ch.Magic {
			if a.ID == id {
				return true
			}
		}
	}

	return false
}

var _ = d2s.QualityMagic

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
