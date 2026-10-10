package d2mapgen

import "testing"

// Sewers of Lut Gholein: the down stairs room leads to the next sewer level, the dock room of level 1 back to town.
func TestSewerRoomExits(t *testing.T) {
	cases := []struct {
		level int
		file  string
		want  int
	}{
		{47, "Act2/Sewer/SewSDown.ds1", 48},
		{48, "Act2/Sewer/SewSDown.ds1", 49},
		{49, "Act2/Sewer/SewSDown.ds1", 0},
		{47, "Act2/Sewer/SewNSDock.ds1", 40},
		{48, "Act2/Sewer/SewNSDock.ds1", 0},
		{47, "Act2/Sewer/SewEUp.ds1", 0},
	}

	for _, c := range cases {
		if got := mazeRoomExit(c.level, c.file); got != c.want {
			t.Errorf("mazeRoomExit(%d, %s) = %d, want %d", c.level, c.file, got, c.want)
		}
	}
}
