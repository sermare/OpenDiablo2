package d2itemdesc

import (
	"reflect"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// These tests build a small table set in code (no game data needed) with
// the same rows and strings the game's tables have.

var testStrings = map[string]string{
	"ModStr1a": "to Strength", "ModStr1d": "to Energy", "ModStr1b": "to Dexterity", "ModStr1c": "to Vitality",
	"Moditem2allattrib": "to all Attributes",
	"Modstr2v":          "Enhanced Defense", "ModStr2j": "Enhanced Maximum Damage", "ModStr2k": "Enhanced Minimum Damage",
	"ModStr1g": "to Minimum Damage", "ModStr1f": "to Maximum Damage", "ModStr1i": "Defense",
	"ModStr1j": "Fire Resist", "ModStr1l": "Lightning Resist", "ModStr1k": "Cold Resist", "ModStr1n": "Poison Resist",
	"strModAllResistances": "All Resistances +%d",
	"ModStr2l":             "Replenish Life", "ModStr3h": "Requirements", "ModStr3a": "to Amazon Skill Levels",
	"ModStr3b": "to Sorceress Skill Levels", "ModStr3g": "Increased Chance of Blocking",
	"ModStre9s": "Indestructible", "ModStre9c": "(Based on Character Level)", "ModStr1u": "to Life",
	"ModStre10d": "(%d/%d Charges)", "ItemExpansiveChanc1": "%d%% Chance to cast level %d %s on striking",
	"ModitemAura": "Level %d %s Aura When Equipped", "ModStre9t": "Repairs %d durability per second",
	"ModStre9u":  "Repairs %d durability in %d seconds",
	"StrSklTabA": "+%d to Bow and Crossbow Skills", "StrSklTabB": "+%d to Fire Skills",
	"AmaOnly": "(Amazon Only)", "SorOnly": "(Sorceress Only)",
	"strModFireDamageRange": "Adds %d-%d fire damage", "strModFireDamage": "+%d fire damage",
	"strModColdDamageRange": "Adds %d-%d cold damage", "strModColdDamage": "+%d cold damage",
	"strModPoisonDamageRange": "Adds %d-%d poison damage over %d seconds",
	"strModPoisonDamage":      "+%d poison damage over %d seconds",
	"ModStr1p":                "to Minimum Fire Damage", "ModStr1o": "to Maximum Fire Damage",
	"SkillFireBolt": "Fire Bolt", "SkillTeleport": "Teleport", "SkillFanaticism": "Fanaticism", "SkillIceBlast": "Ice Blast",
	"ItemStats1h": "Defense:", "ItemStats1d": "Durability:", "ItemStats1j": "of", "ItemStats1e": "Required Strength:",
	"ItemStats1f": "Required Dexterity:", "ItemStats1p": "Required Level:", "ItemStats1i": "Quantity:",
	"ItemStats1l": "One-Hand Damage:", "ItemStats1m": "Two-Hand Damage:", "ItemStats1n": "Throw Damage:",
	"ItemStats1o": "Smite Damage:", "ItemStats1r": "Chance to Block: ", "ItemStats1b": "Unidentified",
	"to": "to", "Socketable": "Socketed", "Hiquality": "Superior", "Cracked": "Cracked", "Crude": "Crude",
	"HiqualityFormat": "%0 %1", "LowqualityFormat": "%0 %1", "MagicFormat": "%0 %1 %2", "RareFormat": "%0 %1",
	"Cost": "Cost: ", "Sell": "Sell value: ", "Repair": "Repair cost: ",
	"Sturdy": "Sturdy", "of the Fox": "of the Fox", "Beast": "Beast", "Hold": "Hold",
	"ExInsertSockets": "Can be Inserted into Socketed Items", "GemXp1": "Helms:", "GemXp2": "Shields:", "GemXp3": "Weapons:",
	"Runeword130":     "Spirit",
	"ItemStatsrejuv1": "Heals 35% Life and Mana",
	"Civerb's Ward":   "Civerb's Ward", "Civerb's Vestments": "Civerb's Vestments", "Civerb's Icon": "Civerb's Icon",
	"Civerb's Cudgel": "Civerb's Cudgel",
}

func testTables() *Tables {
	t := &Tables{
		Tr:           func(k string) string { return testStrings[k] },
		Stats:        map[int]*StatDef{},
		StatByName:   map[string]*StatDef{},
		Props:        map[string][]PropSlot{},
		Bases:        map[string]*BaseItem{},
		Gems:         map[string]*GemDef{},
		Skills:       map[int]*SkillDef{},
		Sets:         map[string]*SetDef{},
		types:        map[string][]string{"shld": {"armo"}, "shie": {"shld"}, "pala": {"shld"}, "swor": {"weap"}},
		typeClass:    map[string]string{"pala": "pal", "amaz": "ama"},
		EtherealText: "Ethereal (Cannot be Repaired)",
	}

	for _, d := range []*StatDef{
		{ID: 0, Name: "strength", Priority: 67, Func: 1, Val: 1, StrPos: "ModStr1a", StrNeg: "ModStr1a", Grp: 1, GrpFunc: 1, GrpVal: 1, GrpStrPos: "Moditem2allattrib", GrpStrNeg: "Moditem2allattrib"},
		{ID: 1, Name: "energy", Priority: 61, Func: 1, Val: 1, StrPos: "ModStr1d", StrNeg: "ModStr1d", Grp: 1, GrpFunc: 1, GrpVal: 1, GrpStrPos: "Moditem2allattrib", GrpStrNeg: "Moditem2allattrib"},
		{ID: 2, Name: "dexterity", Priority: 65, Func: 1, Val: 1, StrPos: "ModStr1b", StrNeg: "ModStr1b", Grp: 1, GrpFunc: 1, GrpVal: 1, GrpStrPos: "Moditem2allattrib", GrpStrNeg: "Moditem2allattrib"},
		{ID: 3, Name: "vitality", Priority: 63, Func: 1, Val: 1, StrPos: "ModStr1c", StrNeg: "ModStr1c", Grp: 1, GrpFunc: 1, GrpVal: 1, GrpStrPos: "Moditem2allattrib", GrpStrNeg: "Moditem2allattrib"},
		{ID: 7, Name: "maxhp", Priority: 59, Func: 1, Val: 1, StrPos: "ModStr1u", StrNeg: "ModStr1u"},
		{ID: 16, Name: "item_armor_percent", Priority: 74, Func: 4, Val: 1, StrPos: "Modstr2v", StrNeg: "Modstr2v"},
		{ID: 17, Name: "item_maxdamage_percent", Priority: 129, Func: 3, StrPos: "ModStr2j", StrNeg: "ModStr2j"},
		{ID: 18, Name: "item_mindamage_percent", Priority: 130, Func: 3, StrPos: "ModStr2k", StrNeg: "ModStr2k"},
		{ID: 20, Name: "toblock", Priority: 134, Func: 2, Val: 1, StrPos: "ModStr3g", StrNeg: "ModStr3g"},
		{ID: 21, Name: "mindamage", Priority: 127, Func: 1, Val: 1, StrPos: "ModStr1g", StrNeg: "ModStr1g"},
		{ID: 22, Name: "maxdamage", Priority: 126, Func: 1, Val: 1, StrPos: "ModStr1f", StrNeg: "ModStr1f"},
		{ID: 23, Name: "secondary_mindamage", Priority: 124, Func: 1, Val: 1, StrPos: "ModStr1g", StrNeg: "ModStr1g"},
		{ID: 31, Name: "armorclass", Priority: 71, Func: 1, Val: 1, StrPos: "ModStr1i", StrNeg: "ModStr1i"},
		{ID: 39, Name: "fireresist", Priority: 36, Func: 4, Val: 2, StrPos: "ModStr1j", StrNeg: "ModStr1j", Grp: 2, GrpFunc: 19, GrpStrPos: "strModAllResistances", GrpStrNeg: "strModAllResistances"},
		{ID: 41, Name: "lightresist", Priority: 38, Func: 4, Val: 2, StrPos: "ModStr1l", StrNeg: "ModStr1l", Grp: 2, GrpFunc: 19, GrpStrPos: "strModAllResistances", GrpStrNeg: "strModAllResistances"},
		{ID: 43, Name: "coldresist", Priority: 40, Func: 4, Val: 2, StrPos: "ModStr1k", StrNeg: "ModStr1k", Grp: 2, GrpFunc: 19, GrpStrPos: "strModAllResistances", GrpStrNeg: "strModAllResistances"},
		{ID: 45, Name: "poisonresist", Priority: 34, Func: 4, Val: 2, StrPos: "ModStr1n", StrNeg: "ModStr1n", Grp: 2, GrpFunc: 19, GrpStrPos: "strModAllResistances", GrpStrNeg: "strModAllResistances"},
		{ID: 48, Name: "firemindam", Priority: 102, Func: 1, Val: 1, StrPos: "ModStr1p", StrNeg: "ModStr1p"},
		{ID: 49, Name: "firemaxdam", Priority: 101, Func: 1, Val: 1, StrPos: "ModStr1o", StrNeg: "ModStr1o"},
		{ID: 54, Name: "coldmindam", Priority: 96, Func: 1, Val: 1, StrPos: "ModStr1k", StrNeg: "ModStr1k"},
		{ID: 55, Name: "coldmaxdam", Priority: 95, Func: 1, Val: 1, StrPos: "ModStr1k", StrNeg: "ModStr1k"},
		{ID: 56, Name: "coldlength", Priority: 0},
		{ID: 57, Name: "poisonmindam", Priority: 92, Func: 1, Val: 1, StrPos: "ModStr1n", StrNeg: "ModStr1n"},
		{ID: 58, Name: "poisonmaxdam", Priority: 91, Func: 1, Val: 1, StrPos: "ModStr1n", StrNeg: "ModStr1n"},
		{ID: 59, Name: "poisonlength", Priority: 0},
		{ID: 74, Name: "hpregen", Priority: 56, Func: 1, Val: 2, StrPos: "ModStr2l", StrNeg: "ModStr2l"},
		{ID: 83, Name: "item_addclassskills", Priority: 150, Func: 13, Val: 1},
		{ID: 91, Name: "item_req_percent", Priority: 0, Func: 4, Val: 2, StrPos: "ModStr3h", StrNeg: "ModStr3h"},
		{ID: 107, Name: "item_singleskill", Priority: 81, Func: 27},
		{ID: 97, Name: "item_nonclassskill", Priority: 81, Func: 28},
		{ID: 151, Name: "item_aura", Priority: 159, Func: 16, StrPos: "ModitemAura", StrNeg: "ModitemAura"},
		{ID: 152, Name: "item_indesctructible", Priority: 160, Func: 3, StrPos: "ModStre9s", StrNeg: "ModStre9s"},
		{ID: 188, Name: "item_addskill_tab", Priority: 151, Func: 14},
		{ID: 198, Name: "item_skillonhit", Priority: 160, Func: 15, StrPos: "ItemExpansiveChanc1", StrNeg: "ItemExpansiveChanc1"},
		{ID: 204, Name: "item_charged_skill", Priority: 1, Func: 24, StrPos: "ModStre10d", StrNeg: "ModStre10d"},
		{ID: 216, Name: "item_hp_perlevel", Priority: 57, Func: 6, Val: 1, StrPos: "ModStr1u", StrNeg: "ModStr1u", Str2: "ModStre9c", Op: 2, OpParam: 3},
		{ID: 252, Name: "item_replenish_durability", Priority: 1, Func: 11, StrPos: "ModStre9t", StrNeg: "ModStre9t"},
	} {
		t.Stats[d.ID], t.StatByName[d.Name] = d, d
	}

	t.Classes = []ClassDef{
		{Name: "Amazon", AllSkills: "ModStr3a", SkillTabs: [3]string{"StrSklTabA", "", ""}, ClassOnly: "AmaOnly"},
		{Name: "Sorceress", AllSkills: "ModStr3b", SkillTabs: [3]string{"", "", "StrSklTabB"}, ClassOnly: "SorOnly"},
	}
	t.Skills = map[int]*SkillDef{
		36:  {Code: "Fire Bolt", Name: "SkillFireBolt", Class: 1},
		54:  {Code: "Teleport", Name: "SkillTeleport", Class: 1},
		123: {Code: "Fanaticism", Name: "SkillFanaticism", Class: 3},
		245: {Code: "Ice Blast", Name: "SkillIceBlast", Class: 1},
	}

	return t
}

func TestFormatSubstitution(t *testing.T) {
	tests := []struct {
		format string
		parts  []string
		want   string
	}{
		{"%0 %1 %2", []string{"Sturdy", "Cap", "of the Fox"}, "Sturdy Cap of the Fox"},
		{"%0 %1 %2", []string{"", "Cap", "of the Fox"}, "Cap of the Fox"},
		{"%0 %1 %2", []string{"Sturdy", "Cap", ""}, "Sturdy Cap"},
		{"%2 %1 %0", []string{"A", "B", "C"}, "C B A"}, // a language that reorders the parts
		{"%1", []string{"A", "B"}, "B"},
	}

	for _, tc := range tests {
		if got := Format(tc.format, tc.parts...); got != tc.want {
			t.Errorf("Format(%q, %v) = %q, want %q", tc.format, tc.parts, got, tc.want)
		}
	}
}

func TestStatLines(t *testing.T) {
	tb := testTables()

	tests := []struct {
		name  string
		stats []Stat
		ctx   *Context
		want  []string
	}{
		{"func1 val1", []Stat{{ID: 0, Value: 5}}, nil, []string{"+5 to Strength"}},
		{"negative value keeps its sign", []Stat{{ID: 0, Value: -5}}, nil, []string{"-5 to Strength"}},
		{"func1 val2 puts the value after", []Stat{{ID: 74, Value: 20}}, nil, []string{"Replenish Life +20"}},
		{"func4 val1 percent", []Stat{{ID: 16, Value: 101}}, nil, []string{"+101% Enhanced Defense"}},
		{"func4 val2 percent after", []Stat{{ID: 39, Value: 28}}, nil, []string{"Fire Resist +28%"}},
		{"requirements percent negative", []Stat{{ID: 91, Value: -50}}, nil, []string{"Requirements -50%"}},
		{"func2 val1", []Stat{{ID: 20, Value: 10}}, nil, []string{"10% Increased Chance of Blocking"}},
		{"func3 val0 has no number", []Stat{{ID: 152, Value: 1}}, nil, []string{"Indestructible"}},
		{"all attributes group", []Stat{{ID: 0, Value: 5}, {ID: 1, Value: 5}, {ID: 2, Value: 5}, {ID: 3, Value: 5}}, nil,
			[]string{"+5 to all Attributes"}},
		{"attributes differ so no group", []Stat{{ID: 0, Value: 5}, {ID: 1, Value: 5}, {ID: 2, Value: 5}, {ID: 3, Value: 6}}, nil,
			[]string{"+5 to Strength", "+5 to Dexterity", "+6 to Vitality", "+5 to Energy"}},
		{"all resistances group", []Stat{{ID: 39, Value: 20}, {ID: 41, Value: 20}, {ID: 43, Value: 20}, {ID: 45, Value: 20}}, nil,
			[]string{"All Resistances +20"}},
		{"three resistances stay single", []Stat{{ID: 39, Value: 20}, {ID: 41, Value: 20}, {ID: 43, Value: 20}}, nil,
			[]string{"Cold Resist +20%", "Lightning Resist +20%", "Fire Resist +20%"}},
		{"fire damage range", []Stat{{ID: 48, Value: 1}, {ID: 49, Value: 6}}, nil, []string{"Adds 1-6 fire damage"}},
		{"fire damage single value", []Stat{{ID: 48, Value: 4}, {ID: 49, Value: 4}}, nil, []string{"+4 fire damage"}},
		{"cold damage ignores the freeze length", []Stat{{ID: 54, Value: 3}, {ID: 55, Value: 7}, {ID: 56, Value: 25}}, nil,
			[]string{"Adds 3-7 cold damage"}},
		{"poison damage over time", []Stat{{ID: 57, Value: 256}, {ID: 58, Value: 512}, {ID: 59, Value: 250}}, nil,
			[]string{"Adds 250-500 poison damage over 10 seconds"}},
		{"min without max falls back", []Stat{{ID: 48, Value: 4}}, nil, []string{"+4 to Minimum Fire Damage"}},
		{"enhanced damage equal", []Stat{{ID: 17, Value: 50}, {ID: 18, Value: 50}}, nil, []string{"+50% Enhanced Damage"}},
		{"enhanced damage differs", []Stat{{ID: 17, Value: 60}, {ID: 18, Value: 50}}, nil,
			[]string{"+50% Enhanced Minimum Damage", "+60% Enhanced Maximum Damage"}},
		{"min damage mirrored by secondary is listed once", []Stat{{ID: 21, Value: 3}, {ID: 23, Value: 3}}, nil,
			[]string{"+3 to Minimum Damage"}},
		{"class skills", []Stat{{ID: 83, Param: 1, Value: 3}}, nil, []string{"+3 to Sorceress Skill Levels"}},
		{"skill tab with class", []Stat{{ID: 188, Param: 0 | 0<<3, Value: 2}}, nil,
			[]string{"+2 to Bow and Crossbow Skills (Amazon Only)"}},
		{"skill tab sorceress", []Stat{{ID: 188, Param: 2 | 1<<3, Value: 1}}, nil,
			[]string{"+1 to Fire Skills (Sorceress Only)"}},
		{"single skill with class", []Stat{{ID: 107, Param: 54, Value: 3}}, nil, []string{"+3 to Teleport (Sorceress Only)"}},
		{"non class skill", []Stat{{ID: 97, Param: 54, Value: 3}}, nil, []string{"+3 to Teleport"}},
		{"skill on striking", []Stat{{ID: 198, Param: 36<<6 | 5, Value: 10}}, nil,
			[]string{"10% Chance to cast level 5 Fire Bolt on striking"}},
		{"aura", []Stat{{ID: 151, Param: 123, Value: 12}}, nil, []string{"Level 12 Fanaticism Aura When Equipped"}},
		{"charges", []Stat{{ID: 204, Param: 245<<6 | 14, Value: 60 | 60<<8}}, nil,
			[]string{"Level 14 Ice Blast (60/60 Charges)"}},
		{"repair rate", []Stat{{ID: 252, Value: 20}}, nil, []string{"Repairs 1 durability in 5 seconds"}},
		{"per level scales with the hero", []Stat{{ID: 216, Value: 16}}, &Context{CharLevel: 40}, []string{"+80 to Life (Based on Character Level)"}},
		{"sorted by descpriority", []Stat{{ID: 7, Value: 10}, {ID: 31, Value: 5}, {ID: 16, Value: 30}}, nil,
			[]string{"+30% Enhanced Defense", "+5 Defense", "+10 to Life"}},
		{"same stat id and param add up", []Stat{{ID: 7, Value: 10}, {ID: 7, Value: 5}}, nil, []string{"+15 to Life"}},
		{"unknown stat is skipped", []Stat{{ID: 9999, Value: 1}}, nil, nil},
	}

	for _, tc := range tests {
		got := tb.StatLines(Merge(tc.stats), tc.ctx)
		if !reflect.DeepEqual(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}

func TestPriceLine(t *testing.T) {
	tb := testTables()

	tests := []struct {
		kind  PriceKind
		price int
		gold  int
		want  Line
	}{
		{PriceBuy, 120, 500, Line{"Cost: 120", Gold}},
		{PriceBuy, 800, 500, Line{"Cost: 800", Red}},
		{PriceBuy, 800, -1, Line{"Cost: 800", Gold}},
		{PriceSell, 33, 0, Line{"Sell value: 33", Gold}},
		{PriceRepair, 7, 0, Line{"Repair cost: 7", Gold}},
	}

	for _, tc := range tests {
		if got := tb.PriceLine(tc.kind, tc.price, tc.gold); got != tc.want {
			t.Errorf("PriceLine(%v, %d, %d) = %+v, want %+v", tc.kind, tc.price, tc.gold, got, tc.want)
		}
	}
}

func describeTables() *Tables {
	t := testTables()

	t.Bases["cap"] = &BaseItem{Code: "cap", Name: "Cap", Type: "helm", Armor: true, MinAC: 3, MaxAC: 5, ReqStr: 0, Durability: 12}
	t.Bases["plt"] = &BaseItem{Code: "plt", Name: "Plate Mail", Type: "tors", Armor: true, MinAC: 108, MaxAC: 150, ReqStr: 65, Durability: 60, ReqLevel: 25}
	t.Bases["lrg"] = &BaseItem{Code: "lrg", Name: "Large Shield", Type: "shie", Armor: true, MinAC: 12, MaxAC: 16, Block: 24, ReqStr: 34, Durability: 24}
	t.Bases["ssd"] = &BaseItem{Code: "ssd", Name: "Short Sword", Type: "swor", Weapon: true, MinDam: 2, MaxDam: 7, ReqStr: 25, ReqDex: 25, Durability: 24, MissMin: 0}
	t.Bases["bsd"] = &BaseItem{Code: "bsd", Name: "Bastard Sword", Type: "swor", Weapon: true, MinDam: 7, MaxDam: 19, OneOrTwo: true, TwoMin: 15, TwoMax: 36, ReqStr: 62, Durability: 32}
	t.Bases["jav"] = &BaseItem{Code: "jav", Name: "Javelin", Type: "jave", Weapon: true, MinDam: 1, MaxDam: 5, MissMin: 6, MissMax: 11, Stackable: true, ReqStr: 25, Durability: 0, NoDurability: true}
	t.Bases["rin"] = &BaseItem{Code: "rin", Name: "Ring", Type: "ring", NoDurability: true}
	t.Bases["aam"] = &BaseItem{Code: "aam", Name: "Ancestral Spear", Type: "amaz", Weapon: true, MinDam: 5, MaxDam: 10}
	t.Bases["rvs"] = &BaseItem{Code: "rvs", Name: "Rejuvenation Potion", Type: "rpot"}
	t.Bases["gsw"] = &BaseItem{Code: "gsw", Name: "Flawless Sapphire", Type: "gem0", ReqLevel: 18}

	t.Prefixes = []AffixDef{{}, {Name: "Sturdy", LevelReq: 3}}
	t.Suffixes = []AffixDef{{}, {Name: "of the Fox", LevelReq: 9}}
	t.RarePrefixes = []string{"Beast"}
	t.RareSuffix = []string{"Hold"}
	t.Uniques = []UniqueDef{{Index: "Civerb's Cudgel", Code: "ssd", LevelReq: 11}}
	t.Runewords = make([]RunewordDef, 200)
	t.Runewords[155-runewordIDBase] = RunewordDef{Key: "Runeword130", Display: "Spirit"}
	t.SetItems = []SetItemDef{
		{Index: "Civerb's Ward", Set: "Civerb's Vestments", Code: "lrg", LevelReq: 9},
		{Index: "Civerb's Icon", Set: "Civerb's Vestments", Code: "rin", LevelReq: 9},
	}
	t.Sets["Civerb's Vestments"] = &SetDef{
		Index: "Civerb's Vestments", Name: "Civerb's Vestments", Items: []string{"Civerb's Ward", "Civerb's Icon"},
		Partial: [4][]PropSpec{{{Code: "str", Min: 4, Max: 4}}},
		Full:    []PropSpec{{Code: "hp", Min: 50, Max: 50}},
	}

	t.Props["str"] = []PropSlot{{Func: 1, Stat: "strength"}}
	t.Props["hp"] = []PropSlot{{Func: 1, Stat: "maxhp"}}
	t.Props["res-fire"] = []PropSlot{{Func: 1, Stat: "fireresist"}}
	t.Props["dmg-fire"] = []PropSlot{{Func: 15, Stat: "firemindam"}, {Func: 16, Stat: "firemaxdam"}}
	t.Gems["gsw"] = &GemDef{Name: "Flawless Sapphire", Code: "gsw",
		Weapon: []PropSpec{{Code: "dmg-fire", Min: 5, Max: 9}},
		Helm:   []PropSpec{{Code: "hp", Min: 24, Max: 24}},
		Shield: []PropSpec{{Code: "res-fire", Min: 22, Max: 22}},
	}
	t.types["rpot"] = nil

	return t
}

func lineTexts(ls []Line) []string { return texts(ls) }

func TestDescribe(t *testing.T) {
	tb := describeTables()
	hero := &Context{HasHero: true, CharLevel: 20, Strength: 50, Dexterity: 30, Class: 1}

	tests := []struct {
		name string
		it   d2s.Item
		ctx  *Context
		want []Line
	}{
		{"normal armor", d2s.Item{Code: "cap", Quality: d2s.QualityNormal, Identified: true, Defense: 4, MaxDurability: 12, Durability: 12}, nil,
			[]Line{{"Cap", White}, {"Defense: 4", White}, {"Durability: 12 of 12", White}}},
		{"superior name and blue defense", d2s.Item{Code: "cap", Quality: d2s.QualityHigh, Identified: true, Defense: 5, MaxDurability: 12, Durability: 12}, nil,
			[]Line{{"Superior Cap", White}, {"Defense: 5", Blue}, {"Durability: 12 of 12", White}}},
		{"low quality is gray", d2s.Item{Code: "cap", Quality: d2s.QualityLow, LowQualityID: 1, Identified: true, Defense: 3, MaxDurability: 10, Durability: 3}, nil,
			[]Line{{"Cracked Cap", Gray}, {"Defense: 3", White}, {"Durability: 3 of 10", White}}},
		{"requirements met are white", d2s.Item{Code: "plt", Quality: d2s.QualityNormal, Identified: true, Defense: 120, MaxDurability: 60, Durability: 60}, hero,
			[]Line{{"Plate Mail", White}, {"Defense: 120", White}, {"Durability: 60 of 60", White},
				{"Required Strength: 65", Red}, {"Required Level: 25", Red}}},
		{"requirements met are white 2", d2s.Item{Code: "plt", Quality: d2s.QualityNormal, Identified: true, Defense: 120, MaxDurability: 60, Durability: 60},
			&Context{HasHero: true, CharLevel: 30, Strength: 80},
			[]Line{{"Plate Mail", White}, {"Defense: 120", White}, {"Durability: 60 of 60", White},
				{"Required Strength: 65", White}, {"Required Level: 25", White}}},
		{"ethereal lowers strength and shows the line", d2s.Item{Code: "plt", Quality: d2s.QualityNormal, Identified: true, Ethereal: true, Defense: 180, MaxDurability: 30, Durability: 30}, nil,
			[]Line{{"Plate Mail", Gray}, {"Defense: 180", Blue}, {"Durability: 30 of 30", White},
				{"Required Strength: 55", White}, {"Required Level: 25", White}, {"Ethereal (Cannot be Repaired)", Blue}}},
		{"shield block and socket count", d2s.Item{Code: "lrg", Quality: d2s.QualityNormal, Identified: true, Socketed: true, TotalSockets: 2, Defense: 14, MaxDurability: 24, Durability: 20}, nil,
			[]Line{{"Large Shield", Gray}, {"Defense: 14", White}, {"Chance to Block: 24%", White}, {"Durability: 20 of 24", White},
				{"Required Strength: 34", White}, {"Socketed (2)", Blue}}},
		{"weapon damage with enhancement", d2s.Item{Code: "ssd", Quality: d2s.QualityMagic, Identified: true, MagicPrefix: 1, MagicSuffix: 1, MaxDurability: 24, Durability: 24,
			Properties: []d2s.Property{{ID: 17, Value: 100}, {ID: 18, Value: 100}, {ID: 22, Value: 3}}}, nil,
			[]Line{{"Sturdy Short Sword of the Fox", Blue}, {"One-Hand Damage: 4 to 17", Blue}, {"Durability: 24 of 24", White},
				{"Required Strength: 25", White}, {"Required Dexterity: 25", White}, {"Required Level: 9", White},
				{"+100% Enhanced Damage", Blue}, {"+3 to Maximum Damage", Blue}}},
		{"one or two handed shows both", d2s.Item{Code: "bsd", Quality: d2s.QualityNormal, Identified: true, MaxDurability: 32, Durability: 32}, nil,
			[]Line{{"Bastard Sword", White}, {"One-Hand Damage: 7 to 19", White}, {"Two-Hand Damage: 15 to 36", White},
				{"Durability: 32 of 32", White}, {"Required Strength: 62", White}}},
		{"throwing weapon quantity", d2s.Item{Code: "jav", Quality: d2s.QualityNormal, Identified: true, Quantity: 40}, nil,
			[]Line{{"Javelin", White}, {"One-Hand Damage: 1 to 5", White}, {"Throw Damage: 6 to 11", White},
				{"Quantity: 40", White}, {"Required Strength: 25", White}}},
		{"unidentified magic item", d2s.Item{Code: "cap", Quality: d2s.QualityMagic, Identified: false, MagicPrefix: 1, Defense: 4, MaxDurability: 12, Durability: 12,
			Properties: []d2s.Property{{ID: 7, Value: 10}}}, nil,
			[]Line{{"Cap", Blue}, {"Defense: 4", White}, {"Durability: 12 of 12", White}, {"Required Level: 3", White}, {"Unidentified", Red}}},
		{"rare item has two name lines", d2s.Item{Code: "cap", Quality: d2s.QualityRare, Identified: true, RareName1: 156, RareName2: 1, Defense: 4, MaxDurability: 12, Durability: 12,
			Properties: []d2s.Property{{ID: 7, Value: 10}}}, nil,
			[]Line{{"Beast Hold", Yellow}, {"Cap", Yellow}, {"Defense: 4", White}, {"Durability: 12 of 12", White}, {"+10 to Life", Blue}}},
		{"unique item", d2s.Item{Code: "ssd", Quality: d2s.QualityUnique, Identified: true, UniqueID: 0, MaxDurability: 24, Durability: 24,
			Properties: []d2s.Property{{ID: 16, Value: 5}}}, nil,
			[]Line{{"Civerb's Cudgel", Gold}, {"Short Sword", Gold}, {"One-Hand Damage: 2 to 7", White}, {"Durability: 24 of 24", White},
				{"Required Strength: 25", White}, {"Required Dexterity: 25", White}, {"Required Level: 11", White}, {"+5% Enhanced Defense", Blue}}},
		{"runeword: name gold, base gray, sockets from runes", d2s.Item{Code: "lrg", Quality: d2s.QualityNormal, Identified: true, Socketed: true, Runeword: true, RunewordID: 155,
			TotalSockets: 1, Defense: 14, MaxDurability: 24, Durability: 24, RunewordProperties: []d2s.Property{{ID: 0, Value: 2}},
			Children: []d2s.Item{{Code: "gsw", Location: d2s.LocationSocketed}}}, nil,
			[]Line{{"Spirit", Gold}, {"Large Shield", Gray}, {"Defense: 14", White}, {"Chance to Block: 24%", White}, {"Durability: 24 of 24", White},
				{"Required Strength: 34", White}, {"Required Level: 18", White}, {"+2 to Strength", Blue}, {"Fire Resist +22%", Blue}, {"Socketed (1)", Blue}}},
		{"personalized name", d2s.Item{Code: "cap", Quality: d2s.QualityNormal, Identified: true, Personalized: true, PersonalName: "Sergio", Defense: 4, MaxDurability: 12, Durability: 12}, nil,
			[]Line{{"Sergio's Cap", White}, {"Defense: 4", White}, {"Durability: 12 of 12", White}}},
		{"indestructible hides durability", d2s.Item{Code: "cap", Quality: d2s.QualityMagic, Identified: true, Defense: 4, MaxDurability: 12, Durability: 12,
			Properties: []d2s.Property{{ID: 152, Value: 1}}}, nil,
			[]Line{{"Cap", Blue}, {"Defense: 4", White}, {"Indestructible", Blue}}},
		{"class only item in red for another class", d2s.Item{Code: "aam", Quality: d2s.QualityNormal, Identified: true}, hero,
			[]Line{{"Ancestral Spear", White}, {"One-Hand Damage: 5 to 10", White}, {"(Amazon Only)", Red}}},
		{"rejuvenation potion", d2s.Item{Code: "rvs", Quality: d2s.QualityNormal, Identified: true}, nil,
			[]Line{{"Rejuvenation Potion", White}, {"Heals 35% Life and Mana", White}}},
		{"gem lists its effects per kind of item", d2s.Item{Code: "gsw", Quality: d2s.QualityNormal, Identified: true}, nil,
			[]Line{{"Flawless Sapphire", White}, {"Can be Inserted into Socketed Items", White}, {"Weapons: Adds 5-9 fire damage", Blue},
				{"Helms: +24 to Life", Blue}, {"Shields: Fire Resist +22%", Blue}, {"Required Level: 18", White}}},
	}

	for _, tc := range tests {
		it := tc.it
		if it.Quality == 0 {
			it.Quality = d2s.QualityNormal
		}

		got := tb.Describe(&it, tc.ctx)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %v\nwant %v", tc.name, fmtLines(got), fmtLines(tc.want))
		}
	}
}

func fmtLines(ls []Line) string {
	var b strings.Builder

	for _, l := range ls {
		b.WriteString("\n   [" + l.Color.String() + "] " + l.Text)
	}

	return b.String()
}

func TestDescribeSetBonuses(t *testing.T) {
	tb := describeTables()
	ward := d2s.Item{Code: "lrg", Quality: d2s.QualitySet, Identified: true, SetID: 0, SetListMask: 0b00001, Defense: 14, MaxDurability: 24, Durability: 24,
		Properties:    []d2s.Property{{ID: 16, Value: 15}},
		SetProperties: [][]d2s.Property{{{ID: 7, Value: 21}}}}

	head := []Line{{"Civerb's Ward", Green}, {"Large Shield", Gold}, {"Defense: 16", Blue}, {"Chance to Block: 24%", White},
		{"Durability: 24 of 24", White}, {"Required Strength: 34", White}, {"Required Level: 9", White}, {"+15% Enhanced Defense", Blue}}

	tests := []struct {
		name string
		worn map[string]bool
		want []Line
	}{
		{"nothing worn: all pieces missing, no bonus", nil, append(append([]Line{}, head...),
			Line{"", White}, Line{"Civerb's Vestments", Gold}, Line{"Civerb's Ward", Red}, Line{"Civerb's Icon", Red})},
		{"two pieces: item and set bonus active", map[string]bool{"Civerb's Ward": true, "Civerb's Icon": true},
			append(append([]Line{}, head...), Line{"+21 to Life", Green},
				Line{"", White}, Line{"Civerb's Vestments", Gold}, Line{"Civerb's Ward", Green}, Line{"Civerb's Icon", Green},
				Line{"+4 to Strength", Green}, Line{"+50 to Life", Gold})},
	}

	for _, tc := range tests {
		it := ward
		got := tb.Describe(&it, &Context{WornSetItems: tc.worn})

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %v\nwant %v", tc.name, fmtLines(got), fmtLines(tc.want))
		}
	}
}

func TestEvalPropFunctions(t *testing.T) {
	tb := describeTables()
	tb.Props["skilltab"] = []PropSlot{{Func: 10, Stat: "item_addskill_tab"}}
	tb.Props["sor"] = []PropSlot{{Func: 21, Stat: "item_addclassskills"}}
	tb.Props["dmg%"] = []PropSlot{{Func: 7}}
	tb.Props["dmg-min"] = []PropSlot{{Func: 5}}
	tb.Props["indestruct"] = []PropSlot{{Func: 20}}
	tb.Props["res-all"] = []PropSlot{{Func: 1, Stat: "fireresist"}, {Func: 3, Stat: "lightresist"}, {Func: 3, Stat: "coldresist"}, {Func: 3, Stat: "poisonresist"}}

	tests := []struct {
		spec PropSpec
		want []Stat
	}{
		{PropSpec{Code: "str", Min: 3, Max: 3}, []Stat{{ID: 0, Value: 3}}},
		{PropSpec{Code: "res-all", Max: 10}, []Stat{{ID: 39, Value: 10}, {ID: 41, Value: 10}, {ID: 43, Value: 10}, {ID: 45, Value: 10}}},
		{PropSpec{Code: "dmg-fire", Min: 5, Max: 9}, []Stat{{ID: 48, Value: 5}, {ID: 49, Value: 9}}},
		{PropSpec{Code: "dmg%", Max: 30}, []Stat{{ID: 17, Value: 30}, {ID: 18, Value: 30}}},
		{PropSpec{Code: "dmg-min", Max: 2}, []Stat{{ID: 21, Value: 2}}},
		{PropSpec{Code: "skilltab", Param: "4", Max: 1}, []Stat{{ID: 188, Param: 1 | 1<<3, Value: 1}}}, // class 1 tab 1
		{PropSpec{Code: "sor", Max: 2}, []Stat{{ID: 83, Param: 1, Value: 2}}},
		{PropSpec{Code: "indestruct"}, []Stat{{ID: 152, Value: 1}}},
		{PropSpec{Code: "nosuchprop", Max: 1}, nil},
	}

	for _, tc := range tests {
		if got := tb.EvalProp(tc.spec); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("EvalProp(%+v) = %+v, want %+v", tc.spec, got, tc.want)
		}
	}
}
