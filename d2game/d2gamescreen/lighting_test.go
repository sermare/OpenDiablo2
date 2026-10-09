package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
)

func TestParseAutoTime(t *testing.T) {
	for _, tc := range []struct {
		in         string
		phase, deg int
		ok         bool
	}{
		{"", 0, 0, false}, {"day", 2, -1, true}, {"DUSK", 4, -1, true}, {"night", 0, -1, true},
		{"3", 3, -1, true}, {"dusk@185", 4, 185, true}, {"7", 0, 0, false}, {"dusk@x", 0, 0, false},
		{"bogus", 0, 0, false},
	} {
		p, d, ok := parseAutoTime(tc.in)
		if ok != tc.ok || (ok && (p != tc.phase || d != tc.deg)) {
			t.Errorf("%q: got %d,%d,%v want %d,%d,%v", tc.in, p, d, ok, tc.phase, tc.deg, tc.ok)
		}
	}
}

func TestAutoTimeFreezesClock(t *testing.T) {
	t.Setenv("OD2_AUTOTIME", "dusk")

	c := newDayClock()
	c.Advance(100000)

	if c.Phase() != d2daynight.PhaseNight4 {
		t.Errorf("phase %d", c.Phase())
	}

	if a := c.Ambient(); a != (d2daynight.RGB{R: 0xc2, G: 0x98, B: 0xc1}) {
		t.Errorf("dusk ambient %v", a)
	}
}

func TestBaseLight(t *testing.T) {
	t.Setenv("OD2_AUTOTIME", "day")
	day := newDayClock()

	// Levels.txt colour wins (cave: intensity 0, white)
	if c := baseLight(0, 255, 255, 255, 0, 8, day); c.Intensity != 0 || c.R != 255 {
		t.Errorf("cave %v", c)
	}

	// no colour: the day/night ambient (start of day: full intensity, white)
	if c := baseLight(0, 0, 0, 0, 0, 1, day); c.Intensity != 255 || c.G != 255 {
		t.Errorf("town %v", c)
	}

	// Act 5 outdoors is capped
	if c := baseLight(0, 0, 0, 0, 4, 109, day); c.Intensity != act5MaxIntensity {
		t.Errorf("act5 %v", c)
	}

	if c := baseLight(0, 0, 0, 0, 0, levelFixedLight, day); c.Intensity != levelFixedValue || c.R != 0xf5 {
		t.Errorf("level 0x78 %v", c)
	}
}

// The Act 2 outdoors (town 40, desert 41-46) have no Levels.txt colour and follow the day/night
// ambient. They used to be looked up through the level TYPE (12, 16) and read the dark white-lit
// cave rows of levels 12 and 16 (Pit Level 1/2).
func TestAct2OutdoorsFollowDayNight(t *testing.T) {
	t.Setenv("OD2_AUTOTIME", "day")
	day := newDayClock()

	for id := 40; id <= 46; id++ {
		if c := baseLight(0, 0, 0, 0, 1, id, day); c.Intensity != 255 || c.R != 255 {
			t.Errorf("level %d: %v", id, c)
		}
	}

	// the dungeons keep the dark white-lit base from Levels.txt
	if c := baseLight(0, 255, 255, 255, 1, 55, day); c.Intensity != 0 || c.B != 255 {
		t.Errorf("tomb %v", c)
	}
}
