package d2level

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Kind says how a link is taken.
type Kind int

// Link kinds.
const (
	// KindTile is a stair, hole, door or warp tile: the player interacts with
	// a room tile of the source level (C2S 0x13 with unit type 5, distance < 5)
	// and SERVER_EnterWarpTile moves him to the destination level.
	KindTile Kind = iota
	// KindEdge is a seamless border between two neighbouring outdoor levels:
	// the player just walks across, no level change happens in the original.
	KindEdge
)

// Source records how well a link is known.
type Source int

// Link sources.
const (
	// SourceLevelsTxt links come from the Vis/Warp columns of Levels.txt. That
	// the two levels are connected is data; which tile of the source level
	// carries the link (the tile's warp id selects the slot) is UNVERIFIED for
	// slots whose WarpN value repeats.
	SourceLevelsTxt Source = iota
	// SourceDRLG links are the verified neighbours of the Act 1 world search
	// (drlg2.md, cluster tables 0x6f1d00 and 0x6f1df0).
	SourceDRLG
)

// Link is a directed connection between two levels.
type Link struct {
	From, To int
	// Warp is the LvlWarp.txt id of the tile in From that leads to To (-1 for
	// edges and slots without one).
	Warp   int
	Kind   Kind
	Source Source
}

// drlgEdges are the placement neighbours of the Act 1 world search (verified,
// drlg2.md section B.1): {level, reference level}. Cluster 1: Stony Field (4)
// is the anchor, Cold Plains (3) sits next to it, Blood Moor (2) next to Cold
// Plains, the Rogue Encampment (1) next to Blood Moor and the Burial Grounds
// (17) next to Cold Plains. Cluster 2: Monastery (26) is the anchor, Tamoe
// Highland (7) below it, Black Marsh (6) next to Tamoe and Dark Wood (5) next
// to Black Marsh.
var drlgEdges = [][2]int{
	{3, 4}, {2, 3}, {1, 2}, {17, 3},
	{7, 26}, {6, 7}, {5, 6},
	// Act 2 desert (DRLG_InitActLevelLinks case 1, drlg-act23-outdoor.md 3.1:
	// the pass 3 vis registrations with warp -1): Lut Gholein, Rocky Waste, Dry
	// Hills, Far Oasis, Lost City, Valley of Snakes in a chain. The Canyon of the
	// Magi (46) has no seamless neighbour.
	{41, 40}, {42, 41}, {43, 42}, {44, 43}, {45, 44},
}

var allLinks []Link

func init() {
	for _, e := range levelsTxtLinks {
		allLinks = append(allLinks, Link{From: e[0], To: e[1], Warp: e[2], Kind: KindTile, Source: SourceLevelsTxt})
	}

	for _, e := range drlgEdges {
		allLinks = append(allLinks,
			Link{From: e[0], To: e[1], Warp: -1, Kind: KindEdge, Source: SourceDRLG},
			Link{From: e[1], To: e[0], Warp: -1, Kind: KindEdge, Source: SourceDRLG})
	}
}

// Links returns every known directed link.
func Links() []Link {
	return append([]Link(nil), allLinks...)
}

// LinksFrom returns the links leaving a level.
func LinksFrom(level int) []Link {
	var out []Link

	for _, l := range allLinks {
		if l.From == level {
			out = append(out, l)
		}
	}

	return out
}

// Destination returns the level a tile with LvlWarp id warp leads to from
// level: the first Vis slot whose Warp slot matches (UNVERIFIED when several
// slots share a warp id, see SourceLevelsTxt).
func Destination(level, warp int) (int, bool) {
	for _, l := range allLinks {
		if l.From == level && l.Kind == KindTile && l.Warp == warp {
			return l.To, true
		}
	}

	return 0, false
}

// TileLinkTo returns the first tile link from one level to another.
func TileLinkTo(from, to int) (Link, bool) {
	for _, l := range allLinks {
		if l.From == from && l.To == to && l.Kind == KindTile {
			return l, true
		}
	}

	return Link{}, false
}

// Reachable reports whether to can be reached from from by following links.
func Reachable(from, to int) bool {
	seen := map[int]bool{from: true}
	queue := []int{from}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur == to {
			return true
		}

		for _, l := range LinksFrom(cur) {
			if !seen[l.To] {
				seen[l.To] = true
				queue = append(queue, l.To)
			}
		}
	}

	return false
}

// ParseWaypointColumn reads the Waypoint column of a Levels.txt and returns
// waypoint bit -> level id (255 means "no waypoint").
func ParseWaypointColumn(levelsTxt []byte) (map[int]int, error) {
	sc := bufio.NewScanner(bytes.NewReader(levelsTxt))
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	if !sc.Scan() {
		return nil, errors.New("empty Levels.txt")
	}

	idCol, wpCol := -1, -1

	for i, h := range strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t") {
		switch h {
		case "Id":
			idCol = i
		case "Waypoint":
			wpCol = i
		}
	}

	if idCol < 0 || wpCol < 0 {
		return nil, errors.New("Levels.txt has no Id/Waypoint column")
	}

	out := map[int]int{}

	for sc.Scan() {
		f := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
		if idCol >= len(f) || wpCol >= len(f) {
			continue
		}

		id, err1 := strconv.Atoi(strings.TrimSpace(f[idCol]))
		bit, err2 := strconv.Atoi(strings.TrimSpace(f[wpCol]))

		if err1 == nil && err2 == nil && bit >= 0 && bit < 255 {
			out[bit] = id
		}
	}

	return out, sc.Err()
}

// ParseLevelLinks reads the Vis0..7 / Warp0..7 columns of a Levels.txt and
// returns the same {from, to, warp} triples as the embedded table (levels
// above 132 are skipped, as in the embedded data).
func ParseLevelLinks(levelsTxt []byte) ([][3]int, error) {
	sc := bufio.NewScanner(bytes.NewReader(levelsTxt))
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	if !sc.Scan() {
		return nil, errors.New("empty Levels.txt")
	}

	col := map[string]int{}
	for i, h := range strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t") {
		col[h] = i
	}

	need := []string{"Id"}
	for k := 0; k < 8; k++ {
		need = append(need, fmt.Sprintf("Vis%d", k), fmt.Sprintf("Warp%d", k))
	}

	for _, n := range need {
		if _, ok := col[n]; !ok {
			return nil, fmt.Errorf("Levels.txt has no %q column", n)
		}
	}

	var out [][3]int

	seen := map[[3]int]bool{}
	num := func(f []string, name string, def int) int {
		if i := col[name]; i < len(f) {
			if v, err := strconv.Atoi(strings.TrimSpace(f[i])); err == nil {
				return v
			}
		}

		return def
	}

	for sc.Scan() {
		f := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")

		id := num(f, "Id", -1)
		if id < 0 || id > 132 {
			continue
		}

		for k := 0; k < 8; k++ {
			vis := num(f, fmt.Sprintf("Vis%d", k), 0)
			if vis == 0 {
				continue
			}

			e := [3]int{id, vis, num(f, fmt.Sprintf("Warp%d", k), -1)}
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}

	return out, sc.Err()
}

// Warp ids of LvlWarp.txt that matter for outdoor entrances.
const (
	WarpWildernessToCaveMax = 3 // ids 0..3: "Act 1 Wilderness to Cave Cliff/Floor L/R"
	WarpCaveDown            = 5 // "Act 1 Cave Down"
)

// CaveEntranceDestination returns the level a wilderness level's cave entrance
// leads to: the target of its "Wilderness to Cave" slot (LvlWarp ids 0..3).
// The cave entrance presets (Act1/Caves/DenEnt*.ds1 ...) come in variants whose
// special tile carries different styles (5 and 6 seen), so the tile is
// recognised by its preset and resolved here.
func CaveEntranceDestination(level int) (int, bool) {
	for _, l := range allLinks {
		if l.From == level && l.Kind == KindTile && l.Warp >= 0 && l.Warp <= WarpWildernessToCaveMax {
			return l.To, true
		}
	}

	return 0, false
}

// upWarps are the LvlWarp ids that lead up (towards the town); the other
// ids of a dungeon level lead down.
var upWarps = map[int]bool{4: true, 8: true, 11: true, 13: true, 16: true, 17: true,
	// Act 2: sewers 21/22, palace 25..27, arcane 30/31, tombs and the Valley of
	// the Kings exits 45, maggot lair 48 (the "up" slots of Levels.txt rows 47..72)
	21: true, 22: true, 25: true, 26: true, 27: true, 30: true, 31: true, 45: true, 48: true}

// isOutdoor reports a level that borders others on seamless edges.
func isOutdoor(level int) bool { return len(EdgeNeighbors(level)) > 0 }

// TileDestination is Destination for the special tile of a DS1 preset, by the
// style the tile carries. UNVERIFIED (the exe resolves the tile through a table
// that is not decoded): what the Act 1 presets show is
//
//   - outdoor levels: the cave entrance presets (Act1/Caves/DenEnt.ds1 and its
//     siblings) carry style 5, "Cave Down", while Levels.txt lists the
//     wilderness slots with the ids 0..3 ("Wilderness to Cave"); a style 5 tile
//     leads to the slot's level. Other styles are matched against the LvlWarp
//     id of the slots (Burial Grounds: 6 and 7);
//   - dungeon levels: style 0 is the "up" exit of the entry stairs, style 1 the second
//     up exit when the level has one (level 10) and else the "next" stairs of
//     crypts, jail and catacombs, and style 4 is the cave "down" exit (4+n for the next
//     ones). The LvlWarp id is NOT the style there (style 4 leads down,
//     LvlWarp 4 is "Cave Up").
func TileDestination(level, style int) (int, bool) {
	if isOutdoor(level) {
		if to, ok := Destination(level, style); ok {
			return to, true
		}

		if style != WarpCaveDown {
			return 0, false
		}

		for _, l := range allLinks {
			if l.From == level && l.Kind == KindTile && l.Warp >= 0 && l.Warp <= WarpWildernessToCaveMax {
				return l.To, true
			}
		}

		return 0, false
	}

	var ups, downs []int

	for _, l := range allLinks {
		if l.From != level || l.Kind != KindTile || l.Warp < 0 {
			continue
		}

		if upWarps[l.Warp] {
			ups = append(ups, l.To)
		} else {
			downs = append(downs, l.To)
		}
	}

	switch {
	case style == 0 && len(ups) > 0:
		return ups[0], true
	case style == 1 && len(ups) > 1:
		return ups[1], true // the second up exit (Underground Passage level 1 leads to both its outdoor ends)
	case style == 1 && len(downs) > 0:
		return downs[0], true // the "next" stairs of crypts, jail and catacombs
	case style >= downStyleBase && style-downStyleBase < len(downs):
		return downs[style-downStyleBase], true
	}

	return 0, false
}

// downStyleBase is the tile style of the first "down" exit of a dungeon level.
const downStyleBase = 4

// SingleTileDestination returns the level that every tile link of a level
// leads to, when it has links to one level only (the desert levels of Act 2:
// Rocky Waste 41 -> Stony Tomb 55, Dry Hills 42 -> Halls of the Dead 56, ...).
func SingleTileDestination(level int) (int, bool) {
	to := 0

	for _, l := range allLinks {
		if l.From != level || l.Kind != KindTile {
			continue
		}

		if to != 0 && l.To != to {
			return 0, false
		}

		to = l.To
	}

	return to, to != 0
}
