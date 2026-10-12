package ebiten

import (
	"image/color"
	"testing"
)

// The burn effect multiplies the sprite into the destination: transparent pixels must become white (the
// neutral factor), opaque ones must keep their colour (w51: black boxes around the Duriel's Lair flies).
func TestBurnFillMatrix(t *testing.T) {
	m := burnFillMatrix()

	tests := []struct {
		name string
		in   color.RGBA
		want color.RGBA
	}{
		{"transparent", color.RGBA{}, color.RGBA{255, 255, 255, 255}},
		{"opaque black", color.RGBA{0, 0, 0, 255}, color.RGBA{0, 0, 0, 255}},
		{"opaque colour", color.RGBA{200, 100, 50, 255}, color.RGBA{200, 100, 50, 255}},
	}

	for _, tc := range tests {
		got := color.RGBAModel.Convert(m.Apply(tc.in)).(color.RGBA)
		if got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
