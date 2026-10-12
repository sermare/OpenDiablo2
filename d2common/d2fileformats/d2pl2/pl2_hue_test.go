package d2pl2

import "testing"

func grayPL2() *PL2 {
	p := &PL2{}

	for i := range p.BasePalette.Colors {
		v := uint8(i)
		p.BasePalette.Colors[i] = PL2Color{R: v, G: v, B: v}
	}

	return p
}

func TestHueTint(t *testing.T) {
	p := grayPL2()

	// palette entries 200.. are blue-ish; row 1 sends every index to 220
	for i := 200; i < 256; i++ {
		p.BasePalette.Colors[i] = PL2Color{R: 40, G: 90, B: 255}
	}

	for i := 0; i < 256; i++ {
		p.HueVariations[1].Indices[i] = 220
	}

	r, g, b, ok := p.HueTint(1)
	if !ok || b != 1 || !(r < g && g < b) {
		t.Fatalf("blue row: %v %v %v %v", r, g, b, ok)
	}

	if r < 0.2 {
		t.Fatalf("floor broken %v", r)
	}

	// an all-zero row maps everything to black: not usable
	r, g, b, ok = grayPL2().HueTint(0)
	if ok || r != 1 || g != 1 || b != 1 {
		t.Fatalf("degenerate row must be refused: %v %v %v %v", r, g, b, ok)
	}

	for _, s := range []int{-1, HueRows} {
		if _, _, _, ok := p.HueTint(s); ok {
			t.Fatalf("shift %d accepted", s)
		}
	}

	var nilp *PL2
	if _, _, _, ok := nilp.HueTint(1); ok {
		t.Fatal("nil PL2 accepted")
	}
}

func TestHueTintIdentity(t *testing.T) {
	p := grayPL2()
	for i := 0; i < 256; i++ {
		p.HueVariations[5].Indices[i] = uint8(i)
	}

	r, g, b, ok := p.HueTint(5)
	if !ok || r != 1 || g != 1 || b != 1 {
		t.Fatalf("identity row: %v %v %v %v", r, g, b, ok)
	}
}
