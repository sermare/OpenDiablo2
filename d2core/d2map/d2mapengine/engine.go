package d2mapengine

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

const (
	logPrefix = "Map Engine"
)

// MapEngine loads the tiles which make up the isometric map and the entities
type MapEngine struct {
	asset *d2asset.AssetManager
	*d2mapstamp.StampFactory
	*d2mapentity.MapEntityFactory
	seed          int64                            // The map seed
	entities      map[string]d2interface.MapEntity // Entities on the map
	tiles         []MapTile
	size          d2geom.Size               // Size of the map, in tiles
	levelType     d2records.LevelTypeRecord // Level type of this map
	dt1TileData   []d2dt1.Tile              // DT1 tile data
	startSubTileX int                       // Starting X position
	startSubTileY int                       // Starting Y position
	dt1Files      []string                  // List of DS1 strings
	dt1Starts     []int                     // index into dt1TileData of the first tile of each dt1Files entry
	dt1Rank       []int                     // for every dt1TileData entry: its position among the tiles of its (style, sequence, type)
	dt1KeyCount   map[[3]int32]int          // tiles seen per (style, sequence, type) while filling dt1Rank
	objBlock      *ObjectCollision          // run-time collision of doors

	gridPaths     bool        // PathFind uses d2path (see UseCollisionPaths)
	startOverride *[2]float64 // hero spawn tile set by a level generator (see SetStartPosition)
	world         World       // where this map sits in the Act 1 world (see SetWorld)
	warpDest      map[[2]int]int

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/789
	IsLoading bool // (temp) Whether we have processed the GenerateMapPacket(only for remote client)

	*d2util.Logger
}

const (
	subtilesPerTile = 5
)

// CreateMapEngine creates a new instance of the map engine and returns a pointer to it.
func CreateMapEngine(l d2util.LogLevel, asset *d2asset.AssetManager) *MapEngine {
	entity, _ := d2mapentity.NewMapEntityFactory(asset)
	stamp := d2mapstamp.NewStampFactory(asset, l, entity)

	engine := &MapEngine{
		asset:            asset,
		MapEntityFactory: entity,
		StampFactory:     stamp,
		// This will be set to true when we are using a remote client connection, and then set to false after we process the GenerateMapPacket
		IsLoading: false,
		objBlock:  NewObjectCollision(),
	}

	engine.Logger = d2util.NewLogger()
	engine.Logger.SetLevel(l)
	engine.Logger.SetPrefix(logPrefix)

	return engine
}

// GetStartingPosition returns the starting position on the map in sub-tiles.
func (m *MapEngine) GetStartingPosition() (x, y int) {
	return m.startSubTileX, m.startSubTileY
}

// SetStartPosition fixes the hero spawn point (tile units, tile centre) for
// generators that know where the level is entered (see d2mapgen real maps).
func (m *MapEngine) SetStartPosition(x, y float64) {
	m.startOverride = &[2]float64{x, y}
}

// World places the loaded map in the Act 1 world: world tile = map tile +
// (OriginX, OriginY), and Rects are the world rectangles of the levels that
// can border it (d2level.EdgeExit). Level 0 means the map has no place in the
// world (the old generators, mazes).
type World struct {
	Level            int
	OriginX, OriginY int
	Rects            map[int]d2level.Rect
}

// SetWorld records where the map sits in the world; generators of seamless
// outdoor levels call it after ResetMap.
func (m *MapEngine) SetWorld(w World) { m.world = w }

// World returns the world placement of the map (Level 0 when it has none).
func (m *MapEngine) World() World { return m.world }

// ResetMap clears all map and entity data and reloads it from the cached files.
func (m *MapEngine) ResetMap(levelType d2enum.RegionIdType, width, height int) {
	m.entities = make(map[string]d2interface.MapEntity)
	m.objBlock = NewObjectCollision()
	m.levelType = *m.asset.Records.Level.Types[levelType]
	m.size = d2geom.Size{Width: width, Height: height}
	m.tiles = make([]MapTile, width*height)
	m.dt1TileData = make([]d2dt1.Tile, 0)
	m.dt1Files = make([]string, 0)
	m.dt1Starts = make([]int, 0)
	m.dt1Rank = nil
	m.dt1KeyCount = map[[3]int32]int{}
	m.startOverride = nil
	m.world = World{}
	m.warpDest = nil
	m.gridPaths = false

	for idx := range m.levelType.Files {
		m.addDT1(m.levelType.Files[idx])
	}
}

func (m *MapEngine) addDT1(fileName string) {
	if fileName == "" || fileName == "0" {
		return
	}

	fileName = strings.ToLower(fileName)
	for i := 0; i < len(m.dt1Files); i++ {
		if m.dt1Files[i] == fileName {
			return
		}
	}

	dt1, err := m.asset.LoadDT1(fileName)
	if err != nil {
		m.Error(err.Error())
	}

	m.dt1Starts = append(m.dt1Starts, len(m.dt1TileData))
	m.dt1TileData = append(m.dt1TileData, dt1.Tiles...)
	m.extendRank()
	m.dt1Files = append(m.dt1Files, fileName)
}

// AddDS1 loads DT1 files and performs string replacements on them. It
// appends the tile data and files to MapEngine.dt1TileData and
// MapEngine.dt1Files.
func (m *MapEngine) AddDS1(fileName string) {
	if fileName == "" || fileName == "0" {
		return
	}

	ds1, err := m.asset.LoadDS1(fileName)
	if err != nil {
		m.Fatalf("Loading ds1: %v", err)
	}

	for idx := range ds1.Files {
		dt1File := ds1.Files[idx]
		dt1File = strings.ToLower(dt1File)
		dt1File = strings.ReplaceAll(dt1File, "c:", "")       // Yes they did...
		dt1File = strings.ReplaceAll(dt1File, ".tg1", ".dt1") // Yes they did...
		dt1File = strings.ReplaceAll(dt1File, "\\d2\\data\\global\\tiles\\", "")
		m.addDT1(strings.ReplaceAll(dt1File, "\\", "/"))
	}
}

// LevelType returns the level type of this map.
func (m *MapEngine) LevelType() d2records.LevelTypeRecord {
	return m.levelType
}

// SetSeed sets the seed of the map for generation.
func (m *MapEngine) SetSeed(seed int64) {
	m.Infof("Setting map engine seed to %d", seed)
	m.seed = seed
}

// Size returns the size of the map in sub-tiles.
func (m *MapEngine) Size() d2geom.Size {
	return m.size
}

// Tile returns the TileRecord containing the data
// for a single map tile.
func (m *MapEngine) Tile(x, y int) *MapTile {
	return &m.tiles[x+(y*m.size.Width)]
}

// Tiles returns a pointer to a slice contaning all
// map tile data.
func (m *MapEngine) Tiles() *[]MapTile {
	return &m.tiles
}

// PlaceStamp places a map stamp at the specified location, creating both entities
// and tiles. Stamps are pre-defined map areas, see d2mapstamp.
func (m *MapEngine) PlaceStamp(stamp *d2mapstamp.Stamp, tileOffsetX, tileOffsetY int) {
	stampSize := stamp.Size()

	mapW := m.size.Width
	mapH := m.size.Height

	xMax := tileOffsetX + stampSize.Width
	yMax := tileOffsetY + stampSize.Height

	if (tileOffsetX < 0) || (tileOffsetY < 0) || (xMax > mapW) || (yMax > mapH) {
		panic("Tried placing a stamp outside the bounds of the map")
	}

	// Copy over the map tile data
	for y := 0; y < stampSize.Height; y++ {
		for x := 0; x < stampSize.Width; x++ {
			targetTileIndex := m.tileCoordinateToIndex(x+tileOffsetX, y+tileOffsetY)
			stampTile := *stamp.Tile(x, y)
			m.tiles[targetTileIndex].RegionType = stamp.RegionID()
			m.tiles[targetTileIndex].Components = stampTile
			m.tiles[targetTileIndex].PrepareTile(x, y, m)
		}
	}

	// Copy over the entities
	stampEntities := stamp.Entities(tileOffsetX, tileOffsetY)
	for idx := range stampEntities {
		e := stampEntities[idx]
		m.entities[e.ID()] = e
		m.trackEntity(e)
	}
}

// PlaceStampClipped places only the top-left w x h tiles of a stamp (DS1 files
// are one tile row/column larger than the preset room they belong to, and the
// extra edge must not overwrite the neighbouring room). Entities outside the
// clipped area are dropped. Tile variants are chosen from map coordinates so
// that equal rooms do not repeat the same pattern. Tiles that fall outside
// the map are ignored.
func (m *MapEngine) PlaceStampClipped(stamp *d2mapstamp.Stamp, tileOffsetX, tileOffsetY, w, h int) {
	size := stamp.Size()
	if w > size.Width {
		w = size.Width
	}

	if h > size.Height {
		h = size.Height
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mx, my := x+tileOffsetX, y+tileOffsetY
			if mx < 0 || my < 0 || mx >= m.size.Width || my >= m.size.Height {
				continue
			}

			t := &m.tiles[m.tileCoordinateToIndex(mx, my)]
			t.RegionType = stamp.RegionID()
			t.Components = *stamp.Tile(x, y)
			t.PrepareTile(mx, my, m)
		}
	}

	for _, e := range stamp.Entities(tileOffsetX, tileOffsetY) {
		pos := e.GetPosition()
		tx, ty := int(pos.X())/subtilesPerTile, int(pos.Y())/subtilesPerTile

		if tx < tileOffsetX || ty < tileOffsetY || tx >= tileOffsetX+w || ty >= tileOffsetY+h {
			continue
		}

		m.entities[e.ID()] = e
	}
}

// SetTile puts one tile that a level generator built itself (not copied from a
// DS1 stamp) on the map and resolves its graphics and sub-tile flags. Tiles
// outside the map are ignored.
func (m *MapEngine) SetTile(x, y int, region d2enum.RegionIdType, c d2mapstamp.Tile) {
	if x < 0 || y < 0 || x >= m.size.Width || y >= m.size.Height {
		return
	}

	t := &m.tiles[m.tileCoordinateToIndex(x, y)]
	*t = MapTile{RegionType: region, Components: c}
	t.PrepareTile(x, y, m)
}

// BlockEmptyTiles marks every sub-tile of tiles that have no visible floor
// as blocked for walking and sight. In the original game void (no floor) is
// not walkable; without this the hero and monsters could leave a cave level
// through the black space between rooms. Only level generators for
// enclosed dungeon levels call it (the town keeps its old behaviour).
func (m *MapEngine) BlockEmptyTiles() {
	for i := range m.tiles {
		t := &m.tiles[i]
		if hasFloor(t) {
			continue
		}

		for s := range t.SubTiles {
			t.SubTiles[s].BlockWalk = true
			t.SubTiles[s].BlockPlayerWalk = true
			t.SubTiles[s].BlockLOS = true
		}
	}
}

func hasFloor(t *MapTile) bool {
	for i := range t.Components.Floors {
		f := &t.Components.Floors[i]
		if !f.Hidden() && f.Prop1 != 0 {
			return true
		}
	}

	return false
}

// converts x,y tile coordinate into index in MapEngine.tiles
func (m *MapEngine) tileCoordinateToIndex(x, y int) int {
	return x + (y * m.size.Width)
}

// SubTileAt gets the flags for the given subtile
func (m *MapEngine) SubTileAt(subX, subY int) *d2dt1.SubTileFlags {
	if subX < 0 || subY < 0 || subX/subtilesPerTile >= m.size.Width || subY/subtilesPerTile >= m.size.Height {
		// outside the map is a wall
		return &d2dt1.SubTileFlags{BlockWalk: true, BlockPlayerWalk: true, BlockLOS: true}
	}

	tile := m.TileAt(subX/subtilesPerTile, subY/subtilesPerTile)

	return tile.GetSubTileFlags(subX%subtilesPerTile, subY%subtilesPerTile)
}

// TileAt returns a pointer to the data for the map tile at the given
// x and y index.
func (m *MapEngine) TileAt(tileX, tileY int) *MapTile {
	idx := m.tileCoordinateToIndex(tileX, tileY)
	if idx < 0 || idx >= len(m.tiles) {
		return nil
	}

	return &m.tiles[idx]
}

// Entities returns a pointer a slice of all map entities.
func (m *MapEngine) Entities() map[string]d2interface.MapEntity {
	return m.entities
}

// Seed returns the map generation seed.
func (m *MapEngine) Seed() int64 {
	return m.seed
}

// AddEntity adds an entity to a slice containing all entities.
func (m *MapEngine) AddEntity(entity d2interface.MapEntity) {
	m.entities[entity.ID()] = entity
	m.trackEntity(entity)
}

// RemoveEntity removes an entity from the map engine
func (m *MapEngine) RemoveEntity(entity d2interface.MapEntity) {
	if entity == nil {
		return
	}

	delete(m.entities, entity.ID())
	m.objBlock.Clear(entity.ID())
}

// GetTiles returns a slice of all tiles matching the given style,
// sequence and tileType.
func (m *MapEngine) GetTiles(style, sequence int, tileType d2enum.TileType) []d2dt1.Tile {
	tiles := make([]d2dt1.Tile, 0)

	for idx := range m.dt1TileData {
		if m.dt1TileData[idx].Style != int32(style) || m.dt1TileData[idx].Sequence != int32(sequence) ||
			m.dt1TileData[idx].Type != int32(tileType) {
			continue
		}

		tiles = append(tiles, m.dt1TileData[idx])
	}

	// special tiles (type 10: start position, warp and other markers) are logical markers that
	// have no graphics in the area's dt1 files (they only exist in e.g. warp.dt1), so their
	// absence is expected and not worth a warning
	if len(tiles) == 0 {
		if tileType.Special() {
			return nil
		}

		m.Warningf("Unknown tile ID [%d %d %d]", style, sequence, tileType)
		return nil
	}

	return tiles
}

// GetStartPosition returns the spawn point on entering the current map.
func (m *MapEngine) GetStartPosition() (x, y float64) {
	if m.startOverride != nil {
		return m.startOverride[0], m.startOverride[1]
	}

	for tileY := 0; tileY < m.size.Height; tileY++ {
		for tileX := 0; tileX < m.size.Width; tileX++ {
			tile := m.tiles[tileX+(tileY*m.size.Width)].Components
			for idx := range tile.Walls {
				if tile.Walls[idx].Type.Special() && tile.Walls[idx].Style == 30 {
					// nolint:gomnd // constant
					return float64(tileX) + 0.5, float64(tileY) + 0.5
				}
			}
		}
	}

	return m.GetCenterPosition()
}

// GetCenterPosition returns the center point of the map.
func (m *MapEngine) GetCenterPosition() (x, y float64) {
	// nolint:gomnd // half of size
	return float64(m.size.Width) / 2.0, float64(m.size.Height) / 2.0
}

// Advance calls the Advance() method for all entities,
// processing a single tick.
func (m *MapEngine) Advance(tickTime float64) {
	if m.IsLoading {
		// https://github.com/OpenDiablo2/OpenDiablo2/issues/789
		return
	}

	for ID := range m.entities {
		m.entities[ID].Advance(tickTime)
	}
}

// TileExists returns true if the tile at the given coordinates exists.
func (m *MapEngine) TileExists(tileX, tileY int) bool {
	tileIndex := m.tileCoordinateToIndex(tileX, tileY)

	if valid := (tileIndex >= 0) && (tileIndex <= len(m.tiles)); valid {
		tile := m.tiles[tileIndex].Components
		numFeatures := len(tile.Floors)
		numFeatures += len(tile.Shadows)
		numFeatures += len(tile.Walls)
		numFeatures += len(tile.Substitutions)

		return numFeatures > 0
	}

	return false
}

// GenerateMap clears the map and places the specified stamp.
func (m *MapEngine) GenerateMap(regionType d2enum.RegionIdType, levelPreset, fileIndex int) {
	region := m.LoadStamp(regionType, levelPreset, fileIndex)
	regionSize := region.Size()
	m.ResetMap(regionType, regionSize.Width, regionSize.Height)
	m.PlaceStamp(region, 0, 0)
}

// GetTileData returns the tile with the given style, sequence, tileType and index.
func (m *MapEngine) GetTileData(style, sequence int, tileType d2enum.TileType, index byte) *d2dt1.Tile {
	for idx := range m.dt1TileData {
		if m.dt1TileData[idx].Style == int32(style) && m.dt1TileData[idx].Sequence == int32(sequence) &&
			m.dt1TileData[idx].Type == int32(tileType) && m.dt1TileData[idx].RarityFrameIndex == int32(index) {
			return &m.dt1TileData[idx]
		}
	}

	return nil
}
