package d2player

import (
	"reflect"
	"testing"
)

func TestTrimAffixes(t *testing.T) {
	tests := []struct {
		name         string
		pre, suf     []string
		n            int
		wantP, wantS []string
	}{
		{"fewer than n", []string{"a"}, []string{"x"}, 4, []string{"a"}, []string{"x"}},
		{"alternates", []string{"a", "b", "c"}, []string{"x", "y", "z"}, 4, []string{"a", "b"}, []string{"x", "y"}},
		{"odd count takes the prefix first", []string{"a", "b", "c"}, []string{"x", "y", "z"}, 3, []string{"a", "b"}, []string{"x"}},
		{"only prefixes", []string{"a", "b", "c"}, nil, 2, []string{"a", "b"}, nil},
		{"only suffixes", nil, []string{"x", "y", "z"}, 2, nil, []string{"x", "y"}},
		{"one", []string{"a"}, []string{"x"}, 1, []string{"a"}, nil},
	}

	for _, tt := range tests {
		p, s := trimAffixes(tt.pre, tt.suf, tt.n)
		if !reflect.DeepEqual(p, tt.wantP) || !reflect.DeepEqual(s, tt.wantS) {
			t.Errorf("%s: got %v %v want %v %v", tt.name, p, s, tt.wantP, tt.wantS)
		}
	}
}
