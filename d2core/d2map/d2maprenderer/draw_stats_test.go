package d2maprenderer

import "testing"

func TestDrawStatsString(t *testing.T) {
	got := DrawStats{Floors: 3, Walls: 2, UpperWalls: 1, Shadows: 4, Roofs: 5, RoofsFading: 1, Entities: 6, LightSources: 2, Lit: true}.String()
	want := "floors=3 walls=2 upperwalls=1 shadows=4 roofs=5 roofsfading=1 wallsfading=0 entities=6 lightsources=2 lit=true"

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
