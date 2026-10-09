package d2records

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Oracle for "skill -> stats on a unit" (passive stats, aura stats, buffs and
// how they merge with item stats). Needs D2_TABLES (skills.txt, States.txt,
// ItemStatCost.txt of patch_d2). The expected numbers are written out from the
// skills.txt columns (Param1..8 and the calc strings), independent of the
// d2calc evaluator, using the skillcalc.txt formulas:
//
//	lnAB(l) = A + (l-1)*B        dmAB(l) = (110*l*(B-A))/(100*(l+6)) + A
//
// Verified (binary notes skills-2.md 4.x): the aura stat list of Frozen Armor
// = aurastat1..6 with aurastatcalc1..6, duration = auralencalc frames;
// SKILL_ApplyPassiveStatsToStatList applies passivestat1..5 with
// passivecalc1..5. UNVERIFIED rules are called out in the table comments.
//
// Exe findings (1.14b, see verify-passive-stats.md in d2-re-notes), all VERIFIED:
//   - 0x5c4d70 SKILL_ApplyPassiveStatsToStatList does NOT test the passive flag:
//     it applies passivestat1..5 (rec +0x98, calcs +0xa4) with nonzero calc
//     values. Its only callers are the buff cast 0x5c7540 (srvdofunc 18: Frozen
//     Armor, Fade, Burst of Speed...), Blaze 0x5c7d10 and Whirlwind 0x5d7a00, so
//     Fade's "fade" and Resist Fire's maxfireresist land on the buff's statlist
//     at cast time. The path that applies true passives (Iron Skin, Natural
//     Resistance) on load or on a skill-level change is NOT located.
//   - 0x5c4c60 SKILL_ApplyAuraStatsToStatList: aurastat1..6 (rec +0x54, calcs
//     +0x68) in column order; each nonzero calc is stored with
//     STATS_SetBaseStatValue (a repeated stat in the same list is overwritten,
//     not summed); stat 0x44 also sets 0x45. No ItemStatCost op at that point.
//   - 0x56c740 SKILL_CreateTimedStateStatList never reads States.txt group. The
//     group exclusion is FUN_0056a480 (flag 1), called by the buff cast 0x5c7540
//     before the statlist is built: it ends every state of the same nonzero
//     group, the state itself included. Hence Frozen/Shiver/Chilling/Bone Armor
//     (and justhit) replace each other, as do Burst of Speed and Fade.
//   - 0x006256e0 / 0x00625760 stat getters return the stored aggregate of the
//     unit's lists; ops 1, 11, 13 are applied when the lists change
//     (0x626430, MulDiv(total of the op stat, value, 100) added to the derived
//     stat), not at read time. Percent stats of different sources are summed.
//   - 0x006225a0 GetDefense: (stat 31 + dex/4) base; percent = stat 171 + stat 16
//     (+ the Holy Shield calc1 when state 0x65 and a hand item qualify); base +
//     base*pct/100 (minus for base <= 0); then a stat 182 term on the total.

func ln(a, b, l int) int { return a + (l-1)*b }
// dm is the exe's diminishing-returns routine 0x646ed0 (emulator-verified by
// the skill calc golden): the integer division 110*l/(l+6) comes FIRST, then
// a + t*(b-a)/100, capped at b. The older txt formula differs by 1 at times.
func dm(a, b, l int) int {
	r := (110*l/(l+6))*(b-a)/100 + a
	if r > b {
		r = b
	}

	return r
}

type statExp struct {
	stat string
	f    func(l int) int
}

type skillCase struct {
	skill string
	// syn are the base levels of other skills (synergies read blvl).
	syn map[string]int

	aura    []statExp                         // aurastat1..6, in column order
	passive []statExp                         // passivestat1..5 evaluated by the calc (see passiveApplied)
	length  func(l int, s map[string]int) int // auralencalc, nil when the column is empty
	rng     func(l int) int                   // aurarangecalc, nil when empty
	// passiveApplied: does Pipeline.PassiveStats return the passivestat
	// columns? It does only for rows with the passive flag (Iron Skin,
	// Natural Resistance). Rows such as Resist Fire, Holy Fire, Holy Shock
	// and Fade carry passivestat columns without the flag; the game applies
	// them regardless of the flag, but only through the buff cast (0x5c4d70
	// has no flag test; see the header).
	passiveApplied bool
}

func same(l func(int) int, names ...string) []statExp {
	var out []statExp
	for _, n := range names {
		out = append(out, statExp{n, l})
	}

	return out
}

var skillCases = []skillCase{
	{skill: "Iron Skin", passiveApplied: true,
		passive: []statExp{{"skill_armor_percent", func(l int) int { return ln(30, 10, l) }}}},
	{skill: "Natural Resistance", passiveApplied: true,
		passive: same(func(l int) int { return dm(0, 80, l) }, "fireresist", "lightresist", "coldresist", "poisonresist")},
	{skill: "Resist Fire",
		aura: []statExp{{"fireresist", func(l int) int { return dm(35, 150, l) }}, {"maxfireresist", func(l int) int { return l }}},
		// passivestat1 maxfireresist = blvl/2 (not applied by the engine, U)
		passive: []statExp{{"maxfireresist", func(l int) int { return l / 2 }}},
		rng:     func(l int) int { return ln(16, 2, l) }},
	{skill: "Might", aura: []statExp{{"damagepercent", func(l int) int { return ln(40, 10, l) }}},
		rng: func(l int) int { return ln(16, 2, l) }},
	{skill: "Holy Fire", rng: func(l int) int { return ln(6, 1, l) },
		// passivecalc1/2 = enms/exms * Param5(6) / 256; evaluated in TestPassiveElemCalcs
		passive: []statExp{{"firemindam", nil}, {"firemaxdam", nil}}},
	{skill: "Holy Shock", rng: func(l int) int { return ln(6, 1, l) },
		passive: []statExp{{"lightmindam", func(int) int { return 1 }}, {"lightmaxdam", nil}}},
	{skill: "Frozen Armor", syn: map[string]int{"Shiver Armor": 2, "Chilling Armor": 4},
		aura: []statExp{{"skill_armor_percent", func(l int) int { return ln(30, 5, l) }}},
		length: func(l int, s map[string]int) int {
			return ln(3000, 300, l) + (s["Shiver Armor"]+s["Chilling Armor"])*250
		}},
	{skill: "Shiver Armor", syn: map[string]int{"Frozen Armor": 3, "Chilling Armor": 1},
		aura: []statExp{{"skill_armor_percent", func(l int) int { return ln(45, 6, l) }}},
		length: func(l int, s map[string]int) int {
			return ln(3000, 300, l) + (s["Frozen Armor"]+s["Chilling Armor"])*250
		}},
	{skill: "Battle Orders", syn: map[string]int{"Shout": 5, "Battle Command": 2},
		aura: same(func(l int) int { return ln(35, 3, l) }, "item_maxmana_percent", "item_maxhp_percent", "skill_staminapercent"),
		// ln12 + (Shout+Battle Command)*par8(125): frames
		length: func(l int, s map[string]int) int { return ln(750, 250, l) + (s["Shout"]+s["Battle Command"])*125 }},
	{skill: "Shout", syn: map[string]int{"Battle Orders": 5, "Battle Command": 2},
		aura: []statExp{{"skill_armor_percent", func(l int) int { return ln(100, 10, l) }}},
		length: func(l int, s map[string]int) int {
			return ln(500, 250, l) + (s["Battle Orders"]+s["Battle Command"])*125
		}},
	// "Burst of Speed" is the skills.txt row named Quickness (id 258)
	{skill: "Quickness",
		aura:   []statExp{{"velocitypercent", func(l int) int { return dm(15, 70, l) }}, {"attackrate", func(l int) int { return dm(15, 60, l) }}},
		length: func(l int, _ map[string]int) int { return ln(3000, 300, l) }},
	{skill: "Fade",
		aura: append(same(func(l int) int { return dm(10, 75, l) }, "fireresist", "coldresist", "lightresist", "poisonresist"),
			statExp{"curse_resistance", func(l int) int { return dm(40, 90, l) }},
			statExp{"damageresist", func(l int) int { return ln(1, 1, l) }}),
		length: func(l int, _ map[string]int) int { return ln(3000, 300, l) },
		// passivestat1 fade = 2 (stat "fade"); not applied (flag-less), U
		passive: []statExp{{"fade", func(int) int { return 2 }}}},
}

func loadItemStatCost(t *testing.T) *d2statlist.Defs {
	t.Helper()

	buf, err := os.ReadFile(filepath.Join(os.Getenv("D2_TABLES"), "patch_d2", "ItemStatCost.txt"))
	if err != nil {
		t.Skip(err)
	}

	defs, err := d2statlist.ParseItemStatCost(buf)
	if err != nil {
		t.Fatal(err)
	}

	return defs
}

func caseUnit(reg *d2skill.Registry, c skillCase, l int) calcUnit {
	u := calcUnit{reg: reg, levels: map[string]int{c.skill: l}}
	for k, v := range c.syn {
		u.levels[k] = v
	}

	return u
}

func TestSkillStatColumns(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()
	defs := loadItemStatCost(t)
	pipe := &d2skill.Pipeline{Skills: reg}

	for _, c := range skillCases {
		sk := reg.ByName(c.skill)
		if sk == nil {
			t.Errorf("%s: not in skills.txt", c.skill)
			continue
		}

		// the stat NAMES of the columns, in order, and their ItemStatCost ids
		var gotAura []string

		for i := 1; i <= 6; i++ {
			if sk.AuraStat[i] != "" {
				gotAura = append(gotAura, sk.AuraStat[i])
			}
		}

		var wantAura []string
		for _, e := range c.aura {
			wantAura = append(wantAura, e.stat)
		}

		if !equalStrings(gotAura, wantAura) {
			t.Errorf("%s aurastat names %v, want %v", c.skill, gotAura, wantAura)
		}

		for _, e := range append(append([]statExp{}, c.aura...), c.passive...) {
			if defs.ID(e.stat) < 0 {
				t.Errorf("%s: stat %q is not in ItemStatCost", c.skill, e.stat)
			}
		}

		for _, l := range []int{1, 5, 12, 20} {
			u := caseUnit(reg, c, l)
			env := d2skill.NewEnv(sk, l, u, reg)

			for i, e := range c.aura {
				if got, want := env.Eval(sk.AuraStatCalc[i+1]), e.f(l); got != want {
					t.Errorf("%s L%d aurastat%d %s = %d, want %d", c.skill, l, i+1, e.stat, got, want)
				}
			}

			if c.length != nil {
				if got, want := env.Eval(sk.AuraLenCalc), c.length(l, c.syn); got != want {
					t.Errorf("%s L%d auralencalc = %d, want %d", c.skill, l, got, want)
				}
			}

			if c.rng != nil {
				if got, want := env.Eval(sk.AuraRangeCalc), c.rng(l); got != want {
					t.Errorf("%s L%d aurarangecalc = %d, want %d", c.skill, l, got, want)
				}
			}

			for i, e := range c.passive {
				if e.f == nil {
					continue
				}

				if got, want := env.Eval(sk.PassiveCalc[i+1]), e.f(l); got != want {
					t.Errorf("%s L%d passivecalc%d %s = %d, want %d", c.skill, l, i+1, e.stat, got, want)
				}
			}

			// PassiveStats only reports skills with the passive flag
			ps := pipe.PassiveStats(u, sk.ID)
			if c.passiveApplied != (len(ps) > 0) {
				t.Errorf("%s L%d PassiveStats returned %d mods, passiveApplied=%v", c.skill, l, len(ps), c.passiveApplied)
			}
		}
	}
}

// TestPassiveElemCalcs: Holy Fire / Holy Shock passive damage = elemental
// min/max (8.8) * Param5 / 256 (stats firemindam.. get whole damage points).
func TestPassiveElemCalcs(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()

	for _, name := range []string{"Holy Fire", "Holy Shock"} {
		sk := reg.ByName(name)
		for _, l := range []int{1, 10, 20} {
			u := calcUnit{reg: reg, levels: map[string]int{name: l}}
			env := d2skill.NewEnv(sk, l, u, reg)
			wantMax := int(sk.ElemMax(env, l)) * 6 / 256

			if got := env.Eval(sk.PassiveCalc[2]); got != wantMax {
				t.Errorf("%s L%d passive max = %d, want exms*6/256 = %d", name, l, got, wantMax)
			}

			if name == "Holy Fire" {
				if got, want := env.Eval(sk.PassiveCalc[1]), int(sk.ElemMin(env, l))*6/256; got != want {
					t.Errorf("%s L%d passive min = %d, want %d", name, l, got, want)
				}
			}
		}
	}
}

func TestSkillStatIDsAndOps(t *testing.T) {
	defs := loadItemStatCost(t)

	ids := map[string]int{"skill_armor_percent": 171, "skill_staminapercent": 162, "skill_passive_staminapercent": 163,
		"item_maxhp_percent": 76, "item_maxmana_percent": 77, "fireresist": 39, "maxfireresist": 40, "lightresist": 41,
		"coldresist": 43, "poisonresist": 45, "damagepercent": 25, "damageresist": 36, "firemindam": 48, "firemaxdam": 49,
		"lightmindam": 50, "lightmaxdam": 51}
	for n, id := range ids {
		if got := defs.ID(n); got != id {
			t.Errorf("stat %s id = %d, want %d", n, got, id)
		}
	}

	// ItemStatCost "op" of the stats skills produce (op 11 = percent of the op
	// stat, op 1 = percent for the stamina skill stats; 0 = no op: summed as is)
	wantOp := map[int]struct {
		op     int
		target string
	}{
		76: {11, "maxhp"}, 77: {11, "maxmana"}, 162: {1, "maxstamina"}, 163: {1, "maxstamina"},
		16: {13, "armorclass"}, 171: {0, ""}, 39: {0, ""}, 40: {0, ""}, 25: {0, ""}, 36: {0, ""},
	}
	for id, w := range wantOp {
		sd, ok := defs.Def(id)
		if !ok || sd.Op != w.op {
			t.Errorf("stat %d op = %d, want %d", id, sd.Op, w.op)
		}

		if w.target != "" && (len(sd.OpStats) == 0 || sd.OpStats[0] != w.target) {
			t.Errorf("stat %d op stat1 = %v, want %s", id, sd.OpStats, w.target)
		}
	}

	// distribution of op over the 1.14 table: ops 0, 3, 10, 12 and 14 are unused
	count := map[int]int{}

	for id := 0; id < 1024; id++ {
		if sd, ok := defs.Def(id); ok {
			count[sd.Op]++
		}
	}

	var used []int

	for op, n := range count {
		if op != 0 {
			used = append(used, op)
		}

		_ = n
	}

	sort.Ints(used)

	if want := []int{1, 2, 4, 5, 6, 7, 8, 9, 11, 13}; !equalInts(used, want) {
		t.Errorf("ops used by the table %v, want %v", used, want)
	}

	wantCount := map[int]int{1: 2, 2: 33, 4: 2, 5: 2, 6: 34, 7: 2, 8: 1, 9: 1, 11: 2, 13: 5}
	for op, n := range wantCount {
		if count[op] != n {
			t.Errorf("stats with op %d: %d, want %d", op, count[op], n)
		}
	}
}

// TestMergeWithItems folds the skill stats into the item stat list the way
// d2statlist.Compute does (Env.Skill) and checks the totals.
func TestMergeWithItems(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()
	defs := loadItemStatCost(t)

	hero := d2statlist.Hero{Class: d2statlist.Class{LifePerVit: 8, ManaPerEne: 8, StaminaPerVit: 4}, Level: 30, Dex: 40,
		BaseLife: 200, BaseMana: 100, BaseStam: 200}
	armor := d2statlist.Item{Slot: d2statlist.SlotTorso, Defense: 100,
		Props: []d2statlist.Prop{{ID: d2statlist.StatArmorPct, Value: 50}, {ID: d2statlist.StatMaxHPPct, Value: 10},
			{ID: d2statlist.StatFireResist, Value: 20}}}

	skill := func(name string, l int, syn map[string]int) *d2statlist.List {
		sk := reg.ByName(name)
		u := calcUnit{reg: reg, levels: map[string]int{name: l}}
		for k, v := range syn {
			u.levels[k] = v
		}

		env := d2skill.NewEnv(sk, l, u, reg)
		out := d2statlist.NewList()

		for i := 1; i <= 6; i++ {
			if sk.AuraStat[i] == "" {
				continue
			}

			p, ok := defs.NamedProp(sk.AuraStat[i], env.Eval(sk.AuraStatCalc[i]))
			if !ok {
				t.Fatalf("%s: unknown stat %s", name, sk.AuraStat[i])
			}

			out.AddProps([]d2statlist.Prop{p})
		}

		return out
	}

	base := d2statlist.Compute(hero, []d2statlist.Item{armor}, nil)
	// (100*150% + 40/4) = 160 hmm: item enhanced defense 150, + dex/4 = 10
	if base.Defense != 160 {
		t.Fatalf("base defense %d", base.Defense)
	}

	// Shout L1 (+100% defense) + Battle Orders L1 (+35% life, mana, stamina)
	// + Resist Fire L1 aura: skill percents add to the item percent (10%).
	sk := skill("Shout", 1, nil)
	sk.Merge(skill("Battle Orders", 1, nil))
	sk.Merge(skill("Resist Fire", 1, nil))

	got := d2statlist.Compute(hero, []d2statlist.Item{armor}, &d2statlist.Env{Skill: sk})

	if want := 160 + 160*100/100; got.Defense != want {
		t.Errorf("defense with Shout = %d, want %d (stat 171 applies to (armor + dex/4))", got.Defense, want)
	}

	// life: (200 + vit part 0) * (100+10+35)/100 ; the percents are summed first
	if want := 200 * 145 / 100; got.MaxLife != want {
		t.Errorf("max life = %d, want %d", got.MaxLife, want)
	}

	if want := 100 * 135 / 100; got.MaxMana != want {
		t.Errorf("max mana = %d, want %d", got.MaxMana, want)
	}

	if want := 200 * 135 / 100; got.MaxStamina != want {
		t.Errorf("max stamina = %d, want %d", got.MaxStamina, want)
	}

	// Resist Fire L1: 35 (dm34 at L1: (110*115)/(700)+35 = 53), item +20
	if want := 20 + dm(35, 150, 1); got.Resist[d2statlist.ResFire] != want {
		t.Errorf("fire resist = %d, want %d", got.Resist[d2statlist.ResFire], want)
	}

	if want := 75 + 1; got.MaxResist[d2statlist.ResFire] != want {
		t.Errorf("max fire resist = %d, want %d", got.MaxResist[d2statlist.ResFire], want)
	}
}

// TestStateGroups: Frozen, Shiver and Chilling Armor share States.txt group 1
// and replace each other; Burst of Speed (quickness) and Fade share group 2
// (UNVERIFIED that the game replaces on apply, see the notes).
func TestStateGroups(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(root, "states", "patch_d2", "States.txt"))
	if err != nil {
		t.Skip(err)
	}

	rm := &RecordManager{}
	if err = statesLoader(rm, d2txt.LoadDataDictionary(buf)); err != nil {
		t.Fatal(err)
	}

	grp := func(n string) int {
		if r := rm.States[n]; r != nil {
			return r.Group
		}

		return -1
	}

	for n, g := range map[string]int{"frozenarmor": 1, "shiverarmor": 1, "chillingarmor": 1, "quickness": 2, "fade": 2,
		"might": 0, "shout": 0, "battleorders": 0, "ironskin": 0, "naturalresistance": 0, "holyfire": 0, "holyshock": 0,
		"resistfire": 0} {
		if grp(n) != g {
			t.Errorf("state %s group = %d, want %d", n, grp(n), g)
		}
	}

	defs := d2state.Defs{}
	for n, r := range rm.States {
		defs[n] = d2state.Def{Name: n, Group: r.Group}
	}

	// the buff cast (0x5c7540) calls 0x56a480 (ClearGroup) before it builds
	// the new statlist, so a new armor ends the old one (verified).
	s := d2state.New()
	s.SetDefs(defs)
	s.Apply(0, d2state.Instance{Name: "frozenarmor", Mods: []d2state.StatMod{{Stat: "skill_armor_percent", Value: 30}}})
	s.Apply(1, d2state.Instance{Name: "might", Mods: []d2state.StatMod{{Stat: "damagepercent", Value: 40}}})
	s.ClearGroup(2, "shiverarmor")
	s.Apply(2, d2state.Instance{Name: "shiverarmor", Mods: []d2state.StatMod{{Stat: "skill_armor_percent", Value: 45}}})

	if s.Active(3, "frozenarmor") || !s.Active(3, "shiverarmor") || !s.Active(3, "might") {
		t.Errorf("states after Shiver Armor: %v", s.Names(3))
	}

	if got := s.DefensePct(3); got != 45 {
		t.Errorf("defense pct = %d, want 45 (armors do not stack)", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
