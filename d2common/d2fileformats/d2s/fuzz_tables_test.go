package d2s

import (
	"encoding/binary"
	"testing"
)

// FuzzItemTables feeds arbitrary table text and a compiled itemstatcost.bin to the table loader.
func FuzzItemTables(f *testing.F) {
	bin := make([]byte, 4+2*statRowSize)
	binary.LittleEndian.PutUint32(bin, 2)

	f.Add([]byte(miniStatCost), []byte(miniArmor), []byte(miniWeapons), []byte(miniMisc), []byte(miniItemTypes))
	f.Add(bin, []byte(miniArmor), []byte(miniWeapons), []byte(miniMisc), []byte(miniItemTypes))
	f.Add([]byte{}, []byte{}, []byte{}, []byte{}, []byte{})

	f.Fuzz(func(t *testing.T, stat, armor, weapons, misc, types []byte) {
		tb, err := NewItemTables(stat, armor, weapons, misc, types)
		if err != nil {
			return
		}

		tb.CharStatInfo(0)
		tb.Stat(31)
		tb.ItemKindOf("key")
		tb.IsStackable("key")
	})
}
