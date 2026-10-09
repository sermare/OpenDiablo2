//go:build ignore

// d2s-revive clears the hardcore and died flags of a .d2s and fills life and mana
// (for the playthrough tests: the sample character is a hardcore corpse).
//
//	go run scripts/d2s-revive.go in.d2s out.d2s $D2_TABLES
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	dir := os.Args[3]
	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		panic(names)
	}

	tb, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"), read("armor.txt"), read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		panic(err)
	}

	c, err := d2s.Parse(data, tb)
	if err != nil {
		panic(err)
	}

	c.Header.Status &^= d2s.StatusHardcore | d2s.StatusDied
	c.Body.SetStat(d2s.StatCurrentHP, c.Body.Attributes.MaxHP<<8)
	c.Body.SetStat(d2s.StatCurrentMana, c.Body.Attributes.MaxMana<<8)

	out, err := d2s.Write(c, tb)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(os.Args[2], out, 0o600); err != nil {
		panic(err)
	}

	fmt.Printf("status now %#x hp=%d\n", c.Header.Status, c.Body.Attributes.MaxHP)
}
