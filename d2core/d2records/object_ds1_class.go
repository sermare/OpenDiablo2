package d2records

import (
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

const (
	ds1ObjectsPerAct = 0x96
	ds1Acts          = 5
)

// ds1ObjectOverrides are the two entries where the community lookup table
// differs from the 5 x 150 object table of Game.exe 1.14b (0x744af8),
// checked entry by entry: act 1 id 114 is objects.txt 0 in the exe and act 3
// id 104 is 342 (the lookup has no row for it).
var ds1ObjectOverrides = map[[2]int]int{{0, 114}: 0, {2, 104}: 342}

var (
	ds1ObjectOnce  sync.Once
	ds1ObjectTable [ds1Acts][ds1ObjectsPerAct]int
)

// DS1ObjectClass maps the id of a type-2 object of a DS1 file (0-based act,
// id below 0x96) to its objects.txt id the way the game's DS1 parser does
// (DRLG_ParseDS1Data). -1 means the object is dropped.
func DS1ObjectClass(act, id int) int {
	ds1ObjectOnce.Do(func() {
		for a := range ds1ObjectTable {
			for i := range ds1ObjectTable[a] {
				ds1ObjectTable[a][i] = -1
			}
		}

		for i := range objectLookups {
			o := &objectLookups[i]
			if o.Type != d2enum.ObjectTypeItem || o.Act < 1 || o.Act > ds1Acts || o.Id < 0 || o.Id >= ds1ObjectsPerAct {
				continue
			}

			ds1ObjectTable[o.Act-1][o.Id] = o.ObjectsTxtId
		}

		for k, v := range ds1ObjectOverrides {
			ds1ObjectTable[k[0]][k[1]] = v
		}
	})

	if act < 0 || act >= ds1Acts || id < 0 || id >= ds1ObjectsPerAct {
		return -1
	}

	return ds1ObjectTable[act][id]
}
