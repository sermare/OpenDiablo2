package d2asset

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2tbl"
)

type stringer string

func (s stringer) String() string { return string(s) }

func TestTranslateString(t *testing.T) {
	patch := d2tbl.TextDictionary{"shared": "patch", "#121": "HELL", "onlyPatch": "p"}
	expansion := d2tbl.TextDictionary{"shared": "expansion", "onlyExp": "e", "#803": "EXPANSION CHARACTER"}
	base := d2tbl.TextDictionary{"shared": "base", "onlyBase": "b", "#1612": "CANCEL", "#1613": "COPYRIGHT"}

	// loadStrings in d2app loads patch, expansion, base, in that order.
	am := &AssetManager{tables: []d2tbl.TextDictionary{patch, expansion, base}}

	tests := []struct {
		name  string
		input interface{}
		mod   int
		want  string
	}{
		{"patch over expansion over base", "shared", 0, "patch"},
		{"expansion over base", "onlyExp", 0, "e"},
		{"base only", "onlyBase", 0, "b"},
		{"missing key falls back to the key itself", "nope", 0, "nope"},
		{"missing numeric key falls back to the key", "#99999", 0, "#99999"},
		{"empty key", "", 0, ""},
		{"keys are case sensitive", "SHARED", 0, "SHARED"},
		{"stringer", stringer("onlyBase"), 0, "b"},
		{"enum label", d2enum.CancelLabel, 0, "CANCEL"},
		{"enum label with locale modifier", d2enum.CancelLabel - 1, 1, "CANCEL"},
		{"enum label index past the table does not panic", 1000, 0, "#-1"},
		{"negative enum index does not panic", -5, 0, "#-1"},
		{"unsupported input type", 1.5, 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			am.languageModifier = tc.mod

			if got := am.TranslateString(tc.input); got != tc.want {
				t.Fatalf("TranslateString(%v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
