package d2statlist

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

// BaseInfo is what armor.txt and weapons.txt say about a base item.
type BaseInfo struct {
	Weapon    *WeaponBase // nil for armor and misc items
	BaseBlock int         // armor.txt "block" (shields)
	MinAC     int
	MaxAC     int
}

// Bases maps item codes to their base data.
type Bases map[string]BaseInfo

// ParseBases reads armor.txt and weapons.txt.
func ParseBases(armor, weapons []byte) (Bases, error) {
	b := Bases{}

	rows, col, err := readTSV(armor)
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		code := tsvCell(r, col, "code")
		if code == "" {
			continue
		}

		b[code] = BaseInfo{
			BaseBlock: tsvInt(r, col, "block"),
			MinAC:     tsvInt(r, col, "minac"),
			MaxAC:     tsvInt(r, col, "maxac"),
		}
	}

	rows, col, err = readTSV(weapons)
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		code := tsvCell(r, col, "code")
		if code == "" {
			continue
		}

		w := &WeaponBase{
			Min: tsvInt(r, col, "mindam"), Max: tsvInt(r, col, "maxdam"),
			TwoMin: tsvInt(r, col, "2handmindam"), TwoMax: tsvInt(r, col, "2handmaxdam"),
			TwoHanded: tsvInt(r, col, "2handed") != 0,
			StrBonus:  tsvInt(r, col, "strbonus"), DexBonus: tsvInt(r, col, "dexbonus"),
		}

		if w.Max == 0 && w.Min == 0 {
			// ranged and throwing weapons keep their damage in the missile columns
			w.Min, w.Max = tsvInt(r, col, "minmisdam"), tsvInt(r, col, "maxmisdam")
			w.Ranged = true
		}

		b[code] = BaseInfo{Weapon: w}
	}

	return b, nil
}

func readTSV(data []byte) ([][]string, map[string]int, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	if len(all) < 2 {
		return nil, nil, errors.New("d2statlist: empty table")
	}

	col := map[string]int{}

	for i, h := range all[0] {
		h = strings.ToLower(strings.TrimSpace(h))
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	return all[1:], col, nil
}

func tsvCell(row []string, col map[string]int, name string) string {
	if i, ok := col[name]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}

	return ""
}

func tsvInt(row []string, col map[string]int, name string) int {
	v, _ := strconv.Atoi(tsvCell(row, col, name))

	return v
}
