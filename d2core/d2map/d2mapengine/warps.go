package d2mapengine

// WarpTile is a special tile of the loaded map that may be a stair or exit.
// DS1 files mark them with special (type 10/11) wall tiles; style 30 is the
// player start marker and is not a warp. That a warp tile's style equals the
// LvlWarp.txt id of the exit is UNVERIFIED (the notes only say the server
// looks the destination up by tile id with 0x6192c0).
type WarpTile struct {
	TileX, TileY int
	Style        int
	// Dest is the level the generator that placed the tile says it leads to
	// (0: unknown, resolve it from Style).
	Dest int
}

// startMarkerStyle is the style of the player start special tile.
const startMarkerStyle = 30

// SetWarpDestination records where the warp tile at a map tile leads, for tiles
// whose style does not say (cave entrances of the outdoor levels).
func (m *MapEngine) SetWarpDestination(tileX, tileY, level int) {
	if m.warpDest == nil {
		m.warpDest = map[[2]int]int{}
	}

	m.warpDest[[2]int{tileX, tileY}] = level
}

// WarpTiles lists the candidate warp tiles of the current map.
func (m *MapEngine) WarpTiles() []WarpTile {
	var out []WarpTile

	for y := 0; y < m.size.Height; y++ {
		for x := 0; x < m.size.Width; x++ {
			for _, w := range m.tiles[x+y*m.size.Width].Components.Walls {
				if w.Type.Special() && w.Style != startMarkerStyle {
					out = append(out, WarpTile{TileX: x, TileY: y, Style: int(w.Style), Dest: m.warpDest[[2]int{x, y}]})
				}
			}
		}
	}

	return out
}
