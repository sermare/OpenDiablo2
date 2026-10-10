package herogen

import (
	"os"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// The real-data tests need D2_TABLES (extracted game tables); D2S_SAMPLE_BODY
// (the sample Sorceress) adds the template check. Nothing is committed.

func realTables(t *testing.T) *Tables {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	tb, err := LoadTables(dir)
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func sampleTemplate(t *testing.T, tb *Tables) (*Template, *d2s.Character) {
	t.Helper()

	path := os.Getenv("D2S_SAMPLE_BODY")
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(data, tb.Save)
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := TemplateFrom(c)
	if err != nil {
		t.Fatal(err)
	}

	return tmpl, c
}

func TestAllowedPoints(t *testing.T) {
	// the real level 94 Sorceress: 470 attribute points allocated, 105 skill points spent
	if got := AllowedStatPoints(94); got != 470 {
		t.Errorf("stat points at 94 = %d, want 470", got)
	}

	if got := AllowedSkillPoints(94); got != 105 {
		t.Errorf("skill points at 94 = %d, want 105", got)
	}

	if AllowedStatPoints(1) != LamEsenStatPoints || AllowedSkillPoints(1) != QuestSkillPoints {
		t.Error("level 1 keeps only the quest grants")
	}
}

func TestSkillArray(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr bool
		index   int
		points  byte
	}{
		{"first barbarian skill", Spec{Class: d2s.Barbarian, Skills: map[int]int{126: 3}}, false, 0, 3},
		{"last barbarian skill", Spec{Class: d2s.Barbarian, Skills: map[int]int{155: 20}}, false, 29, 20},
		{"sorceress skill on a barbarian", Spec{Class: d2s.Barbarian, Skills: map[int]int{36: 1}}, true, 0, 0},
		{"past the class", Spec{Class: d2s.Barbarian, Skills: map[int]int{156: 1}}, true, 0, 0},
		{"over twenty", Spec{Class: d2s.Barbarian, Skills: map[int]int{126: 21}}, true, 0, 0},
		{"sorceress first", Spec{Class: d2s.Sorceress, Skills: map[int]int{36: 1}}, false, 0, 1},
	}

	for _, tt := range tests {
		got, err := skillArray(tt.spec)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: err = %v, want error %v", tt.name, err, tt.wantErr)
			continue
		}

		if !tt.wantErr && got[tt.index] != tt.points {
			t.Errorf("%s: skill[%d] = %d, want %d", tt.name, tt.index, got[tt.index], tt.points)
		}
	}
}

func TestSkillBlock(t *testing.T) {
	b := skillBlock(Spec{Left: 0, Right: 151, Hotkeys: []int{149, -1, 1000 + 147}})

	if b.Right.Skill != 151 || b.Left.Skill != 0 || b.RightSwap.Skill != 151 {
		t.Errorf("mouse skills %+v %+v", b.Left, b.Right)
	}

	if b.Hotkeys[0].Skill != 149 || b.Hotkeys[1].Skill != 0xFFFF || b.Hotkeys[2].Skill != 147|0x8000 ||
		b.Hotkeys[3].Skill != 0xFFFF {
		t.Errorf("hotkeys %+v", b.Hotkeys[:4])
	}
}

func TestCheckPlacement(t *testing.T) {
	worn := func(slot uint8) d2s.Item {
		return d2s.Item{Code: "cap", Location: d2s.LocationEquipped, Equipped: slot}
	}
	belt := func(cell uint8) d2s.Item { return d2s.Item{Code: "hp5", Location: d2s.LocationBelt, X: cell} }
	inv := func(x, y uint8) d2s.Item {
		return d2s.Item{Code: "tbk", Location: d2s.LocationStored, Page: PageInventory, X: x, Y: y}
	}

	if err := checkPlacement([]d2s.Item{worn(1), worn(2), belt(0), belt(1), inv(0, 0), inv(1, 0)}); err != nil {
		t.Errorf("distinct places: %v", err)
	}

	for name, items := range map[string][]d2s.Item{
		"two helms":      {worn(1), worn(1)},
		"two belt cells": {belt(3), belt(3)},
		"two cells":      {inv(2, 1), inv(2, 1)},
	} {
		if err := checkPlacement(items); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// miniStatCost has the stats the property conversion test uses.
const miniStatCost = "Stat\tID\tSigned\tEncode\tSave Bits\tSave Add\tSave Param Bits\tCSvBits\tCSvParam\n" +
	"maxhp\t7\t0\t0\t9\t32\t\t\t\n" +
	"maxdamage%\t17\t0\t0\t9\t0\t\t\t\n" +
	"mindamage%\t18\t0\t0\t9\t0\t\t\t\n" +
	"armorclass\t31\t1\t0\t11\t10\t\t\t\n" +
	"item_nonclassskill\t97\t0\t1\t6\t0\t9\t\t\n"

func miniTablesForProps(t *testing.T) *Tables {
	t.Helper()

	save, err := d2s.NewItemTables([]byte(miniStatCost), []byte("name\tcode\ttype\n"), []byte("name\tcode\ttype\n"),
		[]byte("name\tcode\ttype\n"), []byte("ItemType\tCode\tEquiv1\tEquiv2\n"))
	if err != nil {
		t.Fatal(err)
	}

	shift := make([]int, 300)
	shift[7] = 8 // life is kept in 1/256 points

	return &Tables{Save: save, Creator: &d2drop.Creator{Props: &d2drop.PropTables{ValShift: shift}}}
}

func TestPropertiesConversion(t *testing.T) {
	tb := miniTablesForProps(t)

	writes := []d2drop.StatWrite{
		{Kind: 'S', Stat: 31, Value: 518},          // a unit stat is not a property
		{Kind: 'M', Stat: 18, Value: 245},          // the follower comes first in the roll order
		{Kind: 'M', Stat: 17, Value: 245},          // its leader
		{Kind: 'M', Stat: 7, Value: 25 << 8},       // unshifted by the stat's ValShift
		{Kind: 'L', Stat: 97, Param: 54, Value: 3}, // an add...
		{Kind: 'L', Stat: 97, Param: 54, Value: 2}, // ...adds up per parameter
		{Kind: 'L', Stat: 97, Param: 55, Value: 1}, // another parameter stays separate
		{Kind: 'L', Stat: 3000, Value: 1},          // not a saved stat
		{Kind: 'M', Stat: 31, Value: 100},          // a set overrides
		{Kind: 'M', Stat: 31, Value: 120},
	}

	got, err := tb.properties(writes)
	if err != nil {
		t.Fatal(err)
	}

	want := []d2s.Property{
		{ID: 7, Value: 25}, {ID: 17, Value: 245}, {ID: 18, Value: 245}, {ID: 31, Value: 120},
		{ID: 97, Param: 54, Value: 5}, {ID: 97, Param: 55, Value: 1},
	}

	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("property %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	if _, err := tb.properties([]d2drop.StatWrite{{Kind: 'M', Stat: 17, Value: 5}}); err == nil {
		t.Error("a damage leader without its follower was accepted")
	}
}

func TestBuildItemErrors(t *testing.T) {
	tb := realTables(t)

	for name, spec := range map[string]ItemSpec{
		"unknown unique":     {Unique: "No Such Unique"},
		"wrong base":         {Unique: "The Grandfather", Code: "cap"},
		"ladder only":        {Unique: "Tyrael's Might"},
		"unknown plain code": {Code: "zzz"},
	} {
		if _, err := tb.buildItem(spec, 0); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestGenerateRejectsBadSpecs(t *testing.T) {
	tb := realTables(t)

	mutate := func(f func(*Spec)) Spec {
		s := Barbarian("NokkaBarb")
		f(&s)

		return s
	}

	for name, spec := range map[string]Spec{
		"too many stat points":  mutate(func(s *Spec) { s.Strength += 1 }),
		"below class start":     mutate(func(s *Spec) { s.Strength = 10 }),
		"level 0":               mutate(func(s *Spec) { s.Level = 0 }),
		"level 100":             mutate(func(s *Spec) { s.Level = 100 }),
		"too many skill points": mutate(func(s *Spec) { s.Skills = map[int]int{147: 20, 151: 20, 127: 20, 149: 20, 126: 20, 153: 20} }),
		"foreign skill":         mutate(func(s *Spec) { s.Skills = map[int]int{36: 1} }),
		"bad name":              mutate(func(s *Spec) { s.Name = "x" }),
		"two weapons":           mutate(func(s *Spec) { s.Items = append(s.Items, s.Items[0]) }),
	} {
		if _, err := tb.Generate(spec, nil); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
