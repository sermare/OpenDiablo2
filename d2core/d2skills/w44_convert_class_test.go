package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestConvertRefusedByClass(t *testing.T) {
	stats := d2records.MonStats{
		"imp1":   {Key: "imp1", ID: convertRefusedBaseClass},
		"zombie": {Key: "zombie", ID: 3},
	}

	cases := []struct {
		name  string
		base  string
		state bool
		want  bool
	}{
		{"imp family with the state", "imp1", true, true},
		{"case is ignored", "IMP1", true, true},
		{"imp family without the state", "imp1", false, false},
		{"other class with the state", "zombie", true, false},
		{"unknown class", "nope", true, false},
	}

	for _, c := range cases {
		if got := convertRefusedByClass(stats, c.base, c.state); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}

	if convertRefusedByClass(nil, "imp1", true) {
		t.Error("empty table must not refuse")
	}
}
