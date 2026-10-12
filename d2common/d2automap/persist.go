package d2automap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Sidecar files. Game.exe 1.14b keeps the automap outside the .d2s, in the save folder next to the character:
// "<name>.map" (AUTOMAP_OpenMapFile 0x4535b0) and up to four "<name>.ma0".."<name>.ma3" (one per recent game
// id). The .map file is 24 bytes: u32 0xc (size of the rest), u32 ring index of the slot to reuse next, and four
// u32 game ids. A game id that is not in the four slots takes the slot at the ring index (the ring advances,
// that slot's .maN file is deleted); an unreadable header resets everything and starts the ring at slot 1.
// (VERIFIED from the decompile: OpenMapFile, SaveLevelLayerToFile 0x453880, LoadLevelLayerFromFile 0x453df0,
// FindOrCreateLevelLayer 0x4543e0 as the caller that reloads a level.)
//
// A .maN file starts with a 400 byte index of u32 file offsets keyed by level id (0 = level not stored), followed
// by one record per level: 8 u32 of header and the cell lists as u16 triples. VERIFIED: the index size, the u16
// triples and the three or four length-prefixed lists per level, with the level id checked on load and a cel
// number at or above a limit discarding the level. UNVERIFIED: the exact meaning of the other header fields and
// which of the two coordinate words is x. This port is NOT byte compatible with the exe's files: the index here
// has maxIndexLevels entries (the exe's 400 bytes cannot hold the Act 5 levels, 109..136, unless the exe indexes
// them differently), the header is {next=0, level, 0, 0, floorBytes, wallBytes, objectBytes, 0} and a triple is
// (cel, x, y), x and y as int16 (cells are scale 10).
const (
	headerSize     = 24
	headerTag      = 0xc
	ringSlots      = 4
	maxIndexLevels = 140
	indexSize      = maxIndexLevels * 4
	recordHeader   = 32
	tripleSize     = 6
	maxCel         = 0x7fff
)

// Sidecar names the sidecar files of one character in one game.
type Sidecar struct {
	Dir    string // the character's save folder
	Name   string // the character name (file stem)
	GameID uint32 // identifies the game (the hero's map seed)
}

// Valid reports whether the sidecar has somewhere to live.
func (s Sidecar) Valid() bool { return s.Dir != "" && s.Name != "" }

func (s Sidecar) mapPath() string { return filepath.Join(s.Dir, s.Name+".map") }

func (s Sidecar) slotPath(i int) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s.ma%d", s.Name, i))
}

type ring struct {
	next uint32
	ids  [ringSlots]uint32
}

func readRing(path string) (ring, bool) {
	var r ring

	b, err := os.ReadFile(path)
	if err != nil || len(b) != headerSize || binary.LittleEndian.Uint32(b) != headerTag {
		return r, false
	}

	r.next = binary.LittleEndian.Uint32(b[4:])
	for i := range r.ids {
		r.ids[i] = binary.LittleEndian.Uint32(b[8+4*i:])
	}

	if r.next >= ringSlots {
		return r, false
	}

	return r, true
}

func (r ring) bytes() []byte {
	b := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(b, headerTag)
	binary.LittleEndian.PutUint32(b[4:], r.next)

	for i, id := range r.ids {
		binary.LittleEndian.PutUint32(b[8+4*i:], id)
	}

	return b
}

func (r ring) find(id uint32) int {
	for i, v := range r.ids {
		if v == id {
			return i
		}
	}

	return -1
}

// slot returns the slot file of this game, or -1 when the game has none (read only, nothing is changed).
func (s Sidecar) slot() int {
	r, ok := readRing(s.mapPath())
	if !ok {
		return -1
	}

	return r.find(s.GameID)
}

// openSlot finds or assigns the slot of this game for writing, evicting like AUTOMAP_OpenMapFile.
func (s Sidecar) openSlot() (int, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return 0, err
	}

	r, ok := readRing(s.mapPath())
	if !ok {
		for i := 0; i < ringSlots; i++ {
			_ = os.Remove(s.slotPath(i))
		}

		r = ring{next: 1}
		r.ids[0] = s.GameID

		return 0, replaceFile(s.mapPath(), r.bytes())
	}

	if i := r.find(s.GameID); i >= 0 {
		return i, nil
	}

	i := int(r.next)
	r.ids[i] = s.GameID
	r.next = (r.next + 1) % ringSlots
	_ = os.Remove(s.slotPath(i))

	return i, replaceFile(s.mapPath(), r.bytes())
}

// SaveLevels merges the given levels into the game's slot file (levels stored earlier stay, the same level is
// replaced) and writes it atomically.
func (s Sidecar) SaveLevels(levels map[int]*Model) error {
	if !s.Valid() {
		return errors.New("automap sidecar: no folder or name")
	}

	i, err := s.openSlot()
	if err != nil {
		return err
	}

	all := decodeSlot(readOrNil(s.slotPath(i)))

	for id, m := range levels {
		if id < 1 || id > maxIndexLevels || m == nil {
			continue
		}

		all[id] = m.Cells()
	}

	return replaceFile(s.slotPath(i), encodeSlot(all))
}

// LoadLevel returns the stored cells of one level, nil when the game has no such level (missing, corrupt or
// evicted files all read as "nothing stored", never as an error).
func (s Sidecar) LoadLevel(level int) *Model {
	if !s.Valid() || level < 1 || level > maxIndexLevels {
		return nil
	}

	i := s.slot()
	if i < 0 {
		return nil
	}

	cells, ok := decodeSlot(readOrNil(s.slotPath(i)))[level]
	if !ok {
		return nil
	}

	m := NewModel()
	for _, c := range cells {
		m.Add(c.Kind, c.X, c.Y, c.Cel)
	}

	return m
}

func readOrNil(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return b
}

func encodeSlot(all map[int][]Cell) []byte {
	ids := make([]int, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	out := make([]byte, indexSize)

	for _, id := range ids {
		var lists [numLayers][]byte

		for _, c := range all[id] {
			if c.Kind < 0 || c.Kind >= numLayers {
				continue
			}

			t := make([]byte, tripleSize)
			binary.LittleEndian.PutUint16(t, uint16(c.Cel))
			binary.LittleEndian.PutUint16(t[2:], uint16(int16(c.X)))
			binary.LittleEndian.PutUint16(t[4:], uint16(int16(c.Y)))
			lists[c.Kind] = append(lists[c.Kind], t...)
		}

		binary.LittleEndian.PutUint32(out[(id-1)*4:], uint32(len(out)))

		h := make([]byte, recordHeader)
		binary.LittleEndian.PutUint32(h[4:], uint32(id))
		binary.LittleEndian.PutUint32(h[16:], uint32(len(lists[LayerFloor])))
		binary.LittleEndian.PutUint32(h[20:], uint32(len(lists[LayerWall])))
		binary.LittleEndian.PutUint32(h[24:], uint32(len(lists[LayerObject])))
		out = append(out, h...)

		for _, l := range lists {
			out = append(out, l...)
		}
	}

	return out
}

// decodeSlot reads a slot file. Anything out of bounds drops only the affected level.
func decodeSlot(b []byte) map[int][]Cell {
	all := map[int][]Cell{}
	if len(b) < indexSize {
		return all
	}

	for id := 1; id <= maxIndexLevels; id++ {
		off := int(binary.LittleEndian.Uint32(b[(id-1)*4:]))
		if off == 0 || off < indexSize || off > len(b)-recordHeader {
			continue
		}

		if int(binary.LittleEndian.Uint32(b[off+4:])) != id {
			continue
		}

		var n [numLayers]int

		total := 0

		for l := range n {
			n[l] = int(binary.LittleEndian.Uint32(b[off+16+4*l:]))
			if n[l]%tripleSize != 0 || n[l] > len(b) {
				total = -1

				break
			}

			total += n[l]
		}

		if total < 0 || total > len(b)-off-recordHeader {
			continue
		}

		pos := off + recordHeader
		cells := make([]Cell, 0, total/tripleSize)
		bad := false

		for l := Layer(0); l < numLayers && !bad; l++ {
			for end := pos + n[l]; pos < end; pos += tripleSize {
				cel := int(binary.LittleEndian.Uint16(b[pos:]))
				if cel >= maxCel {
					bad = true // the exe drops the level when a cel is out of range

					break
				}

				cells = append(cells, Cell{
					Cel: cel, Kind: l,
					X: int(int16(binary.LittleEndian.Uint16(b[pos+2:]))),
					Y: int(int16(binary.LittleEndian.Uint16(b[pos+4:]))),
				})
			}
		}

		if !bad {
			all[id] = cells
		}
	}

	return all
}

// replaceFile writes through a synced temporary file in the same folder and renames it over path.
func replaceFile(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}

	tmp := f.Name()

	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}

	if cerr := f.Close(); err == nil {
		err = cerr
	}

	if err == nil {
		err = os.Chmod(tmp, 0o644)
	}

	if err == nil {
		err = os.Rename(tmp, path)
	}

	if err != nil {
		_ = os.Remove(tmp)
	}

	return err
}
