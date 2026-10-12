package d2mapengine

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"

// FloorSurface is the footstep surface (1 dirt, 2 inside stone, 3 outside stone, 4 sand, 5 snow, 6 wood; 0 none) of
// the floor under the sub-tile (subX, subY), from the material flags of its first visible floor tile
// (UNITSND_FindFloorTileAtPos 0x4c77a0 and UNITSND_GetFootstepSurfaceType 0x4c7820, VERIFIED order of the tests; that
// the exe's tile record flag word is the DT1 material word is inferred from the flag values, UNVERIFIED). The caller
// falls back to the SoundEnviron Material 1 column when this returns 0.
func (m *MapEngine) FloorSurface(subX, subY int) int {
	if subX < 0 || subY < 0 {
		return 0
	}

	c := m.TileAt(subX/subtilesPerTile, subY/subtilesPerTile)
	if c == nil {
		return 0
	}

	for i := range c.Components.Floors {
		f := &c.Components.Floors[i]
		if f.Hidden() || f.Prop1 == 0 {
			continue
		}

		opts := m.GetTiles(int(f.Style), int(f.Sequence), 0)
		if len(opts) == 0 {
			continue
		}

		idx := int(f.RandomIndex)
		if idx >= len(opts) {
			idx = 0
		}

		var mat d2dt1.MaterialFlags = opts[idx].MaterialFlags

		return mat.FootstepSurface()
	}

	return 0
}
