package d2mapstamp

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	subtilesPerTile = 5
)

// Stamp represents a pre-fabricated map stamp that can be placed on a map.
type Stamp struct {
	factory     *StampFactory
	entity      *d2mapentity.MapEntityFactory
	regionPath  string // The file path of the region
	regionID    d2enum.RegionIdType
	levelType   d2records.LevelTypeRecord   // The level type id for this stamp
	levelPreset d2records.LevelPresetRecord // The level preset id for this stamp
	tiles       []d2dt1.Tile                // The tiles contained on this stamp
	ds1         *d2ds1.DS1                  // The backing DS1 file for this stamp
}

// Size returns the size of the stamp in tiles.
func (mr *Stamp) Size() d2geom.Size {
	return d2geom.Size{Width: mr.ds1.Width(), Height: mr.ds1.Height()}
}

// LevelPreset returns the level preset ID.
func (mr *Stamp) LevelPreset() d2records.LevelPresetRecord {
	return mr.levelPreset
}

// LevelType returns the level type ID.
func (mr *Stamp) LevelType() d2records.LevelTypeRecord {
	return mr.levelType
}

// RegionID returns the regionID
func (mr *Stamp) RegionID() d2enum.RegionIdType {
	return mr.regionID
}

// RegionPath returns the file path of the region.
func (mr *Stamp) RegionPath() string {
	return mr.regionPath
}

// Objects returns the DS1 object list (monster, object and other placements
// in sub-tile coordinates relative to the stamp).
func (mr *Stamp) Objects() []d2ds1.Object {
	return mr.ds1.Objects
}

// Act returns the act number stored in the DS1 (1 to 5), which selects the
// monpreset / object lookup tables.
func (mr *Stamp) Act() int {
	return int(mr.ds1.Act)
}

// Tile represents a map tile, which can have a variable amount of floors, walls, shadows as layers.
// Typically, there will be an Orientation layer for each wall layer.
type Tile struct {
	Walls         []d2ds1.Tile
	Orientations  []d2ds1.Tile
	Floors        []d2ds1.Tile
	Shadows       []d2ds1.Tile
	Substitutions []d2ds1.Tile
}

// Tile returns the tile at the given x and y tile coordinates.
func (mr *Stamp) Tile(x, y int) *Tile {
	t := &Tile{
		Walls:         make([]d2ds1.Tile, len(mr.ds1.Walls)),
		Floors:        make([]d2ds1.Tile, len(mr.ds1.Floors)),
		Shadows:       make([]d2ds1.Tile, len(mr.ds1.Shadows)),
		Substitutions: make([]d2ds1.Tile, len(mr.ds1.Substitutions)),
	}

	for idx := range mr.ds1.Walls {
		t.Walls[idx] = *mr.ds1.Walls[idx].Tile(x, y)
	}

	for idx := range mr.ds1.Floors {
		t.Floors[idx] = *mr.ds1.Floors[idx].Tile(x, y)
	}

	for idx := range mr.ds1.Shadows {
		t.Shadows[idx] = *mr.ds1.Shadows[idx].Tile(x, y)
	}

	for idx := range mr.ds1.Substitutions {
		t.Substitutions[idx] = *mr.ds1.Substitutions[idx].Tile(x, y)
	}

	return t
}

// TileData returns the tile data for the tile with given style, sequence and type.
func (mr *Stamp) TileData(style, sequence int32, tileType d2enum.TileType) *d2dt1.Tile {
	for idx := range mr.tiles {
		tile := &mr.tiles[idx]
		if tile.Style == style && tile.Sequence == sequence && tile.Type == int32(tileType) {
			return tile
		}
	}

	return nil
}

// Entities spawns all entities and objects in this tile on the map.
func (mr *Stamp) Entities(tileOffsetX, tileOffsetY int) []d2interface.MapEntity {
	entities := make([]d2interface.MapEntity, 0)

	for _, object := range mr.ds1.Objects {
		if object.Type == int(d2enum.ObjectTypeCharacter) {
			monPreset := mr.factory.asset.Records.Monster.Presets[mr.ds1.Act][object.ID]
			monstat := mr.factory.asset.Records.Monster.Stats[monPreset]
			// If monstat is nil here it is a place_ type object, idk how to handle those yet.
			// (See monpreset and monplace txts for reference)
			if monstat != nil {
				// Temorary use of Lookup.
				// nolint:gomnd // constant modifier
				npcX, npcY := (tileOffsetX*5)+object.X, (tileOffsetY*5)+object.Y
				npc, err := mr.newNPC(npcX, npcY, monstat)

				if err == nil {
					npc.SetPaths(convertPaths(tileOffsetX, tileOffsetY, object.Paths))
					entities = append(entities, npc)
				}
			}
		}

		if object.Type == int(d2enum.ObjectTypeItem) {
			// For objects the DS1 ID to objectID is hardcoded in the game
			// use the lookup table
			lookup := mr.factory.asset.Records.FindObject(int(mr.ds1.Act), object.Type, object.ID)

			if lookup == nil {
				continue
			}

			objectRecord := mr.factory.asset.Records.Object.Details[lookup.ObjectsTxtId]

			if objectRecord != nil {
				// nolint:gomnd // constant
				entity, err := mr.newObject((tileOffsetX*5)+object.X, (tileOffsetY*5)+object.Y, objectRecord)
				if err != nil {
					mr.factory.Warningf("skipping object %d at (%d,%d): %v", object.ID, object.X, object.Y, err)
					continue
				}

				entities = append(entities, entity)
			}
		}
	}

	return entities
}

// newNPC creates an NPC; some monster animation data in the game files is not
// accepted by the loaders (they panic), which must not take the whole map down.
func (mr *Stamp) newNPC(x, y int, monstat *d2records.MonStatRecord) (npc *d2mapentity.NPC, err error) {
	defer func() {
		if r := recover(); r != nil {
			npc, err = nil, fmt.Errorf("%v", r)
			mr.factory.Warningf("skipping monster %s at (%d,%d): %v", monstat.Key, x, y, r)
		}
	}()

	return mr.entity.NewNPC(x, y, monstat, 0)
}

func (mr *Stamp) newObject(x, y int, rec *d2records.ObjectDetailRecord) (obj *d2mapentity.Object, err error) {
	defer func() {
		if r := recover(); r != nil {
			obj, err = nil, fmt.Errorf("%v", r)
		}
	}()

	return mr.entity.NewObject(x, y, rec, d2resource.PaletteUnits)
}

func convertPaths(tileOffsetX, tileOffsetY int, paths []d2path.Path) []d2path.Path {
	result := make([]d2path.Path, len(paths))
	for i := 0; i < len(paths); i++ {
		result[i].Action = paths[i].Action
		result[i].Position = d2vector.NewPosition(
			paths[i].Position.X()+float64(tileOffsetX*subtilesPerTile),
			paths[i].Position.Y()+float64(tileOffsetY*subtilesPerTile))
	}

	return result
}
