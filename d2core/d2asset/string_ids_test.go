package d2asset

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2tbl"
)

func TestStringByID(t *testing.T) {
	am := &AssetManager{idTables: []idTable{
		{base: stringIDPatchBase, m: d2tbl.IDStrings{5: "patched"}},
		{base: stringIDExpansionBase, m: d2tbl.IDStrings{7: "exp"}},
		{base: 0, m: d2tbl.IDStrings{0x1055: "Game: ", 0x1422: "Hell"}},
	}}

	tests := []struct {
		id   int
		want string
		ok   bool
	}{
		{0x1055, "Game: ", true}, {0x1422, "Hell", true}, {10005, "patched", true},
		{20007, "exp", true}, {0x58ca, "", false}, {-3, "", false},
	}

	for _, tc := range tests {
		if got, ok := am.StringByID(tc.id); got != tc.want || ok != tc.ok {
			t.Errorf("id %d: got (%q,%v) want (%q,%v)", tc.id, got, ok, tc.want, tc.ok)
		}
	}

	if idTableBase("data\\local\\lng\\eng\\patchstring.tbl") != 10000 || idTableBase("x/string.tbl") != 0 {
		t.Error("table bases")
	}
}
