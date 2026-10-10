package d2mapgen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// isMonasteryLevel reports the levels joined by Levels.txt Vis links with Warp -1 (no warp tile): Monastery Gate 26,
// Outer Cloister 27, Barracks 28, Inner Cloister 32, Cathedral 33. They share a world and are crossed on their
// borders (drlgoutdoor.MonasteryRects).
func isMonasteryLevel(id int) bool {
	switch id {
	case 26, 27, 28, 32, 33:
		return true
	}

	return false
}

// monasteryWorld returns the rectangles of the Monastery levels (and the Act 1 wilderness neighbours) for the
// edge crossing, and level 27's exit side (the Barracks joint; 0 when the world search did not set it, as the
// emulator reports it).
func monasteryWorld(tb *d2drlg.Tables, seed uint32, diff d2drlg.Difficulty) (map[int]d2level.Rect, int, error) {
	rs, side, err := drlgoutdoor.MonasteryRects(tb, seed, diff)
	if err != nil {
		return nil, 0, err
	}

	out := make(map[int]d2level.Rect, len(rs))
	for id, r := range rs {
		out[id] = d2level.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	}

	return out, side, nil
}
