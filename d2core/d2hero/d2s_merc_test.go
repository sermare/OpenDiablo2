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
