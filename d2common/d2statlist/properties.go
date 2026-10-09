package d2statlist

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

// PropertyEntry is one stat of a Properties.txt row.
type PropertyEntry struct {
	Func int
	Stat string
	Val  int // the "val" column (class index for function 21)
}

// PropertyTable is Properties.txt: property code -> stat entries (up to 7).
type PropertyTable struct {
	rows map[string][]PropertyEntry
	defs *Defs
}

// ParseProperties reads Properties.txt; defs resolves stat names to ids.
func ParseProperties(data []byte, defs *Defs) (*PropertyTable, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, errors.New("d2statlist: empty Properties")
	}

	col := map[string]int{}

	for i, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}

	t := &PropertyTable{rows: map[string][]PropertyEntry{}, defs: defs}
	get := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}

		return ""
	}

	for _, row := range rows[1:] {
		code := get(row, "code")
		if code == "" {
			continue
		}

		for i := 1; i <= 7; i++ {
			n := strconv.Itoa(i)
			fn, _ := strconv.Atoi(get(row, "func"+n))
			stat := get(row, "stat"+n)

			if fn == 0 && stat == "" {
				continue
			}

			val, _ := strconv.Atoi(get(row, "val"+n))
			t.rows[code] = append(t.rows[code], PropertyEntry{Func: fn, Stat: stat, Val: val})
		}
	}

	return t, nil
}

// Expand turns a property (code, param, min, max) into stat contributions.
// roll picks the value in [min,max] (nil takes max). resolveSkill maps a
// non-numeric param (a skill name) to its id; may be nil.
//
// Function meanings (itemgen.md section 2 and the Properties.txt column
// notes; functions 1-7 verified as handler addresses, the semantics of the
// rest are UNVERIFIED community documentation):
//
//	1,2,3,5,6,7,8,9,15,16  value -> stat (3 and 9 repeat the previous function)
//	10   skill tab: stat 188 with parameter class*8+tab
//	11   chance to cast on event: value = chance, parameter = skill id
//	14   sockets
//	17   param used as the value
//	21   class skills: parameter = the entry's val column
//	22   single skill: parameter = skill id
//
// Functions that produce no stat (13 durability, 20 indestructible, 23
// ethereal, 24 states, 19 charges, 12/36 random skill) return nothing.
func (t *PropertyTable) Expand(code string, param string, min, max int, roll func(min, max int) int,
	resolveSkill func(string) int) []Prop {
	if roll == nil {
		roll = func(_, max int) int { return max }
	}

	if max < min {
		min, max = max, min
	}

	p, _ := strconv.Atoi(param)
	if param != "" && resolveSkill != nil {
		if _, err := strconv.Atoi(param); err != nil {
			p = resolveSkill(param)
		}
	}

	var out []Prop

	value := roll(min, max)

	for _, e := range t.rows[code] {
		id := t.defs.ID(e.Stat)
		if id < 0 {
			continue
		}

		switch e.Func {
		case 1, 2, 3, 5, 6, 7, 8, 9, 15, 16:
			out = append(out, Prop{ID: id, Value: int64(value)})
		case 10:
			out = append(out, Prop{ID: id, Param: (p/3)*8 + p%3, Value: int64(value)})
		case 11:
			out = append(out, Prop{ID: id, Param: p, Value: int64(value)})
		case 14:
			out = append(out, Prop{ID: id, Value: int64(value)})
		case 17:
			out = append(out, Prop{ID: id, Value: int64(p)})
		case 21:
			out = append(out, Prop{ID: id, Param: e.Val, Value: int64(value)})
		case 22:
			out = append(out, Prop{ID: id, Param: p, Value: int64(value)})
		}
	}

	return out
}
