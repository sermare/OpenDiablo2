package d2maprenderer

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2lightmap"
)

func TestFloorSubtile(t *testing.T) {
	for _, tc := range []struct {
		name         string
		x, y         float64
		wantU, wantV float64
	}{
		{"apex", 80, 0, 0, 0},
		{"left corner", 0, 40, 0, 5},
		{"right corner", 160, 40, 5, 0},
		{"bottom corner", 80, 80, 5, 5},
		{"centre", 80, 40, 2.5, 2.5},
		{"outside clamps", -50, 200, 5, 5},
	} {
		u, v := floorSubtile(tc.x, tc.y)
		if math.Abs(u-tc.wantU) > 1e-9 || math.Abs(v-tc.wantV) > 1e-9 {
			t.Errorf("%s: (%v,%v) want (%v,%v)", tc.name, u, v, tc.wantU, tc.wantV)
		}
	}
}

func TestWallBaseSubtile(t *testing.T) {
	for _, tc := range []struct{ x, wantU, wantV float64 }{
		{0, 0, 5}, {80, 5, 5}, {160, 5, 0}, {40, 2.5, 5}, {120, 5, 2.5},
	} {
		u, v := wallBaseSubtile(tc.x)
		if math.Abs(u-tc.wantU) > 1e-9 || math.Abs(v-tc.wantV) > 1e-9 {
			t.Errorf("x=%v: (%v,%v) want (%v,%v)", tc.x, u, v, tc.wantU, tc.wantV)
		}
	}
}

func TestTintUsesShadeRow(t *testing.T) {
	l := newLighting()
	l.enabled = true

	if c := l.tint(d2lightmap.Cell{Intensity: 255, R: 255, G: 200, B: 100}); c.R != 255 || c.G != 200 || c.B != 100 {
		t.Errorf("full bright tint %v", c)
	}

	if c := l.tint(d2lightmap.Cell{Intensity: 0, R: 255, G: 255, B: 255}); c.R != 0 {
		t.Errorf("dark tint %v", c)
	}

	half := l.tint(d2lightmap.Cell{Intensity: 128, R: 255, G: 255, B: 255})
	if half.R < 120 || half.R > 140 {
		t.Errorf("half tint %v", half)
	}
}

func TestWallFadeIsLinear500ms(t *testing.T) {
	l := newLighting()
	key := wallKey{1, 2, 0}
	l.elapsed = 0.1

	// not covering and never faded: untouched
	if a := l.fadeFor(key, false); a != 1 || len(l.fades) != 0 {
		t.Fatalf("alpha %v fades %d", a, len(l.fades))
	}

	var a float64
	for i := 0; i < 5; i++ { // 5 * 100 ms = 500 ms
		a = l.fadeFor(key, true)
	}

	if math.Abs(a-wallFadeTarget) > 1e-9 {
		t.Errorf("after 500 ms alpha %v, want %v", a, wallFadeTarget)
	}

	if mid := l.fadeFor(wallKey{9, 9, 0}, true); math.Abs(mid-(1-0.1/wallFadeSeconds*(1-wallFadeTarget))) > 1e-9 {
		t.Errorf("first step alpha %v", mid)
	}

	for i := 0; i < 6; i++ {
		a = l.fadeFor(key, false)
	}

	if a != 1 || l.fades[key] != nil {
		t.Errorf("did not fade back in: %v", a)
	}
}

func TestAdvanceLightingRebuildsEachTick(t *testing.T) {
	l := newLighting()
	l.enabled = true
	mr := &MapRenderer{light: l}

	mr.SetLightInput(LightInput{HeroX: 10, HeroY: 10, Base: d2lightmap.Cell{Intensity: 50, R: 255, G: 255, B: 255}})
	mr.advanceLighting(0.05)

	if l.lm.OriginX != 10*subtilesPerTile-24 {
		t.Fatalf("origin %d", l.lm.OriginX)
	}

	c := l.lm.Sample(10*subtilesPerTile, 10*subtilesPerTile)
	if c.Intensity <= 50 {
		t.Errorf("hero light missing: %v", c)
	}

	if far := l.lm.Sample(10*subtilesPerTile+20, 10*subtilesPerTile); far.Intensity != 50 {
		t.Errorf("far cell %v should be base", far)
	}
}

// A level change must not keep the previous level's tile images or wall fades.
func TestResetLevelCaches(t *testing.T) {
	mr := &MapRenderer{light: newLighting()}
	mr.setImageCacheRecord(0, 1, 0, 0, nil)
	mr.blankShadows = map[uint32]bool{1: true}
	mr.light.fades[wallKey{1, 2, 3}] = &wallFade{alpha: 0.5}

	mr.resetLevelCaches()

	if mr.getImageCacheRecord(0, 1, 0, 0) != nil || len(mr.imageCacheRecords) != 0 {
		t.Error("image cache survived a level change")
	}

	if len(mr.blankShadows) != 0 || len(mr.light.fades) != 0 {
		t.Error("shadow or fade state survived a level change")
	}

	(&MapRenderer{}).resetLevelCaches() // no lighting yet must not panic
}
