package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestExportMercFields(t *testing.T) {
	data, tables, state := realSave(t)

	if state.Merc == nil {
		t.Skip("sample save has no mercenary")
	}

	state.Merc.Dead = true
	state.Merc.Experience += 12345

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	orig, _ := d2s.Parse(data, tables)
	m := got.Header.Mercenary

	if !m.Dead || m.Experience != orig.Header.Mercenary.Experience+12345 || m.ID != orig.Header.Mercenary.ID ||
		m.Type != orig.Header.Mercenary.Type || m.NameID != orig.Header.Mercenary.NameID {
		t.Errorf("merc = %+v (was %+v)", m, orig.Header.Mercenary)
	}

	// a replacement merc drops the old merc's items
	state.Merc = &MercState{ID: 0x1234, Type: 0, NameID: 3, Experience: 100, Replaced: true}

	out, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err = d2s.Parse(out, tables)
	if err != nil || got.Header.Mercenary.ID != 0x1234 || len(got.MercItems) != 0 {
		t.Errorf("replacement merc: err=%v header=%+v items=%d", err, got.Header.Mercenary, len(got.MercItems))
	}
}

// TestMercFromHeader pins which header contents count as a merc. The notes
// (hirelings.md) say a merc exists iff id|exp|nameId are not all zero (V by
// decompile); the engine keys on the id alone, which is equivalent for every
// real save because a hired merc always has a non-zero seed (UNVERIFIED for
// hand-edited saves; the 'jf' item section is likewise only read for id != 0).
func TestMercFromHeader(t *testing.T) {
	tests := []struct {
		name string
		in   d2s.Mercenary
		want *MercState
	}{
		{"none", d2s.Mercenary{}, nil},
		{"alive", d2s.Mercenary{ID: 7, NameID: 3, Type: 11, Experience: 100},
			&MercState{ID: 7, NameID: 3, Type: 11, Experience: 100}},
		{"dead", d2s.Mercenary{Dead: true, ID: 7, Type: 2, Experience: 1},
			&MercState{Dead: true, ID: 7, Type: 2, Experience: 1}},
	}

	for _, tc := range tests {
		got := MercFromHeader(tc.in)
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
