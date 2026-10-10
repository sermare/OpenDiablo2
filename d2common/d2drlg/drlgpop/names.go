package drlgpop

import (
	"fmt"
	"strconv"
	"strings"
)

// Place categories of a monpreset.txt row (the converter at 0x65bbd0 looks
// the Place name up in three tables, in this order).
const (
	// CatPlace is a monplace.txt code (place_group50, place_fallen ...); the
	// in-memory monster id is monstatsCount + superUniqueCount + index.
	CatPlace = 0
	// CatMonster is a monstats.txt Id; the id is the class.
	CatMonster = 1
	// CatSuper is a superuniques.txt Superunique key; the id is
	// monstatsCount + index.
	CatSuper = 2
)

// PresetEntry is one resolved monpreset.txt row.
type PresetEntry struct {
	Cat int
	ID  int
}

// Names holds what the DS1 loader needs to turn the monster ids of a DS1
// file into the ids the game uses at run time: the three name tables, the
// two counts and the per-act monpreset rows.
//
//	MonstatsCount  *(DataTables+0xa80)  number of monstats rows (734 in 1.14b, including the "Expansion" separator row)
//	SuperCount     0x96474c, read by FUN_006571b0: number of superuniques rows (66)
//
// Verified against the emulator (the real loaders run with these tables).
type Names struct {
	MonstatsCount int
	SuperCount    int
	// Presets[act] are the rows of monpreset.txt of act+1 in file order.
	Presets [5][]PresetEntry
	// Place holds the monplace.txt codes in row order.
	Place []string
	// MonKeys are the monstats Id keys by class id and SuperKeys the
	// superuniques keys by index (separator rows removed, original case).
	MonKeys, SuperKeys []string
}

// ParseNames builds Names from the text tables: monstats.txt (Id column,
// rows in order), superuniques.txt (Superunique column), monplace.txt (code
// column) and monpreset.txt (Act, Place). superCount is the superuniques
// record count of the compiled table; 0 counts the non-empty keys.
func ParseNames(monstats, superuniques, monplace, monpreset []byte, superCount int) (*Names, error) {
	ms, err := readTSV(monstats)
	if err != nil {
		return nil, fmt.Errorf("monstats: %w", err)
	}

	su, err := readTSV(superuniques)
	if err != nil {
		return nil, fmt.Errorf("superuniques: %w", err)
	}

	mp, err := readTSV(monplace)
	if err != nil {
		return nil, fmt.Errorf("monplace: %w", err)
	}

	pr, err := readTSV(monpreset)
	if err != nil {
		return nil, fmt.Errorf("monpreset: %w", err)
	}

	n := &Names{SuperCount: superCount}

	// The compiled .bin tables the game reads have no "Expansion" separator
	// row of the .txt files: the indexes behind it shift down by one.
	mon := map[string]int{}
	nameIDs := 0

	for _, r := range ms.rows {
		k := strings.ToLower(strings.TrimSpace(ms.str(r, "Id")))
		if k == "expansion" {
			continue
		}

		// ids are the ordinal among distinct names (see d2monreg.ParseTables)
		if _, dup := mon[k]; k != "" && !dup {
			mon[k] = nameIDs
			nameIDs++
		}

		n.MonKeys = append(n.MonKeys, strings.TrimSpace(ms.str(r, "Id")))
		n.MonstatsCount++
	}

	sup := map[string]int{}
	nsup := 0

	for _, r := range su.rows {
		k := strings.ToLower(strings.TrimSpace(su.str(r, "Superunique")))
		if k == "expansion" || k == "" {
			continue
		}

		if _, dup := sup[k]; !dup {
			sup[k] = nsup
		}

		n.SuperKeys = append(n.SuperKeys, strings.TrimSpace(su.str(r, "Superunique")))
		nsup++
	}

	if n.SuperCount == 0 {
		n.SuperCount = nsup
	}

	place := map[string]int{}

	for i, r := range mp.rows {
		k := strings.ToLower(strings.TrimSpace(mp.str(r, "code")))
		if k != "" {
			n.Place = append(n.Place, k)

			if _, dup := place[k]; !dup {
				place[k] = i
			}
		}
	}

	for _, r := range pr.rows {
		act, err := strconv.Atoi(strings.TrimSpace(pr.str(r, "Act")))
		if err != nil || act < 1 || act > 5 {
			continue
		}

		name := strings.ToLower(strings.TrimSpace(pr.str(r, "Place")))
		// a name found nowhere keeps the converter's zeroed record: place id 0
		e := PresetEntry{Cat: CatPlace, ID: 0}

		// the converter tries the tables in this order and the first hit wins
		if i, ok := sup[name]; ok {
			e = PresetEntry{CatSuper, i}
		} else if i, ok := mon[name]; ok {
			e = PresetEntry{CatMonster, i}
		} else if i, ok := place[name]; ok {
			e = PresetEntry{CatPlace, i}
		}

		n.Presets[act-1] = append(n.Presets[act-1], e)
	}

	return n, nil
}

type tsv struct {
	col  map[string]int
	rows [][]string
}

func readTSV(data []byte) (*tsv, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
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

	return r[i]
}
