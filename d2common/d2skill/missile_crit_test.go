package d2skill

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// countRoller wraps seq and counts the consumed steps.
type countRoller struct {
	seq
	n int
}

func (c *countRoller) Roll(n int32) uint32 { c.n++; return c.seq.Roll(n) }

func TestRollMissileStrikeOrder(t *testing.T) {
	in := d2combat.StrikeInput{CriticalChance: 10, DeadlyChance: 20, WeaponChance: 30}

	tests := []struct {
		name  string
		vals  []uint32
		want  bool
		steps int
	}{
		{"critical first", []uint32{9}, true, 1},
		{"critical fails, deadly hits", []uint32{10, 19}, true, 2},
		{"both fail, mastery hits (last)", []uint32{10, 20, 29}, true, 3},
		{"all fail", []uint32{10, 20, 30}, false, 3},
	}

	for _, tc := range tests {
		r := &countRoller{seq: seq{vals: tc.vals}}
		if got := d2combat.RollMissileStrike(r, in); got != tc.want || r.n != tc.steps {
			t.Errorf("%s: got %v in %d steps, want %v in %d", tc.name, got, r.n, tc.want, tc.steps)
		}
	}

	// zero chances: no step consumed (documented deviation from the exe's unconditional 0x151 roll)
	r := &countRoller{}
	if d2combat.RollMissileStrike(r, d2combat.StrikeInput{}) || r.n != 0 {
		t.Errorf("zero chances consumed %d steps", r.n)
	}

	// without a weapon the mastery is skipped
	r = &countRoller{}
	if d2combat.RollMissileStrike(r, d2combat.StrikeInput{WeaponChance: 99, SkipWeapon: true}) || r.n != 0 {
		t.Errorf("skip weapon consumed %d", r.n)
	}
}

func TestMissileCritGate(t *testing.T) {
	f := newFixture(map[string]int{"Attack": 1})
	f.u.stats["passive_critical_strike"] = 100

	weapon := &Skill{DamageSpec: DamageSpec{SrcDam: 128}}
	spell := &Skill{DamageSpec: DamageSpec{SrcDam: 0}}

	tests := []struct {
		name string
		sk   *Skill
		ms   *d2missile.Spec
		want bool
	}{
		{"weapon skill, plain missile", weapon, &d2missile.Spec{}, true},
		{"weapon skill, record row not given", weapon, nil, true},
		{"spell without SrcDam never crits", spell, &d2missile.Spec{}, false},
		{"missile SrcDamage -1 turns it off (poisonjavcloud)", weapon, &d2missile.Spec{SrcDam: -1}, false},
		{"missile SrcDamage 128 keeps it (explodingarrow)", weapon, &d2missile.Spec{SrcDam: 128}, true},
		{"no skill", nil, nil, false},
	}

	for _, tc := range tests {
		if got := missileCrit(f.u, tc.sk, tc.ms); got != tc.want {
			t.Errorf("%s: %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestMissileCritMasteryNeedsHero(t *testing.T) {
	f := newFixture(map[string]int{"Attack": 1})
	sk := &Skill{DamageSpec: DamageSpec{SrcDam: 128}}

	// roller 0 < 10: the mastery crit succeeds; a unit without masteries has none
	if !missileCrit(masteryHero{f.u, map[MasteryKind]int{MasteryCrit: 10}}, sk, nil) {
		t.Error("mastery crit did not fire")
	}

	if missileCrit(f.u, sk, nil) {
		t.Error("crit without any chance")
	}
}

// TestMissileCritDoubles: a Magic Arrow of a hero with critical strike is
// doubled once per missile after the percent bonus; a hero without crit stats
// is bit-for-bit unchanged and consumes no random step.
func TestMissileCritDoubles(t *testing.T) {
	cast := func(stats map[string]int, vals ...uint32) (*d2missile.Missile, *countRoller) {
		f := newFixture(map[string]int{"Magic Arrow": 3})
		f.u.base = f.u.levels
		cr := &countRoller{seq: seq{vals: vals}}
		f.u.roller = cr

		for k, v := range stats {
			f.u.stats[k] = v
		}

		_, do := f.cast("Magic Arrow", 20, 0)
		if !do.OK || len(do.Missiles) != 1 {
			t.Fatalf("%+v", do)
		}

		return do.Missiles[0], cr
	}

	plain, pr := cast(nil)
	zero, zr := cast(map[string]int{"passive_critical_strike": 0, "item_deadlystrike": 0})

	if plain.Damage != zero.Damage || plain.Damage.Crit || pr.n != zr.n {
		t.Fatalf("zero-crit regression: %+v (%d steps) vs %+v (%d steps)", plain.Damage, pr.n, zero.Damage, zr.n)
	}

	// step count of the cast without crit stats: crit stats add exactly one roll
	base := pr.n

	crit, cr := cast(map[string]int{"passive_critical_strike": 20}, make([]uint32, base+4)...)
	if !crit.Damage.Crit || cr.n != base+1 {
		t.Fatalf("critical strike 20, roll 0: crit=%v steps %d want %d", crit.Damage.Crit, cr.n, base+1)
	}

	// the descriptor is the same; rolling it doubles the physical part and flags it
	rp := plain.Damage.Roll(&seq{})
	rc := crit.Damage.Roll(&seq{})

	if rp.Physical == 0 || rc.Physical != 2*rp.Physical || rc.Result&d2combat.ResultCritical == 0 || rp.Result&d2combat.ResultCritical != 0 {
		t.Fatalf("plain %+v crit %+v", rp, rc)
	}

	// a failed roll (99 >= 20) leaves the damage alone
	miss, _ := cast(map[string]int{"passive_critical_strike": 20}, append(make([]uint32, 0), 99, 99, 99, 99, 99, 99, 99, 99)...)
	if miss.Damage.Crit {
		t.Fatalf("roll 99 against 20 crit")
	}

	// percent bonuses are applied before the doubling (0x5a6690 order)
	crit.Damage.DamagePct = 50

	if got, want := crit.Damage.Roll(&seq{}).Physical, 2*(rp.Physical+rp.Physical/2); got != want {
		t.Fatalf("pct then double: %d want %d", got, want)
	}
}

// TestRealMissileCritGate pins the crit gate on the real tables: the weapon
// missiles of the Amazon can crit, the spell missiles and the javelin clouds
// cannot (D2_TABLES; skipped when unset).
func TestRealMissileCritGate(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	skills := map[string]row{}
	for _, r := range loadSkillRows(t, filepath.Join(dir, "skills", "patch_d2", "skills.txt")) {
		skills[r["skill"]] = r
	}

	missileSrc := map[string]int{}
	for _, r := range loadSkillRows(t, filepath.Join(dir, "skills", "patch_d2", "missiles.txt")) {
		if r["Missile"] == "" {
			continue
		}

		n := 0
		if r["SrcDamage"] != "" {
			n, _ = strconv.Atoi(r["SrcDamage"])
		}

		missileSrc[r["Missile"]] = n
	}

	f := newFixture(map[string]int{})
	f.u.stats["passive_critical_strike"] = 100

	crits := func(skill, missile string) bool {
		r, ok := skills[skill]
		if !ok {
			t.Fatalf("no skill %s", skill)
		}

		n, _ := strconv.Atoi(r["SrcDam"])

		return missileCrit(f.u, &Skill{DamageSpec: DamageSpec{SrcDam: n}}, &d2missile.Spec{SrcDam: missileSrc[missile]})
	}

	tests := []struct {
		skill, missile string
		want           bool
	}{
		{"Magic Arrow", "magicarrow", true},
		{"Fire Arrow", "firearrow", true},
		{"Multiple Shot", "multipleshotarrow", true},
		{"Multiple Shot", "multipleshotbolt", true},
		{"Guided Arrow", "guidedarrow", true},
		{"Strafe", "strafearrow", true},
		{"Strafe", "strafebolt", true},
		{"Exploding Arrow", "explodingarrow", true},
		{"Poison Javelin", "poisonjav", true},
		{"Lightning Fury", "lightningfury", true},
		{"Fire Bolt", "firebolt", false},
		{"Charged Bolt", "chargedbolt", false},
		{"Fire Ball", "fireball", false},
		// the javelin sub-missiles carry SrcDamage -1
		{"Poison Javelin", "poisonjavcloud", false},
		{"Plague Javelin", "plaguejavcloud", false},
		{"Lightning Fury", "furylightning", false},
	}

	for _, tc := range tests {
		if got := crits(tc.skill, tc.missile); got != tc.want {
			t.Errorf("%s / %s: crit gate %v want %v", tc.skill, tc.missile, got, tc.want)
		}
	}
}
