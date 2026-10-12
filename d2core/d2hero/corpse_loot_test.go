package d2hero

import "testing"

type fakeLoot struct {
	need    map[string]bool // codes whose requirements fail
	worn    map[int]string
	invRoom int
	belt    map[string]bool
	blocked map[string]bool
	inv     []string
}

func (f *fakeLoot) MeetsRequirements(it *StoredItem) bool { return !f.need[it.Code] }
func (f *fakeLoot) CanTake(it *StoredItem) bool           { return !f.blocked[it.Code] }
func (f *fakeLoot) TryEquip(slot int, it *StoredItem) bool {
	if f.worn[slot] != "" {
		return false
	}

	f.worn[slot] = it.Code

	return true
}
func (f *fakeLoot) TryBelt(it *StoredItem) bool { return f.belt[it.Code] }
func (f *fakeLoot) TryInventory(it *StoredItem) bool {
	if f.invRoom <= 0 {
		return false
	}

	f.invRoom--
	f.inv = append(f.inv, it.Code)

	return true
}

func eq(slot int, code string) StoredItem { return StoredItem{Code: code, Page: PageEquipped, X: slot} }

func TestLootCorpse(t *testing.T) {
	cases := []struct {
		name     string
		items    []StoredItem
		host     fakeLoot
		wantLeft int
		wantInv  int
		wantWorn map[int]string
	}{
		{"all worn back", []StoredItem{eq(1, "cap"), eq(3, "qui")}, fakeLoot{worn: map[int]string{}, invRoom: 1}, 0, 0,
			map[int]string{1: "cap", 3: "qui"}},
		{"slot busy goes to inventory", []StoredItem{eq(1, "cap")}, fakeLoot{worn: map[int]string{1: "skp"}, invRoom: 1}, 0, 1,
			map[int]string{1: "skp"}},
		{"no room is incomplete", []StoredItem{eq(1, "cap")}, fakeLoot{worn: map[int]string{1: "skp"}}, 1, 0, map[int]string{1: "skp"}},
		{"requirements fail then worn in pass two", []StoredItem{eq(3, "plt")},
			fakeLoot{worn: map[int]string{}, need: map[string]bool{"plt": true}, invRoom: 1}, 0, 0, map[int]string{3: "plt"}},
		{"cursor item and quest gate", []StoredItem{{Code: "bks", Page: PageInventory}, eq(1, "cap")},
			fakeLoot{worn: map[int]string{}, blocked: map[string]bool{"bks": true}, invRoom: 3}, 1, 0, map[int]string{1: "cap"}},
		{"belt potion", []StoredItem{{Code: "hp1", Page: PageInventory}},
			fakeLoot{worn: map[int]string{}, belt: map[string]bool{"hp1": true}}, 0, 0, map[int]string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Corpse{Items: tc.items}
			h := tc.host

			left, complete := LootCorpse(c, &h)
			if len(left) != tc.wantLeft || complete != (tc.wantLeft == 0) || len(c.Items) != tc.wantLeft {
				t.Fatalf("left=%v complete=%v", left, complete)
			}

			if len(h.inv) != tc.wantInv {
				t.Fatalf("inventory %v want %d", h.inv, tc.wantInv)
			}

			for s, code := range tc.wantWorn {
				if h.worn[s] != code {
					t.Fatalf("slot %d = %q want %q", s, h.worn[s], code)
				}
			}
		})
	}
}

func TestCanLootCorpse(t *testing.T) {
	for _, tc := range []struct {
		owner, looter string
		party, want   bool
	}{{"a", "a", false, true}, {"a", "b", false, false}, {"a", "b", true, true}} {
		if got := CanLootCorpse(tc.owner, tc.looter, tc.party); got != tc.want {
			t.Errorf("%+v: %v", tc, got)
		}
	}
}
