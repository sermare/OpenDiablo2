package d2drop

import "testing"

// The Cow King's Leathers set (Sets.txt row 0x1d) is a candidate of the set item pick only when
// request flag 1 is set (FlagCowDropper); VERIFIED against ITEMGEN_PickSetItemLod 0x5c0350.
func TestCowKingSetNeedsCowDropperFlag(t *testing.T) {
	cases := []struct {
		name  string
		flags RequestFlags
		want  bool
	}{
		{"no flag", 0, false},
		{"ethereal flag only", FlagForceEthereal, false},
		{"cow dropper", FlagCowDropper, true},
		{"cow dropper plus others", FlagCowDropper | FlagForceSockets, true},
	}

	for _, c := range cases {
		cr := &Creator{Uniques: &UniqueTables{SetItems: []SetItem{{Name: "Cow King's Hide", Set: cowKingSet, Version: 100, Lvl: 1, Rarity: 1, Code: "stu"}}}}
		st := &itemState{c: cr, base: &BaseItem{Code: "stu"}, ilvl: 30}
		st.req.Version = 100
		st.req.Flags = c.flags

		if got := cr.pickSetLod(st); got != c.want {
			t.Errorf("%s: pick = %v, want %v", c.name, got, c.want)
		}
	}
}
