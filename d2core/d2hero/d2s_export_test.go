package d2hero

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// realSave loads the real-save oracle (D2S_SAMPLE_BODY with the tables in
// D2_TABLES) and builds the hero state ImportD2S would produce from it,
// without needing the game's MPQs.
func realSave(t *testing.T) ([]byte, *d2s.ItemTables, *HeroState) {
	t.Helper()

	path, dir := os.Getenv("D2S_SAMPLE_BODY"), os.Getenv("D2_TABLES")
	if path == "" || dir == "" {
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

	tables, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"),
		read("armor.txt"), read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	a := c.Body.Attributes
	stash := loadedStashGold(a.StashedGold)
	state := &HeroState{
		HeroName: c.Header.Name, Act: 1, MapSeed: c.Header.MapSeed, D2SBase: data, Gold: int(a.Gold), StashGold: &stash,
		Stats: &HeroStatsState{
			Level: int(a.Level), Experience: int(a.Experience),
			Strength: int(a.Strength), Energy: int(a.Energy), Dexterity: int(a.Dexterity), Vitality: int(a.Vitality),
			StatsPoints: int(a.UnusedStats), SkillPoints: int(a.UnusedSkillPoints),
			Health: int(a.CurrentHP), MaxHealth: int(a.MaxHP), Mana: int(a.CurrentMana), MaxMana: int(a.MaxMana),
			MaxStamina: int(a.MaxStamina),
		},
		Progress: &HeroProgress{Quests: quests(c.Body), Waypoints: c.Body.Waypoints, NPC: *c.Body.NPCFlags()},
		Merc:     MercFromHeader(c.Header.Mercenary),
	}

	if diff, _, ok := c.Header.ActiveDifficulty(); ok {
		state.Difficulty = d2enum.DifficultyType(diff)
	}

	return data, tables, state
}

func TestExportUnchangedIsByteIdentical(t *testing.T) {
	data, tables, state := realSave(t)

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	if !bytes.Equal(out, data) {
		t.Fatalf("export of an unchanged hero differs from the original (len %d vs %d)", len(out), len(data))
	}
}

func TestExportChangedFields(t *testing.T) {
	data, tables, state := realSave(t)

	state.Gold = 123456
	state.Stats.Level++
	state.Stats.Experience += 1000
	state.Stats.StatsPoints += 5
	state.Stats.Health -= 10
	state.Progress.Waypoints.Set(2, d2s.WPLutGholein, true)
	state.Progress.Quests[0].SetCompleted(2, 1)
	state.MapSeed = 0xDEADBEEF

	when := time.Unix(1700000000, 0)

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{LastPlayed: when})
	if err != nil {
		t.Fatal(err)
	}

	orig, _ := d2s.Parse(data, tables)

	got, err := d2s.Parse(out, tables) // also validates size and checksum
	if err != nil {
		t.Fatal(err)
	}

	a, o := got.Body.Attributes, orig.Body.Attributes
	if a.Gold != 123456 || a.Level != o.Level+1 || got.Header.Level != orig.Header.Level+1 ||
		a.Experience != o.Experience+1000 || a.UnusedStats != o.UnusedStats+5 || a.CurrentHP != o.CurrentHP-10 {
		t.Errorf("attributes not written: %+v (was %+v)", a, o)
	}

	if a.Strength != o.Strength || a.MaxHP != o.MaxHP || a.StashedGold != o.StashedGold {
		t.Errorf("untouched attributes changed: %+v (was %+v)", a, o)
	}

	if !got.Body.Waypoints.Has(2, d2s.WPLutGholein) || !got.Body.QuestRecord(0).Completed(2, 1) {
		t.Error("progress not written")
	}

	if got.Header.MapSeed != 0xDEADBEEF {
		t.Errorf("seed = %x", got.Header.MapSeed)
	}

	if ts := uint32(got.Header.Raw[lastPlayedOffset]) | uint32(got.Header.Raw[lastPlayedOffset+1])<<8 |
		uint32(got.Header.Raw[lastPlayedOffset+2])<<16 | uint32(got.Header.Raw[lastPlayedOffset+3])<<24; ts != 1700000000 {
		t.Errorf("last played = %d", ts)
	}

	// everything the engine does not own is kept
	if len(got.Items) != len(orig.Items) || !bytes.Equal(got.Trailing, orig.Trailing) ||
		got.Header.Mercenary != orig.Header.Mercenary || got.Body.SkillPoints != orig.Body.SkillPoints {
		t.Error("items, mercenary, skills or trailing data changed")
	}

	// and a second export of the exported file is stable
	again, _, err := ExportD2SWithOptions(state, out, tables, ExportOptions{LastPlayed: when})
	if err != nil || !bytes.Equal(again, out) {
		t.Errorf("export is not idempotent (err=%v)", err)
	}
}

func TestExportDifficultyAndGoldClamp(t *testing.T) {
	data, tables, state := realSave(t)

	state.Difficulty = d2enum.DifficultyHell
	state.Gold = 1 << 30

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) == 0 {
		t.Error("expected a gold clamp warning")
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if d, _, _ := got.Header.ActiveDifficulty(); d != 2 {
		t.Errorf("difficulty = %d", d)
	}

	if got.Body.Attributes.Gold != maxGold {
		t.Errorf("gold = %d", got.Body.Attributes.Gold)
	}
}

func TestExportWithoutOriginal(t *testing.T) {
	if _, err := ExportD2S(&HeroState{HeroName: "x"}, nil, nil); err != ErrNoOriginal {
		t.Fatalf("err = %v", err)
	}
}
