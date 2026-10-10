package d2util

import (
	"reflect"
	"testing"
)

func TestSplitIntoLinesWithMaxWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want []string
	}{
		{"latin", "aaa bbb ccc", 8, []string{" aaa bbb", "ccc"}},
		// decoded latin-1 glyph codes (bytes >= 0x80) are one column each, not two
		{"accents", "äää ööö üüü", 8, []string{" äää ööö", "üüü"}},
		// double byte glyphs count as two columns and are cut between glyphs
		{"cjk", "芠芢芤芦芨", 4, []string{"芠芢", "芤芦", "芨"}},
		{"longword", "abcdefgh", 3, []string{"abc", "def", "gh"}},
	}

	for _, tt := range tests {
		if got := SplitIntoLinesWithMaxWidth(tt.in, tt.max); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}
