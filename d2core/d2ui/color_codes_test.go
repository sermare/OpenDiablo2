package d2ui

import "testing"

func TestConvertColorCodes(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", "Mana Cost: ", "Mana Cost: "},
		{"leading code", "\xffc3+10 to Strength", "[blue]+10 to Strength"},
		{"text before code is white", "Fire: \xffc1hot", "[white]Fire: [red]hot"},
		{"two codes", "\xffc4Unique\xffc0 base", "[gold]Unique[white] base"},
		{"utf8 form", "ÿc9Rare", "[yellow]Rare"},
		{"every selector", "\xffc0a\xffc1b\xffc2c\xffc3d\xffc4e\xffc5f\xffc6g\xffc7h\xffc8i\xffc9j\xffc:k\xffc;l",
			"[white]a[red]b[green]c[blue]d[gold]e[grey]f[black]g[tan]h[orange]i[yellow]j[darkgreen]k[purple]l"},
		{"unknown selector dropped", "\xffcZtext", "[white]text"},
		{"trailing intro", "abc\xffc", "[white]abc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ConvertColorCodes(tc.in); got != tc.want {
				t.Fatalf("ConvertColorCodes(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Every token the code table produces must have an RGB the label can draw.
func TestColorCodeTokensHaveColors(t *testing.T) {
	for sel, tok := range ColorCodeTokens {
		if getColor(tok) == nil {
			t.Errorf("selector %q -> %s has no colour", sel, tok)
		}
	}
}
