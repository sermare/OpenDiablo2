package d2vendor

import "testing"

func TestAllActsVendors(t *testing.T) {
	want := map[int]struct {
		name            string
		repairs, gamble bool
	}{
		148: {"Akara", false, false}, 154: {"Charsi", true, false}, 147: {"Gheed", false, true},
		178: {"Fara", true, false}, 177: {"Drognan", false, false}, 199: {"Elzix", false, true}, 202: {"Lysander", false, false},
		253: {"Hralti", true, false}, 254: {"Alkor", false, true}, 255: {"Ormus", false, false}, 252: {"Asheara", false, false},
		257: {"Halbu", true, false}, 405: {"Jamella", false, true},
		511: {"Larzuk", true, false}, 513: {"Malah", false, false}, 512: {"Drehya", false, true},
	}

	if len(All()) != len(want) {
		t.Fatalf("%d vendors, want %d", len(All()), len(want))
	}

	for class, w := range want {
		v, ok := ByClassID(class)
		if !ok || v.Name != w.name || v.Repairs != w.repairs || v.Gambles != w.gamble {
			t.Errorf("class %d: %+v ok=%v, want %+v", class, v, ok, w)
		}
	}

	if v, ok := ByName("hratli"); !ok || v.ClassID != 253 {
		t.Error("ByName must find Hratli by his npc.txt id")
	}

	if _, ok := ByClassID(514); ok {
		t.Error("Nihlathak has no item columns and must not be a vendor")
	}
}
