package drlgoutdoor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// dt1Tile is the part of a DT1 tile header the room tile pick reads
// (offsets in the 0x60 byte header: material 6, orientation 0x14, main index
// 0x18, sub index 0x1c, rarity 0x20).
type dt1Tile struct {
	orient, style, seq, rarity int32
	material                   uint16
}

type dt1Key struct{ orient, style, seq int32 }

// DT1 is a tile file reduced to the headers and the lookup index of
// TILE_AddDt1ToLibrary (0x603b10): tiles are filed under (orientation, style,
// sequence); a query returns the tiles of one key most recently loaded first,
// that is in reverse file order (0x60c2f0 / 0x60c290 push at the head).
type DT1 struct {
	Name  string
	tiles []dt1Tile
	idx   map[dt1Key][]int32
}

// ParseDT1 reads the tile headers of a DT1 file.
func ParseDT1(name string, b []byte) (*DT1, error) {
	if len(b) < 0x114 || binary.LittleEndian.Uint32(b) != 7 {
		return nil, errors.New("drlgoutdoor: not a version 7 DT1 file")
	}

	n := int(binary.LittleEndian.Uint32(b[0x10c:]))
	off := int(binary.LittleEndian.Uint32(b[0x110:]))

	if n < 0 || off < 0 || off+n*0x60 > len(b) {
		return nil, fmt.Errorf("drlgoutdoor: %s: bad tile table", name)
	}

	d := &DT1{Name: name, tiles: make([]dt1Tile, n), idx: map[dt1Key][]int32{}}

	for i := 0; i < n; i++ {
		h := b[off+0x60*i:]
		t := dt1Tile{
			material: binary.LittleEndian.Uint16(h[6:]),
			orient:   int32(binary.LittleEndian.Uint32(h[0x14:])),
			style:    int32(binary.LittleEndian.Uint32(h[0x18:])),
			seq:      int32(binary.LittleEndian.Uint32(h[0x1c:])),
			rarity:   int32(binary.LittleEndian.Uint32(h[0x20:])),
		}
		d.tiles[i] = t
		k := dt1Key{t.orient, t.style, t.seq}
		d.idx[k] = append([]int32{int32(i)}, d.idx[k]...)
	}

	return d, nil
}

// TileRef names one tile of a loaded DT1.
type TileRef struct {
	DT  *DT1
	Idx int
}

// File is the DT1 path, lower case with forward slashes.
func (t TileRef) File() string {
	if t.DT == nil {
		return ""
	}

	return t.DT.Name
}

func (t TileRef) tile() *dt1Tile { return &t.DT.tiles[t.Idx] }

// Style, Sequence and Orientation are the lookup key of the tile.
func (t TileRef) Style() int       { return int(t.tile().style) }
func (t TileRef) Sequence() int    { return int(t.tile().seq) }
func (t TileRef) Orientation() int { return int(t.tile().orient) }
func (t TileRef) Rarity() int      { return int(t.tile().rarity) }

// Library is a room's tile library (room+0x68): up to 32 DT1 slots queried in
// load order.
type Library struct{ slots []*DT1 }

// query is TILE_QueryLibrary (0x603bd0) with a result limit.
func (lb *Library) query(orient, style, seq int32, limit int) []TileRef {
	var out []TileRef

	for _, d := range lb.slots {
		if len(out) >= limit {
			break
		}

		for _, i := range d.idx[dt1Key{orient, style, seq}] {
			if len(out) >= limit {
				break
			}

			out = append(out, TileRef{d, int(i)})
		}
	}

	return out
}

// normDT1 turns a LvlTypes file name into the cache key.
func normDT1(f string) string { return strings.ToLower(strings.ReplaceAll(f, "\\", "/")) }

// DT1 loads and caches the tile headers of a DT1 file. The file is read with
// the same loader as the DS1 patterns (rooted at data/global/tiles).
func (e *Env) DT1(file string) (*DT1, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := normDT1(file)
	if d, ok := e.dt1[key]; ok {
		return d, nil
	}

	if e.DS1 == nil {
		return nil, errors.New("drlgoutdoor: no file loader")
	}

	b, err := e.DS1(file)
	if err != nil {
		return nil, fmt.Errorf("drlgoutdoor: %s: %w", file, err)
	}

	d, err := ParseDT1(key, b)
	if err != nil {
		return nil, err
	}

	if e.dt1 == nil {
		e.dt1 = map[string]*DT1{}
	}

	e.dt1[key] = d

	return d, nil
}

// fixed tail of every room library (DRLG_LoadRoomTileLibrary, 0x671ef0).
var libraryTail = []string{"Act1/Outdoors/Blank.dt1", "Act1/Barracks/InvisWal.dt1", "Act1/Barracks/Warp.dt1"}

// RoomLibrary builds the tile library of a room from its Dt1Mask (R50): bit k
// selects the File k+1 column of the level type, then the three fixed files.
func (l *Level) RoomLibrary(r *Room) (*Library, error) {
	lt, ok := l.env.Tables.LvlType(l.LType)
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: no LvlTypes row %d", l.LType)
	}

	lb := &Library{}

	for k, m := 0, r.R50; m != 0 && k < 32; k, m = k+1, m>>1 {
		if m&1 == 0 {
			continue
		}

		f := lt.Slots[k]
		if f == "" {
			return nil, fmt.Errorf("drlgoutdoor: LvlTypes %d has no file in slot %d", l.LType, k)
		}

		d, err := l.env.DT1(f)
		if err != nil {
			return nil, err
		}

		lb.slots = append(lb.slots, d)
	}

	for _, f := range libraryTail {
		d, err := l.env.DT1(f)
		if err != nil {
			return nil, err
		}

		lb.slots = append(lb.slots, d)
	}

	return lb, nil
}
