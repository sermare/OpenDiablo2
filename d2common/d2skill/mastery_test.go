package d2skill

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// equivTable is a miniature itemtypes tree (real codes, real parents).
var equivTable = map[string][2]string{
	"swor": {"mele", ""}, "axe": {"mele", ""}, "mele": {"weap", ""}, "weap": {"", ""},
	"jave": {"comb", "spea"}, "tkni": {"comb", "knif"}, "comb": {"mele", "thro"}, "thro": {"weap", ""},
	"spea": {"mele", ""}, "knif": {"mele", ""},
}

func equivFn(c string) (string, string) { e := equivTable[c]; return e[0], e[1] }

func TestTypeIs(t *testing.T) {
	tests := []struct {
		have, want string
		ok         bool
	}{
		{"swor", "swor", true}, {"swor", "mele", true}, {"swor", "weap", true}, {"swor", "axe", false},
		{"jave", "thro", true}, {"jave", "spea", true}, {"tkni", "thro", true}, {"swor", "thro", false}, {"", "swor", false},
	}

	for _, tc := range tests {
		if got := TypeIs(equivFn, tc.have, tc.want); got != tc.ok {
			t.Errorf("TypeIs(%q,%q) = %v", tc.have, tc.want, got)
		}
	}
}

func TestMasteryValue(t *testing.T) {
	mods := []TruePassiveMod{
		{Stat: "passive_mastery_melee_dmg", Value: 40, Param: "swor"},
		{Stat: "passive_mastery_melee_dmg", Value: 70, Param: "mele"}, // an ancestor type also matches
		{Stat: "passive_mastery_melee_dmg", Value: 99, Param: "axe"},
		{Stat: "passive_mastery_melee_th", Value: 55, Param: "swor"},
		{Stat: "passive_mastery_throw_crit", Value: 12, Param: "thro"},
		{Stat: "passive_mastery_melee_crit", Value: 30, Param: ""}, // not weapon-keyed: never a mastery
	}

	tests := []struct {
		name string
		kind MasteryKind
		wt   string
		want int
		// thrown selects the throw family (0x159..), else the melee family
		thrown bool
	}{
		{"sword damage takes the largest matching entry", MasteryDamage, "swor", 70, false},
		{"axe", MasteryDamage, "axe", 99, false},
		{"sword to-hit", MasteryToHit, "swor", 55, false},
		{"axe has no to-hit entry", MasteryToHit, "axe", 0, false},
		{"throwing knife crit, thrown", MasteryCrit, "tkni", 12, true},
		{"throwing knife crit, thrown reads no melee family", MasteryDamage, "tkni", 0, true},
		{"throwing knife crit, melee lookup ignores the throw family", MasteryCrit, "tkni", 0, false},
		{"sword crit: unkeyed entry ignored", MasteryCrit, "swor", 0, false},
		{"no weapon", MasteryDamage, "", 0, false},
	}

	for _, tc := range tests {
		got := MasteryValue(mods, tc.kind, tc.thrown, func(p string) bool { return tc.wt != "" && TypeIs(equivFn, tc.wt, p) })
		if got != tc.want {
			t.Errorf("%s: %d want %d", tc.name, got, tc.want)
		}
	}

	if MasteryValue(nil, MasteryDamage, false, nil) != 0 {
		t.Error("no masteries")
	}
}

// masteryHero is a test hero with a fixed mastery lookup.
type masteryHero struct {
	*testUnit
	m map[MasteryKind]int
}

func (h masteryHero) Mastery(k MasteryKind, _ *Skill) int { return h.m[k] }

func TestStrikeUsesMastery(t *testing.T) {
	// weapon 5-9, all rolls 0: plain Attack = 5.0, hit
	strike := func(wrap func(*testUnit) Unit) *MeleeResult {
		f := newFixture(map[string]int{"Attack": 1})
		f.u.base = f.u.levels
		target := &testTarget{id: "zombie", alive: true, level: 1, defense: 10}
		id := f.id("Attack")
		tg := Target{Unit: target}
		u := wrap(f.u)
		f.p.Start(u, id, tg)

		return f.p.Do(u, id, tg).Melee
	}

	plain := strike(func(u *testUnit) Unit { return u })

	// regression: a hero without masteries (Mastery all 0) is unchanged
	zero := strike(func(u *testUnit) Unit { return masteryHero{u, nil} })
	if plain.Damage != zero.Damage || plain.Total != zero.Total || plain.Chance != zero.Chance {
		t.Fatalf("zero masteries changed the strike:\n%+v\n%+v", plain, zero)
	}

	// damage +50%
	dm := strike(func(u *testUnit) Unit { return masteryHero{u, map[MasteryKind]int{MasteryDamage: 50}} })
	if want := int32(5*256 + 5*256*50/100); dm.Damage.Physical != want {
		t.Errorf("damage mastery: %d want %d", dm.Damage.Physical, want)
	}

	// crit: the roll is 0 < 10, so the physical damage doubles
	cr := strike(func(u *testUnit) Unit { return masteryHero{u, map[MasteryKind]int{MasteryCrit: 10}} })
	if cr.Damage.Physical != 2*plain.Damage.Physical {
		t.Errorf("crit mastery: %d vs %d", cr.Damage.Physical, plain.Damage.Physical)
	}

	// to-hit (0x57ba63, verified): the mastery is a PERCENT of the attack
	// rating, summed with item_tohit_percent: +100% on AR 300 rolls like AR 600
	hi := func(m, ar int) int {
		f := newFixture(map[string]int{"Attack": 1})
		f.u.base = f.u.levels
		f.u.ar = ar
		target := &testTarget{id: "z", alive: true, level: 20, defense: 600}
		id := f.id("Attack")
		u := masteryHero{f.u, map[MasteryKind]int{MasteryToHit: m}}
		tg := Target{Unit: target}
		f.p.Start(u, id, tg)

		return f.p.Do(u, id, tg).Melee.Chance
	}

	if a, b, c := hi(0, 300), hi(100, 300), hi(0, 600); b <= a || b != c {
		t.Errorf("to-hit mastery chance: base %d, +100%% %d, AR 600 %d", a, b, c)
	}

	// the percent is additive with item_tohit_percent, not multiplicative
	f := newFixture(map[string]int{"Attack": 1})
	f.u.base = f.u.levels
	f.u.ar = 300
	f.u.stats["item_tohit_percent"] = 50
	id := f.id("Attack")
	tg := Target{Unit: &testTarget{id: "z", alive: true, level: 20, defense: 600}}
	u := masteryHero{f.u, map[MasteryKind]int{MasteryToHit: 50}}
	f.p.Start(u, id, tg)
	both := f.p.Do(u, id, tg).Melee.Chance

	if want := hi(0, 600); both != want {
		t.Errorf("tohit 50+50 percent: chance %d want %d (AR 300 * 2)", both, want)
	}
}

// TestDamageMasteryAdditiveWithED: the damage mastery shares one percent pool
// with damagepercent (0x5792ab): 50 ED + 50 mastery = +100%, not x2.25.
func TestDamageMasteryAdditiveWithED(t *testing.T) {
	f := newFixture(map[string]int{"Attack": 1})
	f.u.base = f.u.levels
	f.u.stats["damagepercent"] = 50
	id := f.id("Attack")
	tg := Target{Unit: &testTarget{id: "zombie", alive: true, level: 1, defense: 10}}
	u := masteryHero{f.u, map[MasteryKind]int{MasteryDamage: 50}}
	f.p.Start(u, id, tg)

	got := f.p.Do(u, id, tg).Melee.Damage.Physical
	if want := int32(5 * 256 * 2); got != want {
		t.Errorf("ED 50 + mastery 50: %d want %d", got, want)
	}
}

// TestMissileMasteryGate: MISSILE_BuildDamageDescriptor adds the damage
// mastery only for skills with a non-zero SrcDam.
func TestMissileMasteryGate(t *testing.T) {
	f := newFixture(map[string]int{"Attack": 1})
	u := masteryHero{f.u, map[MasteryKind]int{MasteryDamage: 40}}

	if got := missileMastery(u, &Skill{DamageSpec: DamageSpec{SrcDam: 128}}); got != 40 {
		t.Errorf("weapon skill: %d", got)
	}

	if got := missileMastery(u, &Skill{DamageSpec: DamageSpec{SrcDam: 0}}); got != 0 {
		t.Errorf("spell skill: %d", got)
	}

	if got := missileMastery(f.u, &Skill{DamageSpec: DamageSpec{SrcDam: 128}}); got != 0 {
		t.Errorf("no mastery unit: %d", got)
	}
}

// loadSkillRows reads skills.txt (tab separated, header row) into rows.
func loadSkillRows(t *testing.T, path string) []row {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)

	var hdr []string

	var out []row

	for sc.Scan() {
		cells := strings.Split(sc.Text(), "\t")
		if hdr == nil {
			hdr = cells

			continue
		}

		r := row{}

		for i, h := range hdr {
			if i < len(cells) && cells[i] != "" {
				r[h] = cells[i]
			}
		}

		out = append(out, r)
	}

	return out
}

// TestRealMasteryRows checks the Barbarian masteries and the weapon passives of
// the real skills.txt (D2_TABLES; skipped when unset).
func TestRealMasteryRows(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	reg := NewRegistry()
	keyed := map[string]bool{}

	for _, r := range loadSkillRows(t, filepath.Join(dir, "skills", "patch_d2", "skills.txt")) {
		if r["skill"] == "" || r["Id"] == "" {
			continue
		}

		reg.Add(skillFromRow(r))

		if r["passiveitype"] != "" {
			keyed[r["skill"]] = true
		}
	}

	// the weapon-keyed passives of 1.14b: skill -> weapon type and stat family
	tests := []struct {
		skill, itype, th, dmg, crit string
		weapon                      string // a weapon type that must match
		other                       string // one that must not
	}{
		{"Sword Mastery", "swor", "melee_th", "melee_dmg", "melee_crit", "swor", "axe"},
		{"Axe Mastery", "axe", "melee_th", "melee_dmg", "melee_crit", "axe", "swor"},
		{"Mace Mastery", "blun", "melee_th", "melee_dmg", "melee_crit", "club", "swor"},
		{"Pole Arm Mastery", "pole", "melee_th", "melee_dmg", "melee_crit", "pole", "spea"},
		{"Spear Mastery", "spea", "melee_th", "melee_dmg", "melee_crit", "spea", "pole"},
		{"Throwing Mastery", "thro", "throw_th", "throw_dmg", "throw_crit", "tkni", "swor"},
		{"Claw Mastery", "h2h", "melee_th", "melee_dmg", "melee_crit", "h2h", "swor"},
	}

	// itemtypes parents used by the matches above (real codes: club and tkni
	// are below blun and thro; the structure is checked by TestTypeIs)
	real := map[string][2]string{"club": {"blun", ""}, "tkni": {"comb", "knif"}, "comb": {"mele", "thro"}}
	eq := func(c string) (string, string) { return real[c][0], real[c][1] }

	for _, tc := range tests {
		sk := reg.ByName(tc.skill)
		if sk == nil {
			t.Errorf("%s missing", tc.skill)

			continue
		}

		if sk.PassiveIType != tc.itype || sk.PassiveState == "" {
			t.Errorf("%s: itype %q state %q", tc.skill, sk.PassiveIType, sk.PassiveState)
		}

		var vals [3][]int

		for _, lvl := range []int{1, 2, 10, 20} {
			f := &fixture{}
			f.reg = reg
			f.u = newUnit(reg, map[string]int{tc.skill: lvl})
			f.u.base = f.u.levels
			f.p = &Pipeline{Skills: reg}

			mods := f.p.TruePassiveStats(f.u, sk.ID)
			if len(mods) != 3 {
				t.Fatalf("%s lvl %d: %d mods", tc.skill, lvl, len(mods))
			}

			for i, name := range []string{tc.th, tc.dmg, tc.crit} {
				if mods[i].Stat != "passive_mastery_"+name || mods[i].Param != tc.itype {
					t.Errorf("%s: slot %d is %+v", tc.skill, i, mods[i])
				}

				vals[i] = append(vals[i], mods[i].Value)
			}

			is := func(want string) func(string) bool {
				return func(p string) bool { return TypeIs(eq, want, p) }
			}

			for k := MasteryToHit; k <= MasteryCrit; k++ {
				if MasteryValue(mods, k, tc.itype == "thro", is(tc.weapon)) != mods[int(k)].Value {
					t.Errorf("%s lvl %d kind %d: not read for weapon %s", tc.skill, lvl, k, tc.weapon)
				}

				if MasteryValue(mods, k, tc.itype == "thro", is(tc.other)) != 0 {
					t.Errorf("%s lvl %d kind %d: read for weapon %s", tc.skill, lvl, k, tc.other)
				}
			}
		}

		// every stat is positive and does not shrink as the skill level rises
		for i := range vals {
			for j, v := range vals[i] {
				if v <= 0 || (j > 0 && v < vals[i][j-1]) {
					t.Errorf("%s stat %d values by level: %v", tc.skill, i, vals[i])

					break
				}
			}
		}
	}

	// Weapon Block is keyed to h2h too (it is the only other keyed row); the
	// Amazon passives (Critical Strike, Penetrate, Dodge...) carry no
	// passiveitype, so they are plain totals, not weapon-keyed.
	if !keyed["Weapon Block"] {
		t.Error("Weapon Block should be keyed")
	}

	if len(keyed) != len(tests)+1 {
		t.Errorf("weapon-keyed skills: %v", keyed)
	}

	for _, n := range []string{"Critical Strike", "Penetrate", "Dodge", "Avoid", "Evade"} {
		if keyed[n] {
			t.Errorf("%s should not be weapon-keyed", n)
		}
	}
}

// TestRealSkillThrows pins the skill half of the throw family test (0x646a90)
// on real skills.txt rows and the real ItemTypes tree (D2_TABLES).
func TestRealSkillThrows(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	parents := map[string][2]string{}

	var hdr []string

	f, err := os.Open(filepath.Join(dir, "d2exp", "ItemTypes.txt"))
	if err != nil {
		t.Skip("no ItemTypes.txt")
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		cells := strings.Split(sc.Text(), "\t")
		if hdr == nil {
			hdr = cells

			continue
		}

		r := map[string]string{}
		for i, h := range hdr {
			if i < len(cells) {
				r[h] = cells[i]
			}
		}

		e1, e2 := r["Equiv1"], r["Equiv2"]
		if e1 == "0" {
			e1 = ""
		}

		parents[r["Code"]] = [2]string{e1, e2}
	}

	typeIs := func(have, want string) bool {
		return TypeIs(func(c string) (string, string) { return parents[c][0], parents[c][1] }, have, want)
	}

	reg := NewRegistry()

	for _, r := range loadSkillRows(t, filepath.Join(dir, "skills", "patch_d2", "skills.txt")) {
		if r["skill"] != "" && r["Id"] != "" {
			reg.Add(skillFromRow(r))
		}
	}

	throws := []string{"Throw", "Left Hand Throw", "Double Throw", "Poison Javelin", "Plague Javelin", "Lightning Bolt", "Lightning Fury"}
	plain := []string{"Attack", "Left Hand Swing", "Jab", "Power Strike", "Impale", "Fend", "Charged Strike", "Fire Arrow", "Bash", "Zeal", "Sword Mastery"}

	for _, n := range throws {
		sk := reg.ByName(n)
		if sk == nil || !SkillThrows(sk, typeIs) {
			t.Errorf("%s should select the throw family", n)
		}
	}

	for _, n := range plain {
		sk := reg.ByName(n)
		if sk == nil || SkillThrows(sk, typeIs) {
			t.Errorf("%s should select the melee family", n)
		}
	}

	// the damage mastery gate on real rows: weapon skills have SrcDam
	if reg.ByName("Fire Arrow").SrcDam <= 0 || reg.ByName("Charged Strike").SrcDam != 0 {
		t.Error("SrcDam of Fire Arrow / Charged Strike")
	}
}
