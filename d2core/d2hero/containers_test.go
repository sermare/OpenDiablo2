package d2hero

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func allKnown(string) bool { return true }

func TestStoredFromD2S(t *testing.T) {
	tests := []struct {
		name     string
		item     d2s.Item
		known    func(string) bool
		wantPage int
		skip     bool
	}{
		{"inventory", d2s.Item{Code: "cm1 ", Location: d2s.LocationStored, Page: 1, X: 3, Y: 2, Quality: 4, Level: 20}, allKnown, PageInventory, false},
		{"stash", d2s.Item{Code: "r07 ", Location: d2s.LocationStored, Page: 5, X: 5, Y: 7, Simple: true}, allKnown, PageStash, false},
		{"cube", d2s.Item{Code: "box ", Location: d2s.LocationStored, Page: 4}, allKnown, PageCube, false},
		{"belt", d2s.Item{Code: "hp5 ", Location: d2s.LocationBelt, X: 5, Simple: true}, allKnown, PageBelt, false},
		{"equipped", d2s.Item{Code: "hlm ", Location: d2s.LocationEquipped, Equipped: 1}, allKnown, 0, true},
		{"cursor", d2s.Item{Code: "hlm ", Location: d2s.LocationCursor}, allKnown, 0, true},
		{"unknown page", d2s.Item{Code: "hlm ", Location: d2s.LocationStored, Page: 9}, allKnown, 0, true},
		{"ear", d2s.Item{Code: "ear ", Ear: true, Location: d2s.LocationStored, Page: 1}, allKnown, 0, true},
		{"no record", d2s.Item{Code: "zzz ", Location: d2s.LocationStored, Page: 1}, func(string) bool { return false }, 0, true},
	}

	for _, tt := range tests {
		got, skip := StoredFromD2S(&tt.item, tt.known)
		if (skip != "") != tt.skip {
			t.Errorf("%s: skip = %q", tt.name, skip)
			continue
		}

		if tt.skip {
			continue
		}

		if got.Page != tt.wantPage || got.X != int(tt.item.X) || got.Y != int(tt.item.Y) || got.Code != trimCode(tt.item.Code) {
			t.Errorf("%s: got %+v", tt.name, got)
		}

		if got.D2S == nil || !got.Origin {
			t.Errorf("%s: d2s original / origin flag missing", tt.name)
		}
	}
}

func TestStoredFromD2SQualityAndDurability(t *testing.T) {
	it := d2s.Item{Code: "plt", Location: d2s.LocationStored, Page: 1, Quality: 0, MaxDurability: 60, Durability: 30, Quantity: 0}
	got, _ := StoredFromD2S(&it, allKnown)

	if got.Quality != int(d2s.QualityNormal) {
		t.Errorf("quality 0 should import as normal, got %d", got.Quality)
	}

	if got.Durability == nil || *got.Durability != 30 {
		t.Errorf("durability = %v", got.Durability)
	}
}

func TestContainersJSONRoundTripAndBackwardsCompat(t *testing.T) {
	d := 12
	state := &HeroState{HeroName: "x", Containers: &HeroContainers{Items: []StoredItem{
		{Code: "cap", Page: PageInventory, X: 1, Y: 2, Quality: 4, ILvl: 9, Seed: 77, Prefixes: []string{"Crimson"}, Durability: &d},
		{Code: "hp1", Page: PageBelt, X: 3},
		{Code: "box", Page: PageStash, X: 0, Y: 0, Identified: true},
	}}}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}

	var got HeroState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.Containers, state.Containers) {
		t.Fatalf("round trip differs:\n%s", data)
	}

	if n := len(got.Containers.Page(PageStash)); n != 1 {
		t.Errorf("stash page has %d items", n)
	}

	// an old hero file has no containers and must load (and save) without them
	var old HeroState
	if err := json.Unmarshal([]byte(`{"heroName":"old","act":1}`), &old); err != nil || old.Containers != nil {
		t.Fatalf("old file: %v %v", err, old.Containers)
	}

	out, _ := json.Marshal(&old)

	var m map[string]json.RawMessage
	_ = json.Unmarshal(out, &m)

	if _, ok := m["containers"]; ok {
		t.Error("nil containers should be omitted")
	}

	// an empty but saved container set stays non-nil
	empty, _ := json.Marshal(&HeroState{Containers: &HeroContainers{Items: []StoredItem{}}})

	var back HeroState
	_ = json.Unmarshal(empty, &back)

	if back.Containers == nil {
		t.Error("saved empty containers must stay non-nil")
	}
}

func TestExportD2SItemsMovesItems(t *testing.T) {
	orig := d2s.Item{Code: "cm1 ", Location: d2s.LocationStored, Page: 1, X: 3, Y: 2, Quality: 4}
	s, _ := StoredFromD2S(&orig, allKnown)

	// the player moved the charm to the stash and added a new item
	s.Page, s.X, s.Y = PageStash, 4, 5
	c := &HeroContainers{Items: []StoredItem{s, {Code: "hp1", Page: PageInventory}}}

	out, skipped := ExportD2SItems(c)
	if len(out) != 1 || len(skipped) != 1 || skipped[0].Code != "hp1" {
		t.Fatalf("export = %d items, skipped %v", len(out), skipped)
	}

	if out[0].Location != d2s.LocationStored || out[0].Page != 5 || out[0].X != 4 || out[0].Y != 5 || out[0].Code != "cm1 " {
		t.Errorf("exported %+v", out[0])
	}

	if orig.Page != 1 || orig.X != 3 {
		t.Error("export changed the original item")
	}
}

func TestExportD2SItemsCarriesIdentified(t *testing.T) {
	orig := d2s.Item{Code: "cm1 ", Location: d2s.LocationStored, Page: 1, Quality: 4}
	s, _ := StoredFromD2S(&orig, allKnown)

	if s.Identified {
		t.Fatal("test item should start unidentified")
	}

	out, _ := ExportD2SItems(&HeroContainers{Items: []StoredItem{s}})
	if out[0].Identified {
		t.Error("an unidentified item was exported as identified")
	}

	// identified in the game (the container snapshot sets the flag)
	s.Identified = true

	out, _ = ExportD2SItems(&HeroContainers{Items: []StoredItem{s}})
	if !out[0].Identified {
		t.Error("the identified flag was lost")
	}

	if orig.Identified {
		t.Error("export changed the original item")
	}

	// the flag survives the JSON hero file
	b, err := json.Marshal(&HeroContainers{Items: []StoredItem{s}})
	if err != nil {
		t.Fatal(err)
	}

	var back HeroContainers
	if err := json.Unmarshal(b, &back); err != nil || !back.Items[0].Identified {
		t.Errorf("json round trip: %v %+v", err, back)
	}
}

// realSample loads the sample save and tables named by D2_TABLES and D2S_SAMPLE_BODY.
func realSample(t *testing.T) ([]byte, *d2s.ItemTables) {
	t.Helper()

	dir, path := os.Getenv("D2_TABLES"), os.Getenv("D2S_SAMPLE_BODY")
	if dir == "" || path == "" {
		t.Skip("set D2_TABLES and D2S_SAMPLE_BODY to run")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		t.Skipf("none of %v found in %s", names, dir)

		return nil
	}

	tables, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"), read("armor.txt"),
		read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data, tables
}

// The sample save (nokkasorc) holds 20 inventory items, 18 stash items (one of
// them the Horadric Cube), no cube items and 11 belt potions (expected parse:
// nokkasorc.json). Importing and exporting must keep every item, page and cell,
// and the exported list must survive the .d2s writer and parser.
func TestRealSampleContainersRoundTripThroughWriter(t *testing.T) {
	data, tables := realSample(t)

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	var containers HeroContainers

	for i := range c.Items {
		if s, skip := StoredFromD2S(&c.Items[i], allKnown); skip == "" {
			containers.Items = append(containers.Items, s)
		}
	}

	inv, belt, cube, stash := len(containers.Page(PageInventory)), len(containers.Page(PageBelt)),
		len(containers.Page(PageCube)), len(containers.Page(PageStash))
	t.Logf("pages: inv=%d belt=%d cube=%d stash=%d", inv, belt, cube, stash)

	if filepath.Base(os.Getenv("D2S_SAMPLE_BODY")) == "nokkasorc" && (inv != 20 || belt != 11 || cube != 0 || stash != 18) {
		t.Errorf("nokkasorc.json has 20/11/0/18 inventory/belt/cube/stash items")
	}

	exported, skipped := ExportD2SItems(&containers)
	if len(skipped) != 0 {
		t.Fatalf("imported items must export, %d skipped", len(skipped))
	}

	if len(exported) != len(containers.Items) {
		t.Fatalf("exported %d of %d", len(exported), len(containers.Items))
	}

	// every original stored/belt item comes back unchanged
	k := 0

	for i := range c.Items {
		if c.Items[i].Location != d2s.LocationStored && c.Items[i].Location != d2s.LocationBelt {
			continue
		}

		if !reflect.DeepEqual(exported[k], c.Items[i]) {
			t.Fatalf("item %d (%s) changed on export", k, c.Items[i].Code)
		}

		k++
	}

	// write them back in place of the originals and parse the result
	var rebuilt []d2s.Item

	k = 0

	for i := range c.Items {
		if c.Items[i].Location == d2s.LocationStored || c.Items[i].Location == d2s.LocationBelt {
			rebuilt = append(rebuilt, exported[k])
			k++
		} else {
			rebuilt = append(rebuilt, c.Items[i])
		}
	}

	c.Items = rebuilt

	out, err := d2s.Write(c, tables)
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != string(data) {
		t.Fatal("rebuilding the item list from the containers changed the file")
	}
}
