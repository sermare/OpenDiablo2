package d2sfx

// Row is the subset of a Sounds.txt row that drives playback. In the original
// the table is an array of 0x8e-byte records and the row ordinal equals the
// Index column (verified), so lookups are direct.
type Row struct {
	Handle    string
	Index     int
	FileName  string
	Volume    int // 0..255
	GroupSize int
	Loop      bool
	FadeIn    int // game ticks (25 per second)
	FadeOut   int // game ticks
	DeferInst bool
	StopInst  bool
	Duration  int // game ticks, 0 = until the file ends
	Compound  int // tick window for merging same-group requests, <0 = always
	Falloff   int // 0..4, selects the audible radius, see FalloffRadius
	Priority  int // 0 means the sound never plays (verified)
	AsyncOnly bool
	Stream    bool
	Stereo    bool
	Tracking  bool
	Solo      bool
	MusicVol  bool // scaled by the music volume instead of the sfx volume
}

// Table is an index-addressed set of rows with the group-head post-pass done.
type Table struct {
	rows []Row
	head []int
}

// NewTable builds a Table; each row is placed at its Index.
func NewTable(rows []Row) *Table {
	max := -1

	for i := range rows {
		if rows[i].Index > max {
			max = rows[i].Index
		}
	}

	t := &Table{rows: make([]Row, max+1), head: make([]int, max+1)}

	for i := range t.rows {
		t.rows[i].Index = i
		t.head[i] = i
	}

	for i := range rows {
		t.rows[rows[i].Index] = rows[i]
	}

	// Post-pass (verified in SOUND_LoadSoundsTxt): rows inside a Group Size span
	// record the span's first row as their group head. Dedupe/Stop/Defer compare heads.
	for i := range t.rows {
		g := t.rows[i].GroupSize
		for j := 1; j < g && i+j < len(t.rows); j++ {
			// The real Sounds.txt has members that repeat the group size (e.g. the
			// kurast night events); the earlier head keeps the claim (unverified).
			if t.head[i+j] == i+j && t.head[i] == i {
				t.head[i+j] = i
			}
		}
	}

	return t
}

// Len returns the number of rows.
func (t *Table) Len() int { return len(t.rows) }

// Get returns the row at index, or nil when out of range (the original also
// treats out-of-range ids as "no row").
func (t *Table) Get(index int) *Row {
	if index < 0 || index >= len(t.rows) {
		return nil
	}

	return &t.rows[index]
}

// Head returns the group head index of a row.
func (t *Table) Head(index int) int {
	if index < 0 || index >= len(t.head) {
		return index
	}

	return t.head[index]
}

// FalloffRadius maps the Falloff column to the audible radius (verified,
// FUN_0047e570): 0 -> 400, 1 and anything else -> 700, 2 -> 1000, 3 -> 1500,
// 4 -> 2000. Units are those of the hero-relative delta with y doubled, which
// looks like screen pixels (inferred).
func FalloffRadius(falloff int) float64 {
	switch falloff {
	case 0:
		return 400
	case 2:
		return 1000
	case 3:
		return 1500
	case 4:
		return 2000
	default:
		return 700
	}
}
