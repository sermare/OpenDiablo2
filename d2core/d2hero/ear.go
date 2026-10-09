package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// earVersion is the item version field written for an ear. UNVERIFIED: the
// value of the 1.14 saves read so far for extended items; the reader ignores
// it for ears.
const earVersion = 101

// EarD2S is the .d2s form of the ear of a slain hero. Kept in StoredItem.D2S
// it is written by ExportD2SItems exactly like an imported item, so an ear a
// player earned in a hardcore kill survives in the exported save.
func EarD2S(name string, level int, class d2s.Class) d2s.Item {
	if level < 1 {
		level = 1
	}

	if level > 99 {
		level = 99
	}

	return d2s.Item{
		Ear: true, Identified: true, Version: earVersion, Location: d2s.LocationStored, Page: PageInventory,
		Code: "ear", EarInfo: &d2s.EarInfo{Class: uint8(class), Level: uint8(level), Name: name},
	}
}
