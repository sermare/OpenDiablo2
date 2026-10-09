package d2statlist

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

// StatDef is the part of an ItemStatCost row the stat list needs.
type StatDef struct {
	ID       int
	Name     string
	ValShift int // value << ValShift is how the game stores it (8 for life/mana/stamina)
	Op       int // "op" column: how the stat feeds another stat (see Defs.PerLevel)
	OpParam  int
	OpStats  []string // "op stat1..3" names
}

// Defs is the parsed ItemStatCost table.
type Defs struct {
	byID   map[int]StatDef
	byName map[string]int
}

// ParseItemStatCost reads the tab separated ItemStatCost.txt of the 1.14
// patch_d2.mpq.
func ParseItemStatCost(data []byte) (*Defs, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, errors.New("d2statlist: empty ItemStatCost")
	}

	col := map[string]int{}

	for i, h := range rows[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	for _, need := range []string{"stat", "id", "op", "op param", "op stat1"} {
		if _, ok := col[need]; !ok {
			return nil, errors.New("d2statlist: ItemStatCost lacks column " + need)
		}
	}

	d := &Defs{byID: map[int]StatDef{}, byName: map[string]int{}}
	get := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}

		return ""
	}

	for _, row := range rows[1:] {
		id, err := strconv.Atoi(get(row, "id"))
		if err != nil {
			continue
		}

		sd := StatDef{ID: id, Name: get(row, "stat")}
		sd.ValShift, _ = strconv.Atoi(get(row, "valshift"))
		sd.Op, _ = strconv.Atoi(get(row, "op"))
		sd.OpParam, _ = strconv.Atoi(get(row, "op param"))

		for _, n := range []string{"op stat1", "op stat2", "op stat3"} {
			if s := get(row, n); s != "" {
				sd.OpStats = append(sd.OpStats, s)
			}
		}

		d.byID[id] = sd
		d.byName[sd.Name] = id
	}

	return d, nil
}

// ID returns the id of a stat name, or -1.
func (d *Defs) ID(name string) int {
	if d == nil {
		return -1
	}

	if id, ok := d.byName[name]; ok {
		return id
	}

	return -1
}

// Def returns a stat's row.
func (d *Defs) Def(id int) (StatDef, bool) {
	if d == nil {
		return StatDef{}, false
	}

	sd, ok := d.byID[id]

	return sd, ok
}

// PerLevel describes a "per character level" stat: op 2, 4 or 5 in
// ItemStatCost with op param p: the value is (stat * clvl) >> p added to the
// first op stat (op 5: as a percent of it). Verified table columns; the
// formula (value*clvl)>>param is the documented meaning (UNVERIFIED in the
// binary: the handler for ops was not decompiled).
type PerLevel struct {
	Stat   int
	Target int
	Shift  int
	Pct    bool
}

// PerLevel lists every per level stat, ordered by stat id.
func (d *Defs) PerLevel() []PerLevel {
	if d == nil {
		return nil
	}

	var out []PerLevel

	for id := 0; id < 1024; id++ {
		sd, ok := d.byID[id]
		if !ok || len(sd.OpStats) == 0 {
			continue
		}

		if sd.Op != 2 && sd.Op != 4 && sd.Op != 5 {
			continue
		}

		t := d.ID(sd.OpStats[0])
		if t < 0 {
			continue
		}

		out = append(out, PerLevel{Stat: id, Target: t, Shift: sd.OpParam, Pct: sd.Op == 5})
	}

	return out
}
