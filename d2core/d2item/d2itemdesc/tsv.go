package d2itemdesc

import (
	"bytes"
	"strconv"
	"strings"
)

// tsv is a tab separated game table (the .txt files of the excel folder)
// with its header row, addressed by column name (case insensitive).
type tsv struct {
	cols map[string]int
	rows [][]string
}

func parseTSV(data []byte) *tsv {
	t := &tsv{cols: make(map[string]int)}

	data = bytes.ReplaceAll(data, []byte("\r"), nil)
	lines := strings.Split(string(data), "\n")

	if len(lines) == 0 {
		return t
	}

	for i, h := range strings.Split(lines[0], "\t") {
		k := strings.ToLower(strings.TrimSpace(h))
		if _, dup := t.cols[k]; !dup && k != "" {
			t.cols[k] = i
		}
	}

	for _, l := range lines[1:] {
		if l == "" {
			continue
		}

		t.rows = append(t.rows, strings.Split(l, "\t"))
	}

	return t
}

// str returns the cell of row r in the named column ("" when absent).
func (t *tsv) str(r []string, col string) string {
	i, ok := t.cols[strings.ToLower(col)]
	if !ok || i >= len(r) {
		return ""
	}

	return strings.TrimSpace(r[i])
}

// num returns the cell as an integer (0 when absent or not a number).
func (t *tsv) num(r []string, col string) int {
	n, err := strconv.Atoi(t.str(r, col))
	if err != nil {
		return 0
	}

	return n
}

func (t *tsv) has(col string) bool {
	_, ok := t.cols[strings.ToLower(col)]
	return ok
}
