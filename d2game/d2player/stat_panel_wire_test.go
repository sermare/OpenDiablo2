package d2player

import "testing"

func TestResistLine(t *testing.T) {
	tests := []struct {
		name         string
		shown, total int
		text         string
		col          int
	}{
		{"plain", 30, 75, "30", StatFontWhite},
		{"at cap gold", 75, 75, "75", StatFontGold},
		{"over cap clamped", 80, 75, "75", StatFontGold},
		{"raised cap 90, 89 not gold", 89, 90, "89", StatFontWhite},
		{"cap 95 ceiling", 95, 95, "95", StatFontGold},
		{"negative red", -20, 75, "-20", StatFontRed},
		{"floor -100", -130, 75, "-100", StatFontRed},
		{"zero white", 0, 75, "0", StatFontWhite},
	}

	for _, tc := range tests {
		txt, col := resistLine(tc.shown, tc.total)
		if txt != tc.text || col != tc.col {
			t.Errorf("%s: got %q/%d want %q/%d", tc.name, txt, col, tc.text, tc.col)
		}
	}
}

func TestStatFontRGBA(t *testing.T) {
	if c := StatFontRGBA(StatFontWhite); c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("white %v", c)
	}

	if StatFontRGBA(StatFontRed) == StatFontRGBA(StatFontBlue) || StatFontRGBA(StatFontGold) == StatFontRGBA(StatFontWhite) {
		t.Error("colours not distinct")
	}
}
