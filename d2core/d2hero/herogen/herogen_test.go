package herogen

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
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

// TestBarbarianHero generates the preset and proves the file: it parses, it
// writes back byte for byte, the numbers follow charstats and the tables, the
// gear can be worn and the skills have their prerequisites.
func TestBarbarianHero(t *testing.T) {
	tb := realTables(t)
	tmpl, sample := sampleTemplate(t, tb)

	hero, err := tb.Generate(Barbarian("NokkaBarb"), tmpl)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(hero.Data, tb.Save)
	if err != nil {
		t.Fatalf("the generated file does not parse: %v", err)
	}

	again, err := d2s.Write(c, tb.Save)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(again, hero.Data) {
		t.Error("parse and write do not reproduce the generated file")
	}

	second, err := tb.Generate(Barbarian("NokkaBarb"), tmpl)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(second.Data, hero.Data) {
		t.Error("two runs give different files")
	}

	checkHeader(t, c)
	checkNumbers(t, tb, c, hero)
	checkGear(t, tb, c)
	checkSkills(t, tb, c)

	if sample != nil {
		h, s := c.Header, sample.Header
		if h.MapSeed != s.MapSeed || h.Difficulty != s.Difficulty || h.Mercenary != s.Mercenary {
			t.Errorf("the template's map seed, difficulty or mercenary were not copied: %+v vs %+v", h, s)
		}

		if c.Body.Waypoints != sample.Body.Waypoints || c.Body.Quests != sample.Body.Quests {
			t.Error("the template's waypoints or quests were not copied")
		}
	}
}

func checkHeader(t *testing.T, c *d2s.Character) {
	t.Helper()

	h := c.Header

	if h.Class != d2s.Barbarian || h.Level != 94 || h.SkillCount != 30 {
		t.Errorf("class %v level %d skills %d", h.Class, h.Level, h.SkillCount)
	}

	if err := d2s.ValidateName(h.Name); err != nil {
		t.Errorf("name %q: %v", h.Name, err)
	}

	if !h.IsExpansion() || h.IsHardcore() || h.IsDead() || h.IsNewCharacter() {
		t.Errorf("status %#x: want a living softcore expansion hero", h.Status)
	}

	if _, _, ok := h.ActiveDifficulty(); !ok {
		t.Error("no active difficulty")
	}

	sb := h.SkillBlock()
	if sb.Right.Skill != SkillWhirlwind || sb.Left.Skill != 0 {
		t.Errorf("mouse skills %+v / %+v", sb.Left, sb.Right)
	}
}

func checkNumbers(t *testing.T, tb *Tables, c *d2s.Character, hero *Hero) {
	t.Helper()

	cls := tb.Classes["Barbarian"]
	a := c.Body.Attributes

	if int(a.Level) != 94 || tb.Exp["Barbarian"].LevelFor(int64(a.Experience)) != 94 {
		t.Errorf("level %d, experience %d is level %d", a.Level, a.Experience, tb.Exp["Barbarian"].LevelFor(int64(a.Experience)))
	}

	if a.UnusedStats != 0 || a.UnusedSkillPoints != 0 {
		t.Errorf("unspent points: stats %d skills %d (the preset spends everything)", a.UnusedStats, a.UnusedSkillPoints)
	}

	spent := int(a.Strength+a.Dexterity+a.Vitality+a.Energy) - (cls.InitStr + cls.InitDex + cls.InitVit + cls.InitEne)
	if spent != AllowedStatPoints(94) {
		t.Errorf("%d attribute points spent, want %d", spent, AllowedStatPoints(94))
	}

	// the stored maxima are the class formula (+20 life from the Potion of Life), without items
	life, mana, stam := cls.BaseMax(94, int(a.Vitality), int(a.Energy))
	if int(a.MaxHP) != life+PotionOfLifeLife || int(a.MaxMana) != mana || int(a.MaxStamina) != stam {
		t.Errorf("stored maxima %d/%d/%d, charstats say %d(+20)/%d/%d", a.MaxHP, a.MaxMana, a.MaxStamina, life, mana, stam)
	}

	// the current values are the totals with the worn items: full health, like a save written by the game
	tot := hero.Totals
	if int(a.CurrentHP) != tot.MaxLife || int(a.CurrentMana) != tot.MaxMana || int(a.CurrentStamina) != tot.MaxStamina {
		t.Errorf("current %d/%d/%d, totals %d/%d/%d", a.CurrentHP, a.CurrentMana, a.CurrentStamina, tot.MaxLife,
			tot.MaxMana, tot.MaxStamina)
	}

	if tot.MaxLife < 1500 || tot.MaxLife <= int(a.MaxHP) {
		t.Errorf("life %d: the gear should add to %d", tot.MaxLife, a.MaxHP)
	}

	if tot.DamageMin <= 0 || tot.DamageMax <= tot.DamageMin || tot.AttackRating <= 0 || tot.Defense <= 0 {
		t.Errorf("combat numbers: damage %d-%d attack rating %d defense %d", tot.DamageMin, tot.DamageMax,
			tot.AttackRating, tot.Defense)
	}

	// the same totals from the items parsed back from the file
	items := statItems(c.Items, tb.Bases)
	h := d2statlist.Hero{
		Class: cls, Level: 94, Str: int(a.Strength), Dex: int(a.Dexterity), Vit: int(a.Vitality), Ene: int(a.Energy),
		BaseLife: int(a.MaxHP), BaseMana: int(a.MaxMana), BaseStam: int(a.MaxStamina), Difficulty: 2,
	}

	if back := d2statlist.Compute(h, items, nil); back.MaxLife != tot.MaxLife || back.DamageMax != tot.DamageMax {
		t.Errorf("totals after reading the file: life %d damage %d, generated %d / %d", back.MaxLife, back.DamageMax,
			tot.MaxLife, tot.DamageMax)
	}
}

func checkGear(t *testing.T, tb *Tables, c *d2s.Character) {
	t.Helper()

	cr := tb.Creator
	a := c.Body.Attributes
	worn := map[uint8]string{}
	belt := 0

	for i := range c.Items {
		it := &c.Items[i]
		code := strings.TrimSpace(it.Code)

		switch it.Location {
		case d2s.LocationBelt:
			belt++
		case d2s.LocationEquipped:
			worn[it.Equipped] = code

			if it.Quality != d2s.QualityUnique || !it.Identified {
				t.Errorf("%s in slot %d: quality %d identified %v", code, it.Equipped, it.Quality, it.Identified)
			}

			u := cr.Uniques.Uniques[it.UniqueID]
			base := cr.Items.ByCode[code]

			switch {
			case u.Code != code || u.Ladder || !u.Enabled:
				t.Errorf("%s: unique row %d (%s) is %s, ladder %v enabled %v", code, it.UniqueID, u.Name, u.Code, u.Ladder, u.Enabled)
			case u.LvlReq > 94 || base.LevelReq > 94:
				t.Errorf("%s (%s) needs level %d", code, u.Name, u.LvlReq)
			case base.ReqStr > int(a.Strength) || base.ReqDex > int(a.Dexterity):
				t.Errorf("%s (%s) needs strength %d dexterity %d, the hero has %d/%d", code, u.Name, base.ReqStr,
					base.ReqDex, a.Strength, a.Dexterity)
			}

			if base.Durability > 0 && it.Durability == 0 && !hasProperty(it, statIndestruct) {
				t.Errorf("%s (%s) is broken", code, u.Name)
			}
		}
	}

	for slot := SlotHead; slot <= SlotGloves; slot++ {
		if slot == SlotLeftHand {
			continue // the two-hand sword leaves it free
		}

		if worn[slot] == "" {
			t.Errorf("nothing worn in slot %d", slot)
		}
	}

	if worn[SlotRightHand] != "7gd" || !cr.Items.ByCode["7gd"].TwoHanded {
		t.Errorf("weapon %q: want The Grandfather, a two-hand sword", worn[SlotRightHand])
	}

	if belt != 16 {
		t.Errorf("%d belt potions, want 16", belt)
	}
}

func hasProperty(it *d2s.Item, id int) bool {
	for _, p := range it.Properties {
		if p.ID == id {
			return true
		}
	}

	return false
}

// checkSkills proves every skill the hero has points in meets its required
// level and prerequisites in skills.txt, and that the 30-skill array is in
// skill id order.
func checkSkills(t *testing.T, tb *Tables, c *d2s.Character) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(tb.Dir, "skills", "patch_d2", "skills.txt"))
	if err != nil {
		t.Skipf("skills.txt: %v", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	type row struct {
		name   string
		reqLvl int
		reqs   []string
		maxLvl int
	}

	byID := map[int]row{}
	byName := map[string]int{}

	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")
		if len(f) <= col["maxlvl"] || f[col["charclass"]] != "bar" {
			continue
		}

		id, _ := strconv.Atoi(f[col["Id"]])
		rl, _ := strconv.Atoi(f[col["reqlevel"]])
		ml, _ := strconv.Atoi(f[col["maxlvl"]])
		r := row{name: f[col["skill"]], reqLvl: rl, maxLvl: ml}

		for _, k := range []string{"reqskill1", "reqskill2", "reqskill3"} {
			if f[col[k]] != "" {
				r.reqs = append(r.reqs, f[col[k]])
			}
		}

		byID[id] = r
		byName[r.name] = id
	}

	first := FirstSkillID(d2s.Barbarian)
	if byName["Bash"] != first {
		t.Fatalf("Bash is skill %d, the preset assumes the class starts at %d", byName["Bash"], first)
	}

	total := 0

	for i, pts := range c.Body.SkillPoints {
		if pts == 0 {
			continue
		}

		total += int(pts)
		r, ok := byID[first+i]

		switch {
		case !ok:
			t.Errorf("skill index %d (id %d) is not a Barbarian skill", i, first+i)
		case int(pts) > r.maxLvl:
			t.Errorf("%s has %d points, maximum %d", r.name, pts, r.maxLvl)
		case r.reqLvl > 94:
			t.Errorf("%s needs level %d", r.name, r.reqLvl)
		}

		for _, req := range r.reqs {
			if c.Body.SkillPoints[byName[req]-first] == 0 {
				t.Errorf("%s needs %s, which has no points", r.name, req)
			}
		}
	}

	if total != AllowedSkillPoints(94) {
		t.Errorf("%d skill points spent, want %d", total, AllowedSkillPoints(94))
	}

	for _, name := range []string{"Frenzy", "Whirlwind", "Battle Orders", "Sword Mastery"} {
		if c.Body.SkillPoints[byName[name]-first] != 20 {
			t.Errorf("%s is not at 20", name)
		}
	}
}
