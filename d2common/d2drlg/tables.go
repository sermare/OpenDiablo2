package d2drlg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Difficulty indexes the per-difficulty columns (normal, nightmare, hell).
type Difficulty int

// Difficulties.
const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

// LevelRec is the subset of a Levels.txt row the generator reads.
type LevelRec struct {
	Name             string
	ID, Act          int
	SizeX, SizeY     [3]int
	OffsetX, OffsetY int
	Depend           int
	DrlgType         int // 1 maze, 2 preset, 3 outdoor
	LevelType        int // LvlTypes id
	Vis              [8]int
	Warp             [8]int
	// SubType, SubTheme, SubWaypoint and SubShrine are the LvlSub groups used
	// for the sub-themes and the waypoint/shrine patterns of outdoor rooms.
	SubType, SubTheme, SubWaypoint, SubShrine int
}

// MazeRec is a LvlMaze.txt row.
type MazeRec struct {
	Name         string
	Level        int
	Rooms        [3]int // Rooms, Rooms(N), Rooms(H)
	SizeX, SizeY int
	Merge        int
}

// PrestRec is a LvlPrest.txt row (or compiled record).
type PrestRec struct {
	Name         string
	Def          int
	LevelID      int
	SizeX, SizeY int
	Files        int
	File         [6]string
	Dt1Mask      int
	// Populate, Logicals, Outdoors, Animate, KillEdge and FillBlanks are the
	// LvlPrest columns of the same name.
	Populate, Logicals, Outdoors, Animate, KillEdge, FillBlanks int
}

// LvlTypeRec is a LvlTypes.txt row.
type LvlTypeRec struct {
	Name  string
	ID    int
	Files []string
	// Slots are the 32 File columns by position ("0" and empty cells are
	// ""); the room tile library addresses them by bit index (File k+1 is
	// slot k).
	Slots [32]string
}

// SubRec is a LvlSub.txt row.
type SubRec struct {
	Name     string
	Type     int
	File     string
	CheckAll int
	BordType int
	GridSize int
	Dt1Mask  int
	Prob     [5]int
	Trials   [5]int
	Max      [5]int
}

// Levels gives access to Levels.txt.
type Levels interface {
	Level(id int) (LevelRec, bool)
}

// LvlMaze gives access to LvlMaze.txt, keyed by level id.
type LvlMaze interface {
	Maze(levelID int) (MazeRec, bool)
}

// LvlPrest gives access to LvlPrest, keyed by the Def column (not the row).
type LvlPrest interface {
	PrestByDef(def int) (PrestRec, bool)
}

// LvlTypes gives access to LvlTypes.txt.
type LvlTypes interface {
	LvlType(id int) (LvlTypeRec, bool)
}

// LvlSub gives access to LvlSub.txt rows of one Type, in file order.
type LvlSub interface {
	SubRows(typ int) []SubRec
}

// Source bundles all tables the generators need.
type Source interface {
	Levels
	LvlMaze
	LvlPrest
	LvlTypes
	LvlSub
}

// Raw holds the unparsed bytes of the table files. Any may be nil.
type Raw struct {
	Levels, LvlMaze, LvlPrest, LvlPrestBin, LvlTypes, LvlSub []byte
	// LvlWarp is the expansion LvlWarp.txt (optional; the outdoor tile build
	// needs it for the cave entrance tiles).
	LvlWarp []byte
}

// WarpRec is a LvlWarp.txt row: the tile description of a level exit.
type WarpRec struct {
	ID         int
	LitVersion int
	Tiles      int
	// Dir is the Direction column: 'b' both, 'l' or 'r'.
	Dir byte
}

// LvlWarps gives access to LvlWarp.txt. Tables implements it; it is a
// separate interface so other Source implementations need not.
type LvlWarps interface {
	WarpRec(id int) (WarpRec, bool)
}

// Tables is the in-memory implementation of Source.
type Tables struct {
	levels map[int]LevelRec
	mazes  map[int]MazeRec
	prest  map[int]PrestRec
	types  map[int]LvlTypeRec
	subs   map[int][]SubRec
	warps  map[int]WarpRec
	// PrestN is the number of LvlPrest records loaded.
	PrestN int
}

// Level implements Levels.
func (t *Tables) Level(id int) (LevelRec, bool) { r, ok := t.levels[id]; return r, ok }

// Maze implements LvlMaze.
func (t *Tables) Maze(id int) (MazeRec, bool) { r, ok := t.mazes[id]; return r, ok }

// PrestByDef implements LvlPrest.
func (t *Tables) PrestByDef(def int) (PrestRec, bool) { r, ok := t.prest[def]; return r, ok }

// LvlType implements LvlTypes.
func (t *Tables) LvlType(id int) (LvlTypeRec, bool) { r, ok := t.types[id]; return r, ok }

// WarpRec implements LvlWarps.
func (t *Tables) WarpRec(id int) (WarpRec, bool) { r, ok := t.warps[id]; return r, ok }

// SubRows implements LvlSub.
func (t *Tables) SubRows(typ int) []SubRec { return t.subs[typ] }

type txtTable struct {
	col  map[string]int
	rows [][]string
}

func parseTxt(data []byte) (*txtTable, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		return nil, errors.New("d2drlg: empty table")
	}

	t := &txtTable{col: map[string]int{}}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := t.col[h]; !dup {
			t.col[h] = i
		}
	}

	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}

		t.rows = append(t.rows, strings.Split(l, "\t"))
	}

	return t, nil
}

func (t *txtTable) str(r []string, name string) string {
	i, ok := t.col[name]
	if !ok || i >= len(r) {
		return ""
	}

	return r[i]
}

func (t *txtTable) num(r []string, name string) int {
	v, err := strconv.Atoi(strings.TrimSpace(t.str(r, name)))
	if err != nil {
		return 0
	}

	return v
}

func (t *txtTable) has(r []string, name string) bool {
	_, err := strconv.Atoi(strings.TrimSpace(t.str(r, name)))
	return err == nil
}

// The 1.14b lvlprest.bin is a uint32 record count followed by 432 byte
// records (verified against the txt for every non-separator row of patch_d2).
const (
	binRecordSize = 432
	binFilesOff   = 64
	binFile1Off   = 68
	binFileLen    = 60
	binDt1Off     = 428
)

// ParseLvlPrestBin decodes the compiled 1.14b lvlprest.bin (no names).
func ParseLvlPrestBin(data []byte) ([]PrestRec, error) {
	if len(data) < 4 {
		return nil, errors.New("d2drlg: lvlprest.bin too short")
	}

	n := int(binary.LittleEndian.Uint32(data))
	if len(data) != 4+n*binRecordSize {
		return nil, fmt.Errorf("d2drlg: lvlprest.bin size %d does not match %d records of %d bytes", len(data), n, binRecordSize)
	}

	out := make([]PrestRec, n)

	for i := range out {
		r := data[4+i*binRecordSize : 4+(i+1)*binRecordSize]
		u := func(off int) int { return int(int32(binary.LittleEndian.Uint32(r[off:]))) }
		p := PrestRec{Def: u(0), LevelID: u(4), SizeX: u(40), SizeY: u(44), Files: u(binFilesOff), Dt1Mask: u(binDt1Off), Populate: u(8), Logicals: u(0xc), Outdoors: u(0x10),
			Animate: u(0x14), KillEdge: u(0x18), FillBlanks: u(0x1c)}

		for k := 0; k < 6; k++ {
			s := r[binFile1Off+k*binFileLen : binFile1Off+(k+1)*binFileLen]
			if z := strings.IndexByte(string(s), 0); z >= 0 {
				s = s[:z]
			}

			if str := string(s); str != "0" {
				p.File[k] = str
			}
		}

		out[i] = p
	}

	return out, nil
}

func parsePrestTxt(data []byte) ([]PrestRec, error) {
	t, err := parseTxt(data)
	if err != nil {
		return nil, err
	}

	var out []PrestRec

	for _, r := range t.rows {
		if !t.has(r, "Def") { // "Expansion" separator row
			continue
		}

		p := PrestRec{Name: t.str(r, "Name"), Def: t.num(r, "Def"), LevelID: t.num(r, "LevelId"),
			SizeX: t.num(r, "SizeX"), SizeY: t.num(r, "SizeY"), Files: t.num(r, "Files"), Dt1Mask: t.num(r, "Dt1Mask"),
			Populate: t.num(r, "Populate"), Logicals: t.num(r, "Logicals"), Outdoors: t.num(r, "Outdoors"),
			Animate: t.num(r, "Animate"), KillEdge: t.num(r, "KillEdge"), FillBlanks: t.num(r, "FillBlanks")}

		for k := 0; k < 6; k++ {
			if f := t.str(r, "File"+strconv.Itoa(k+1)); f != "0" {
				p.File[k] = f
			}
		}

		out = append(out, p)
	}

	return out, nil
}

// Load builds Tables from raw bytes. LvlPrestBin, when given, overrides the
// numeric and file columns of the txt (it is authoritative); names come from
// the txt when the record counts agree.
func Load(raw Raw) (*Tables, error) {
	t := &Tables{levels: map[int]LevelRec{}, mazes: map[int]MazeRec{}, prest: map[int]PrestRec{},
		types: map[int]LvlTypeRec{}, subs: map[int][]SubRec{}, warps: map[int]WarpRec{}}

	if raw.Levels != nil {
		lt, err := parseTxt(raw.Levels)
		if err != nil {
			return nil, err
		}

		for _, r := range lt.rows {
			if !lt.has(r, "Id") || lt.str(r, "Name") == "Null" {
				continue
			}

			l := LevelRec{Name: lt.str(r, "Name"), ID: lt.num(r, "Id"), Act: lt.num(r, "Act"),
				OffsetX: lt.num(r, "OffsetX"), OffsetY: lt.num(r, "OffsetY"), Depend: lt.num(r, "Depend"),
				DrlgType: lt.num(r, "DrlgType"), LevelType: lt.num(r, "LevelType")}
			l.SizeX = [3]int{lt.num(r, "SizeX"), lt.num(r, "SizeX(N)"), lt.num(r, "SizeX(H)")}
			l.SizeY = [3]int{lt.num(r, "SizeY"), lt.num(r, "SizeY(N)"), lt.num(r, "SizeY(H)")}

			for i := range l.Vis {
				l.Vis[i] = lt.num(r, "Vis"+strconv.Itoa(i))
				l.Warp[i] = lt.num(r, "Warp"+strconv.Itoa(i))
			}

			l.SubType, l.SubTheme = lt.num(r, "SubType"), lt.num(r, "SubTheme")
			l.SubWaypoint, l.SubShrine = lt.num(r, "SubWaypoint"), lt.num(r, "SubShrine")

			t.levels[l.ID] = l
		}
	}

	if raw.LvlMaze != nil {
		mt, err := parseTxt(raw.LvlMaze)
		if err != nil {
			return nil, err
		}

		for _, r := range mt.rows {
			m := MazeRec{Name: mt.str(r, "Name"), Level: mt.num(r, "Level"),
				Rooms: [3]int{mt.num(r, "Rooms"), mt.num(r, "Rooms(N)"), mt.num(r, "Rooms(H)")},
				SizeX: mt.num(r, "SizeX"), SizeY: mt.num(r, "SizeY"), Merge: mt.num(r, "Merge")}
			if _, dup := t.mazes[m.Level]; m.Level > 0 && !dup {
				t.mazes[m.Level] = m
			}
		}
	}

	var prest []PrestRec

	if raw.LvlPrest != nil {
		var err error
		if prest, err = parsePrestTxt(raw.LvlPrest); err != nil {
			return nil, err
		}
	}

	if raw.LvlPrestBin != nil {
		bin, err := ParseLvlPrestBin(raw.LvlPrestBin)
		if err != nil {
			return nil, err
		}

		if len(prest) == len(bin) {
			for i := range bin {
				bin[i].Name = prest[i].Name
			}
		}

		prest = bin
	}

	for _, p := range prest {
		if _, dup := t.prest[p.Def]; !dup { // first record with a Def wins
			t.prest[p.Def] = p
		}
	}

	t.PrestN = len(prest)

	if raw.LvlTypes != nil {
		tt, err := parseTxt(raw.LvlTypes)
		if err != nil {
			return nil, err
		}

		for _, r := range tt.rows {
			lt := LvlTypeRec{Name: tt.str(r, "Name"), ID: tt.num(r, "Id")}

			for k := 1; k <= 32; k++ {
				if f := tt.str(r, "File "+strconv.Itoa(k)); f != "" && f != "0" {
					lt.Files = append(lt.Files, f)
					lt.Slots[k-1] = f
				}
			}

			t.types[lt.ID] = lt
		}
	}

	if raw.LvlWarp != nil {
		wt, err := parseTxt(raw.LvlWarp)
		if err != nil {
			return nil, err
		}

		for _, r := range wt.rows {
			if !wt.has(r, "Id") {
				continue
			}

			w := WarpRec{ID: wt.num(r, "Id"), LitVersion: wt.num(r, "LitVersion"), Tiles: wt.num(r, "Tiles")}
			if d := wt.str(r, "Direction"); d != "" {
				w.Dir = d[0]
			}

			t.warps[w.ID] = w
		}
	}

	if raw.LvlSub != nil {
		st, err := parseTxt(raw.LvlSub)
		if err != nil {
			return nil, err
		}

		for _, r := range st.rows {
			if !st.has(r, "Type") { // "Expansion" separator row
				continue
			}

			s := SubRec{Name: st.str(r, "Name"), Type: st.num(r, "Type"), File: st.str(r, "File"),
				CheckAll: st.num(r, "CheckAll"), BordType: st.num(r, "BordType"), GridSize: st.num(r, "GridSize"), Dt1Mask: st.num(r, "Dt1Mask")}

			for k := 0; k < 5; k++ {
				n := strconv.Itoa(k)
				s.Prob[k], s.Trials[k], s.Max[k] = st.num(r, "Prob"+n), st.num(r, "Trials"+n), st.num(r, "Max"+n)
			}

			t.subs[s.Type] = append(t.subs[s.Type], s)
		}
	}

	return t, nil
}
