package d2mapgen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The hero's entry into a level without a fixed arrival (Levels.txt
// Position=0), read from FUN_0066dfe0 (called through FUN_0061ad90): the rooms
// are walked from the head of the room list (the last room allocated first)
// and the first match of these rules wins,
//
//  1. a room with the waypoint chunk flag (0x30000): the first waypoint object
//     (objects.txt SubClass bit 0x40) in it, else the room centre;
//  2. a room with an exit slot flag (0xff0, a Vis slot without a Warp): its
//     centre;
//  3. the room that holds the level centre;
//  4. a random room (the room-list order stands in for the roll).
//
// The result is a point that the placement then moves to the nearest free
// spot. The rule is read from the disassembly; it has no emulator golden.
const (
	roomFlagWaypoint = 0x30000
	roomFlagExitMask = 0xff0
	objectSubWaypt   = 0x40
)

// entryCand is a room as the entry rule sees it (tiles of the map).
type entryCand struct {
	x, y, w, h int
	flags      uint32
	// waypoint is the position (tile) of the first waypoint object in the room.
	waypoint *[2]int
}

// pickEntry applies the rules to cands (head of the room list first) for a
// level whose centre is (cx, cy); the tile it returns is the point the hero is
// placed next to.
func pickEntry(cands []entryCand, cx, cy int) (x, y int, how string, ok bool) {
	centre := func(c entryCand) (int, int) { return c.x + c.w/2, c.y + c.h/2 }

	for _, c := range cands {
		if c.flags&roomFlagWaypoint == 0 {
			continue
		}

		if c.waypoint != nil {
			return c.waypoint[0], c.waypoint[1], "at the waypoint", true
		}

		x, y = centre(c)

		return x, y, "in the waypoint room", true
	}

	for _, c := range cands {
		if c.flags&roomFlagExitMask != 0 {
			x, y = centre(c)

			return x, y, "in the first exit room", true
		}
	}

	for _, c := range cands {
		if cx >= c.x && cx < c.x+c.w && cy >= c.y && cy < c.y+c.h {
			x, y = centre(c)

			return x, y, "in the room at the level centre", true
		}
	}

	if len(cands) > 0 {
		x, y = centre(cands[0])

		return x, y, "in the first room", true
	}

	return 0, 0, "", false
}

// waypointObjects returns the tiles (map coordinates) of the waypoint objects
// of a stamp placed at tile (ox, oy).
func (g *MapGenerator) waypointObjects(stamp *d2mapstamp.Stamp, ox, oy int) [][2]int {
	var out [][2]int

	act := stamp.Act() - 1
	if act < 0 {
		act = 0
	}

	for _, o := range stamp.Objects() {
		if o.Type != int(d2enum.ObjectTypeItem) {
			continue
		}

		id := d2records.DS1ObjectClass(act, o.ID)
		if id < 0 {
			continue
		}

		if rec := g.asset.Records.Object.Details[id]; rec != nil && rec.SubClass&objectSubWaypt != 0 {
			out = append(out, [2]int{ox + o.X/subtilesPerTile, oy + o.Y/subtilesPerTile})
		}
	}

	return out
}

// entryCands builds the candidates of an outdoor level: its rooms, last
// allocated first, with the waypoint objects (map tiles) that fall inside.
func entryCands(lv *drlgoutdoor.Level, rect drlgoutdoor.Rect, wps [][2]int) []entryCand {
	cands := make([]entryCand, 0, len(lv.Rooms))

	for i := len(lv.Rooms) - 1; i >= 0; i-- {
		r := lv.Rooms[i]
		c := entryCand{x: r.X - rect.X, y: r.Y - rect.Y, w: r.W, h: r.H, flags: r.Flags}

		for _, w := range wps {
			if w[0] >= c.x && w[0] < c.x+c.w && w[1] >= c.y && w[1] < c.y+c.h {
				w := w
				c.waypoint = &w

				break
			}
		}

		cands = append(cands, c)
	}

	return cands
}
