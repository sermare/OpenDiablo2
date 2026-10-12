package d2player

import (
	"image/color"
	"testing"
)

// Golden values: the marker colours are the entries of the Act 1 palette nearest to the RGB triples of
// AUTOMAP_InitMarkerPaletteColors 0x455d30, by the metric of D2CMP_FindNearestPaletteIndex 0x604300 (squared RGB
// distance, first minimum wins). They were computed from the game's palette once; no game data is read here.
func TestAutomapMarkerColorsGolden(t *testing.T) {
	cases := []struct {
		name string
		got  color.RGBA
		want [3]uint8
	}{
		{"self (0,0,255)", automapColorSelf, [3]uint8{36, 96, 216}},
		{"other (255,0,0)", automapColorOther, [3]uint8{252, 44, 0}},
		{"corpse (255,0,255)", automapColorCorpse, [3]uint8{164, 32, 252}},
		{"party (0,255,0)", automapColorParty, [3]uint8{24, 252, 0}},
		{"minion (0x44,0x70,0x74)", automapColorMinion, [3]uint8{68, 112, 116}},
		{"npc (0xf4,0xf4,0xf4)", automapColorNPC, [3]uint8{244, 244, 244}},
		{"portal (0xf4,0xf4,0)", automapColorPortal, [3]uint8{244, 192, 76}},
	}

	for _, c := range cases {
		if got := [3]uint8{c.got.R, c.got.G, c.got.B}; got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestAutomapActColors(t *testing.T) {
	for act, want := range map[int][3]uint8{1: {164, 32, 252}, 2: {252, 20, 248}, 3: {164, 32, 252}, 4: {164, 32, 252}, 5: {164, 32, 252}} {
		c := automapCorpseColor(act)
		if got := [3]uint8{c.R, c.G, c.B}; got != want {
			t.Errorf("corpse act %d: %v want %v", act, got, want)
		}
	}

	for act, want := range map[int][3]uint8{1: {244, 192, 76}, 2: {244, 192, 76}, 3: {244, 192, 76}, 4: {252, 224, 52}, 5: {244, 192, 76}} {
		c := automapPortalColor(act)
		if got := [3]uint8{c.R, c.G, c.B}; got != want {
			t.Errorf("portal act %d: %v want %v", act, got, want)
		}
	}
}
