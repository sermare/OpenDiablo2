package d2level

import "testing"

// Numeric golden for the Durance of Hate special tiles, measured with the DRLG oracle (the real Game.exe DRLG run
// under emulation, 24 games = 8 seeds x 3 difficulties, 120 warp rooms of levels 100..102, no disagreement):
// a preset room whose special tile has style k gets the room flag 0x10<<k, DRLG_BuildRoomNeighborLinks (0x66f030)
// links each flag bit k via DRLG_LinkRoomToAdjacentLevel (0x66eed0) to Levels.txt Vis[k] / Warp[k].
// Columns: LvlPrest Def, DS1 file, the style k the oracle found, the LvlWarp id of the link, and the destination
// level of the link when the file sits in level 100 / 101 / 102 (0 = the oracle never placed it there).
func TestDuranceTileDestinations(t *testing.T) {
	type row struct {
		def   int
		file  string
		style int
		warp  int
		to100 int
		to101 int
		to102 int
	}

	for _, r := range []row{
		{784, "mephwwarpu", 3, 66, 83, 100, 0},
		{785, "mephewarpu", 2, 65, 83, 100, 0},
		{786, "mephswarpu", 3, 66, 83, 100, 0},
		{787, "mephnwarpu", 2, 65, 83, 100, 0},
		{788, "mephwwarpd", 1, 68, 101, 102, 0},
		{789, "mephewarpd", 0, 67, 101, 102, 0},
		{790, "mephswarpd", 1, 68, 101, 102, 0},
		{791, "mephnwarpd", 0, 67, 101, 102, 0},
		{796, "mephcomp", 3, 66, 0, 0, 101},
	} {
		for level, want := range map[int]int{100: r.to100, 101: r.to101, 102: r.to102} {
			if want == 0 {
				continue
			}

			got, ok := TileDestination(level, r.style)
			if !ok || got != want {
				t.Errorf("%s (def %d) in level %d: style %d -> %d %v, want %d", r.file, r.def, level, r.style, got, ok, want)
			}
		}
	}

	// the Levels.txt Vis slots {Vis0, Vis1, Vis2, Vis3} of 100, 101, 102 are the style table; styles >= 4 lead nowhere
	for level, want := range map[int][4]int{100: {101, 101, 83, 83}, 101: {102, 102, 100, 100}, 102: {0, 0, 101, 101}} {
		for style, to := range want {
			got, ok := TileDestination(level, style)
			if (to == 0) == ok || got != to {
				t.Errorf("TileDestination(%d, %d) = %d %v, want %d", level, style, got, ok, to)
			}
		}

		for _, style := range []int{4, 5, -1} {
			if got, ok := TileDestination(level, style); ok {
				t.Errorf("TileDestination(%d, %d) = %d, want none", level, style, got)
			}
		}
	}
}
