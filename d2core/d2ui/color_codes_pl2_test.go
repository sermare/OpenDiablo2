package d2ui

import "testing"

// The colours of the three tokens the engine had no name for, read from the 13 text shift rows of the shipped pal.pl2
// (row 7, 10 and 11 mapped through the palette at the brightest font index; acts 1, 3, 5).
func TestColorCodeRGBFromPl2(t *testing.T) {
	for _, tc := range []struct {
		got  uint32
		want [3]uint8
	}{
		{colorTanAlpha, [3]uint8{216, 184, 100}},
		{colorDarkGreenAlpha, [3]uint8{24, 100, 8}},
		{colorPurpleAlpha, [3]uint8{164, 32, 252}},
	} {
		r, g, b := uint8(tc.got>>24), uint8(tc.got>>16), uint8(tc.got>>8)
		if [3]uint8{r, g, b} != tc.want || uint8(tc.got) != 0xff {
			t.Errorf("%#x != %v", tc.got, tc.want)
		}
	}
}
