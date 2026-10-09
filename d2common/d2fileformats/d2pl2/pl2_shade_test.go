package d2pl2

import (
	"os"
	"testing"
)

func TestShadeRow(t *testing.T) {
	for in, want := range map[int]int{-5: 0, 0: 0, 7: 0, 8: 1, 128: 16, 248: 31, 255: 31, 999: 31} {
		if got := ShadeRow(in); got != want {
			t.Errorf("ShadeRow(%d)=%d want %d", in, got, want)
		}
	}
}

func TestShadeFactorsSynthetic(t *testing.T) {
	p := &PL2{}
	for i := range p.BasePalette.Colors {
		p.BasePalette.Colors[i] = PL2Color{R: uint8(i), G: uint8(i), B: uint8(i)}
	}

	// row r maps index i -> i*r/31: brightness r/31 exactly (up to rounding)
	for r := 0; r < ShadeRows; r++ {
		for i := 0; i < 256; i++ {
			p.LightLevelVariations[r].Indices[i] = uint8(i * r / (ShadeRows - 1))
		}
	}

	f := p.ShadeFactors()
	lin := LinearShadeFactors()

	if f[31] != 1 || f[0] != 0 {
		t.Errorf("ends %v %v", f[0], f[31])
	}

	for r := range f {
		if d := f[r] - lin[r]; d > 0.03 || d < -0.03 {
			t.Errorf("row %d factor %.3f, linear %.3f", r, f[r], lin[r])
		}
	}
}

// TestShadeFactorsReal checks a real act PL2 (D2_PL2=path to pal.pl2).
func TestShadeFactorsReal(t *testing.T) {
	path := os.Getenv("D2_PL2")
	if path == "" {
		t.Skip("D2_PL2 not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	p, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}

	f := p.ShadeFactors()
	if f[31] < 0.99 {
		t.Errorf("row 31 should be identity, factor %v", f[31])
	}

	if f[0] > 0.1 {
		t.Errorf("row 0 should be nearly black, factor %v", f[0])
	}

	for r := 1; r < ShadeRows; r++ {
		if f[r]+1e-9 < f[r-1] {
			t.Errorf("not monotonic at %d: %v < %v", r, f[r], f[r-1])
		}
	}

	t.Logf("shade factors: %.2f", f)
}
