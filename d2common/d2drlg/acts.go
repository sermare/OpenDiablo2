package d2drlg

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// ActExtras are the values DRLG_CreateDrlg (0x644210) draws from the DRLG seed
// right after the base-seed step, before any level is generated (drlg.md
// section 1). Verified against the emulated game binary (see oracle_test.go).
type ActExtras struct {
	// TombA and TombB are the two Act 2 (act index 1) special tomb level ids
	// (66..72). TombA is stored at drlg+0x94 (room count x3 in FillMazeRooms),
	// TombB at drlg+0x484 (x2).
	TombA, TombB int
	// Flip is the Act 3 (act index 2) bit stored at drlg+0x474.
	Flip int
	// Seed is the DRLG seed after the extras were drawn.
	Seed d2rand.Seed
}

// DrawActExtras derives the per-act extras for a game seed. act is the act
// index, 0..4.
func DrawActExtras(gameSeed uint32, act int) ActExtras {
	_, s := d2rand.DrlgBaseSeed(gameSeed)
	out := ActExtras{}

	switch act {
	case 1:
		for {
			r1 := s.Step() % 7
			r2 := s.Step() % 7

			if r1 != r2 {
				out.TombA, out.TombB = 66+int(r1), 66+int(r2)

				break
			}
		}
	case 2:
		out.Flip = int(s.Step() & 1)
	}

	out.Seed = *s

	return out
}
