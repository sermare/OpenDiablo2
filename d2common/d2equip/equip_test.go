package d2equip

import (
	"os"
	"path/filepath"
	"testing"
)

const miniTypes = "ItemType\tCode\tEquiv1\tEquiv2\tRepair\tBody\tBodyLoc1\tBodyLoc2\tShoots\tQuiver\tClass\n" +
	"Shield\tshie\tshld\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Armor\ttors\tarmo\t\t1\t1\ttors\ttors\t\t\t\n" +
	"Bow Quiver\tbowq\tmisl\t\t0\t1\trarm\tlarm\t\tbow\t\n" +
	"Xbow Quiver\txboq\tmisl\t\t0\t1\trarm\tlarm\t\txbow\t\n" +
	"Ring\tring\tmisc\t\t0\t1\trrin\tlrin\t\t\t\n" +
	"Amulet\tamul\tmisc\t\t0\t1\tneck\tneck\t\t\t\n" +
	"Belt\tbelt\tarmo\t\t1\t1\tbelt\tbelt\t\t\t\n" +
	"Boots\tboot\tarmo\t\t1\t1\tfeet\tfeet\t\t\t\n" +
	"Gloves\tglov\tarmo\t\t1\t1\tglov\tglov\t\t\t\n" +
	"Helm\thelm\tarmo\t\t1\t1\thead\thead\t\t\t\n" +
	"Primal Helm\tphlm\thelm\tbarb\t1\t1\thead\thead\t\t\t\n" +
	"Charm\tchar\tmisc\t\t0\t0\t\t\t\t\t\n" +
	"Sword\tswor\tmele\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Axe\taxe\tmele\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Bow\tbow\tmiss\t\t0\t1\trarm\tlarm\tbowq\t\t\n" +
	"Crossbow\txbow\tmiss\t\t1\t1\trarm\tlarm\txboq\t\t\n" +
	"Staff\tstaf\trod\t\t1\t1\trarm\tlarm\t\t\t\n" +
	"Amazon Bow\tabow\tbow\tamaz\t0\t1\trarm\tlarm\tbowq\t\t\n" +
	"Hand to Hand\th2h\tmele\tassn\t1\t1\trarm\tlarm\t\t\t\n" +
	"Orb\torb\tweap\tsorc\t1\t1\trarm\tlarm\t\t\t\n" +
	"Auric Shields\tashd\tshld\tpala\t1\t1\trarm\tlarm\t\t\t\n" +
	"Any Armor\tarmo\t\t\t0\t0\t\t\t\t\t\n" +
	"Any Shield\tshld\tarmo\tseco\t0\t0\t\t\t\t\t\n" +
	"Melee Weapon\tmele\tweap\t\t0\t0\t\t\t\t\t\n" +
	"Missile Weapon\tmiss\tweap\t\t0\t0\t\t\t\t\t\n" +
	"Weapon\tweap\t\t\t0\t0\t\t\t\t\t\n" +
	"Staves And Rods\trod\tblun\t\t0\t0\t\t\t\t\t\n" +
	"Blunt\tblun\tmele\t\t0\t0\t\t\t\t\t\n" +
	"Missile\tmisl\tmisc\t\t0\t0\t\t\t\t\t\n" +
	"Miscellaneous\tmisc\t\t\t0\t0\t\t\t\t\t\n" +
	"Amazon Item\tamaz\tclas\t\t0\t0\t\t\t\t\tama\n" +
	"Barbarian Item\tbarb\tclas\t\t0\t0\t\t\t\t\tbar\n" +
	"Assassin Item\tassn\tclas\t\t0\t0\t\t\t\t\tass\n" +
	"Sorceress Item\tsorc\tclas\t\t0\t0\t\t\t\t\tsor\n" +
	"Paladin Item\tpala\tclas\t\t0\t0\t\t\t\t\tpal\n" +
	"Class Specific\tclas\t\t\t0\t0\t\t\t\t\t\n"

func testRules(t *testing.T) Rules {
	t.Helper()

	ty, err := ParseTypes([]byte(miniTypes))
	if err != nil {
		t.Fatal(err)
	}

	return Rules{Types: ty}
}

func armor(code, typ string) *Item { return &Item{Code: code, Type: typ, Identified: true} }

func weapon(code, typ string, two bool) *Item {
	return &Item{Code: code, Type: typ, Weapon: true, TwoHanded: two, Identified: true}
}

func TestTypesInheritance(t *testing.T) {
	r := testRules(t)

	for _, c := range []struct {
		code, anc string
		want      bool
	}{
		{"shie", "shld", true}, {"shie", "armo", true}, {"ashd", "pala", true}, {"abow", "bow", true},
		{"abow", "weap", true}, {"abow", "mele", false}, {"staf", "mele", true}, {"phlm", "armo", true},
		{"ring", "armo", false}, {"nothing", "armo", false}, {"armo", "armo", true},
	} {
		if got := r.Types.IsA(c.code, c.anc); got != c.want {
			t.Errorf("IsA(%s,%s)=%v want %v", c.code, c.anc, got, c.want)
		}
	}

	if got := r.Types.ClassOf("abow"); got != "ama" {
		t.Errorf("ClassOf(abow)=%q", got)
	}

	if got := r.Types.ClassOf("bow"); got != "" {
		t.Errorf("ClassOf(bow)=%q", got)
	}
}

func TestFitsLoc(t *testing.T) {
	r := testRules(t)

	for _, c := range []struct {
		typ  string
		loc  Loc
		want bool
	}{
		{"helm", LocHead, true}, {"helm", LocTorso, false}, {"phlm", LocHead, true},
		{"tors", LocTorso, true}, {"ring", LocRightRing, true}, {"ring", LocLeftRing, true}, {"ring", LocNeck, false},
		{"amul", LocNeck, true}, {"belt", LocBelt, true}, {"boot", LocFeet, true}, {"glov", LocGloves, true},
		{"swor", LocRightHand, true}, {"swor", LocSwapRight, true}, {"shie", LocLeftHand, true},
		{"shie", LocSwapLeft, true}, {"shie", LocHead, false}, {"char", LocBelt, false}, {"ashd", LocLeftHand, true},
		{"unknown", LocHead, false},
	} {
		if got := r.FitsLoc(c.typ, c.loc); got != c.want {
			t.Errorf("FitsLoc(%s,%v)=%v want %v", c.typ, c.loc, got, c.want)
		}
	}
}

func TestPlaceHands(t *testing.T) {
	r := testRules(t)
	sword, axe := weapon("ssd", "swor", false), weapon("axe", "axe", false)
	bow, quiver := weapon("bow", "bow", true), armor("aqv", "bowq")
	xquiver := armor("cqv", "xboq")
	shield := armor("buc", "shie")
	staff := weapon("sst", "staf", true)
	bastard := weapon("bas", "swor", true)
	bastard.OneOrTwo = true
	claw := weapon("clw", "h2h", false)

	for _, c := range []struct {
		name   string
		class  string
		body   map[Loc]*Item
		it     *Item
		loc    Loc
		ok     bool
		reason Reason
		disp   []Loc
	}{
		{"shield alone", ClassPaladin, nil, shield, LocLeftHand, true, ReasonOK, nil},
		{"sword + shield", ClassPaladin, map[Loc]*Item{LocRightHand: sword}, shield, LocLeftHand, true, ReasonOK, nil},
		{"two-hander displaces shield", ClassPaladin, map[Loc]*Item{LocLeftHand: shield}, staff, LocRightHand, true, ReasonOK, []Loc{LocLeftHand}},
		{"shield displaces two-hander", ClassPaladin, map[Loc]*Item{LocRightHand: staff}, shield, LocLeftHand, true, ReasonOK, []Loc{LocRightHand}},
		{"barbarian bastard sword one handed keeps shield", ClassBarbarian, map[Loc]*Item{LocLeftHand: shield}, bastard, LocRightHand, true, ReasonOK, nil},
		{"paladin bastard sword two handed", ClassPaladin, map[Loc]*Item{LocLeftHand: shield}, bastard, LocRightHand, true, ReasonOK, []Loc{LocLeftHand}},
		{"bow + bow quiver", ClassAmazon, map[Loc]*Item{LocRightHand: bow}, quiver, LocLeftHand, true, ReasonOK, nil},
		{"bow placed next to its quiver", ClassAmazon, map[Loc]*Item{LocLeftHand: quiver}, bow, LocRightHand, true, ReasonOK, nil},
		{"quiver without weapon", ClassAmazon, nil, quiver, LocLeftHand, false, ReasonNoAmmoWeapon, nil},
		{"bolts with a bow", ClassAmazon, map[Loc]*Item{LocRightHand: bow}, xquiver, LocLeftHand, false, ReasonNoAmmoWeapon, nil},
		{"quiver with a sword", ClassAmazon, map[Loc]*Item{LocRightHand: sword}, quiver, LocLeftHand, false, ReasonNoAmmoWeapon, nil},
		{"sorceress weapon in off hand", ClassSorceress, map[Loc]*Item{LocRightHand: sword}, axe, LocLeftHand, false, ReasonDualWield, nil},
		{"barbarian dual wield", ClassBarbarian, map[Loc]*Item{LocRightHand: sword}, axe, LocLeftHand, true, ReasonOK, nil},
		{"barbarian no two-hander off hand", ClassBarbarian, map[Loc]*Item{LocRightHand: sword}, staff, LocLeftHand, false, ReasonDualWield, nil},
		{"assassin claws", ClassAssassin, map[Loc]*Item{LocRightHand: claw}, claw, LocLeftHand, true, ReasonOK, nil},
		{"assassin sword off hand", ClassAssassin, map[Loc]*Item{LocRightHand: claw}, sword, LocLeftHand, false, ReasonDualWield, nil},
		{"swap set pairs with swap set", ClassPaladin, map[Loc]*Item{LocSwapLeft: shield, LocRightHand: staff}, staff, LocSwapRight, true, ReasonOK, []Loc{LocSwapLeft}},
		{"ring in a hand", ClassPaladin, nil, armor("rin", "ring"), LocRightHand, false, ReasonBodyLoc, nil},
		{"helm on the head", ClassPaladin, nil, armor("cap", "helm"), LocHead, true, ReasonOK, nil},
		{"class helm for another class", ClassPaladin, nil, armor("ba3", "phlm"), LocHead, false, ReasonClass, nil},
		{"class helm for its class", ClassBarbarian, nil, armor("ba3", "phlm"), LocHead, true, ReasonOK, nil},
	} {
		d := r.Place(Hero{Class: c.class, Str: 200, Dex: 200, Level: 99}, c.body, c.it, c.loc)
		if d.OK != c.ok || (!c.ok && d.Reason != c.reason) {
			t.Errorf("%s: %+v", c.name, d)
		}

		if c.ok && len(d.Displaced) != len(c.disp) {
			t.Errorf("%s: displaced %v want %v", c.name, d.Displaced, c.disp)
		}

		for i := range c.disp {
			if c.ok && i < len(d.Displaced) && d.Displaced[i] != c.disp[i] {
				t.Errorf("%s: displaced %v want %v", c.name, d.Displaced, c.disp)
			}
		}
	}
}

func TestClassRestriction(t *testing.T) {
	r := testRules(t)
	h := Hero{Class: ClassSorceress, Str: 100, Dex: 100, Level: 50}

	orb := weapon("ob1", "orb", false)
	if d := r.Place(h, nil, orb, LocRightHand); !d.OK {
		t.Errorf("sorceress with an orb: %+v", d)
	}

	h.Class = ClassPaladin
	if d := r.Place(h, nil, orb, LocRightHand); d.OK || d.Reason != ReasonClass {
		t.Errorf("paladin with an orb: %+v", d)
	}

	h.Class = ClassPaladin
	if d := r.Place(h, nil, armor("pa1", "ashd"), LocLeftHand); !d.OK {
		t.Errorf("paladin with an auric shield: %+v", d)
	}

	h.Class = ClassAmazon
	if d := r.Place(h, nil, armor("pa1", "ashd"), LocLeftHand); d.OK || d.Reason != ReasonClass {
		t.Errorf("amazon with an auric shield: %+v", d)
	}
}

func TestRequirements(t *testing.T) {
	r := testRules(t)
	plate := &Item{Code: "plt", Type: "tors", Identified: true, ReqStr: 65, ReqLevel: 30}

	for _, c := range []struct {
		name   string
		h      Hero
		mod    func(*Item)
		ok     bool
		reason Reason
	}{
		{"enough", Hero{Str: 65, Dex: 10, Level: 30}, nil, true, ReasonOK},
		{"weak", Hero{Str: 64, Dex: 10, Level: 30}, nil, false, ReasonStrength},
		{"low level", Hero{Str: 100, Dex: 10, Level: 29}, nil, false, ReasonLevel},
		{"requirements -20%", Hero{Str: 52, Dex: 10, Level: 30}, func(i *Item) { i.ReqPercent = -20 }, true, ReasonOK},
		{"ethereal -10", Hero{Str: 55, Dex: 10, Level: 30}, func(i *Item) { i.Ethereal = true }, true, ReasonOK},
		{"ethereal still short", Hero{Str: 54, Dex: 10, Level: 30}, func(i *Item) { i.Ethereal = true }, false, ReasonStrength},
		{"dex", Hero{Str: 100, Dex: 4, Level: 30}, func(i *Item) { i.ReqDex = 5 }, false, ReasonDexterity},
		{"unidentified", Hero{Str: 100, Dex: 10, Level: 30}, func(i *Item) { i.Identified = false }, false, ReasonUnidentified},
		{"requirements -100% and ethereal needs nothing", Hero{Str: 1, Dex: 1, Level: 30}, func(i *Item) { i.ReqPercent = -100; i.Ethereal = true }, true, ReasonOK},
	} {
		it := *plate
		if c.mod != nil {
			c.mod(&it)
		}

		d := r.Place(c.h, nil, &it, LocTorso)
		if d.OK != c.ok || (!c.ok && d.Reason != c.reason) {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
}

func TestResolveCascade(t *testing.T) {
	r := testRules(t)
	// the helm needs 60 strength; the gloves give +20 strength and need 40; the hero has 45:
	// the helm is off while the gloves are on... and turns on when the gloves lift the hero
	helm := &Item{Code: "hlm", Type: "helm", Identified: true, ReqStr: 60}
	gloves := &Item{Code: "gl", Type: "glov", Identified: true, ReqStr: 40}
	body := map[Loc]*Item{LocHead: helm, LocGloves: gloves}

	hero := func(active func(Loc) bool) Hero {
		h := Hero{Str: 45, Dex: 50, Level: 50}
		if active(LocGloves) {
			h.Str += 20
		}

		return h
	}

	got := Resolve(nil, body, nil, hero, r.Types)
	if !got[LocGloves] || !got[LocHead] {
		t.Errorf("start with everything on: %v", got)
	}

	// without the gloves bonus the helm fails; with the gloves removed from the previous state
	got = Resolve(ActiveState{LocHead: false, LocGloves: true}, body, nil, hero, r.Types)
	if !got[LocHead] {
		t.Errorf("the helm must be switched on when the gloves lift the strength: %v", got)
	}

	// a hero too weak for the gloves: both off
	weak := func(active func(Loc) bool) Hero { return Hero{Str: 30, Dex: 50, Level: 50} }

	got = Resolve(nil, body, nil, weak, r.Types)
	if got[LocHead] || got[LocGloves] {
		t.Errorf("weak hero: %v", got)
	}

	// a broken item is off even when it qualifies
	got = Resolve(nil, body, func(l Loc) bool { return l == LocGloves }, hero, r.Types)
	if got[LocGloves] {
		t.Errorf("broken gloves are on: %v", got)
	}

	// weapon set II items are never active
	got = Resolve(nil, map[Loc]*Item{LocSwapRight: weapon("a", "swor", false)}, nil, hero, r.Types)
	if len(got) != 0 {
		t.Errorf("swap items active: %v", got)
	}
}

func TestPickArmorPiece(t *testing.T) {
	all := func(Loc) bool { return true }

	if TotalArmorWeight(all) != 22 {
		t.Fatalf("total weight %d", TotalArmorWeight(all))
	}

	// from the start of the table, rolls 0..2 head, 3..7 torso, 8..11 right hand ...
	for roll, want := range map[int]Loc{0: LocHead, 2: LocHead, 3: LocTorso, 7: LocTorso, 8: LocRightHand, 12: LocLeftHand,
		16: LocBelt, 18: LocFeet, 20: LocGloves, 21: LocGloves} {
		if got, ok := PickArmorPiece(0, roll, all); !ok || got != want {
			t.Errorf("roll %d: %v want %v", roll, got, want)
		}
	}

	// hands without armor do not count
	noHands := func(l Loc) bool { return l != LocRightHand && l != LocLeftHand }
	if TotalArmorWeight(noHands) != 14 {
		t.Errorf("weight without hands %d", TotalArmorWeight(noHands))
	}

	if got, _ := PickArmorPiece(0, 8, noHands); got != LocBelt {
		t.Errorf("roll 8 without hands: %v", got)
	}

	// starting in the middle wraps around
	if got, _ := PickArmorPiece(6, 0, all); got != LocGloves {
		t.Errorf("start at gloves: %v", got)
	}

	if got, _ := PickArmorPiece(6, 2, all); got != LocHead {
		t.Errorf("wrap: %v", got)
	}

	if _, ok := PickArmorPiece(0, 0, func(Loc) bool { return false }); ok {
		t.Error("nothing to damage must report false")
	}
}

func TestRollLoss(t *testing.T) {
	it := &Item{MaxDurability: 20, Durability: 5}

	if d, lost := RollLoss(it, 3, ChanceWeapon); !lost || d != 4 {
		t.Errorf("roll 3 vs 4%%: %d %v", d, lost)
	}

	if d, lost := RollLoss(it, 4, ChanceWeapon); lost || d != 5 {
		t.Errorf("roll 4 vs 4%%: %d %v", d, lost)
	}

	if d, lost := RollLoss(it, 9, ChanceArmor); !lost || d != 4 {
		t.Errorf("roll 9 vs 10%%: %d %v", d, lost)
	}

	last := &Item{MaxDurability: 20, Durability: 1}
	if d, _ := RollLoss(last, 0, ChanceArmor); d != 0 || !(&Item{MaxDurability: 20}).Broken() {
		t.Errorf("last point: %d", d)
	}

	for name, x := range map[string]*Item{
		"indestructible": {MaxDurability: 20, Durability: 5, Indestructible: true},
		"nodurability":   {MaxDurability: 20, Durability: 5, NoDurability: true},
		"no max":         {Durability: 5},
		"already broken": {MaxDurability: 20, Durability: 0},
	} {
		if _, lost := RollLoss(x, 0, 100); lost {
			t.Errorf("%s lost durability", name)
		}
	}

	if !PropertiesOff(&Item{MaxDurability: 20}) || PropertiesOff(&Item{MaxDurability: 20, Durability: 1}) ||
		PropertiesOff(&Item{}) {
		t.Error("PropertiesOff")
	}
}

func TestWeaponSets(t *testing.T) {
	pairs := map[Loc]Loc{LocRightHand: LocSwapRight, LocLeftHand: LocSwapLeft, LocSwapRight: LocRightHand,
		LocSwapLeft: LocLeftHand, LocHead: LocHead}

	for in, want := range pairs {
		if got := EffectiveLoc(in, 1); got != want {
			t.Errorf("EffectiveLoc(%v,1)=%v want %v", in, got, want)
		}

		if got := EffectiveLoc(in, 0); got != in {
			t.Errorf("EffectiveLoc(%v,0)=%v", in, got)
		}
	}

	if NextSet(0) != 1 || NextSet(1) != 0 {
		t.Error("NextSet")
	}

	a, b := weapon("a", "swor", false), weapon("b", "axe", false)
	body := map[Loc]*Item{LocRightHand: a, LocSwapRight: b}

	if r, _ := HandsOf(body, 0); r != a {
		t.Error("set I hands")
	}

	if r, _ := HandsOf(body, 1); r != b {
		t.Error("set II hands")
	}
}

// TestRealTables parses the real tables when D2_TABLES points at them.
func TestRealTables(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "patch_d2", "ItemTypes.txt"))
	if err != nil {
		t.Skip(err)
	}

	ty, err := ParseTypes(data)
	if err != nil {
		t.Fatal(err)
	}

	r := Rules{Types: ty}

	for _, c := range []struct {
		typ string
		loc Loc
		ok  bool
	}{
		{"shie", LocLeftHand, true}, {"ashd", LocLeftHand, true}, {"head", LocLeftHand, true}, {"orb", LocRightHand, true},
		{"circ", LocHead, true}, {"pelt", LocHead, true}, {"phlm", LocHead, true}, {"cloa", LocTorso, true},
		{"abow", LocRightHand, true}, {"ajav", LocLeftHand, true}, {"h2h2", LocRightHand, true}, {"bowq", LocLeftHand, true},
		{"mboq", LocLeftHand, true}, {"scha", LocBelt, false}, {"ring", LocLeftRing, true}, {"amul", LocNeck, true},
		{"tpot", LocRightHand, true}, {"jewl", LocHead, false},
	} {
		if got := r.FitsLoc(c.typ, c.loc); got != c.ok {
			t.Errorf("real FitsLoc(%s,%v)=%v want %v", c.typ, c.loc, got, c.ok)
		}
	}

	if got := ty.ClassOf("abow"); got != "ama" {
		t.Errorf("ClassOf(abow)=%q", got)
	}

	if got := ty.ClassOf("orb"); got != "sor" {
		t.Errorf("ClassOf(orb)=%q", got)
	}

	weapons, err := os.ReadFile(filepath.Join(dir, "patch_d2", "weapons.txt"))
	if err != nil {
		t.Skip(err)
	}

	armorTxt, _ := os.ReadFile(filepath.Join(dir, "patch_d2", "armor.txt"))

	bases, err := ParseBases(armorTxt, weapons)
	if err != nil {
		t.Fatal(err)
	}

	if b := bases["bsw"]; !b.TwoHanded || !b.OneOrTwo || b.ReqStr == 0 {
		t.Errorf("bastard sword %+v", b)
	}

	if b := bases["plt"]; b.ReqStr != 65 || b.Durability == 0 {
		t.Errorf("plate mail %+v", b)
	}
}
