package drlgoutdoor

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// MazeLevel is a maze level (caves, crypts, catacombs, tombs, sewers, ...)
// wrapped for the tile builder: every chunk of the maze result is a preset
// room (type 2) of the room's own DS1, built by DRLG_BuildPresetRoomTiles
// (0x6696c0) exactly like the preset rooms of the outdoor levels.
type MazeLevel struct {
	Rooms []*Room
	lv    *Level
}

// NewMazeLevel turns a drlgmaze result into preset rooms: the chunk position
// and size, the room seed (the level-seed step the chunk was allocated with),
// the LvlPrest Dt1Mask as tile library mask, the DS1 of the room as the preset
// the chunk is a part of (its rectangle is the room's, so chunks of rooms
// larger than 12x12 index into the DS1 by their offset).
func NewMazeLevel(env *Env, res *drlgmaze.Result, levelID int, gameSeed uint32) (*MazeLevel, error) {
	rec, ok := env.Tables.Level(levelID)
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: level %d unknown", levelID)
	}

	p := Params{ID: levelID, Vis: rec.Vis, Warp: rec.Warp}
	p.BaseSeed, _ = d2rand.DrlgBaseSeed(gameSeed)

	l := &Level{Params: p, env: env, LType: rec.LevelType}
	ml := &MazeLevel{lv: l}

	// the warp bits of the level (a vis slot with no destination), as for the
	// preset levels (GeneratePreset)
	var flags0 uint32

	for k := 0; k < 8; k++ {
		if p.Vis[k] != 0 && p.Warp[k] == -1 {
			flags0 |= 0x10 << uint(k)
		}
	}

	for _, c := range res.Chunks {
		mr := res.Rooms[c.Room]
		if mr.FileName == "" || mr.File < 0 {
			continue
		}

		pr, ok := env.Tables.PrestByDef(mr.Def)
		if !ok {
			return nil, fmt.Errorf("drlgoutdoor: maze Def %d unknown", mr.Def)
		}

		ds, err := env.Pattern(NormalizePrestFile(mr.FileName))
		if err != nil {
			return nil, err
		}

		r := &Room{Type: 2}
		r.Seed.Init(c.Lo)
		r.S4 = r.Seed.Step()
		r.X, r.Y, r.W, r.H = c.X, c.Y, c.W, c.H
		r.PrestDef, r.File = mr.Def, mr.File
		r.PrestX, r.PrestY, r.PrestW, r.PrestH = mr.X, mr.Y, mr.W, mr.H
		r.R50 = uint32(pr.Dt1Mask)

		chx, chy := (c.X-mr.X)/8, (c.Y-mr.Y)/8
		r.Flags = flags0 | presetRectBits(ds, c.X-mr.X, c.Y-mr.Y, c.W, c.H)

		if waypointChunk(ds, chx, chy) {
			r.Flags |= 0x30000
		}

		if pr.Outdoors != 0 {
			r.Flags |= 0x80000
		}

		if pr.Populate == 0 {
			r.Flags |= 0x800000
		}

		ml.Rooms = append(ml.Rooms, r)
	}

	return ml, nil
}

// BuildTiles builds the tile records of every chunk in creation order (the
// result is parallel to Rooms). Records of border cells are merged with the
// rooms built before, like in the game.
func (m *MazeLevel) BuildTiles() ([]*RoomTiles, error) {
	m.lv.Rooms = m.Rooms

	return m.lv.buildTiles(false)
}
