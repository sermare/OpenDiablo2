package d2tbl

import (
	"fmt"
	"os"
	"testing"
)

func TestZZIDs(t *testing.T) {
	for _, f := range []string{"d2data/string.tbl", "d2exp/expansionstring.tbl", "patch_d2/patchstring.tbl"} {
		b, err := os.ReadFile(os.Getenv("HOME") + "/git/d2-tables/" + f)
		if err != nil {
			t.Skip()
		}

		m, err := LoadIDStrings(b, nil)
		fmt.Println(f, len(m), err)

		if len(m) < 3000 && false {
			mn, mx := 1<<30, 0
			for k, v := range m {
				if k < mn {
					mn = k
				}
				if k > mx {
					mx = k
				}
				if k%1000 == 927 || k == 1148 || k == 1149 || k == 1150 || v == "Ladder" {
					fmt.Printf("  key %d %q\n", k, v)
				}
			}
			fmt.Println("  range", mn, mx)
		}
		for _, id := range []int{0x13e6, 0x13e7, 0x13e8, 0x13e9, 0x13e5, 10000 + 0x13e6, 10000 + 0x13e8,927, 1067, 2731, 0x2b7c - 10000, 0x2b7d - 10000, 0x2b7e - 10000, 0x2b7c - 20000, 0x2b7d - 20000, 0x2b7e - 20000} {
			if s, ok := m[id]; ok {
				fmt.Printf("  node %d %q\n", id, s)
			}
		}
	}
}
