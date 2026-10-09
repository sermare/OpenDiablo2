package d2cube

import (
	"strconv"
	"strings"
)

// table is a tab separated game table. Unlike d2txt it keeps the "Expansion"
// separator rows, because the numeric ids used by CubeMain (pre=, suf=) count
// every row of MagicPrefix.txt and MagicSuffix.txt.
type table struct {
	cols map[string]int
	rows [][]string
}

func parseTable(raw []byte) *table {
	text := strings.ReplaceAll(string(raw), "\r", "")
	lines := strings.Split(text, "\n")
	t := &table{cols: map[string]int{}}

	if len(lines) == 0 {
		return t
	}

	for i, h := range strings.Split(lines[0], "\t") {
		k := strings.ToLower(strings.TrimSpace(h))
		if _, dup := t.cols[k]; !dup {
			t.cols[k] = i
		}
	}

	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}

		t.rows = append(t.rows, strings.Split(l, "\t"))
	}

	return t
}

func (t *table) s(row []string, col string) string {
	i, ok := t.cols[strings.ToLower(col)]
	if !ok || i >= len(row) {
		return ""
	}

	// Excel quotes cells that contain a comma ("hpot,qty=3"); the quotes are
	// not part of the value
	v := strings.TrimSpace(row[i])
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = strings.ReplaceAll(v[1:len(v)-1], `""`, `"`)
	}

	return strings.TrimSpace(v)
}

func (t *table) n(row []string, col string) int {
	v, err := strconv.Atoi(t.s(row, col))
	if err != nil {
		return 0
	}

	return v
}

func (t *table) b(row []string, col string) bool { return t.n(row, col) == 1 }
