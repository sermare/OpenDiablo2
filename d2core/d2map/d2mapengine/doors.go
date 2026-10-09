package d2mapengine

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// trackEntity puts the footprint of a closed door into the collision overlay.
func (m *MapEngine) trackEntity(e d2interface.MapEntity) {
	ob, ok := e.(*d2mapentity.Object)
	if !ok || !ob.IsDoor() {
		return
	}

	m.syncDoor(ob)
}

func (m *MapEngine) syncDoor(ob *d2mapentity.Object) {
	if !ob.Blocking() {
		m.objBlock.Clear(ob.ID())
		return
	}

	x, y, w, h := ob.Footprint()
	m.objBlock.Set(ob.ID(), x, y, w, h)
}

// SetDoorOpen opens or closes a door and updates the collision overlay, so a
// closed door blocks walking and an open one does not. It returns whether the
// state changed.
func (m *MapEngine) SetDoorOpen(ob *d2mapentity.Object, open bool) (bool, error) {
	var (
		changed bool
		err     error
	)

	if open {
		changed, err = ob.Open()
	} else {
		changed, err = ob.Close()
	}

	m.syncDoor(ob)

	return changed, err
}

// DoorBlocks reports whether a closed door covers the sub-tile.
func (m *MapEngine) DoorBlocks(subX, subY int) bool {
	return m.objBlock.Blocked(subX, subY)
}

// WalkBlocked reports whether a sub-tile cannot be walked on: outside the map,
// blocked by the DT1 flags of its tile, or covered by a closed door.
func (m *MapEngine) WalkBlocked(subX, subY int) bool {
	if subX < 0 || subY < 0 {
		return true
	}

	if m.objBlock.Blocked(subX, subY) {
		return true
	}

	tile := m.TileAt(subX/subtilesPerTile, subY/subtilesPerTile)
	if tile == nil || subX/subtilesPerTile >= m.size.Width {
		return true
	}

	return tile.GetSubTileFlags(subX%subtilesPerTile, subY%subtilesPerTile).BlockWalk
}

// NearestWalkable searches outward from a sub-tile for the closest sub-tile
// that is not blocked, up to radius sub-tiles (the original uses a collision
// mask and a search radius of 0x32 when placing a unit after a level change).
// ok is false if nothing was found.
func (m *MapEngine) NearestWalkable(subX, subY, radius int) (x, y int, ok bool) {
	for r := 0; r <= radius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue // only the ring at distance r
				}

				if !m.WalkBlocked(subX+dx, subY+dy) {
					return subX + dx, subY + dy, true
				}
			}
		}
	}

	return 0, 0, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
