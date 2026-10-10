package d2level

import "testing"

// Kurast Bazaar (80), Upper Kurast (81) and the Causeway (82) each have two temples behind the same LvlWarp id: the
// temple presets stand for the first and second link; only their entrance tiles (styles 2 and 3) are exits.
func TestTempleEntranceByPreset(t *testing.T) {
	cases := []struct {
		level int
		path  string
		style int
		want  int
	}{
		{80, "Act3/Kurast/BurbsTemple2.ds1", 2, 94},
		{80, "Act3/Kurast/BurbsTemple3.ds1", 3, 95},
		{81, "Act3/Kurast/BurbsTemple2.ds1", 2, 96},
		{81, "Act3/Kurast/BurbsTemple3.ds1", 3, 97},
		{82, "Act3/Kurast/BurbsTemple2.ds1", 2, 98},
		{82, "Act3/Kurast/BurbsTemple3.ds1", 3, 99},
		{80, "Act3/Kurast/BurbsTemple2.ds1", 8, 0}, // a door of the preset, not the entrance
		{80, "Act3/Kurast/Burbs16x16_2.ds1", 2, 0}, // not a temple preset
		{79, "Act3/Kurast/BurbsTemple2.ds1", 2, 0}, // Lower Kurast has no temple
	}

	for _, c := range cases {
		got, ok := TempleEntranceByPreset(c.level, c.path, c.style)
		if got != c.want || ok != (c.want != 0) {
			t.Errorf("TempleEntranceByPreset(%d, %s, %d) = %d, %v; want %d", c.level, c.path, c.style, got, ok, c.want)
		}
	}
}
