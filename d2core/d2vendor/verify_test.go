package d2vendor

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Pinning tests for the facts confirmed in Game.exe 1.14b (see
// d2-re-notes/verify-vendor.md). Tests that need real tables use D2_TABLES.

// 0x5746a0: the cap table is indexed by the vendor's act, min(level+5, cap).
func TestVerifiedItemLevelCapPerAct(t *testing.T) {
	want := [5]int{12, 20, 28, 36, 45}
	for act := range want {
		if got := ItemLevel(99, act); got != want[act] {
			t.Errorf("act %d cap %d want %d", act+1, got, want[act])
		}
	}

	for _, tc := range []struct {
		v    string
		want int
	}{{"Akara", 0}, {"Fara", 1}, {"Ormus", 2}, {"Halbu", 3}, {"Larzuk", 4}} {
		v, _ := ByName(tc.v)
		if got := ActIndex(v); got != tc.want {
			t.Errorf("ActIndex(%s)=%d want %d", tc.v, got, tc.want)
		}
	}

	// Only normal difficulty is capped.
	bases := []Base{{Code: "cap", W: 1, H: 1, Vendor: Params{Max: 0}}}
	for diff, want := range []int{12, 104, 104} {
		s := GenerateSeeded(1, append(bases, Base{Code: "x", W: 1, H: 1, Permanent: true, Vendor: Params{Max: 1}}),
			Options{PlayerLevel: 99, Tier: 0, Difficulty: diff})
		if got := s.Items[0].ILvl; got != want {
			t.Errorf("difficulty %d: ilvl %d want %d", diff, got, want)
		}
	}
}

// 0x574780: counts are Min + random(Max+1-Min); the magic maximum gains 1
// below item level 25 and 2..3 from there, drawn before the count.
func TestVerifiedCountRolls(t *testing.T) {
	b := []Base{{Code: "cap", ReqLevel: 1, W: 1, H: 1, Gear: true, CanBeMagic: true,
		Vendor: Params{Min: 2, Max: 5, MagicMin: 1, MagicMax: 3, MagicLevel: 1}}}

	r := &seqRNG{vals: []uint32{0}}
	Generate(r, b, Options{PlayerLevel: 1, Tier: -1}) // ilvl 6

	// regular: Roll(5+1-2 = 4) then per item a quality roll (2 items),
	// magic: Roll(3+1-1 = 3).
	want := []int32{4, 100, 100, 3}
	if len(r.mods) < len(want) {
		t.Fatalf("mods %v", r.mods)
	}

	for i, m := range want {
		if r.mods[i] != m {
			t.Fatalf("roll %d modulus %d want %d (all %v)", i, r.mods[i], m, r.mods)
		}
	}

	// ilvl >= 25: no regular pass; Roll(2) for the extra (2 or 3), Roll(3+extra-1).
	r = &seqRNG{vals: []uint32{1}}
	s := Generate(r, b, Options{PlayerLevel: 40, Tier: -1})

	if len(r.mods) < 2 || r.mods[0] != 2 || r.mods[1] != 3+3-1 {
		t.Errorf("mods %v want [2 5 ...]", r.mods)
	}

	// extra = 1+1 + 1 = 3, count = 1 + 1%5 = 2
	if len(s.Items) != 2 || s.Items[0].Quality != d2drop.QualityMagic {
		t.Errorf("items %d", len(s.Items))
	}
}

func TestVerifiedVersionGate(t *testing.T) {
	b := []Base{{Code: "xpc", W: 1, H: 1, Version: 100, Vendor: Params{Min: 1, Max: 1}}}

	if s := GenerateSeeded(1, b, Options{PlayerLevel: 1, Tier: -1, Classic: true}); len(s.Items) != 0 {
		t.Errorf("classic game stocked an expansion item")
	}

	if s := GenerateSeeded(1, b, Options{PlayerLevel: 1, Tier: -1}); len(s.Items) != 1 {
		t.Errorf("expansion game items %d", len(s.Items))
	}
}

// 0x574110 / 0x574cf0: potions climb to hp4 / hp5 on non-normal difficulty
// from player level 26; the vendor then counts them as sold.
func TestVerifiedPotionTiers(t *testing.T) {
	hp1 := Base{Code: "hp1", W: 1, H: 1, Permanent: true, NightmareUpgrade: "hp4", HellUpgrade: "hp5", Vendor: Params{Min: 1, Max: 1}}
	resolve := func(c string) (Base, bool) { return Base{Code: c, W: 1, H: 1}, true }

	for _, tc := range []struct {
		diff, level int
		want        string
	}{{0, 40, "hp1"}, {1, 25, "hp1"}, {1, 26, "hp4"}, {2, 26, "hp5"}, {2, 80, "hp5"}} {
		s := GenerateSeeded(7, []Base{hp1}, Options{PlayerLevel: tc.level, Tier: -1, Difficulty: tc.diff, Resolve: resolve})
		if len(s.Items) != 1 || s.Items[0].Code != tc.want {
			t.Errorf("difficulty %d level %d: %v want %s", tc.diff, tc.level, s.Items, tc.want)
		}
	}

	for _, c := range []string{"hp4", "hp5", "mp4", "mp5"} {
		if VendorSells(0, c, nil) || !VendorSells(1, c, nil) || !VendorSells(2, c, nil) {
			t.Errorf("VendorSells rule for %s", c)
		}
	}

	if !VendorSells(0, "tsc", []string{"isc", "tsc"}) || VendorSells(1, "hp1", []string{"tsc"}) {
		t.Error("permanent list rule")
	}
}

func TestVerifiedExceptionalUpgradeOdds(t *testing.T) {
	b := &Base{Code: "cap", Uber: "xap", Ultra: "uap"}
	// nightmare: roll < ilvl*64+4000 -> exceptional
	r := &seqRNG{vals: []uint32{4000 + 10*64 - 1}}
	if got := upgradeCode(r, b, Options{PlayerLevel: 30, Difficulty: 1}, 10); got != "xap" || r.mods[0] != 100000 {
		t.Errorf("got %s mods %v", got, r.mods)
	}

	r = &seqRNG{vals: []uint32{4000 + 10*64}}
	if got := upgradeCode(r, b, Options{PlayerLevel: 30, Difficulty: 1}, 10); got != "cap" {
		t.Errorf("got %s", got)
	}

	// hell: elite below ilvl*16+1000 (only with the gate flag), else exceptional below ilvl*128+5000
	r = &seqRNG{vals: []uint32{999}}
	if got := upgradeCode(r, b, Options{PlayerLevel: 30, Difficulty: 2, EliteUpgrade: true}, 0); got != "uap" {
		t.Errorf("got %s", got)
	}

	r = &seqRNG{vals: []uint32{999}}
	if got := upgradeCode(r, b, Options{PlayerLevel: 30, Difficulty: 2}, 0); got != "xap" {
		t.Errorf("got %s", got)
	}
}

// 0x534c20: the flag is set strictly after 240 s.
func TestVerifiedRestockInterval(t *testing.T) {
	if RestockDue(240000) || !RestockDue(240001) {
		t.Error("restock boundary")
	}
}

// 0x576290: Cain charges 100 gold per unidentified item, free with quest flag 4.
func TestVerifiedIdentifyPrice(t *testing.T) {
	if IdentifyPrice(3, false) != 300 || IdentifyPrice(3, true) != 0 || IdentifyPrice(0, false) != 0 {
		t.Error("identify price")
	}

	if !IsCain(244) || !IsCain(265) || !IsCain(520) || IsCain(148) {
		t.Error("Cain classes")
	}
}

// Real data: Min columns are populated, the potion upgrade chain exists, the
// may-be-magic flag covers every vendor magic column, and the quest columns
// land on the right side (0x62f100 / 0x658220).
func TestVerifiedOnRealTables(t *testing.T) {
	rec := loadRecords(t)

	for _, p := range [][3]string{{"hp1", "hp4", "hp5"}, {"hp3", "hp4", "hp5"}, {"mp2", "mp4", "mp5"}, {"hp4", "xxx", "hp5"}} {
		icr := rec.Item.All[p[0]]
		if icr == nil || icr.NightmareUpgrade != p[1] || icr.HellUpgrade != p[2] {
			t.Errorf("%s upgrade columns %+v", p[0], icr)
		}
	}

	minSeen := 0

	for _, v := range All() {
		for _, b := range BasesFor(rec, v) {
			if b.Vendor.Min > 0 {
				minSeen++
			}

			if b.Vendor.MagicMax > 0 && b.Vendor.MagicLevel < 255 && !b.CanBeMagic { // MagicLvl 255 never rolls
				t.Errorf("%s: %s has magic columns but the adapter says it cannot be magic", v.Name, b.Code)
			}
		}
	}

	if minSeen == 0 {
		t.Error("no Min column set in the real tables")
	}

	// Quest group A of Gheed: questbuymult 1024 (vendor pays side), questsellmult 922 (player pays side).
	gheed, _ := ByName("Gheed")
	q := &d2s.QuestRecord{}
	n := NPCPricing(rec, gheed, q)

	if n.Quest[0].Buy != 922 || n.Quest[0].Sell != 1024 {
		t.Errorf("quest group A buy/sell %d/%d want 922/1024", n.Quest[0].Buy, n.Quest[0].Sell)
	}
}
