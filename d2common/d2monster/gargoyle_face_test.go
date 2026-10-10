package d2monster

import "testing"

func TestGargoyleFacePoint(t *testing.T) {
	tests := []struct {
		name           string
		ox, oy, tx, ty int
		wx, wy         int
	}{
		{"mostly vertical", 100, 100, 103, 110, 100, 110},
		{"mostly horizontal", 100, 100, 110, 103, 110, 100},
		{"tie goes horizontal", 100, 100, 105, 105, 105, 100},
		{"negative", 100, 100, 98, 90, 100, 90},
	}

	for _, tc := range tests {
		if x, y := GargoyleFacePoint(tc.ox, tc.oy, tc.tx, tc.ty); x != tc.wx || y != tc.wy {
			t.Errorf("%s: got (%d,%d) want (%d,%d)", tc.name, x, y, tc.wx, tc.wy)
		}
	}
}

func TestGargoyleFacingByte(t *testing.T) {
	tests := []struct{ dir, want int }{
		{0, 0x1f}, {8, 0x1f}, {9, 0x31}, {24, 0x31}, {25, 0x00}, {40, 0x00}, {41, 0x11}, {56, 0x11}, {57, 0x1f}, {63, 0x1f}, {-1, 0x1f},
	}

	for _, tc := range tests {
		if got := GargoyleFacingByte(tc.dir); got != tc.want {
			t.Errorf("dir %d: got %#x want %#x", tc.dir, got, tc.want)
		}
	}
}

type facerWorld struct {
	*faWorld
	calls [][3]int
	dir   int
}

func (w *facerWorld) FaceTowards(_ *Brain, x, y, f int) { w.calls = append(w.calls, [3]int{x, y, f}) }
func (w *facerWorld) Direction64(*Brain, int, int) int  { return w.dir }

func TestGargoyleThinkFaces(t *testing.T) {
	b, p := faBrain("GargoyleTrap", 0, []int{0}, 20, 100, 7, 3)
	_ = p

	w := &facerWorld{faWorld: newFA(6, false), dir: 30}
	tg := Target{X: 103, Y: 110}
	c := &Ctx{B: b, W: w}
	c.Target = &tg
	c.gargoyleFace(tg)

	if len(w.calls) != 1 || w.calls[0] != [3]int{100, 110, 0x00} {
		t.Errorf("calls = %v", w.calls)
	}
}
