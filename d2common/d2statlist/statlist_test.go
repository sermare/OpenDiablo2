package d2statlist

import (
	"os"
	"path/filepath"
	"testing"
)

var sorc = Class{
	ID: 1, InitStr: 10, InitDex: 25, InitVit: 10, InitEne: 35, InitStamina: 74, HpAdd: 30,
	LifePerVit: 8, ManaPerEne: 8, StaminaPerVit: 4, LifePerLevel: 4, ManaPerLevel: 8, StaminaPerLevel: 4,
	ToHitFactor: -15, BlockFactor: 20,
}

func TestBaseMax(t *testing.T) {
	tests := []struct {
		name                string
		level, vit, ene     int
		life, mana, stamina int
	}{
		{"level 1 sorceress", 1, 10, 35, 40, 35, 74},
		// the real level 94 save: stored 869 (= 849 + 20 quest life), 221, 525
		{"level 94 sorceress", 94, 368, 35, 849, 221, 525},
		{"one vitality point", 1, 11, 35, 42, 35, 75},
		{"one energy point", 1, 10, 36, 40, 37, 74},
	}

	for _, tc := range tests {
		l, m, s := sorc.BaseMax(tc.level, tc.vit, tc.ene)
		if l != tc.life || m != tc.mana || s != tc.stamina {
			t.Errorf("%s: got %d/%d/%d want %d/%d/%d", tc.name, l, m, s, tc.life, tc.mana, tc.stamina)
		}
	}
}

func hero() Hero {
	l, m, s := sorc.BaseMax(30, 50, 60)

	return Hero{Class: sorc, Level: 30, Str: 20, Dex: 40, Vit: 50, Ene: 60, BaseLife: l, BaseMana: m, BaseStam: s}
}

func TestComputeAttributesAndPools(t *testing.T) {
	h := hero()
	base := Compute(h, nil, nil)

	if base.MaxLife != h.BaseLife || base.MaxMana != h.BaseMana || base.MaxStamina != h.BaseStam {
		t.Fatalf("no items must keep the base maxima, got %d/%d/%d", base.MaxLife, base.MaxMana, base.MaxStamina)
	}

	items := []Item{
		{Slot: SlotAmulet, Props: []Prop{{ID: StatVitality, Value: 10}, {ID: StatEnergy, Value: 10}, {ID: StatMaxHP, Value: 20},
			{ID: StatMaxMana, Value: 30}, {ID: StatMaxStamina, Value: 40}}},
		{Slot: SlotBelt, Props: []Prop{{ID: StatMaxHPPct, Value: 10}}},
	}
	got := Compute(h, items, nil)

	// +10 vitality = +20 life and +10 stamina, +20 flat life, then +10% of the total
	wantLife := (h.BaseLife + 20 + 20) * 110 / 100
	wantMana := h.BaseMana + 20 + 30
	wantStam := h.BaseStam + 10 + 40

	if got.MaxLife != wantLife || got.MaxMana != wantMana || got.MaxStamina != wantStam {
		t.Errorf("got %d/%d/%d want %d/%d/%d", got.MaxLife, got.MaxMana, got.MaxStamina, wantLife, wantMana, wantStam)
	}

	if got.Vit != 60 || got.Ene != 70 {
		t.Errorf("attributes with items vit=%d ene=%d", got.Vit, got.Ene)
	}
}

func TestComputeActiveItems(t *testing.T) {
	h := hero()
	str := func(slot int, charm, broken bool) Item {
		return Item{Slot: slot, Charm: charm, Broken: broken, Props: []Prop{{ID: StatStrength, Value: 5}}}
	}

	tests := []struct {
		name string
		it   Item
		want int
	}{
		{"equipped slot", str(SlotHead, false, false), 25},
		{"weapon switch slot is inactive", str(SlotSwapRight, false, false), 20},
		{"inventory charm", str(0, true, false), 25},
		{"item in the stash is not passed", Item{Slot: 0, Props: []Prop{{ID: StatStrength, Value: 5}}}, 20},
		{"broken item", str(SlotHead, false, true), 20},
	}

	for _, tc := range tests {
		if got := Compute(h, []Item{tc.it}, nil).Str; got != tc.want {
			t.Errorf("%s: strength %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestDefense(t *testing.T) {
	h := hero() // dex 40 -> +10
	armor := Item{Slot: SlotTorso, Defense: 100, Props: []Prop{{ID: StatArmorPct, Value: 50}, {ID: StatArmorClass, Value: 30}}}
	// base 100 +50% = 150, flat +30 not enhanced; plus dex/4
	if got := Compute(h, []Item{armor}, nil).Defense; got != 150+30+10 {
		t.Errorf("defense %d", got)
	}

	// enhanced defense of a socketed jewel and the runeword enhances the same item
	armor.Sockets = []SocketedItem{{Code: "jew", Props: []Prop{{ID: StatArmorPct, Value: 10}}}}
	armor.RunewordProps = []Prop{{ID: StatArmorPct, Value: 40}}

	if got := Compute(h, []Item{armor}, nil).Defense; got != 200+30+10 {
		t.Errorf("defense with jewel and runeword %d", got)
	}

	// per level armor: (value*clvl)>>3
	pl := Item{Slot: SlotHead, Props: []Prop{{ID: 214, Value: 8}}}
	if got := Compute(h, []Item{pl}, nil).Defense; got != 30+10 {
		t.Errorf("armor per level %d", got)
	}

	if got := EtherealDefense(101); got != 151 {
		t.Errorf("ethereal defense %d", got)
	}
}

func TestResistances(t *testing.T) {
	h := hero()
	it := Item{Slot: SlotAmulet, Props: []Prop{
		{ID: StatFireResist, Value: 100}, {ID: StatColdResist, Value: 60}, {ID: StatMaxColdRes, Value: 10},
		{ID: StatLightResist, Value: 30}, {ID: StatPoisonResist, Value: -20},
	}}

	tests := []struct {
		diff int
		want [4]int
	}{
		{0, [4]int{75, 60, 30, -20}},
		// the penalty (-40) applies before the cap: fire 100-40 = 60
		{1, [4]int{60, 20, -10, -60}},
		{2, [4]int{0, -40, -70, -100}}, // -100 penalty; floor is -100
	}

	for _, tc := range tests {
		h.Difficulty = tc.diff

		if got := Compute(h, []Item{it}, nil); got.ResistShown != tc.want {
			t.Errorf("difficulty %d: %v want %v", tc.diff, got.ResistShown, tc.want)
		}
	}

	// max resist raises the cap to 85: 90 cold shows 85
	it.Props = []Prop{{ID: StatColdResist, Value: 90}, {ID: StatMaxColdRes, Value: 10}}
	h.Difficulty = 0

	if got := Compute(h, []Item{it}, nil); got.ResistShown[ResCold] != 85 || got.MaxResist[ResCold] != 85 {
		t.Errorf("cold %d cap %d", got.ResistShown[ResCold], got.MaxResist[ResCold])
	}
}

func TestQuestResistBonus(t *testing.T) {
	h := hero()
	base := Compute(h, nil, nil).Resist

	h.QuestResist = 10
	got := Compute(h, nil, nil)

	for i := range got.Resist {
		if got.Resist[i] != base[i]+10 {
			t.Errorf("resist %d: %d, want %d", i, got.Resist[i], base[i]+10)
		}
	}
}

func TestAttackRatingDamageBlockAndSkills(t *testing.T) {
	h := hero() // dex 40
	weapon := Item{Slot: SlotRightHand, Weapon: &WeaponBase{Min: 10, Max: 20, StrBonus: 100},
		Props: []Prop{{ID: StatMaxDmgPct, Value: 100}, {ID: StatMinDmgPct, Value: 100}, {ID: StatMinDamage, Value: 5},
			{ID: StatMaxDamage, Value: 5}, {ID: StatToHit, Value: 50}, {ID: StatAllSkills, Value: 1},
			{ID: StatSkillTab, Param: 1*8 + 2, Value: 2}, {ID: StatClassSkills, Param: 1, Value: 1}}}
	shield := Item{Slot: SlotLeftHand, BaseBlock: 30, Props: []Prop{{ID: StatToBlock, Value: 10}}}

	got := Compute(h, []Item{weapon, shield}, nil)

	// AR: (50 + (40-7)*5 - 15) = 200
	if got.AttackRating != 200 {
		t.Errorf("attack rating %d", got.AttackRating)
	}

	// damage: ((10*2+5) * (100+20 str))/100 = 30 ; ((20*2)+5)*1.2 = 54
	if got.DamageMin != 30 || got.DamageMax != 54 {
		t.Errorf("damage %d-%d", got.DamageMin, got.DamageMax)
	}

	// block (30+10+20 class) * (40-15) / (2*30) = 25
	if got.BlockPct != 25 {
		t.Errorf("block %d", got.BlockPct)
	}

	if b := got.SkillBonus(1, 2); b != 1+1+2 {
		t.Errorf("skill bonus %d", b)
	}

	if b := got.SkillBonus(1, 0); b != 1+1 {
		t.Errorf("skill bonus tab 0 %d", b)
	}

	if b := got.SkillBonus(0, 0); b != 1 {
		t.Errorf("other class %d", b)
	}

	// bare hands
	if bare := Compute(h, nil, nil); bare.DamageMin != 1 || bare.DamageMax != 2 {
		t.Errorf("bare hands %d-%d", bare.DamageMin, bare.DamageMax)
	}

	// ethereal weapons have +50% base damage
	weapon.Ethereal = true
	weapon.Props = nil

	if e := Compute(h, []Item{weapon}, nil); e.DamageMin != 15*120/100 || e.DamageMax != 30*120/100 {
		t.Errorf("ethereal %d-%d", e.DamageMin, e.DamageMax)
	}
}

func TestCriticalCombination(t *testing.T) {
	tt := Totals{DeadlyStrike: 50, CriticalStrike: 50}
	if got := tt.CombinedCritical(); got != 75 {
		t.Errorf("combined %d", got)
	}
}

func TestSetTiersAndGems(t *testing.T) {
	h := hero()
	piece := func(slot int) Item {
		return Item{Slot: slot, SetID: 7, SetLists: [][]Prop{{{ID: StatStrength, Value: 3}}, {{ID: StatStrength, Value: 4}}}}
	}

	for n, want := range map[int]int{1: 20, 2: 20 + 3*2, 3: 20 + (3+4)*3} {
		var items []Item

		for i := 0; i < n; i++ {
			items = append(items, piece(SlotHead+i*2))
		}

		if got := Compute(h, items, nil).Str; got != want {
			t.Errorf("%d pieces: strength %d want %d", n, got, want)
		}
	}

	env := &Env{
		Gems: GemTable{"gsr": {Weapon: []Prop{{ID: StatToHit, Value: 100}}, Helm: []Prop{{ID: StatDexterity, Value: 7}},
			Shield: []Prop{{ID: StatFireResist, Value: 20}}}},
		Sets: SetDefs{7: {Pieces: 2, Partial: [][]Prop{{{ID: StatVitality, Value: 5}}}, Full: []Prop{{ID: StatEnergy, Value: 9}}}},
	}

	helm := Item{Slot: SlotHead, SetID: 7, Sockets: []SocketedItem{{Code: "gsr"}}}
	shield := Item{Slot: SlotLeftHand, BaseBlock: 20, SetID: 7, Sockets: []SocketedItem{{Code: "gsr"}}}
	got := Compute(h, []Item{helm, shield}, env)

	if got.Dex != 47 || got.Resist[ResFire] != 20 || got.Vit != 55 || got.Ene != 69 {
		t.Errorf("gems/sets: dex=%d fire=%d vit=%d ene=%d", got.Dex, got.Resist[ResFire], got.Vit, got.Ene)
	}

	// runewords take the effect from their own list, the runes inside add nothing
	rw := Item{Slot: SlotHead, Runeword: true, Sockets: []SocketedItem{{Code: "gsr"}}}
	if g := Compute(h, []Item{rw}, env); g.Dex != 40 {
		t.Errorf("runeword socket counted: dex %d", g.Dex)
	}
}

func TestPerLevelStats(t *testing.T) {
	h := hero() // level 30
	// +8 life per 8 levels... item_hp_perlevel 8 -> 8*30>>3 = 30
	it := Item{Slot: SlotAmulet, Props: []Prop{{ID: 216, Value: 8}, {ID: 223, Value: 8}, {ID: 231, Value: 8}}}
	got := Compute(h, []Item{it}, nil)

	if got.Vit != 50+30 || got.Resist[ResFire] != 30 {
		t.Errorf("vit=%d fire=%d", got.Vit, got.Resist[ResFire])
	}

	// 30 flat life from the per level stat, 30 vitality = 60 life
	if want := h.BaseLife + 30 + 60; got.MaxLife != want {
		t.Errorf("life %d want %d", got.MaxLife, want)
	}
}

func TestListBasics(t *testing.T) {
	l := NewList()
	l.Add(StatFireResist, 0, 10)
	l.Add(StatSkillTab, 9, 1)
	l.Add(StatSkillTab, 10, 2)

	other := NewList()
	other.Add(StatFireResist, 0, 5)
	l.Merge(other)

	if l.Get(StatFireResist) != 15 || l.Sum(StatSkillTab) != 3 || l.GetParam(StatSkillTab, 10) != 2 || l.Len() != 3 {
		t.Errorf("list %s", l)
	}

	if l.String() != "39=15 188:9=1 188:10=2" {
		t.Errorf("string %q", l.String())
	}
}

const propsTxt = "code\tset1\tval1\tfunc1\tstat1\tset2\tval2\tfunc2\tstat2\n" +
	"res-fire\t\t\t1\tfireresist\t\t\t\t\n" +
	"skilltab\t\t\t10\titem_addskill_tab\t\t\t\t\n" +
	"allskills\t\t\t1\titem_allskills\t\t\t\t\n" +
	"ama\t\t0\t21\titem_addclassskills\t\t\t\t\n" +
	"sor\t\t1\t21\titem_addclassskills\t\t\t\t\n" +
	"dmg%\t\t\t7\titem_maxdamage_percent\t\t\t7\titem_mindamage_percent\n" +
	"skill\t\t\t22\titem_singleskill\t\t\t\t\n"

const statTxt = "Stat\tID\tValShift\top\top param\top base\top stat1\top stat2\top stat3\n" +
	"fireresist\t39\t\t\t\t\t\t\t\n" +
	"item_allskills\t127\t\t\t\t\t\t\t\n" +
	"item_addskill_tab\t188\t\t\t\t\t\t\t\n" +
	"item_addclassskills\t83\t\t\t\t\t\t\t\n" +
	"item_maxdamage_percent\t17\t\t\t\t\t\t\t\n" +
	"item_mindamage_percent\t18\t\t\t\t\t\t\t\n" +
	"item_singleskill\t107\t\t\t\t\t\t\t\n" +
	"maxhp\t7\t8\t\t\t\t\t\t\n" +
	"item_hp_perlevel\t216\t8\t2\t3\t\tmaxhp\t\t\n"

func TestPropertyExpansion(t *testing.T) {
	defs, err := ParseItemStatCost([]byte(statTxt))
	if err != nil {
		t.Fatal(err)
	}

	pt, err := ParseProperties([]byte(propsTxt), defs)
	if err != nil {
		t.Fatal(err)
	}

	pick := func(min, max int) int { return (min + max) / 2 }
	skillID := func(name string) int {
		if name == "Teleport" {
			return 54
		}

		return 0
	}

	tests := []struct {
		code, param string
		min, max    int
		want        []Prop
	}{
		{"res-fire", "", 10, 20, []Prop{{ID: 39, Value: 15}}},
		{"allskills", "", 1, 1, []Prop{{ID: 127, Value: 1}}},
		{"skilltab", "4", 2, 2, []Prop{{ID: 188, Param: 1*8 + 1, Value: 2}}},
		{"sor", "", 1, 1, []Prop{{ID: 83, Param: 1, Value: 1}}},
		{"dmg%", "", 100, 100, []Prop{{ID: 17, Value: 100}, {ID: 18, Value: 100}}},
		{"skill", "Teleport", 1, 1, []Prop{{ID: 107, Param: 54, Value: 1}}},
		{"nope", "", 1, 1, nil},
	}

	for _, tc := range tests {
		got := pt.Expand(tc.code, tc.param, tc.min, tc.max, pick, skillID)
		if len(got) != len(tc.want) {
			t.Errorf("%s: %v want %v", tc.code, got, tc.want)
			continue
		}

		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: %v want %v", tc.code, got, tc.want)
			}
		}
	}

	pl := defs.PerLevel()
	if len(pl) != 1 || pl[0] != (PerLevel{Stat: 216, Target: 7, Shift: 3}) {
		t.Errorf("per level %v", pl)
	}
}

// TestRealTables checks the built-in per level table, and the class formula
// for every class, against the game's tables (D2_TABLES with ItemStatCost.txt
// and CharStats.txt); skipped when unset.
func TestRealTables(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("set D2_TABLES to run")
	}

	isc, err := os.ReadFile(filepath.Join(dir, "ItemStatCost.txt"))
	if err != nil {
		t.Skip(err)
	}

	defs, err := ParseItemStatCost(isc)
	if err != nil {
		t.Fatal(err)
	}

	real := map[int]PerLevel{}
	for _, p := range defs.PerLevel() {
		real[p.Stat] = p
	}

	for _, b := range perLevelTable {
		p, ok := real[b.stat]
		if !ok || p.Target != b.target || p.Shift != b.shift {
			t.Errorf("per level stat %d: built-in {%d >> %d}, table %+v", b.stat, b.target, b.shift, p)
		}
	}

	cs, err := os.ReadFile(filepath.Join(dir, "CharStats.txt"))
	if err != nil {
		t.Skip(err)
	}

	classes, err := ParseClasses(cs)
	if err != nil {
		t.Fatal(err)
	}

	startLife := map[string]int{"Amazon": 50, "Sorceress": 40, "Necromancer": 45, "Paladin": 55, "Barbarian": 55, "Druid": 55, "Assassin": 50}
	for name, want := range startLife {
		c, ok := classes[name]
		if !ok {
			t.Errorf("class %s missing", name)
			continue
		}

		if l, _, _ := c.BaseMax(1, c.InitVit, c.InitEne); l != want {
			t.Errorf("%s starting life %d, want %d", name, l, want)
		}
	}
}
