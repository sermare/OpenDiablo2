package herogen

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// classExpect is what a preset of a class must look like.
type classExpect struct {
	preset string
	class  d2s.Class
	// classType is the item type that only this class wears, and slot where the preset wears it
	classType string
	classSlot uint8
	weapon    string // base code of the right hand weapon
	offhand   string // base code of the left hand item ("" = empty)
	twoHanded bool
	minLife   int
	left      string   // the main attack
	supports  []string // support skills with points (at least three)
	synergies int      // points the synergy skills of the main attack must hold in total (0: the table has none)
	mainRange string   // skills.txt range of the main attack: h2h (melee) or anything else (cast or thrown)
}

var classExpects = []classExpect{
	{"amazon", d2s.Amazon, "ajav", SlotRightHand, "ama", "uml", false, 1200, "Lightning Fury",
		[]string{"Valkyrie", "Critical Strike", "Dopplezon", "Slow Missiles", "Avoid", "Evade"}, 30, "rng"},
	{"sorc", d2s.Sorceress, "orb", SlotRightHand, "obc", "xsh", false, 1000, "Fire Ball",
		[]string{"Frozen Armor", "Teleport", "Warmth", "Fire Mastery"}, 30, "none"},
	{"necro", d2s.Necromancer, "head", SlotLeftHand, "7bw", "nee", false, 1000, "Bone Spear",
		[]string{"Bone Armor", "Raise Skeleton", "Skeleton Mastery", "Clay Golem", "Amplify Damage"}, 30, "none"},
	{"paladin", d2s.Paladin, "ashd", SlotLeftHand, "9ws", "pac", false, 1200, "Zeal",
		[]string{"Fanaticism", "Vigor", "Might", "Vengeance"}, 20, "h2h"},
	{"barb", d2s.Barbarian, "phlm", SlotHead, "7gd", "", true, 1500, "Frenzy",
		[]string{"Battle Orders", "Natural Resistance", "Increased Speed", "Sword Mastery"}, 10, "h2h"},
	{"druid", d2s.Druid, "pelt", SlotHead, "6cs", "", true, 900, "Firestorm",
		[]string{"Oak Sage", "Cyclone Armor", "Summon Spirit Wolf", "Raven", "Wearbear"}, 30, "none"},
	{"assassin", d2s.Assassin, "h2h2", SlotRightHand, "7wb", "7lw", false, 1200, "Dragon Talon",
		[]string{"Claw Mastery", "Quickness", "Fade", "Shadow Warrior", "Weapon Block"}, 0, "h2h"},
}

// TestSkillNamesMatchTable proves the name lists the presets use are the 30 skills of each class in
// skills.txt order, and that FirstSkillID is the id of the first.
func TestSkillNamesMatchTable(t *testing.T) {
	tb := realTables(t)

	if tb.Skills == nil {
		t.Skip("skills.txt not found")
	}

	for c := d2s.Amazon; c <= d2s.Assassin; c++ {
		names := ClassSkillNames(c)

		for i, n := range names {
			r, ok := tb.Skills[FirstSkillID(c)+i]
			if !ok || r.Name != n || r.Class != classTokens[c] {
				t.Errorf("%v skill %d: preset says %q, skills.txt says %+v", c, FirstSkillID(c)+i, n, r)
			}
		}

		if id, ok := SkillID(c, names[29]); !ok || id != FirstSkillID(c)+29 {
			t.Errorf("%v: SkillID of the last skill is %d", c, id)
		}

		if SkillName(c, FirstSkillID(c)+3) != names[3] || SkillName(c, 0) != "" {
			t.Errorf("%v: SkillName", c)
		}
	}
}

func TestPresetNames(t *testing.T) {
	seen := map[d2s.Class]bool{}

	for _, n := range PresetNames() {
		c, ok := ClassOfPreset(n)
		if !ok {
			t.Fatalf("preset %q unknown", n)
		}

		seen[c] = true

		if err := d2s.ValidateName(DefaultName(c)); err != nil {
			t.Errorf("%v: default name %q: %v", c, DefaultName(c), err)
		}

		spec, err := Preset(c, DefaultName(c))
		if err != nil || spec.Class != c || spec.Level != 94 {
			t.Errorf("%s: preset %+v, %v", n, spec, err)
		}
	}

	if len(seen) != 7 {
		t.Errorf("%d classes have a preset, want 7", len(seen))
	}

	for alias, want := range map[string]d2s.Class{
		"Barbarian": d2s.Barbarian, " SIN ": d2s.Assassin, "necromancer": d2s.Necromancer, "sorceress": d2s.Sorceress,
	} {
		if got, ok := ClassOfPreset(alias); !ok || got != want {
			t.Errorf("alias %q = %v, %v", alias, got, ok)
		}
	}

	if _, ok := ClassOfPreset("wizard"); ok {
		t.Error("an unknown preset was accepted")
	}

	if _, err := Preset(d2s.Class(9), "x"); err == nil {
		t.Error("a preset of class 9")
	}
}

func TestSkillMapPanicsOnUnknownName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unknown skill name did not panic")
		}
	}()

	skillMap(d2s.Amazon, SP{"Fire Bolt", 1}) // a Sorceress skill
}

func TestBeltOfFillsFourRows(t *testing.T) {
	items := beltOf([4]string{"a", "b", "c", "d"})
	if len(items) != 16 {
		t.Fatalf("%d cells", len(items))
	}

	for i, it := range items {
		if int(it.Place.X) != i || it.Code != string(rune('a'+i%4)) {
			t.Errorf("cell %d = %+v", i, it)
		}
	}
}

func TestBeltBoxes(t *testing.T) {
	tb := &Tables{BeltTypes: map[string]int{"umc": 6, "lbl": 1, "mbl": 0, "hbl": 3}}

	for code, want := range map[string]int{"umc": 16, "lbl": 8, "mbl": 12, "hbl": 16, "": 4, "xyz": 4} {
		if got := tb.BeltBoxes(code); got != want {
			t.Errorf("belt %q: %d cells, want %d", code, got, want)
		}
	}
}

// TestClassHeroes generates the preset of every class and proves the file: it parses, writes back byte
// for byte, is the same on every run, the numbers follow charstats and the tables, the gear is class
// legal and meets its requirements, and the skills have their prerequisites and a main attack.
func TestClassHeroes(t *testing.T) {
	tb := realTables(t)
	tmpl, sample := sampleTemplate(t, tb)

	for _, ex := range classExpects {
		ex := ex

		t.Run(ex.preset, func(t *testing.T) {
			c, ok := ClassOfPreset(ex.preset)
			if !ok || c != ex.class {
				t.Fatalf("preset %q is class %v, want %v", ex.preset, c, ex.class)
			}

			spec, err := Preset(c, DefaultName(c))
			if err != nil {
				t.Fatal(err)
			}

			hero, err := tb.Generate(spec, tmpl)
			if err != nil {
				t.Fatal(err)
			}

			parsed, err := d2s.Parse(hero.Data, tb.Save)
			if err != nil {
				t.Fatalf("the generated file does not parse: %v", err)
			}

			again, err := d2s.Write(parsed, tb.Save)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(again, hero.Data) {
				t.Error("parse and write do not reproduce the generated file")
			}

			second, err := tb.Generate(spec, tmpl)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(second.Data, hero.Data) {
				t.Error("two runs give different files")
			}

			checkClassHeader(t, parsed, spec)
			checkClassNumbers(t, tb, parsed, hero, ex)
			checkClassGear(t, tb, parsed, spec, ex)
			checkClassSkills(t, tb, parsed, spec, ex)

			if sample != nil {
				h, s := parsed.Header, sample.Header
				if h.MapSeed != s.MapSeed || h.Difficulty != s.Difficulty || h.Mercenary != s.Mercenary {
					t.Errorf("the template's map seed, difficulty or mercenary were not copied: %+v vs %+v", h, s)
				}

				if parsed.Body.Waypoints != sample.Body.Waypoints || parsed.Body.Quests != sample.Body.Quests ||
					parsed.Body.NPC != sample.Body.NPC {
					t.Error("the template's waypoints, quests or NPC flags were not copied")
				}
			}
		})
	}
}

func checkClassHeader(t *testing.T, c *d2s.Character, spec Spec) {
	t.Helper()

	h := c.Header

	if h.Class != spec.Class || h.Level != 94 || h.SkillCount != 30 {
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
	if int(sb.Left.Skill) != spec.Left || int(sb.Right.Skill) != spec.Right {
		t.Errorf("mouse skills %d/%d, want %d/%d", sb.Left.Skill, sb.Right.Skill, spec.Left, spec.Right)
	}

	if int(sb.Hotkeys[0].Skill) != spec.Left {
		t.Errorf("F1 holds skill %d, the main attack is %d", sb.Hotkeys[0].Skill, spec.Left)
	}
}

func checkClassNumbers(t *testing.T, tb *Tables, c *d2s.Character, hero *Hero, ex classExpect) {
	t.Helper()

	name := ex.class.String()
	cls := tb.Classes[name]
	a := c.Body.Attributes

	if int(a.Level) != 94 || tb.Exp[name].LevelFor(int64(a.Experience)) != 94 {
		t.Errorf("level %d, experience %d is level %d", a.Level, a.Experience, tb.Exp[name].LevelFor(int64(a.Experience)))
	}

	if a.UnusedStats != 0 || a.UnusedSkillPoints != 0 {
		t.Errorf("unspent points: stats %d skills %d (the preset spends everything)", a.UnusedStats, a.UnusedSkillPoints)
	}

	spent := int(a.Strength+a.Dexterity+a.Vitality+a.Energy) - (cls.InitStr + cls.InitDex + cls.InitVit + cls.InitEne)
	if spent != AllowedStatPoints(94) {
		t.Errorf("%d attribute points spent, want %d", spent, AllowedStatPoints(94))
	}

	// no attribute is below the class start (a preset that spends a stat down would hide a typo)
	if int(a.Strength) < cls.InitStr || int(a.Dexterity) < cls.InitDex || int(a.Vitality) < cls.InitVit ||
		int(a.Energy) < cls.InitEne {
		t.Errorf("attributes %d/%d/%d/%d below the class start %d/%d/%d/%d", a.Strength, a.Dexterity, a.Vitality,
			a.Energy, cls.InitStr, cls.InitDex, cls.InitVit, cls.InitEne)
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

	if tot.MaxLife < ex.minLife || tot.MaxLife <= int(a.MaxHP) {
		t.Errorf("life %d: want at least %d, and the gear should add to %d", tot.MaxLife, ex.minLife, a.MaxHP)
	}

	if tot.MaxMana < 150 {
		t.Errorf("mana %d: the main attack needs a pool", tot.MaxMana)
	}

	if tot.DamageMin <= 0 || tot.DamageMax <= tot.DamageMin || tot.AttackRating <= 0 || tot.Defense <= 0 {
		t.Errorf("combat numbers: damage %d-%d attack rating %d defense %d", tot.DamageMin, tot.DamageMax,
			tot.AttackRating, tot.Defense)
	}

	// the same totals from the items parsed back from the file
	h := d2statlist.Hero{
		Class: cls, Level: 94, Str: int(a.Strength), Dex: int(a.Dexterity), Vit: int(a.Vitality), Ene: int(a.Energy),
		BaseLife: int(a.MaxHP), BaseMana: int(a.MaxMana), BaseStam: int(a.MaxStamina), Difficulty: 2,
	}

	if back := d2statlist.Compute(h, statItems(c.Items, tb.Bases), nil); back.MaxLife != tot.MaxLife ||
		back.DamageMax != tot.DamageMax || back.MaxMana != tot.MaxMana {
		t.Errorf("totals after reading the file: life %d mana %d damage %d, generated %d / %d / %d", back.MaxLife,
			back.MaxMana, back.DamageMax, tot.MaxLife, tot.MaxMana, tot.DamageMax)
	}
}

func checkClassGear(t *testing.T, tb *Tables, c *d2s.Character, spec Spec, ex classExpect) {
	t.Helper()

	cr := tb.Creator
	worn := map[uint8]string{}
	wornType := map[uint8]string{}
	belt, tomes, annihilus := 0, 0, 0
	cells := map[uint8]string{}

	for i := range c.Items {
		it := &c.Items[i]
		code := strings.TrimSpace(it.Code)

		switch it.Location {
		case d2s.LocationBelt:
			belt++
			cells[it.X] = code

			if !strings.HasPrefix(code, "hp") && !strings.HasPrefix(code, "mp") && code != "rvl" {
				t.Errorf("belt cell %d holds %q, not a potion", it.X, code)
			}
		case d2s.LocationStored:
			switch code {
			case "tbk", "ibk":
				tomes++

				if it.Quantity != 20 {
					t.Errorf("tome %s holds %d scrolls, want 20", code, it.Quantity)
				}
			case "cm1":
				annihilus++
			}
		case d2s.LocationEquipped:
			worn[it.Equipped] = code
			wornType[it.Equipped] = cr.Items.ByCode[code].Type

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
			}

			if base.Durability > 0 && it.Durability == 0 && !hasProperty(it, statIndestruct) {
				t.Errorf("%s (%s) is broken", code, u.Name)
			}

			if tb.Save.IsStackable(code) && it.Quantity < 1 {
				t.Errorf("%s (%s) is a stack of nothing", code, u.Name)
			}
		}
	}

	// the worn set is legal by the game's own equip rules, judged on the attributes of the file
	a := c.Body.Attributes
	fileSpec := spec
	fileSpec.Strength, fileSpec.Dexterity = int(a.Strength), int(a.Dexterity)

	if err := tb.CheckWorn(fileSpec, c.Items); err != nil {
		t.Errorf("worn set: %v", err)
	}

	// and an item the hero's strength does not reach is refused by the same rules (the check is not vacuous)
	weak := fileSpec
	weak.Strength, weak.Dexterity = 1, 1

	if err := tb.CheckWorn(weak, c.Items); err == nil {
		t.Error("a hero with strength 1 can wear the set: CheckWorn checks nothing")
	}

	for slot := SlotHead; slot <= SlotGloves; slot++ {
		if slot == SlotLeftHand && (ex.twoHanded || ex.offhand == "") {
			if worn[slot] != "" {
				t.Errorf("left hand holds %q next to a two hand weapon", worn[slot])
			}

			continue
		}

		if worn[slot] == "" {
			t.Errorf("nothing worn in slot %d", slot)
		}
	}

	if worn[SlotRightHand] != ex.weapon {
		t.Errorf("weapon %q, want %q", worn[SlotRightHand], ex.weapon)
	}

	if ex.offhand != "" && worn[SlotLeftHand] != ex.offhand {
		t.Errorf("left hand %q, want %q", worn[SlotLeftHand], ex.offhand)
	}

	if ex.twoHanded && !cr.Items.ByCode[ex.weapon].TwoHanded {
		t.Errorf("%s is not a two hand weapon", ex.weapon)
	}

	// the class item of the class: javelins, orbs, voodoo heads, auric shields, primal helms, pelts, claws
	if got := wornType[ex.classSlot]; got == "" || !tb.EquipRules.Types.IsA(got, ex.classType) {
		t.Errorf("slot %d holds type %q, want a %q (class item)", ex.classSlot, got, ex.classType)
	}

	if want := classTokens[ex.class]; tb.EquipRules.Types.ClassOf(ex.classType) != want {
		t.Errorf("item type %q is restricted to %q, not %q", ex.classType, tb.EquipRules.Types.ClassOf(ex.classType), want)
	}

	if belt != 16 {
		t.Errorf("%d belt potions, want 16", belt)
	}

	// healing, rejuvenation and mana potions are all in the belt
	kinds := map[string]bool{}

	for _, code := range cells {
		kinds[code] = true
	}

	if !kinds["hp5"] || !kinds["mp5"] || !kinds["rvl"] {
		t.Errorf("belt potions %v: want healing, mana and rejuvenation", kinds)
	}

	// the first column in front is a healing potion: the fight drinks the front of the first column that heals
	if cells[0] != "hp5" {
		t.Errorf("front of the first belt column is %q, want hp5", cells[0])
	}

	if tomes != 2 {
		t.Errorf("%d tomes, want 2", tomes)
	}

	if annihilus != 1 {
		t.Errorf("%d Annihilus, want 1", annihilus)
	}
}

// checkClassSkills proves the build: every skill meets skills.txt (maximum, level, prerequisites), all points
// are spent, the main attack is at 20 and is on the left button, three or more support skills have points
// and the synergies of the main attack are fed.
func checkClassSkills(t *testing.T, tb *Tables, c *d2s.Character, spec Spec, ex classExpect) {
	t.Helper()

	if tb.Skills == nil {
		t.Skip("skills.txt not found")
	}

	first := FirstSkillID(ex.class)
	names := ClassSkillNames(ex.class)
	pts := map[string]int{}
	fileSpec := spec
	fileSpec.Skills = map[int]int{}
	total := 0

	for i, p := range c.Body.SkillPoints {
		if p == 0 {
			continue
		}

		total += int(p)
		pts[names[i]] = int(p)
		fileSpec.Skills[first+i] = int(p)
	}

	if total != AllowedSkillPoints(94) {
		t.Errorf("%d skill points spent, want %d", total, AllowedSkillPoints(94))
	}

	if err := tb.ValidateSkills(fileSpec); err != nil {
		t.Errorf("skills: %v", err)
	}

	// a build that skips a prerequisite is refused (the check is not vacuous)
	broken := fileSpec
	broken.Skills = map[int]int{}

	for id, p := range fileSpec.Skills {
		if names[id-first] != "Corpse Explosion" && names[id-first] != "Plague Javelin" && names[id-first] != "Blaze" &&
			names[id-first] != "Sacrifice" && names[id-first] != "Molten Boulder" && names[id-first] != "Dragon Claw" &&
			names[id-first] != "Double Throw" {
			broken.Skills[id] = p
		}
	}

	if err := tb.ValidateSkills(broken); err == nil {
		t.Error("a build without one of its prerequisites passes")
	}

	if pts[ex.left] != 20 {
		t.Errorf("the main attack %s has %d points, want 20", ex.left, pts[ex.left])
	}

	if int(c.Header.SkillBlock().Left.Skill) != first+indexOf(names, ex.left) {
		t.Errorf("the left button does not hold %s", ex.left)
	}

	row := tb.Skills[first+indexOf(names, ex.left)]
	if row.Passive {
		t.Errorf("the main attack %s is a passive", ex.left)
	}

	if row.Range != ex.mainRange {
		t.Errorf("the main attack %s has range %q, want %q", ex.left, row.Range, ex.mainRange)
	}

	support := 0

	for _, s := range ex.supports {
		if pts[s] > 0 {
			support++
		}
	}

	if support < 3 {
		t.Errorf("only %d of the supports %v have points", support, ex.supports)
	}

	syn := synergiesOf(t, tb, ex.class, ex.left)
	got := 0

	for _, s := range syn {
		got += pts[s]
	}

	if got < ex.synergies {
		t.Errorf("synergies of %s %v hold %d points, want at least %d", ex.left, syn, got, ex.synergies)
	}

	if ex.synergies > 0 && len(syn) == 0 {
		t.Errorf("skills.txt lists no synergy for %s", ex.left)
	}
}

func indexOf(names [30]string, n string) int {
	for i := range names {
		if names[i] == n {
			return i
		}
	}

	return -100
}

var synergyRef = regexp.MustCompile(`skill\('([^']+)'`)

// synergiesOf reads the skills a skill's numbers grow from: the skill('Name'.blvl) terms in the
// calculation columns of its skills.txt row (EDmgSymPerCalc, DmgSymPerCalc, calc1.., Param..).
func synergiesOf(t *testing.T, tb *Tables, class d2s.Class, name string) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(tb.Dir, "skills", "patch_d2", "skills.txt"))
	if err != nil {
		t.Skipf("skills.txt: %v", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	head := strings.Split(lines[0], "\t")
	id := strconv.Itoa(FirstSkillID(class) + indexOf(ClassSkillNames(class), name))

	idCol := -1

	for i, h := range head {
		if h == "Id" {
			idCol = i
			break
		}
	}

	var out []string

	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")
		if idCol < 0 || len(f) <= idCol || f[idCol] != id {
			continue
		}

		seen := map[string]bool{name: true}

		for i := range head {
			if i >= len(f) {
				continue
			}

			for _, m := range synergyRef.FindAllStringSubmatch(f[i], -1) {
				if !seen[m[1]] {
					seen[m[1]] = true
					out = append(out, m[1])
				}
			}
		}
	}

	return out
}

// TestBarbarianHero keeps the checks of the first preset that was proven in the game (levels 94 Barbarian,
// 9d/9g/9h/9i): the Grandfather, the Whirlwind and Frenzy build.
func TestBarbarianHero(t *testing.T) {
	tb := realTables(t)

	hero, err := tb.Generate(Barbarian("NokkaBarb"), nil)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(hero.Data, tb.Save)
	if err != nil {
		t.Fatal(err)
	}

	first := FirstSkillID(d2s.Barbarian)
	for name, want := range map[string]int{
		"Frenzy": 20, "Battle Orders": 20, "Sword Mastery": 20, "Whirlwind": 10, "Natural Resistance": 10,
		"Double Swing": 6, "Taunt": 6,
	} {
		if got := c.Body.SkillPoints[indexOf(ClassSkillNames(d2s.Barbarian), name)]; int(got) != want {
			t.Errorf("%s (%d) has %d points, want %d", name, first+indexOf(ClassSkillNames(d2s.Barbarian), name), got, want)
		}
	}

	if sb := c.Header.SkillBlock(); sb.Left.Skill != SkillFrenzy || sb.Right.Skill != SkillBattleOrders {
		t.Errorf("mouse skills %d / %d", sb.Left.Skill, sb.Right.Skill)
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
