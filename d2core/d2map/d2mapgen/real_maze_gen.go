package d2mapgen

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	realMazeMargin   = 3 // empty tiles around the rooms
	subtilesPerTile  = 5
	middleOfTile     = 3 // the server puts a spawning hero on sub-tile 3 of the tile
	entrySearchTiles = 8 // how far from the entry marker a walkable tile is searched
)

// entryRoom matches the DS1 names of the rooms that hold a level's entry
// (up stairs / warp back): CaveEPre2, CryptNWarpPrev, CatEWUp, CatNSEWExit, DenEnt.
// Verified against the marker tiles in the Act 1 cave, crypt and catacomb DS1
// files (hidden special tiles 21/0 and 0/0); jail rooms are checked in code.
var entryRoom = regexp.MustCompile(`(?i)(pre|prev|up|ent|exit)\d*\.ds1$`)

// GenerateRealMaze replaces the map with the DRLG maze level for the hero's
// seed: every room's chosen preset DS1 is stamped at its room position
// (clipped to the room size), empty space is made unwalkable, the hero spawn
// is put next to the level's entry marker and the DS1 monster placements are
// resolved to monsters (the monster director adopts them).
//
// Exactness caveat: the room layout comes from the d2drlg port, whose
// equality with the real game is not proven. Tile variants, monster
// choices behind place_* markers and the entry rule are approximations (see
// the comments below), so a map may differ from the real game's for the
// same seed.
func (g *MapGenerator) GenerateRealMaze(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	defer d2util.PerfTime(fmt.Sprintf("generate level=%d", levelID))()

	if isOutdoorLevel(levelID) {
		return g.GenerateRealOutdoor(levelID, seed, diff)
	}

	if isPresetLevel(levelID) {
		return g.GenerateRealPreset(levelID, seed, diff)
	}

	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	base, _ := d2rand.DrlgBaseSeed(seed)

	res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: levelID, Difficulty: diff, BaseSeed: base})
	if err != nil {
		return err
	}

	lvl, _ := tb.Level(levelID)
	region := d2enum.RegionIdType(lvl.LevelType)

	if g.asset.Records.Level.Types[region] == nil {
		return fmt.Errorf("level %d: no LvlTypes row %d", levelID, lvl.LevelType)
	}

	w := res.MaxX - res.MinX + 2*realMazeMargin + 1
	h := res.MaxY - res.MinY + 2*realMazeMargin + 1

	g.engine.ResetMap(region, w, h)

	var (
		placed  int
		entries []roomRect
		mon     monsterStats
	)

	levelSeed := d2rand.LevelSeed(base, uint32(levelID))

	for ri, r := range res.Rooms {
		rec, ok := g.asset.Records.Level.Presets[r.Def]
		if !ok || r.File < 0 {
			g.Warningf("real maze: Def %d has no usable preset files, room skipped", r.Def)
			continue
		}

		var files []string

		for _, f := range rec.Files {
			if f != "" && f != "0" {
				files = append(files, f)
			}
		}

		if r.File >= len(files) {
			g.Warningf("real maze: Def %d file index %d out of range (%d files)", r.Def, r.File, len(files))
			continue
		}

		// only the chosen DS1's DT1 list is loaded, on top of the LvlTypes ones
		g.engine.AddDS1(files[r.File])

		stamp := g.engine.LoadStamp(region, r.Def, r.File)
		if stamp == nil {
			continue
		}

		ox, oy := r.X-res.MinX+realMazeMargin, r.Y-res.MinY+realMazeMargin

		g.engine.PlaceStampClipped(stamp, ox, oy, r.W, r.H)

		if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
			g.Infof("AUTOMAP room %d def=%d %s at tile (%d,%d) %dx%d", ri, r.Def, files[r.File], ox, oy, r.W, r.H)
		}

		placed++

		if dest := mazeRoomExit(levelID, files[r.File]); dest != 0 {
			g.setRoomWarps(roomRect{ox, oy, r.W, r.H, files[r.File]}, dest)
		}

		if entryRoom.MatchString(files[r.File]) {
			entries = append(entries, roomRect{ox, oy, r.W, r.H, files[r.File]})
		}

		roomSeed := d2rand.New(levelSeed.Lo + uint32(ri)*0x9E3779B1)
		g.placeMonsters(stamp, levelID, diff, ox, oy, r.W, r.H, roomSeed, &mon)
	}

	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)

	sx, sy, how := g.findEntry(res, entries, realMazeMargin)
	g.engine.SetStartPosition(sx, sy)

	g.Infof("real maze: level %d type %d seed %#x: %d rooms, %d stamps placed, map %dx%d tiles", levelID, lvl.LevelType,
		seed, len(res.Rooms), placed, w, h)
	g.Infof("real maze: hero entry at tile (%.1f,%.1f) %s", sx, sy, how)
	if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
		g.logWalkMap(sx, sy)
	}

	g.Infof("real maze: DS1 monsters: %d direct, %d place_* markers resolved, %d super uniques, %d groups, %d skipped",
		mon.direct, mon.place, mon.super, mon.groups, mon.skipped)

	return nil
}

type roomRect struct {
	x, y, w, h int
	file       string
}

// findEntry picks the hero's start tile: next to the first entry marker
// (special tile) of an entry room, else the middle of the first entry room,
// else the middle of the first room. The marker rule is a heuristic: the real
// game places the arriving hero by the level's warp (Levels.txt Vis/Warp ->
// LvlWarp), which the DRLG port does not provide yet.
func (g *MapGenerator) findEntry(res *drlgmaze.Result, entries []roomRect, margin int) (x, y float64, how string) {
	cand := entries

	if len(cand) == 0 && len(res.Rooms) > 0 {
		r := res.Rooms[len(res.Rooms)-1]
		cand = []roomRect{{r.X - res.MinX + margin, r.Y - res.MinY + margin, r.W, r.H, "(no entry room found, last room)"}}
	}

	for _, rr := range cand {
		cx, cy, ok := g.markerIn(rr)
		if !ok {
			cx, cy = rr.x+rr.w/2, rr.y+rr.h/2
		}

		if tx, ty, ok := g.nearestWalkable(cx, cy, entrySearchTiles); ok {
			return float64(tx) + 0.05, float64(ty) + 0.05, "near " + rr.file
		}
	}

	return float64(res.MaxX-res.MinX) / 2, float64(res.MaxY-res.MinY) / 2, "(fallback: map centre, nothing walkable found)"
}

// mazeRoomExit says where the exit tiles of a special room of a maze lead, when the tile style alone
// does not say. River of Flame (107): the bridge room at the north end (Act4/Diab/BridgeLava.ds1)
// carries the walk-through exit to the Chaos Sanctuary (108); its tiles have the styles 8, 12 and
// 16 (UNVERIFIED which of the six the original uses; all lead to the same level). The south room
// (WarpMesa.ds1, style 0) goes back to the City of the Damned by the slot rule of d2level.TileDestination.
func mazeRoomExit(levelID int, file string) int {
	low := strings.ToLower(file)

	switch {
	case levelID == 107 && strings.Contains(low, "bridgelava"):
		return 108
	case levelID >= 47 && levelID <= 48 && strings.Contains(low, "sewsdown"):
		// the stairs down of the Sewers (OBSERVED: Act2/Sewer/SewSDown.ds1 carries the one tile with the
		// style 2; the slot rule of TileDestination takes style 2 for an exit that is not there)
		return levelID + 1
	case levelID == 47 && strings.Contains(low, "sewnsdock"):
		// the dock end of Sewers Level 1 leads back to Lut Gholein (Levels.txt Vis1 of level 47, LvlWarp 21)
		return 40
	}

	return 0
}

// setRoomWarps makes every exit tile of a room lead to level dest.
func (g *MapGenerator) setRoomWarps(rr roomRect, dest int) {
	for y := rr.y; y < rr.y+rr.h; y++ {
		for x := rr.x; x < rr.x+rr.w; x++ {
			t := g.engine.TileAt(x, y)
			if t == nil {
				continue
			}

			for i := range t.Components.Walls {
				if wl := &t.Components.Walls[i]; wl.Type.Special() && wl.Style != startMarkerStyle {
					g.engine.SetWarpDestination(x, y, dest)
				}
			}
		}
	}
}

// markerIn returns the first special marker tile (wall type 10 or 11, the
// hidden tiles of warp.dt1) in the room.
func (g *MapGenerator) markerIn(rr roomRect) (tx, ty int, ok bool) {
	for y := rr.y; y < rr.y+rr.h; y++ {
		for x := rr.x; x < rr.x+rr.w; x++ {
			t := g.engine.TileAt(x, y)
			if t == nil {
				continue
			}

			for i := range t.Components.Walls {
				wl := &t.Components.Walls[i]
				if wl.Prop1 != 0 && wl.Type.Special() {
					return x, y, true
				}
			}
		}
	}

	return 0, 0, false
}

// nearestWalkable searches outwards (square rings) from a tile for one whose
// spawn sub-tile is walkable.
func (g *MapGenerator) nearestWalkable(cx, cy, radius int) (int, int, bool) {
	walkable := func(x, y int) bool {
		f := g.engine.SubTileAt(x*subtilesPerTile+middleOfTile, y*subtilesPerTile+middleOfTile)
		return !f.BlockWalk && !f.BlockPlayerWalk
	}

	for r := 0; r <= radius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if (dx != -r && dx != r) && (dy != -r && dy != r) {
					continue
				}

				if walkable(cx+dx, cy+dy) {
					return cx + dx, cy + dy, true
				}
			}
		}
	}

	return 0, 0, false
}

// monsterStats counts how DS1 monster placements were resolved.
type monsterStats struct{ direct, place, super, groups, skipped int }

// placeMonsters turns the DS1 monster placements that the stamp itself cannot
// resolve into NPC entities of a hostile monstats row; the monster director
// adopts hostile NPC placements and gives them their AI. Placements whose
// monpreset name is a plain monstats key were already created by the stamp.
//
// What the original does for the rest is not in the data tables, so this is an
// approximation (unverified):
//   - a super unique name (Coldcrow, Bishibosh...) spawns its Class monster
//     (no boss modifiers, no minions);
//   - place_fallen / place_fallenshaman / place_fetish ... spawn the tier 1
//     monstats row of that name;
//   - place_group25/50/75/100 spawn, with that chance, a monster of the level's
//     Levels.txt monster list plus a group of MinGrp..MaxGrp of the same kind;
//   - other markers (nests, npc packs, champions, nothing) are skipped.
func (g *MapGenerator) placeMonsters(stamp *d2mapstamp.Stamp, levelID int, diff d2drlg.Difficulty,
	ox, oy, w, h int, rng *d2rand.Seed, st *monsterStats) {
	rec := g.asset.Records
	presets := rec.Monster.Presets[int32(stamp.Act())]
	levelMons := levelMonsters(rec.Level.Details[levelID], diff)

	for _, o := range stamp.Objects() {
		if o.Type != int(d2enum.ObjectTypeCharacter) || o.ID < 0 || o.ID >= len(presets) {
			continue
		}

		if o.X/subtilesPerTile >= w || o.Y/subtilesPerTile >= h {
			continue
		}

		name := presets[o.ID]
		if rec.Monster.Stats[name] != nil {
			st.direct++ // created by the stamp
			continue
		}

		x, y := ox*subtilesPerTile+o.X, oy*subtilesPerTile+o.Y

		var (
			stat  *d2records.MonStatRecord
			count = 1
		)

		lower := strings.ToLower(name)

		switch {
		case rec.Monster.Unique.Super[name] != nil:
			sup := rec.Monster.Unique.Super[name]
			stat = rec.Monster.Stats[sup.Class]
			st.super++
		case strings.HasPrefix(lower, "place_group"):
			pct := 0
			fmt.Sscanf(lower, "place_group%d", &pct)

			if int(rng.Roll(100)) >= pct || len(levelMons) == 0 {
				st.skipped++
				continue
			}

			stat = rec.Monster.Stats[levelMons[int(rng.Roll(int32(len(levelMons))))]]
			if stat != nil && stat.MinionGroupMax > stat.MinionGroupMin {
				count = stat.MinionGroupMin + int(rng.Roll(int32(stat.MinionGroupMax-stat.MinionGroupMin+1)))
			}

			st.groups++
		case strings.HasPrefix(lower, "place_"):
			kind := strings.TrimPrefix(lower, "place_")
			if stat = rec.Monster.Stats[kind+"1"]; stat == nil {
				stat = rec.Monster.Stats[kind]
			}

			if stat != nil {
				st.place++
			}
		}

		if stat == nil {
			st.skipped++
			continue
		}

		for i := 0; i < count; i++ {
			px, py := x, y

			if i > 0 { // spread a group out around its leader
				px, py = x+(i%3-1)*2, y+(i/3%3-1)*2
			}

			npc, err := g.safeNPC(px, py, stat)
			if err != nil {
				g.Warningf("real maze: could not place %s: %v", stat.Key, err)
				continue
			}

			g.engine.AddEntity(npc)
		}
	}
}

// levelMonsters lists the Levels.txt monster keys (mon1..mon10 / nmon1..) of
// a level for a difficulty, limited to NumMon entries.
func levelMonsters(d *d2records.LevelDetailRecord, diff d2drlg.Difficulty) []string {
	if d == nil {
		return nil
	}

	all := []string{d.MonsterID1Normal, d.MonsterID2Normal, d.MonsterID3Normal, d.MonsterID4Normal,
		d.MonsterID5Normal, d.MonsterID6Normal, d.MonsterID7Normal, d.MonsterID8Normal, d.MonsterID9Normal,
		d.MonsterID10Normal}

	if diff >= 1 { // nightmare and hell use the nmon list
		all = []string{d.MonsterID1Nightmare, d.MonsterID2Nightmare, d.MonsterID3Nightmare, d.MonsterID4Nightmare,
			d.MonsterID5Nightmare, d.MonsterID6Nightmare, d.MonsterID7Nightmare, d.MonsterID8Nightmare,
			d.MonsterID9Nightmare, d.MonsterID10Nightmare}
	}

	out := make([]string, 0, len(all))

	for i, m := range all {
		if m != "" && m != "0" && (d.NumMonsterTypes == 0 || i < d.NumMonsterTypes) {
			out = append(out, m)
		}
	}

	return out
}

// logWalkMap logs one character per tile: '#' no floor, '.' walkable, 'x' floor
// but blocked in the middle, '@' the hero entry, 'm' a monster placement.
// (OD2_AUTOMAP_ASCII=1)
func (g *MapGenerator) logWalkMap(heroX, heroY float64) {
	size := g.engine.Size()
	mon := map[[2]int]bool{}

	for _, e := range g.engine.Entities() {
		p := e.GetPosition()
		mon[[2]int{int(p.X()) / subtilesPerTile, int(p.Y()) / subtilesPerTile}] = true
	}

	for y := 0; y < size.Height; y++ {
		row := make([]byte, size.Width)

		for x := 0; x < size.Width; x++ {
			f := g.engine.SubTileAt(x*subtilesPerTile+middleOfTile, y*subtilesPerTile+middleOfTile)

			switch {
			case int(heroX) == x && int(heroY) == y:
				row[x] = '@'
			case mon[[2]int{x, y}]:
				row[x] = 'm'
			case !g.engine.TileExists(x, y):
				row[x] = '#'
			case f.BlockWalk:
				row[x] = 'x'
			default:
				row[x] = '.'
			}
		}

		g.Infof("AUTOMAP walk %02d %s", y, string(row))
	}
}

// safeNPC creates an NPC, turning a panic of the animation loaders (some
// monster COF files hold weapon classes they do not know) into an error.
func (g *MapGenerator) safeNPC(x, y int, stat *d2records.MonStatRecord) (npc *d2mapentity.NPC, err error) {
	defer func() {
		if r := recover(); r != nil {
			npc, err = nil, fmt.Errorf("%v", r)
		}
	}()

	return g.engine.NewNPC(x, y, stat, 0)
}
