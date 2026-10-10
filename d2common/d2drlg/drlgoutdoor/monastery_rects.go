package drlgoutdoor

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// MonasteryRects returns the world rectangles of the Act 1 levels that are linked by Levels.txt Vis slots whose
// Warp id is -1 (Monastery Gate 26 -> Outer Cloister 27 -> Barracks 28, Inner Cloister 32 -> Cathedral 33), plus the
// neighbours of the Act 1 world search (Tamoe Highland 7 ...), and level 27's exit side (the Barracks joint).
//
// In the original those levels share one DRLG world: the level of a room is the level of the DrlgRoom the hero
// stands in, so a Vis link with Warp -1 is crossed by walking into the neighbouring level's rooms (the same
// mechanism as the seamless borders of the wilderness; no warp tile unit exists). The rectangles:
//   - 26 and 32 sit at their absolute Levels.txt offsets, 27 and 33 at their Depend offsets (27 (3000,960),
//     33 (3996,966), verified against the emulator by ParamsPreset);
//   - 28 is the maze result of drlgmaze with level 27's rect and exit side (DRLG_FinishBarracksLevel,
//     verified against the emulator by the drlgmaze oracle test), the world layout supplies the side.
func MonasteryRects(t *d2drlg.Tables, seed uint32, diff d2drlg.Difficulty) (rects map[int]Rect, side int, err error) {
	lay, err := drlgworld.Generate(t, seed, diff)
	if err != nil {
		return nil, -1, err
	}

	rects = map[int]Rect{}

	for id, p := range lay.Levels {
		rects[id] = Rect{p.Rect.X, p.Rect.Y, p.Rect.W, p.Rect.H}
	}

	for _, id := range []int{26, 27, 32, 33} {
		p, err := ParamsPreset(t, id, seed, diff)
		if err != nil {
			return nil, -1, err
		}

		rects[id] = p.Rect
	}

	// the world search sets level 27's exit side field only for one Black Marsh direction; the emulator's oracle
	// runs show 0 (west) otherwise (maze_act23.json: side27 absent in 24 of 60 Barracks runs)
	side = lay.BarracksExitSide
	if side < 0 {
		side = 0
	}

	r27 := rects[27]
	base, _ := d2rand.DrlgBaseSeed(seed)

	res, err := drlgmaze.Generate(t, drlgmaze.Params{LevelID: 28, Difficulty: diff, BaseSeed: base,
		L27: drlgmaze.Level27{X: r27.X, Y: r27.Y, W: r27.W, H: r27.H, Side: side}})
	if err != nil {
		return nil, side, err
	}

	rects[28] = Rect{res.RectX, res.RectY, res.RectW, res.RectH}

	return rects, side, nil
}
