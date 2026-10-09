package d2s

import (
	"os"
	"testing"
)

func TestPromoteWritesAPlayableSave(t *testing.T) {
	if os.Getenv("D2_TABLES") == "" {
		t.Skip("set D2_TABLES to run")
	}

	tables := loadRealTables(t, false)

	for _, flags := range []NewCharacterFlags{{Expansion: true}, {}, {Expansion: true, Hardcore: true}} {
		for _, class := range []Class{Necromancer, Druid, Amazon} {
			data, err := NewCharacter("Fresh", class, flags, DefaultAppearance(class))
			if err != nil {
				t.Fatal(err)
			}

			c, err := Parse(data, nil)
			if err != nil {
				t.Fatal(err)
			}

			c.Promote()
			c.Body.SetStat(StatStrength, 25)
			c.Body.SetStat(StatLevel, 1)

			out, err := Write(c, tables)
			if err != nil {
				t.Fatalf("%v %+v: %v", class, flags, err)
			}

			back, err := Parse(out, tables)
			if err != nil {
				t.Fatalf("%v %+v: re-parse: %v", class, flags, err)
			}

			if back.Header.IsNewCharacter() || back.Body == nil || back.Body.Attributes.Strength != 25 ||
				back.Header.IsExpansion() != flags.Expansion {
				t.Fatalf("%v %+v: promoted file wrong: %+v", class, flags, back.Header)
			}

			if d, act, ok := back.Header.ActiveDifficulty(); !ok || d != 0 || act != 0 {
				t.Fatalf("active difficulty %d %d %v", d, act, ok)
			}

			if back.Body.Waypoints[0] != 1 {
				t.Fatalf("waypoints %x", back.Body.Waypoints[0])
			}
		}
	}
}
