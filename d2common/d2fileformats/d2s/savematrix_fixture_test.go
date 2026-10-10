package d2s

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestMatrixNokkaFixture compares the parse of the read-only nokka-d2s example (found under
// ~/git/nokka-d2s-ref/examples, no env var needed) with its expected JSON, checks the byte exact
// round trip and then re-writes its 60 real items under every class and flag combination.
func TestMatrixNokkaFixture(t *testing.T) {
	path := homePath("git", "nokka-d2s-ref", "examples", "nokkasorc")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture: %v", err)
	}

	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Skipf("fixture json: %v", err)
	}

	tb := robustTables()
	if tb == nil {
		t.Skip("real tables not found (D2_TABLES)")
	}

	var ref struct {
		Header struct {
			Name   string `json:"name"`
			Class  string `json:"class"`
			Status struct {
				Expansion bool `json:"expansion"`
				Died      bool `json:"died"`
				Hardcore  bool `json:"hardcore"`
			} `json:"status"`
		} `json:"header"`
		Attributes map[string]uint64 `json:"attributes"`
		Items      []struct {
			Type     string `json:"type"`
			ID       uint32 `json:"id"`
			Level    uint8  `json:"level"`
			Quality  uint8  `json:"quality"`
			Location uint8  `json:"location_id"`
			X        uint8  `json:"position_x"`
			Y        uint8  `json:"position_y"`
			Alt      uint8  `json:"alt_position_id"`
		} `json:"items"`
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}

	c, err := Parse(data, tb)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Write(c, tb)
	if err != nil || !bytes.Equal(out, data) {
		t.Fatalf("fixture is not byte-exact after a round trip (%v, first diff 0x%X)", err, firstDiff(out, data))
	}

	h := c.Header
	if h.Name != ref.Header.Name || h.Class.String() != ref.Header.Class || h.IsExpansion() != ref.Header.Status.Expansion ||
		h.IsHardcore() != ref.Header.Status.Hardcore || h.IsDead() != ref.Header.Status.Died {
		t.Fatalf("header %+v differs from %+v", h, ref.Header)
	}

	a := c.Body.Attributes
	for name, got := range map[string]uint64{"strength": a.Strength, "energy": a.Energy, "dexterity": a.Dexterity,
		"vitality": a.Vitality, "level": a.Level, "experience": a.Experience, "gold": a.Gold,
		"stashed_gold": a.StashedGold, "max_hp": a.MaxHP, "max_mana": a.MaxMana, "max_stamina": a.MaxStamina} {
		if got != ref.Attributes[name] {
			t.Errorf("attribute %s = %d, reference %d", name, got, ref.Attributes[name])
		}
	}

	if len(c.Items) != len(ref.Items) {
		t.Fatalf("%d items, reference %d", len(c.Items), len(ref.Items))
	}

	for i, r := range ref.Items {
		it := c.Items[i]
		if it.Code != r.Type || it.ID != r.ID || it.Level != r.Level || it.Quality != r.Quality ||
			it.Location != r.Location || it.X != r.X || it.Y != r.Y || it.Page != r.Alt {
			t.Errorf("item %d: %+v vs reference %+v", i, it, r)
		}
	}

	// the same body and items in every class / flag variant (the layout depends on the flags only)
	for class := Amazon; class <= Assassin; class++ {
		for flags := uint32(0); flags < 8; flags++ {
			v := *c
			hv := *c.Header
			v.Header = &hv
			hv.Class = class
			hv.Status = 0

			if flags&1 != 0 {
				hv.Status |= StatusExpansion
			}

			if flags&2 != 0 {
				hv.Status |= StatusHardcore | StatusDied
			}

			if flags&4 != 0 {
				hv.Status |= StatusLadder
			}

			roundTripExact(t, fmt.Sprintf("nokka/%v/%d", class, flags), &v, tb)
		}
	}
}
