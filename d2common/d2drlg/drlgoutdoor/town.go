package drlgoutdoor

import "fmt"

// townDef maps the Act 2 and Act 3 town levels to their LvlPrest Def
// ("Act 2 - Town" 301, "Act 3 - Town" 529): the towns are preset levels, not
// outdoor generators.
var townDef = map[int]int{40: 301, 75: 529}

// GenerateTown builds the room list of Lut Gholein (40) or Kurast Docks (75):
// AllocPresetMap rolls the file (nothing when the Def has Files == 0), then
// PlacePresetRooms tiles the level with 8x8 chunk rooms, one level-seed step
// each (drlg-act23-outdoor.md 3.2; the rooms and final seed are verified there).
// file is the preset file slot: Params.TownFile of the Act 2 world (1 LutW,
// 2 LutN); Act 3 draws its own (Files == 1).
func GenerateTown(env *Env, p Params, file int) (lv *Level, err error) {
	def, ok := townDef[p.ID]
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: level %d is not an Act 2/3 town", p.ID)
	}

	rec, ok := env.Tables.PrestByDef(def)
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: LvlPrest Def %d unknown", def)
	}

	l := &Level{Params: p, env: env, ctr: map[int]*counter{}}
	if lr, ok := env.Tables.Level(p.ID); ok {
		l.LType = lr.LevelType // the room tile libraries (BuildTiles) come from the level type
	}

	l.Seed = newLevelSeed(p)
	l.W, l.H = p.Rect.W>>3, p.Rect.H>>3

	if rec.Files >= 1 {
		n := int(l.Seed.Roll(int32(rec.Files)))
		if p.ID == 75 {
			file = n
		}
	}

	l.presetSize = [2]int{p.Rect.W, p.Rect.H}
	l.placePresetRooms(def, file, p.Rect.X, p.Rect.Y, 0)

	return l, nil
}
