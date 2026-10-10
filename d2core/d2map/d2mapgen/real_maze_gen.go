package d2mapgen

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
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

	// the two special tombs of Act 2 are drawn from the game seed (the real Tal Rasha's tomb gets the
	// chamber with the orifice, see d2drlg.RealTomb)
	params := drlgmaze.Params{LevelID: levelID, Difficulty: diff, BaseSeed: base}
	if levelID >= 40 && levelID <= 74 {
		ex := d2drlg.DrawActExtras(seed, 1)
		params.TombA, params.TombB = ex.TombA, ex.TombB
	}

	res, err := drlgmaze.Generate(tb, params)
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

	// the original's population: rooms are the DrlgRoom chunks with their seeds
	pop := g.newPopLevel(levelID, seed, diff)
	chunkRooms := map[int][]*popRoom{}

	if pop != nil {
		for _, c := range res.Chunks {
			noPop := false
			if pr, ok := tb.PrestByDef(res.Rooms[c.Room].Def); ok && pr.Populate == 0 {
				noPop = true
			}

			pr := pop.addRoom(c.X-res.MinX+realMazeMargin, c.Y-res.MinY+realMazeMargin, c.W, c.H, c.Seed, noPop)
			chunkRooms[c.Room] = append(chunkRooms[c.Room], pr)
		}
	}

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

		if pop != nil {
			var first *popRoom
			if rs := chunkRooms[ri]; len(rs) > 0 {
				first = rs[0]
			}

			var gate *d2rand.Seed
			if r.GateSteps > 0 {
				gate = &r.GateSeed
			}

			ps := pop.addPreset(stamp, ox, oy, gate, first)
			for _, pr := range chunkRooms[ri] {
				pr.preset = ps
			}

			g.engine.PlaceStampClippedWhere(stamp, ox, oy, r.W, r.H, ps.keepFunc(), false)
		} else {
			g.engine.PlaceStampClipped(stamp, ox, oy, r.W, r.H)
		}

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

		if pop == nil {
			roomSeed := d2rand.New(levelSeed.Lo + uint32(ri)*0x9E3779B1)
			g.placeMonsters(stamp, levelID, diff, ox, oy, r.W, r.H, roomSeed, &mon)
		}
	}

	exact := g.applyExactMazeTiles(levelID, seed, res, region, w, h)

	g.Infof("TILESTATS level=%d exact=%v %s", levelID, exact, g.engine.TileStats())

	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)

	if pop != nil {
		pop.run()
		pop.logSummary("real maze")
	}

	sx, sy, how := g.findEntry(res, entries, realMazeMargin)
	g.engine.SetStartPosition(sx, sy)

	g.Infof("real maze: level %d type %d seed %#x: %d rooms, %d stamps placed, map %dx%d tiles", levelID, lvl.LevelType,
		seed, len(res.Rooms), placed, w, h)
	g.Infof("real maze: hero entry at tile (%.1f,%.1f) %s", sx, sy, how)
	if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
		g.logWalkMap(sx, sy)
	}

	if levelID >= d2level.FirstTalRashaTomb && levelID <= d2level.LastTalRashaTomb {
		orifices := 0

		for _, e := range g.engine.Entities() {
			if o, ok := e.(*d2mapentity.Object); ok && o.Record().Index == 152 { // the Horadric staff orifice
				orifices++
			}
		}

		g.Infof("real maze: Tal Rasha's tomb level %d: real=%v (game seed %d draws %d) orifice objects=%d",
			levelID, levelID == params.TombA, seed, params.TombA, orifices)
	}

	if levelID == 74 {
		pads := map[int]int{}

		for _, e := range g.engine.Entities() {
			if o, ok := e.(*d2mapentity.Object); ok && o.Record().OperateFn == 27 {
				pads[o.Record().Index]++
			}
		}

		g.Infof("real maze: Arcane Sanctuary teleport pads by objects.txt row: %v", pads)
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

			if os.Getenv("OD2_DEBUG_MARKERS") != "" {
				g.Infof("MARKER skipped %s at (%d,%d) level %d", name, x, y, levelID)
			}

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

			if i == 0 && rec.Monster.Unique.Super[name] != nil {
				npc.SetSuperUnique(name) // the director spawns the boss of that name, with its followers
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

	reach := g.engine.ReachableFrom(int(heroX*subtilesPerTile), int(heroY*subtilesPerTile))

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
			case reach != nil && !reach.At(x*subtilesPerTile+middleOfTile, y*subtilesPerTile+middleOfTile):
				row[x] = 'o' // walkable but not connected to the hero's start
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

// applyExactMazeTiles replaces the stamped DS1 tiles of the maze rooms by the tile
// records the game builds for them (drlgoutdoor.MazeLevel: every chunk is a
// preset room of its DS1, DT1 library from the LvlPrest Dt1Mask, rarity pick
// with the room seed, neighbour and border merging). The stamp keeps the
// entities and the marker tiles. When the build fails (a code path of the game
// that is not ported) the stamped tiles stay and false is returned.
func (g *MapGenerator) applyExactMazeTiles(levelID int, seed uint32, res *drlgmaze.Result, region d2enum.RegionIdType, w, h int) bool {
	if os.Getenv("OD2_MAZE_STAMP") == "1" { // debugging switch: the old stamp lookup, for before/after counts
		return false
	}

	env, err := outdoorEnv(g.asset)
	if err != nil {
		g.Infof("maze tiles: level %d: %v", levelID, err)
		return false
	}

	ml, err := drlgoutdoor.NewMazeLevel(env, res, levelID, seed)
	if err == nil {
		var tiles []*drlgoutdoor.RoomTiles

		if tiles, err = ml.BuildTiles(); err == nil {
			rect := drlgoutdoor.Rect{X: res.MinX - realMazeMargin, Y: res.MinY - realMazeMargin, W: w, H: h}
			_, n := g.applyExactTiles(tiles, rect, region, false)
			g.Infof("maze tiles: level %d: exact records for %d rooms", levelID, n)

			return true
		}
	}

	g.Infof("maze tiles: level %d: exact records unavailable (%v); keeping the stamped DS1 tiles", levelID, err)

	return false
}
