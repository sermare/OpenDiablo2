package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// TestExportWorldAct checks that the act of the hero reaches the low bits of
// the active difficulty byte, including the way back to Act I.
func TestExportWorldAct(t *testing.T) {
	tests := []struct {
		name      string
		header    [3]byte
		act       int
		diff      d2enum.DifficultyType
		want      [3]byte
		wantFirst string
	}{
		{"act 1 -> 3", [3]byte{0x80, 0, 0}, 3, d2enum.DifficultyNormal, [3]byte{0x82, 0, 0}, ""},
		{"act 5 -> back to 1", [3]byte{0x84, 0, 0}, 1, d2enum.DifficultyNormal, [3]byte{0x80, 0, 0}, ""},
		{"unknown act keeps the header", [3]byte{0x83, 0, 0}, 0, d2enum.DifficultyNormal, [3]byte{0x83, 0, 0}, ""},
		{"nightmare act 2", [3]byte{0, 0x80, 0}, 2, d2enum.DifficultyNightmare, [3]byte{0, 0x81, 0}, ""},
	}

	for _, tc := range tests {
		h := &d2s.Header{Difficulty: tc.header}
		exportWorld(h, &HeroState{Act: tc.act, Difficulty: tc.diff})

		if h.Difficulty != tc.want {
			t.Errorf("%s: difficulty bytes % x, want % x", tc.name, h.Difficulty, tc.want)
		}
	}
}
