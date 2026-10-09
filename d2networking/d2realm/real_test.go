package d2realm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// realData loads the real level-94 save and the game tables; the tests are
// skipped unless D2_TABLES and D2S_SAMPLE_BODY are set (nothing is committed).
func realData(t *testing.T) ([]byte, *d2s.ItemTables) {
	t.Helper()

	dir, path := os.Getenv("D2_TABLES"), os.Getenv("D2S_SAMPLE_BODY")
	if dir == "" || path == "" {
		t.Skip("set D2_TABLES and D2S_SAMPLE_BODY to run")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		t.Skipf("none of %v in %s", names, dir)

		return nil
	}

	tables, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"), read("armor.txt"),
		read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data, tables
}

// tamper parses the sample, lets edit change it and writes it back. ok is
// false when the writer itself refuses the edit.
func tamper(t *testing.T, data []byte, tables *d2s.ItemTables, edit func(*d2s.Character)) (out []byte, ok bool) {
	t.Helper()

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	edit(c)

	out, err = d2s.Write(c, tables)

	return out, err == nil
}

func TestRealCharacterValidation(t *testing.T) {
	data, tables := realData(t)
	v := Validator{Tables: tables}

	c, err := v.Validate(data)
	if err != nil {
		t.Fatalf("real save rejected: %v", err)
	}

	if len(c.Items) == 0 {
		t.Fatal("no items parsed")
	}

	// without tables the body is still range-checked
	if _, err := (&Validator{}).Validate(data); err != nil {
		t.Fatalf("real save rejected without tables: %v", err)
	}

	setStat := func(id int, v uint64) func(*d2s.Character) {
		return func(c *d2s.Character) { c.Body.SetStat(id, v) }
	}

	bad := map[string]func(*d2s.Character){
		"strength 1023":   setStat(d2s.StatStrength, 1023),
		"unused stats":    setStat(d2s.StatUnusedStats, 900),
		"skill points":    setStat(d2s.StatUnusedSkills, 200),
		"gold over limit": setStat(d2s.StatGold, 990001),
		"stash gold":      setStat(d2s.StatStashedGold, 3000000),
		"stat level":      setStat(d2s.StatLevel, 50),
		"unknown item":    func(c *d2s.Character) { c.Items[0].Code = "zzz" },
		"item level 120":  func(c *d2s.Character) { c.Items[0].Level = 120 },
	}

	rejected := 0

	for name, edit := range bad {
		out, ok := tamper(t, data, tables, edit)
		if !ok {
			t.Logf("%s: the writer refused the edit", name)

			continue
		}

		if _, err := v.Validate(out); err == nil {
			t.Errorf("tampered save %q accepted", name)
		} else {
			rejected++
		}
	}

	if rejected < 5 {
		t.Fatalf("only %d tampered saves reached the validator", rejected)
	}
}

func TestRealCharacterInGames(t *testing.T) {
	data, tables := realData(t)

	_, addr := newServer(t, Config{Tables: tables})
	a := dial(t, addr, "nokka")

	if err := a.Upload(data); err != nil {
		t.Fatalf("upload real save: %v", err)
	}

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	diff, _, _ := c.Header.ActiveDifficulty()

	// level window rules use the real level 94
	_, err = a.Create(CreateGame{Name: "toolow", MinLevel: 95})
	wantCode(t, err, CodeLevelTooLow)
	_, err = a.Create(CreateGame{Name: "toohigh", MaxLevel: 50})
	wantCode(t, err, CodeLevelTooHigh)

	// difficulty above the character's active one is locked
	if diff < int(Hell) {
		_, err = a.Create(CreateGame{Name: "locked", Difficulty: Hell})
		wantCode(t, err, CodeDifficultyLocked)
	}

	// a tampered upload is refused and the stored save is untouched
	if bad, ok := tamper(t, data, tables, func(c *d2s.Character) { c.Body.SetStat(d2s.StatStrength, 1000) }); ok {
		wantCode(t, a.Upload(bad), CodeCharInvalid)
	}

	if c.Header.IsExpansion() {
		g, err := a.Create(CreateGame{Name: "ok", Difficulty: byte(diff), MinLevel: 80, MaxLevel: 99})
		if err != nil {
			t.Fatal(err)
		}

		if !g.Game.Expansion {
			t.Fatal("expansion flag lost")
		}
	}
}
