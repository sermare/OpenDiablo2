package d2txt

import (
	"bytes"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// DataDictionary represents a data file (Excel)
type DataDictionary struct {
	lookup map[string]int
	r      *csv.Reader
	record []string
	Err    error

	// OnMissing, when set, is called once per distinct header name that was
	// asked for but is not in the file. Lookups of such names still read
	// column 0 (legacy behaviour), so this is how callers find typos and
	// wrong-case names. With OD2_TXT_WARN=1 in the environment and no
	// OnMissing set, the name is logged once instead.
	OnMissing func(field string)
	missing   map[string]bool
	missingMu sync.Mutex // guards missing: lookups of absent names write it lazily
}

// Has reports whether the file has a column with exactly this header name.
func (d *DataDictionary) Has(field string) bool {
	_, ok := d.lookup[field]

	return ok
}

// Missing returns the header names that were looked up but do not exist, in
// no particular order.
func (d *DataDictionary) Missing() []string {
	d.missingMu.Lock()
	defer d.missingMu.Unlock()

	out := make([]string, 0, len(d.missing))
	for k := range d.missing {
		out = append(out, k)
	}

	return out
}

// col returns the column index for a header name. A missing name yields 0,
// as before, but is reported once.
func (d *DataDictionary) col(field string) int {
	i, ok := d.lookup[field]
	if ok {
		return i
	}

	d.missingMu.Lock()
	first := !d.missing[field]

	if first {
		if d.missing == nil {
			d.missing = map[string]bool{}
		}

		d.missing[field] = true
	}

	d.missingMu.Unlock()

	if first { // reported outside the lock so the callback may call Missing()
		switch {
		case d.OnMissing != nil:
			d.OnMissing(field)
		case os.Getenv("OD2_TXT_WARN") == "1":
			log.Printf("d2txt: header %q not found, reading column 0", field)
		}
	}

	return 0
}

// LoadDataDictionary loads the contents of a spreadsheet style txt file
func LoadDataDictionary(buf []byte) *DataDictionary {
	cr := csv.NewReader(bytes.NewReader(buf))
	cr.Comma = '\t'
	cr.ReuseRecord = true
	cr.FieldsPerRecord = -1 // real game tables have ragged rows
	cr.LazyQuotes = true

	fieldNames, err := cr.Read()
	if err != nil {
		// an empty or unreadable table: no columns and no rows, with the cause in Err. This used
		// to panic, which crashed the engine on a truncated or empty .txt.
		return &DataDictionary{lookup: map[string]int{}, r: cr, Err: err}
	}

	data := &DataDictionary{
		lookup: make(map[string]int, len(fieldNames)),
		r:      cr,
	}

	for i, name := range fieldNames {
		data.lookup[name] = i
	}

	return data
}

// Next reads the next row, skips Expansion lines or
// returns false when the end of a file is reached or an error occurred
func (d *DataDictionary) Next() bool {
	for {
		var err error

		d.record, err = d.r.Read()

		if err == io.EOF {
			return false
		} else if err != nil {
			d.Err = err
			return false
		}

		// a loop rather than recursion: a file of nothing but Expansion lines must not grow the stack
		if len(d.record) > 0 && d.record[0] == "Expansion" {
			continue
		}

		return true
	}
}

// String gets a string from the given column
func (d *DataDictionary) String(field string) string {
	i := d.col(field)
	if i >= len(d.record) {
		return ""
	}

	return d.record[i]
}

// Number gets a number for the given column
func (d *DataDictionary) Number(field string) int {
	n, err := strconv.Atoi(d.String(field))
	if err != nil {
		return 0
	}

	return n
}

// List splits a delimited list from the given column
func (d *DataDictionary) List(field string) []string {
	str := d.String(field)
	return strings.Split(str, ",")
}

// Bool gets a bool value for the given column
func (d *DataDictionary) Bool(field string) bool {
	// a column holding a number other than 0/1 used to panic here; any positive value counts as true
	return d.Number(field) > 0
}
