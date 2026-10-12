package d2statlist

import "testing"

// ITEM_ApplySetBonusProperties 0x6626a0 (VERIFIED w45): the worn set mates form a bit mask whose
// popcount n (table 0x6eeff0) selects partial tiers 2 .. min(n, pieces-1) and, from n >= pieces, the
// full bonus; pieces is the set's field +0xc, counted by DATATBL_LoadSetsTable 0x635680 as the SetItems
// rows pointing at the set (at most 6).
func TestSetBonusTiersByPiecesWorn(t *testing.T) {
	tier := func(id int) []Prop { return []Prop{{ID: id, Value: 1}} }
	partial := [][]Prop{tier(1), tier(2), tier(3), tier(4)}
	full := []Prop{{ID: 99, Value: 1}}

	tests := []struct {
		pieces, worn int
		want         []int
	}{
		{2, 1, nil},
		{2, 2, []int{99}},
		{3, 2, []int{1, 99}[:1]},
		{3, 3, []int{1, 99}},
		{4, 3, []int{1, 2}},
		{4, 4, []int{1, 2, 99}},
		{5, 4, []int{1, 2, 3}},
		{6, 5, []int{1, 2, 3, 4}},
		{6, 6, []int{1, 2, 3, 4, 99}},
	}

	for _, tc := range tests {
		defs := SetDefs{7: {Pieces: tc.pieces, Partial: partial, Full: full}}

		var got []int
		for _, p := range defs.SetBonus(7, tc.worn) {
			got = append(got, p.ID)
		}

		if len(got) != len(tc.want) {
			t.Errorf("%d piece set, %d worn: got %v want %v", tc.pieces, tc.worn, got, tc.want)
			continue
		}

		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%d piece set, %d worn: got %v want %v", tc.pieces, tc.worn, got, tc.want)
			}
		}
	}
}
