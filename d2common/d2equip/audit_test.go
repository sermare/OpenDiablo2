package d2equip

import (
	"os"
	"path/filepath"
	"testing"
)

// Oracle audit of the item requirement, hand, durability and socket rules
// against the real tables (D2_TABLES, e.g. ~/git/d2-tables). Skipped when unset.

type realData struct {
	rules Rules
	bases Bases
	gems  Gems
}

func loadReal(t *testing.T) realData {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(parts ...string) []byte {
		b, err := os.ReadFile(filepath.Join(append([]string{dir}, parts...)...))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	ty, err := ParseTypes(read("patch_d2", "ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	bases, err := ParseBases(read("patch_d2", "armor.txt"), read("patch_d2", "weapons.txt"), read("patch_d2", "misc.txt"))
	if err != nil {
		t.Fatal(err)
	}

	gems, err := ParseGems(read("itemgen", "patch_d2", "Gems.txt"))
	if err != nil {
		t.Fatal(err)
	}

	return realData{Rules{Types: ty}, bases, gems}
}

func (d realData) item(code string) *Item {
	b := d.bases[code]

	return &Item{
		Code: code, Type: b.Type, Weapon: b.Weapon, TwoHanded: b.TwoHanded, OneOrTwo: b.OneOrTwo, Identified: true,
		ReqStr: b.ReqStr, ReqDex: b.ReqDex, ReqLevel: b.ReqLevel, MaxDurability: b.Durability, Durability: b.Durability,
		NoDurability: b.NoDurability,
	}
}

func TestAuditRequirementsReal(t *testing.T) {
	d := loadReal(t)

	for _, c := range []struct {
		name    string
		code    string
		class   string
		str     int
		dex     int
		level   int
		percent int
		eth     bool
		unid    bool
		want    Reason
	}{
		{"plate ok", "plt", "pal", 65, 1, 1, 0, false, false, ReasonOK},
		{"plate 1 short", "plt", "pal", 64, 50, 99, 0, false, false, ReasonStrength},
		{"plate -20% (52)", "plt", "pal", 52, 50, 99, -20, false, false, ReasonOK},
		{"plate -20% 51 short", "plt", "pal", 51, 50, 99, -20, false, false, ReasonStrength},
		{"plate eth (55)", "plt", "pal", 55, 50, 99, 0, true, false, ReasonOK},
		{"plate eth 54 short", "plt", "pal", 54, 50, 99, 0, true, false, ReasonStrength},
		{"plate eth -20% (42)", "plt", "pal", 42, 50, 99, -20, true, false, ReasonOK},
		{"plate +10% (71)", "plt", "pal", 70, 50, 99, 10, false, false, ReasonStrength},
		{"unidentified first", "plt", "pal", 1, 1, 1, 0, false, true, ReasonUnidentified},
		{"orb for amazon", "ob1", "ama", 99, 99, 99, 0, false, false, ReasonClass},
		{"orb for sorc", "ob1", "sor", 99, 99, 99, 0, false, false, ReasonOK},
		{"amazon bow for barb", "am1", "bar", 99, 99, 99, 0, false, false, ReasonClass},
		{"claw for sin", "ktr", "ass", 99, 99, 99, 0, false, false, ReasonOK},
		{"claw for pal", "ktr", "pal", 99, 99, 99, 0, false, false, ReasonClass},
		{"plain sword any class", "ssd", "sor", 99, 99, 99, 0, false, false, ReasonOK},
		{"zero strength hero", "buc", "pal", 0, 99, 99, 0, false, false, ReasonStrength},
	} {
		it := d.item(c.code)
		if _, ok := d.bases[c.code]; !ok {
			t.Errorf("%s: base %q missing from the tables", c.name, c.code)
			continue
		}

		it.ReqPercent, it.Ethereal, it.Identified = c.percent, c.eth, !c.unid
		if c.name == "orb for sorc" || c.name == "orb for amazon" {
			// level 1 only: levelreq of the base must not be the failing reason here
			it.ReqLevel = 0
		}

		got := Check(Hero{Class: c.class, Str: c.str, Dex: c.dex, Level: c.level}, it, d.rules.Types)
		if got.Reason() != c.want {
			t.Errorf("%s: reason %q (%s) want %q", c.name, got.Reason(), got.Detail(), c.want)
		}
	}
}

func TestAuditRequiredStatTable(t *testing.T) {
	for _, c := range []struct {
		base, pct int
		eth       bool
		want      int
	}{
		{65, 0, false, 65}, {65, -20, false, 52}, {65, -15, false, 56}, {65, -30, false, 46}, {65, 0, true, 55},
		{65, -20, true, 42}, {50, 25, false, 62}, {0, 0, false, 0}, {0, 0, true, -10}, {100, -100, false, 0},
	} {
		if got := RequiredStat(c.base, c.pct, c.eth); got != c.want {
			t.Errorf("RequiredStat(%d,%d,%v)=%d want %d", c.base, c.pct, c.eth, got, c.want)
		}
	}
}

func TestAuditRequiredLevel(t *testing.T) {
	for _, c := range []struct {
		base int
		aff  []int
		want int
	}{
		{0, nil, 0}, {20, nil, 20}, {20, []int{12, 31}, 31}, {45, []int{12, 31}, 45}, {1, []int{5, 70, 33}, 70},
	} {
		if got := RequiredLevel(c.base, c.aff...); got != c.want {
			t.Errorf("RequiredLevel(%d,%v)=%d want %d", c.base, c.aff, got, c.want)
		}
	}
}

func TestAuditHandsReal(t *testing.T) {
	d := loadReal(t)

	two := func(code string) *Item { return d.item(code) }

	// bastard sword is 1or2handed, a great sword is two-handed only for others, a polearm never one-handed
	if b := d.bases["bsw"]; !b.TwoHanded || !b.OneOrTwo {
		t.Fatalf("bsw flags %+v", b)
	}

	for _, c := range []struct {
		name        string
		class       string
		right, left string
		ok          bool // the pair can be held together
	}{
		{"barb 1h + 1h", "bar", "hax", "hax", true},
		{"barb 2h sword + 1h", "bar", "bsw", "hax", true},
		{"barb dual 2h swords", "bar", "bsw", "bsw", true},
		{"barb polearm + shield", "bar", "pax", "buc", false},
		{"barb 1h + shield", "bar", "hax", "buc", true},
		{"barb polearm + 1h in off hand", "bar", "pax", "hax", false},
		{"pal 1h + 1h", "pal", "hax", "hax", false},
		{"pal sword + shield", "pal", "ssd", "buc", true},
		{"pal 2h sword + shield", "pal", "bsw", "buc", false},
		{"sin claw + claw", "ass", "ktr", "ktr", true},
		{"sin claw + sword", "ass", "ktr", "ssd", false},
		{"sin claw + shield", "ass", "ktr", "buc", true},
		{"ama bow + arrows", "ama", "sbw", "aqv", true},
		{"ama bow + bolts", "ama", "sbw", "cqv", false},
		{"ama xbow + bolts", "ama", "lxb", "cqv", true},
		{"arrows without a bow", "ama", "hax", "aqv", false},
	} {
		right, left := two(c.right), two(c.left)
		if _, ok := d.bases[c.left]; !ok {
			t.Errorf("%s: base %q missing", c.name, c.left)
			continue
		}

		// the Place verdict of the left hand item given the right hand one
		dec := d.rules.Place(Hero{Class: c.class, Str: 200, Dex: 200, Level: 99}, map[Loc]*Item{LocRightHand: right}, left, LocLeftHand)
		got := dec.OK && len(dec.Displaced) == 0

		if got != c.ok {
			t.Errorf("%s: held together %v (%+v) want %v", c.name, got, dec, c.ok)
		}
	}
}

func TestAuditBodyLocsReal(t *testing.T) {
	d := loadReal(t)

	for _, c := range []struct {
		code string
		loc  Loc
		ok   bool
	}{
		{"cap", LocHead, true}, {"cap", LocTorso, false}, {"qui", LocTorso, true}, {"lbl", LocBelt, true},
		{"lgl", LocGloves, true}, {"lbt", LocFeet, true}, {"buc", LocLeftHand, true}, {"buc", LocRightHand, true},
		{"buc", LocSwapLeft, true}, {"buc", LocHead, false}, {"hax", LocRightHand, true}, {"hax", LocLeftHand, true},
		{"hax", LocSwapRight, true}, {"hax", LocTorso, false}, {"rin", LocRightRing, true}, {"rin", LocLeftRing, true},
		{"rin", LocNeck, false}, {"amu", LocNeck, true}, {"amu", LocRightRing, false}, {"cm1", LocBelt, false},
		{"aqv", LocLeftHand, true}, {"aqv", LocRightHand, true},
	} {
		b, ok := d.bases[c.code]
		if !ok {
			t.Errorf("base %q missing", c.code)
			continue
		}

		if got := d.rules.FitsLoc(b.Type, c.loc); got != c.ok {
			t.Errorf("FitsLoc(%s/%s,%v)=%v want %v", c.code, b.Type, c.loc, got, c.ok)
		}
	}

	// every base item whose type can be worn resolves to at least one location
	for code, b := range d.bases {
		ty := d.rules.Types.Get(b.Type)
		if ty != nil && ty.Body && b.Type != "torc" && len(d.rules.Types.Locs(b.Type)) == 0 {
			t.Errorf("worn type %s of %s has no location", b.Type, code)
		}
	}
}

func TestAuditDurabilityReal(t *testing.T) {
	d := loadReal(t)

	// the chances are the only pinned numbers: armor 10%, weapon 4% (see durability.go, exe 0x557d90)
	if ChanceArmor != 10 || ChanceWeapon != 4 {
		t.Errorf("chances %d/%d", ChanceArmor, ChanceWeapon)
	}

	for _, c := range []struct {
		name   string
		code   string
		indest bool
		cur    int
		roll   int
		want   int
		lost   bool
	}{
		{"armor hit", "plt", false, 10, 9, 9, true},
		{"armor safe roll", "plt", false, 10, 10, 10, false},
		{"indestructible", "plt", true, 10, 0, 10, false},
		{"already broken", "plt", false, 0, 0, 0, false},
		{"weapon roll 3", "hax", false, 5, 3, 4, true},
		{"weapon roll 4", "hax", false, 5, 4, 5, false},
	} {
		it := d.item(c.code)
		it.Indestructible, it.Durability = c.indest, c.cur

		chance := ChanceArmor
		if it.Weapon {
			chance = ChanceWeapon
		}

		got, lost := RollLoss(it, c.roll, chance)
		if got != c.want || lost != c.lost {
			t.Errorf("%s: dur %d lost %v want %d %v", c.name, got, lost, c.want, c.lost)
		}
	}

	// bases without durability never lose it; every other base has a positive one
	for code, b := range d.bases {
		it := d.item(code)
		if b.NoDurability && it.CanLoseDurability() {
			t.Errorf("%s: nodurability item can lose durability", code)
		}
	}

	if it := d.item("plt"); !it.CanLoseDurability() {
		t.Error("plate mail must be able to lose durability")
	}

	// an ethereal item loses durability like any other (only repair is refused)
	eth := d.item("plt")
	eth.Ethereal = true

	if !eth.CanLoseDurability() {
		t.Error("ethereal items lose durability")
	}
}

func TestAuditSocketsReal(t *testing.T) {
	d := loadReal(t)

	for _, c := range []struct {
		code string
		ilvl int
		diff int
		want int
	}{
		{"plt", 10, 2, 2}, {"hax", 85, 2, 2}, {"hax", 10, 0, 2}, {"dgr", 85, 2, 1}, {"buc", 90, 2, 1},
		{"cap", 10, 2, 2}, {"gsd", 85, 0, 3}, {"gsd", 85, 1, 4}, {"gsd", 85, 2, 6}, {"gsd", 20, 2, 3}, {"gsd", 30, 2, 4},
	} {
		b, ok := d.bases[c.code]
		if !ok {
			t.Errorf("base %q missing", c.code)
			continue
		}

		if got := d.rules.Types.MaxSockets(b, c.ilvl, c.diff); got != c.want {
			t.Errorf("MaxSockets(%s ilvl %d diff %d)=%d want %d (gemsockets %d)", c.code, c.ilvl, c.diff, got, c.want, b.GemSockets)
		}
	}

	for code, b := range d.bases {
		for ilvl := 1; ilvl <= 99; ilvl += 7 {
			for diff := 0; diff < 3; diff++ {
				n := d.rules.Types.MaxSockets(b, ilvl, diff)
				if n < 0 || n > b.GemSockets || n > SocketCapByDifficulty(diff) {
					t.Errorf("%s ilvl %d diff %d: %d sockets (base %d)", code, ilvl, diff, n, b.GemSockets)
				}
			}
		}
	}

	for code, ty := range d.rules.Types.byCode {
		if ty.MaxSock1 > ty.MaxSock25 || ty.MaxSock25 > ty.MaxSock40 {
			t.Errorf("type %s: MaxSock columns not monotonic %d %d %d", code, ty.MaxSock1, ty.MaxSock25, ty.MaxSock40)
		}
	}
}

func TestAuditSocketCount(t *testing.T) {
	for _, c := range []struct {
		max     int
		seed    uint32
		classic bool
		helm    bool
		want    int
	}{
		{0, 5, false, false, 0}, {6, 0, false, false, 1}, {6, 5, false, false, 6}, {6, 6, false, false, 1}, {3, 7, false, false, 2},
		{6, 9, true, false, 3}, {6, 9, true, true, 2}, {2, 9, true, true, 2}, {1, 4, true, false, 1},
	} {
		if got := SocketCount(c.max, c.seed, c.classic, c.helm); got != c.want {
			t.Errorf("SocketCount(%d,%d,%v,%v)=%d want %d", c.max, c.seed, c.classic, c.helm, got, c.want)
		}
	}

	for diff, want := range []int{3, 4, 6, 6} {
		if got := SocketCapByDifficulty(diff); got != want {
			t.Errorf("cap(%d)=%d want %d", diff, got, want)
		}
	}
}

func TestAuditGemSlotsReal(t *testing.T) {
	d := loadReal(t)

	// the slot of every base item agrees with its gemapplytype column
	for code, b := range d.bases {
		if b.GemSockets == 0 || (!b.Weapon && d.rules.Types.IsA(b.Type, "misc")) {
			continue
		}

		slot, ok := d.rules.Types.SlotOf(b.Type)
		if !ok || int(slot) != b.GemApplyType {
			t.Errorf("%s (%s): slot %v ok=%v, gemapplytype %d", code, b.Type, slot, ok, b.GemApplyType)
		}
	}

	for _, c := range []struct {
		gem  string
		slot SocketSlot
		code string // first mod
		min  int
	}{
		{"gpr", SlotWeapon, "fire-min", 15}, {"gpr", SlotArmor, "hp", 38}, {"gpr", SlotShield, "res-fire", 40},
		{"r09", SlotWeapon, "dmg-ltng", 1}, {"r09", SlotArmor, "res-ltng", 30}, {"r33", SlotWeapon, "indestruct", 1},
		{"r33", SlotShield, "indestruct", 1}, {"skc", SlotWeapon, "manasteal", 1}, {"skc", SlotArmor, "regen", 2},
	} {
		g, ok := d.gems[c.gem]
		if !ok {
			t.Errorf("gem %s missing", c.gem)
			continue
		}

		mods := g.ModsFor(c.slot)
		if len(mods) == 0 || mods[0].Code != c.code || mods[0].Min != c.min {
			t.Errorf("gem %s slot %d mods %+v want first %s min %d", c.gem, c.slot, mods, c.code, c.min)
		}
	}

	// every gem gives something in each of the three slots
	for code, g := range d.gems {
		for s := SlotWeapon; s <= SlotShield; s++ {
			if len(g.ModsFor(s)) == 0 {
				t.Errorf("gem %s has no mods for slot %d", code, s)
			}
		}
	}
}
