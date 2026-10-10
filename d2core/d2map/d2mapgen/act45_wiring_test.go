package d2mapgen

import "testing"

// River of Flame (107) wiring found by playing it: the bridge room at the north end carries the exit to the
// Chaos Sanctuary, the others carry none (the south room is resolved by the slot rule).
func TestMazeRoomExit(t *testing.T) {
	cases := []struct {
		level int
		file  string
		want  int
	}{
		{107, "Act4/Diab/BridgeLava.ds1", 108},
		{107, "Act4/Lava/WarpMesa.ds1", 0},
		{107, "Act4/Lava/LavaX.ds1", 0},
		{107, "Act4/Diab/Bridge2.ds1", 0},
		{113, "Act4/Diab/BridgeLava.ds1", 0},
	}

	for _, c := range cases {
		if got := mazeRoomExit(c.level, c.file); got != c.want {
			t.Errorf("mazeRoomExit(%d, %s) = %d, want %d", c.level, c.file, got, c.want)
		}
	}
}

// The maze provider reaches the Act 4 and 5 mazes (it stopped at level 72, so the River of Flame refused to load).
func TestMazeProviderRange(t *testing.T) {
	if maxMazeLevel < 135 {
		t.Errorf("maxMazeLevel = %d, the last maze of Act 5 is level 135", maxMazeLevel)
	}
}
