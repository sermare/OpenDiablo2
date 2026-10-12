package d2asset

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2tbl"
)

// Numeric string ids as the exe uses them (AUTOMAP_DrawAutomap 0x4564a0 and others): ids below 10000
// read string.tbl, 10000.. patchstring.tbl and 20000.. expansionstring.tbl, each by the node's own
// id minus the base; a table that lacks the node falls through to the next lower one (VERIFIED w50 from
// STRTABLE_GetStringById and the real tables: 0x58cb = expansionstring node 2731 "EXPANSION CHARACTER",
// 0x2aaf = patch_d2 patchstring node 927 "LADDER CHARACTER"; the w44 note had the two bases swapped).
const (
	stringIDPatchBase     = 10000
	stringIDExpansionBase = 20000
)

type idTable struct {
	base int
	m    d2tbl.IDStrings
}

// idTableBase gives the id offset of a table by its file name (0 for string.tbl).
func idTableBase(tablePath string) int {
	p := strings.ToLower(tablePath)

	switch {
	case strings.Contains(p, "patchstring"):
		return stringIDPatchBase
	case strings.Contains(p, "expansionstring"):
		return stringIDExpansionBase
	}

	return 0
}

// addIDTable registers the id-keyed view of a loaded table.
func (am *AssetManager) addIDTable(tablePath string, data []byte) {
	m, err := d2tbl.LoadIDStrings(data, am.Locale().Decode)
	if err != nil {
		am.Debugf("no id view of %s: %v", tablePath, err)
		return
	}

	am.idTables = append(am.idTables, idTable{base: idTableBase(tablePath), m: m})
}

// StringByID returns the text of a numeric string id (for example 0x1055, the automap "Game: " line)
// and whether it exists.
func (am *AssetManager) StringByID(id int) (string, bool) {
	best := func(base int) (string, bool) {
		for _, t := range am.idTables {
			if t.base != base {
				continue
			}

			if s, ok := t.m[id-base]; ok {
				return s, true
			}
		}

		return "", false
	}

	// highest base first, like the exe: expansion, then patch, then the base table
	for _, base := range []int{stringIDExpansionBase, stringIDPatchBase, 0} {
		if id < base {
			continue
		}

		if s, ok := best(base); ok {
			return s, true
		}
	}

	return "", false
}
