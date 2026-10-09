// Package d2automap is the pure (display-free) part of the Diablo II automap:
// the AutoMap table (txt and the compiled automap.bin), the lookup from a map
// tile to an automap cell, the isometric projection and the reveal model.
//
// What is VERIFIED against Game.exe 1.14b (UI\automap.cpp, D2Common LvlTbls.cpp,
// found with Ghidra, see d2-re-notes) and what is not is marked in each file.
// Short version of the real algorithm:
//
//   - automap.bin: u32 row count, then 44-byte rows: LevelName[16], TileName[8],
//     Style u8, StartSequence u8, EndSequence u8, pad u8, Cel1..Cel4 i32.
//     0xFF in a byte column and -1 in a cel mean "any" and "unused". (VERIFIED)
//   - The level name is "<act> <LvlTypes name>" (index = LvlTypes.txt Id) and the
//     tile name indexes a fixed table (fl wl wr wtlr ... = DT1 orientation).
//     A tile matches the first row of the level whose tile name equals the
//     tile's orientation, whose style is 0xFF or equal to the tile's main index
//     and whose sequence range holds the tile's sub index. One of the (up to
//     four) cels of the row is picked at random from the level seed. (VERIFIED,
//     except that the orientation is the raw DT1 value: strongly suggested)
//   - A cell is drawn at ((tx-ty)*8, (tx+ty)*4) from the 16x32 frame of
//     MaxiMap.dc6 (MaxiMapS.dc6 for the 8x16 mini map). (VERIFIED)
package d2automap

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Any is the value of a Style, StartSequence or EndSequence that matches everything
// (0xFF in automap.bin, -1 in AutoMap.txt).
const Any = -1

// NumLevelTypes is the number of level names the engine knows (index 0 is "None").
const NumLevelTypes = 36

// levelNames are the engine's LevelName strings, indexed by LvlTypes.txt Id
// (read from Game.exe at 0x6e9300, VERIFIED).
var levelNames = [NumLevelTypes]string{
	"None",
	"1 Town", "1 Wilderness", "1 Cave", "1 Crypt", "1 Monestary", "1 Courtyard", "1 Barracks", "1 Jail",
	"1 Cathedral", "1 Catacombs", "1 Tristram",
	"2 Town", "2 Sewer", "2 Harem", "2 Basement", "2 Desert", "2 Tomb", "2 Lair", "2 Arcane",
	"3 Town", "3 Jungle", "3 Kurast", "3 Spider", "3 Dungeon", "3 Sewer",
	"4 Town", "4 Mesa", "4 Lava",
	"5 Town", "5 Siege", "5 Barricade", "5 Temple", "5 Ice", "5 Baal", "5 Lava",
}

// tileNames are the engine's TileName strings, indexed by tile orientation
// (read from Game.exe at 0x6e9540, VERIFIED as a table; that the index equals the
// DT1 orientation is probable).
var tileNames = [...]string{
	"fl", "wl", "wr", "wtlr", "wtll", "wtr", "wbl", "wbr", "wld", "wrd",
	"wle", "wre", "co", "sh", "tr", "rf", "ld", "rd", "fd", "fi",
}

// LevelIndex returns the LvlTypes id of an AutoMap level name, or -1.
func LevelIndex(name string) int {
	name = strings.TrimSpace(name)
	for i, n := range levelNames {
		if strings.EqualFold(n, name) {
			return i
		}
	}

	return -1
}

// LevelName returns the AutoMap level name of a LvlTypes id.
func LevelName(level int) string {
	if level < 0 || level >= NumLevelTypes {
		return ""
	}

	return levelNames[level]
}

// TileIndex returns the tile orientation of an AutoMap tile name, or -1.
func TileIndex(name string) int {
	name = strings.TrimSpace(name)
	for i, n := range tileNames {
		if strings.EqualFold(n, name) {
			return i
		}
	}

	return -1
}

// Row is one row of AutoMap.txt.
type Row struct {
	Level int // LvlTypes id
	Tile  int // tile orientation
	Style int // Any or the tile's main index
	// Start and End are the sequence (sub index) range, Any when unrestricted.
	Start, End int
	// Cels are the MaxiMap.dc6 frames to choose from (the used leading ones).
	Cels []int
}

// Table is a loaded AutoMap table.
type Table struct {
	rows []Row
	// first/last are the half-open row range of each level. Like the engine, the
	// last contiguous block of a level wins when a level occurs in several.
	first, last [NumLevelTypes]int
}

// Rows returns all rows in file order.
func (t *Table) Rows() []Row { return t.rows }

// LevelRows returns the rows searched for a level.
func (t *Table) LevelRows(level int) []Row {
	if level < 0 || level >= NumLevelTypes || t.first[level] < 0 {
		return nil
	}

	return t.rows[t.first[level]:t.last[level]]
}

func newTable(rows []Row) *Table {
	t := &Table{rows: rows}
	for i := range t.first {
		t.first[i], t.last[i] = -1, -1
	}

	for i := 0; i < len(rows); {
		lvl := rows[i].Level
		j := i + 1

		for j < len(rows) && rows[j].Level == lvl {
			j++
		}

		t.first[lvl], t.last[lvl] = i, j
		i = j
	}

	return t
}

// Lookup returns the first row for the tile, or nil. level is the LvlTypes id of
// the level, tile the orientation, style the DT1 main index, seq the sub index.
func (t *Table) Lookup(level, tile, style, seq int) *Row {
	rows := t.LevelRows(level)
	for i := range rows {
		r := &rows[i]
		if r.Tile != tile {
			continue
		}

		if r.Style != Any && r.Style != style {
			continue
		}

		if r.Start != Any && (seq < r.Start || seq > r.End) {
			continue
		}

		return r
	}

	return nil
}

// Cel picks the MaxiMap frame for a tile; pick is a random number (the engine
// uses the level's seed; callers pass a hash of the tile position). ok is false
// when no row matches (the tile has no automap graphic).
func (t *Table) Cel(level, tile, style, seq int, pick uint32) (cel int, ok bool) {
	r := t.Lookup(level, tile, style, seq)
	if r == nil || len(r.Cels) == 0 {
		return 0, false
	}

	return r.Cels[int(pick%uint32(len(r.Cels)))], true
}

// ParseBin reads the compiled automap.bin.
func ParseBin(b []byte) (*Table, error) {
	const rowSize = 44

	if len(b) < 4 {
		return nil, errors.New("automap.bin: too short")
	}

	n := int(binary.LittleEndian.Uint32(b))
	if n < 0 || len(b) < 4+n*rowSize {
		return nil, fmt.Errorf("automap.bin: %d rows need %d bytes, have %d", n, 4+n*rowSize, len(b))
	}

	rows := make([]Row, 0, n)

	for i := 0; i < n; i++ {
		rec := b[4+i*rowSize : 4+(i+1)*rowSize]

		lvl := LevelIndex(cString(rec[0:16]))
		tile := TileIndex(cString(rec[16:24]))

		if lvl < 0 || tile < 0 {
			return nil, fmt.Errorf("automap.bin row %d: unknown level %q or tile %q", i, cString(rec[0:16]), cString(rec[16:24]))
		}

		r := Row{Level: lvl, Tile: tile, Style: byteOrAny(rec[24]), Start: byteOrAny(rec[25]), End: byteOrAny(rec[26])}

		for c := 0; c < 4; c++ {
			v := int(int32(binary.LittleEndian.Uint32(rec[28+c*4:])))
			if v == -1 {
				break // the engine counts the cels up to the first -1
			}

			r.Cels = append(r.Cels, v)
		}

		if len(r.Cels) == 0 {
			return nil, fmt.Errorf("automap.bin row %d has no cel", i)
		}

		rows = append(rows, r)
	}

	return newTable(rows), nil
}

// ParseTxt reads AutoMap.txt (tab separated, header line first). The line
// "Expansion" that separates the classic and the Lord of Destruction rows is
// skipped, as the compiler of automap.bin does.
func ParseTxt(r io.Reader) (*Table, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var col map[string]int

	var rows []Row

	line := 0

	for sc.Scan() {
		line++

		text := strings.TrimRight(sc.Text(), "\r\n")
		f := strings.Split(text, "\t")

		if col == nil {
			col = map[string]int{}

			for i, h := range f {
				if _, dup := col[h]; !dup {
					col[h] = i
				}
			}

			for _, need := range []string{"LevelName", "TileName", "Style", "StartSequence", "EndSequence", "Cel1", "Cel2", "Cel3", "Cel4"} {
				if _, ok := col[need]; !ok {
					return nil, fmt.Errorf("AutoMap.txt: no %s column", need)
				}
			}

			continue
		}

		if strings.TrimSpace(f[0]) == "" || strings.EqualFold(strings.TrimSpace(f[0]), "Expansion") {
			continue
		}

		get := func(name string) string {
			if i := col[name]; i < len(f) {
				return strings.TrimSpace(f[i])
			}

			return ""
		}

		lvl, tile := LevelIndex(get("LevelName")), TileIndex(get("TileName"))
		if lvl < 0 || tile < 0 {
			return nil, fmt.Errorf("AutoMap.txt line %d: unknown level %q or tile %q", line, get("LevelName"), get("TileName"))
		}

		row := Row{Level: lvl, Tile: tile, Style: numOrAny(get("Style")), Start: numOrAny(get("StartSequence")), End: numOrAny(get("EndSequence"))}

		for _, c := range []string{"Cel1", "Cel2", "Cel3", "Cel4"} {
			v := numOrAny(get(c))
			if v == Any {
				break
			}

			row.Cels = append(row.Cels, v)
		}

		if len(row.Cels) == 0 {
			return nil, fmt.Errorf("AutoMap.txt line %d has no cel", line)
		}

		rows = append(rows, row)
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	if col == nil {
		return nil, errors.New("AutoMap.txt: empty")
	}

	return newTable(rows), nil
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}

	return string(b)
}

func byteOrAny(b byte) int {
	if b == 0xff {
		return Any
	}

	return int(b)
}

func numOrAny(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return Any
	}

	return v
}
