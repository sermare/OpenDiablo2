package d2monreg

import (
	"fmt"
	"strconv"
	"strings"
)

// Monster flag bits of the dword at monstats record +0xc (the column the
// loader sets per boolean column; verified from the loader's descriptor).
const (
	FlagIsSpawn      = 0
	FlagIsMelee      = 1
	FlagNoRatio      = 2
	FlagOpenDoors    = 3
	FlagSetBoss      = 4
	FlagBossXfer     = 5
	FlagBoss         = 6
	FlagPrimeEvil    = 7
	FlagNPC          = 8
	FlagInteract     = 9
	FlagInTown       = 10
	FlagLUndead      = 11
	FlagHUndead      = 12
	FlagDemon        = 13
	FlagFlying       = 14
	FlagKillable     = 15
	FlagSwitchAI     = 16
	FlagNoMultiShot  = 17
	FlagNeverCount   = 18
	FlagPetIgnore    = 19
	FlagDeathDmg     = 20
	FlagGenericSpawn = 21
	FlagZoo          = 22
	FlagPlaceSpawn   = 23
	FlagInventory    = 24
	FlagEnabled      = 25
	FlagNoAura       = 27
	FlagRangedType   = 28
)

var flagColumns = map[string]int{
	"isSpawn": FlagIsSpawn, "isMelee": FlagIsMelee, "noRatio": FlagNoRatio, "opendoors": FlagOpenDoors,
	"SetBoss": FlagSetBoss, "BossXfer": FlagBossXfer, "boss": FlagBoss, "primeevil": FlagPrimeEvil, "npc": FlagNPC,
	"interact": FlagInteract, "inTown": FlagInTown, "lUndead": FlagLUndead, "hUndead": FlagHUndead, "demon": FlagDemon,
	"flying": FlagFlying, "killable": FlagKillable, "switchai": FlagSwitchAI, "nomultishot": FlagNoMultiShot,
	"neverCount": FlagNeverCount, "petIgnore": FlagPetIgnore, "deathDmg": FlagDeathDmg, "genericSpawn": FlagGenericSpawn,
	"zoo": FlagZoo, "placespawn": FlagPlaceSpawn, "inventory": FlagInventory, "enabled": FlagEnabled,
	"noaura": FlagNoAura, "rangedtype": FlagRangedType,
}

// Mon is the part of a monstats.txt row the spawn code reads (record offsets
// of the 0x1a8-byte in-memory record in the comments).
type Mon struct {
	Key         string
	BaseID      int // +2 (-1 when unset)
	NextInClass int // +4
	Ex          int // +0x18 monstats2 row
	Spawn       int // +0x20
	Minion1     int // +0x26
	Minion2     int // +0x28
	PartyMin    int // +0x2c
	PartyMax    int // +0x2d
	Rarity      int // +0x2e
	MinGrp      int // +0x2f
	MaxGrp      int // +0x30
	Sparse      int // +0x31 (sparsePopulate)
	Level       [3]int
	Flags       uint32 // +0xc
}

// Has reports a flag bit.
func (m *Mon) Has(bit int) bool { return m.Flags&(1<<uint(bit)) != 0 }

// Mon2 is the part of a monstats2.txt row the spawn code reads.
type Mon2 struct {
	// Counts are the numbers of variants per component (HDv ... S8v: the
	// number of comma separated codes, at most 12); Bits is the byte at +0x25
	// the loader computes (sum of count-1 over the components with more than
	// one variant, at most 0xfe).
	Counts   [16]int
	Bits     int
	SizeX    int // +8 (collision radius)
	SizeY    int // +9
	SpawnCol int // +0xa
}

// Level is the part of a Levels.txt row the spawn code reads (offsets of the
// 0x220-byte record).
type Level struct {
	ID, Act     int
	WarpDist    int       // +0xc
	MonLvl      [2][3]int // +0x10 (normal game), +0x16 (expansion)
	MonDen      [3]int    // +0x1c
	MonUMin     [3]int    // +0x28
	MonUMax     [3]int    // +0x2b
	MonWndr     int       // +0x2e
	MonSpcWalk  int       // +0x2f
	Quest       int       // +0x30
	RangedSpawn int       // +0x31
	NumMon      int       // +0x32
	// Mon, NMon and UMon are the 25 raw entries of mon1.., nmon1.., umon1..
	// (-1 for empty); the counts are the numbers of valid entries (+0x33,
	// +0x34, +0x35).
	Mon, NMon, UMon                [25]int
	CountMon, CountNMon, CountUMon int
	ObjGrp, ObjPrb                 [8]int
}

// Tables bundles the three tables.
type Tables struct {
	Mons   []Mon
	Mon2s  []Mon2
	Levels []*Level // index = level id (nil when absent)
	// LevelCount is the number of Levels records (DataTables+0xc5c).
	LevelCount int
	byKey      map[string]int
}

// MonByKey returns the class id of a monstats Id (-1 when unknown).
func (t *Tables) MonByKey(k string) int {
	if i, ok := t.byKey[strings.ToLower(strings.TrimSpace(k))]; ok {
		return i
	}

	return -1
}

type tsv struct {
	col  map[string]int
	rows [][]string
}

func readTSV(b []byte) (*tsv, error) {
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("empty table")
	}

	t := &tsv{col: map[string]int{}}

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

func (t *tsv) str(r []string, name string) string {
	i, ok := t.col[name]
	if !ok || i >= len(r) {
		return ""
	}

	return strings.TrimSpace(r[i])
}

func (t *tsv) num(r []string, name string) int {
	v, err := strconv.Atoi(t.str(r, name))
	if err != nil {
		f, ferr := strconv.ParseFloat(t.str(r, name), 64)
		if ferr != nil {
			return 0
		}

		return int(f)
	}

	return v
}

// ParseTables builds the tables from monstats.txt, monstats2.txt and
// Levels.txt. The "Expansion" separator rows of the text files are dropped
// (the compiled tables the game reads do not have them).
func ParseTables(monstats, monstats2, levels []byte) (*Tables, error) {
	ms, err := readTSV(monstats)
	if err != nil {
		return nil, fmt.Errorf("monstats: %w", err)
	}

	m2, err := readTSV(monstats2)
	if err != nil {
		return nil, fmt.Errorf("monstats2: %w", err)
	}

	lv, err := readTSV(levels)
	if err != nil {
		return nil, fmt.Errorf("levels: %w", err)
	}

	t := &Tables{byKey: map[string]int{}}

	var rows [][]string

	// The class ids stored in the compiled tables (levels.bin, monstats.bin) are
	// the ordinal of the NAME among the distinct names, while the in-memory
	// record array is indexed by that id: the second "cr_lancer8" row of
	// monstats.txt (row 723) shifts every later name down by one, so that class
	// 723 is the duplicate row and "overseer6" (row 724) is class 723. The game
	// really does this (levels.bin refers to overseer6 as 723; verified against
	// the emulated loader).
	nameIDs := 0

	for _, r := range ms.rows {
		k := strings.ToLower(ms.str(r, "Id"))
		if k == "expansion" {
			continue
		}

		if _, dup := t.byKey[k]; !dup && k != "" {
			t.byKey[k] = nameIDs
			nameIDs++
		}

		rows = append(rows, r)
	}

	by2 := map[string]int{}

	var rows2 [][]string

	for _, r := range m2.rows {
		k := strings.ToLower(m2.str(r, "Id"))
		if k == "expansion" {
			continue
		}

		if _, dup := by2[k]; !dup && k != "" {
			by2[k] = len(rows2)
		}

		rows2 = append(rows2, r)
	}

	idx := func(tab map[string]int, s string) int {
		if i, ok := tab[strings.ToLower(strings.TrimSpace(s))]; ok {
			return i
		}

		return -1
	}

	for _, r := range rows {
		m := Mon{Key: ms.str(r, "Id"), BaseID: idx(t.byKey, ms.str(r, "BaseId")), NextInClass: idx(t.byKey, ms.str(r, "NextInClass")),
			Ex: idx(by2, ms.str(r, "MonStatsEx")), Spawn: idx(t.byKey, ms.str(r, "spawn")),
			Minion1: idx(t.byKey, ms.str(r, "minion1")), Minion2: idx(t.byKey, ms.str(r, "minion2")),
			PartyMin: ms.num(r, "PartyMin"), PartyMax: ms.num(r, "PartyMax"), Rarity: ms.num(r, "Rarity"),
			MinGrp: ms.num(r, "MinGrp"), MaxGrp: ms.num(r, "MaxGrp"), Sparse: ms.num(r, "sparsePopulate"),
			Level: [3]int{ms.num(r, "Level"), ms.num(r, "Level(N)"), ms.num(r, "Level(H)")}}

		for name, bit := range flagColumns {
			if ms.num(r, name) != 0 {
				m.Flags |= 1 << uint(bit)
			}
		}

		t.Mons = append(t.Mons, m)
	}

	comps := [16]string{"HDv", "TRv", "LGv", "Rav", "Lav", "RHv", "LHv", "SHv", "S1v", "S2v", "S3v", "S4v", "S5v", "S6v", "S7v", "S8v"}

	for _, r := range rows2 {
		m := Mon2{SizeX: m2.num(r, "SizeX"), SizeY: m2.num(r, "SizeY"), SpawnCol: m2.num(r, "spawnCol")}

		for i, c := range comps {
			m.Counts[i] = countCodes(m2.str(r, c))
		}

		bits := 0

		for _, c := range m.Counts {
			if c > 1 {
				bits += c - 1
			}
		}

		if bits > 0xfe {
			bits = 0xfe
		}

		m.Bits = bits
		t.Mon2s = append(t.Mon2s, m)
	}

	t.LevelCount = 0

	for _, r := range lv.rows {
		if strings.ToLower(lv.str(r, "Name")) == "expansion" || lv.str(r, "Id") == "" {
			continue
		}

		l := &Level{ID: lv.num(r, "Id"), Act: lv.num(r, "Act"), WarpDist: lv.num(r, "WarpDist"),
			MonWndr: lv.num(r, "MonWndr"), MonSpcWalk: lv.num(r, "MonSpcWalk"), Quest: lv.num(r, "Quest"),
			RangedSpawn: lv.num(r, "rangedspawn"), NumMon: lv.num(r, "NumMon")}
		l.MonLvl = [2][3]int{
			{lv.num(r, "MonLvl1"), lv.num(r, "MonLvl2"), lv.num(r, "MonLvl3")},
			{lv.num(r, "MonLvl1Ex"), lv.num(r, "MonLvl2Ex"), lv.num(r, "MonLvl3Ex")},
		}
		l.MonDen = [3]int{lv.num(r, "MonDen"), lv.num(r, "MonDen(N)"), lv.num(r, "MonDen(H)")}
		l.MonUMin = [3]int{lv.num(r, "MonUMin"), lv.num(r, "MonUMin(N)"), lv.num(r, "MonUMin(H)")}
		l.MonUMax = [3]int{lv.num(r, "MonUMax"), lv.num(r, "MonUMax(N)"), lv.num(r, "MonUMax(H)")}

		for i := 0; i < 25; i++ {
			n := strconv.Itoa(i + 1)
			l.Mon[i] = idx(t.byKey, lv.str(r, "mon"+n))
			l.NMon[i] = idx(t.byKey, lv.str(r, "nmon"+n))
			l.UMon[i] = idx(t.byKey, lv.str(r, "umon"+n))
		}

		l.CountMon, l.CountNMon, l.CountUMon = countValid(l.Mon[:]), countValid(l.NMon[:]), countValid(l.UMon[:])

		for i := 0; i < 8; i++ {
			l.ObjGrp[i] = lv.num(r, "ObjGrp"+strconv.Itoa(i))
			l.ObjPrb[i] = lv.num(r, "ObjPrb"+strconv.Itoa(i))
		}

		for len(t.Levels) <= l.ID {
			t.Levels = append(t.Levels, nil)
		}

		t.Levels[l.ID] = l
		t.LevelCount++
	}

	return t, nil
}

func countValid(a []int) int {
	n := 0

	for _, v := range a {
		if v >= 0 {
			n++
		}
	}

	return n
}

// countCodes is the converter of the *v columns: the number of comma
// separated codes (at most 12; an empty code ends the list).
func countCodes(s string) int {
	s = strings.Trim(strings.TrimSpace(s), "\"")
	if s == "" {
		return 0
	}

	n := 0

	for _, p := range strings.Split(s, ",") {
		if strings.TrimSpace(p) == "" || n >= 12 {
			break
		}

		n++
	}

	return n
}

// Level returns the Levels record of an id.
func (t *Tables) Level(id int) *Level {
	if id < 0 || id >= len(t.Levels) {
		return nil
	}

	return t.Levels[id]
}
