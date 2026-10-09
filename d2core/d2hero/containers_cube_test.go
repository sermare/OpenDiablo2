package d2hero

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestStoredFromD2SSockets(t *testing.T) {
	it := d2s.Item{Code: "kit ", Location: d2s.LocationStored, Page: 1, Socketed: true, TotalSockets: 3,
		Children: []d2s.Item{{Code: "r08 "}, {Code: "r09 "}}}
	got, skip := StoredFromD2S(&it, allKnown)

	if skip != "" || got.Sockets != 3 || !reflect.DeepEqual(got.Socketed, []string{"r08", "r09"}) {
		t.Fatalf("got %+v skip %q", got, skip)
	}

	plain, _ := StoredFromD2S(&d2s.Item{Code: "kit ", Location: d2s.LocationStored, Page: 1}, allKnown)
	if plain.Sockets != 0 || plain.Socketed != nil {
		t.Errorf("unsocketed item: %+v", plain)
	}
}

func TestStoredCubeFieldsRoundTripJSON(t *testing.T) {
	in := StoredItem{Code: "rin", Page: PageCube, Quality: 8, Crafted: true, Sockets: 2, Socketed: []string{"gcv"}, Runeword: "Steel",
		Mods: []StoredMod{{Code: "thorns", Min: 3, Max: 7, Value: 5}}}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var out StoredItem
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip: %+v != %+v", out, in)
	}
}
