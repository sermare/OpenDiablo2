package d2player

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The numbers are those of UI_DrawHelpScreen 0x492b00 and its helpers (see help_layout.go).

func TestHelpLayoutKey(t *testing.T) {
	if f := HelpFrames800[2]; f.X != 66 || f.Bottom != 20 {
		t.Errorf("frame 2: %v", f)
	}

	if f := HelpFrames800[6]; f.X != 780 || f.Bottom != 512 {
		t.Errorf("frame 6: %v", f)
	}

	if f := HelpFrames640(480)[6]; f.X != 320 || f.Bottom != 432 {
		t.Errorf("640 frame 6: %v", f)
	}

	if r := Mode800.HelpCloseRect(); r != (UIRect{"help", "close", 685, 24, 32, 32}) {
		t.Errorf("close 800 %v", r)
	}

	if r := Mode640.HelpCloseRect(); r.X != 525 || r.Y != 24 {
		t.Errorf("close 640 %v", r)
	}

	if x, b := HelpLegendBullet(7); x != 104 || b != 73+140 {
		t.Errorf("bullet 7: %d %d", x, b)
	}

	if len(HelpLeaders800) != 11 || len(HelpLeaders640) != 9 {
		t.Errorf("leaders %d / %d", len(HelpLeaders800), len(HelpLeaders640))
	}

	// the Go overlay stands on the original's numbers
	if l, ok := HelpLeaderAt(lifeOrbDotX, lifeOrbDotY); !ok || l.Y1 != 486 || l.Y2 != 532 {
		t.Errorf("life orb leader %v %v", l, ok)
	}

	for _, d := range [][2]int{{newStatsDotX, newStatsDotY}, {newSkillDotX, newSkillDotY}, {leftSkillClickToChangeDotX, leftSkillClickToChangeDotY},
		{rightSkillClickToChangeDotX, rightSkillClickToChangeDotY}, {otherScreensDotX, otherScreensDotY}, {manaOrbDotX, manaOrbDotY},
		{staminaBarDotX, staminaBarDotY}, {toggleDotX, toggleDotY}, {barDotX, barDotY}, {beltDotX, beltDotY}} {
		if _, ok := HelpLeaderAt(d[0], d[1]); !ok {
			t.Errorf("no leader for go dot %v", d)
		}
	}

	if closeButtonY != 24 || closeButtonX != 685 {
		t.Errorf("go close button %d,%d", closeButtonX, closeButtonY)
	}
}

// TestHelpLayoutGolden compares the layout with testdata/help_panel.golden. Regenerate with UPDATE_GOLDEN=1 only after
// re-reading the numbers from the executable.
func TestHelpLayoutGolden(t *testing.T) {
	path := filepath.Join("testdata", "help_panel.golden")
	got := strings.Join(HelpLayoutLines(), "\n") + "\n"

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(want) != got {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
		for i := 0; i < len(wl) && i < len(gl); i++ {
			if wl[i] != gl[i] {
				t.Fatalf("golden differs at line %d: got %q want %q", i+1, gl[i], wl[i])
			}
		}

		t.Fatalf("golden has %d lines, layout %d", len(wl), len(gl))
	}
}
