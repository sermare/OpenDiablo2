package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestZoneChangeText(t *testing.T) {
	rec := &d2records.LevelDetailRecord{LevelDisplayName: "Blood Moor"}

	for _, tc := range []struct {
		name      string
		last, cur int
		d         *d2records.LevelDetailRecord
		want      string
		ok        bool
	}{
		{"first entry silent", 0, 2, rec, "", false},
		{"same level silent", 2, 2, rec, "", false},
		{"level change", 1, 2, rec, "Entering The Blood Moor", true},
		{"missing row silent", 1, 2, nil, "", false},
	} {
		got, ok := zoneChangeText(tc.last, tc.cur, tc.d)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: got %q,%v", tc.name, got, ok)
		}
	}
}
