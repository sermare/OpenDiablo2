package d2skill

import "testing"

// Tests of skills_next2.go and of the next-2 changes in skills_next.go /
// class.go (Energy Shield, Revive, Sanctuary, Redemption, Smite with Holy
// Shield, Charge overlay, Corpse Explosion falloff, knockback distance).

func TestEnergyShieldAbsorb(t *testing.T) {
	cases := []struct {
		name                   string
		dmg, pct, ratio, mana  int
		wantAbsorbed, wantCost int
	}{
		{"half, cheap ratio", 100, 50, 16, 1000, 50, 50},
		{"ratio 32 costs two mana per point", 100, 40, 32, 1000, 40, 80},
		{"mana limits it", 100, 50, 16, 20, 20, 20},
		{"mana limit with a dear ratio", 100, 50, 32, 20, 10, 20},
		{"no mana", 100, 50, 16, 0, 0, 0},
		{"no damage", 0, 50, 16, 100, 0, 0},
		{"state without percent", 100, 0, 16, 100, 0, 0},
		{"ratio below one is one", 100, 100, 0, 1000, 100, 6},
	}

	for _, c := range cases {
		a, cost := EnergyShieldAbsorb(c.dmg, c.pct, c.ratio, c.mana)
		if a != c.wantAbsorbed || cost != c.wantCost {
			t.Errorf("%s: absorbed %d cost %d, want %d and %d", c.name, a, cost, c.wantAbsorbed, c.wantCost)
		}
	}
}

func TestReviveLife(t *testing.T) {
	rollMax := func(n int) int { return n - 1 }
	rollMin := func(int) int { return 0 }

	cases := []struct {
		name   string
		lo, hi int
		roll   func(int) int
		want   int
	}{
		{"lowest roll", 40, 60, rollMin, 40},
		{"highest roll", 40, 60, rollMax, 60},
		{"swapped bounds use the minimum", 50, 30, rollMax, 50},
		{"never below one", 0, 0, rollMin, 1},
		{"no roller", 40, 60, nil, 40},
	}

	for _, c := range cases {
		if got := ReviveLife(c.lo, c.hi, c.roll); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

func TestReviveCapped(t *testing.T) {
	cases := []struct {
		corpse, caster, life, wantLvl, wantHP int
	}{
		{10, 20, 300, 10, 300}, // corpse below the caster: unchanged
		{30, 15, 400, 15, 200}, // above: caster level, life scaled by 15/30
		{40, 1, 10, 1, 1},      // never below one life
		{0, 20, 50, 0, 50},     // unknown corpse level: unchanged
	}

	for _, c := range cases {
		l, hp := ReviveCapped(c.corpse, c.caster, c.life)
		if l != c.wantLvl || hp != c.wantHP {
			t.Errorf("corpse %d caster %d: (%d,%d), want (%d,%d)", c.corpse, c.caster, l, hp, c.wantLvl, c.wantHP)
		}
	}
}

func TestFilterAllowsMonster(t *testing.T) {
	const sanctuary, holyFire = 59270, 42883

	cases := []struct {
		name         string
		filter       int
		undead, boss bool
		want         bool
	}{
		{"Sanctuary undead", sanctuary, true, false, true},
		{"Sanctuary living monster", sanctuary, false, false, false},
		{"Sanctuary undead boss", sanctuary, true, true, false},
		{"Holy Fire living monster", holyFire, false, false, true},
		{"Holy Fire boss", holyFire, false, true, true},
		{"empty filter is the default", 0, false, false, true},
		{"players only", FilterPlayers, false, false, false},
	}

	for _, c := range cases {
		if got := FilterAllowsMonster(c.filter, c.undead, c.boss); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}

	if FilterAllowsPlayer(sanctuary) || !FilterAllowsPlayer(holyFire) || !FilterAllowsPlayer(0) {
		t.Error("player filter bit")
	}
}

func TestKnockDistanceFor(t *testing.T) {
	if KnockDistanceFor(10) != 10 || KnockDistanceFor(78) != 5 {
		t.Errorf("distances %d %d", KnockDistanceFor(10), KnockDistanceFor(78))
	}
}

func next2Fixture(levels map[string]int) *classFixture {
	cf := nextFixture(levels)

	for _, r := range []row{
		{"skill": "Sanctuary", "Id": "410", "srvdofunc": "66", "aurafilter": "59270", "aurastate": "sanctuary",
			"aurarangecalc": "ln12", "aurastat1": "item_undeaddamage_percent", "aurastatcalc1": "ln34", "aura": "1",
			"EType": "mag", "EMin": "8", "EMax": "16", "HitShift": "8", "manashift": "8"},
		{"skill": "Redemption", "Id": "411", "srvdofunc": "82", "aurafilter": "4354", "aurastate": "redemption",
			"aurarangecalc": "ln12", "calc1": "dm34", "calc2": "ln56", "calc3": "ln56", "aura": "1", "manashift": "8",
			"mana": "3", "lvlmana": "1"},
		{"skill": "Holy Shield", "Id": "412", "srvdofunc": "18", "aurastate": "holyshield", "MinDam": "3", "MaxDam": "6",
			"MinLevDam1": "2", "MaxLevDam1": "2", "HitShift": "8", "manashift": "8"},
	} {
		cf.reg.Add(skillFromRow(r))
	}

	cf.reg.ByName("Sanctuary").ResultFlags = 11 // as skills.txt: hit | death | knockback

	return cf
}

func TestSanctuaryEffect(t *testing.T) {
	cf := next2Fixture(map[string]int{"Sanctuary": 5})
	id := cf.id("Sanctuary")
	tg := Target{X: 5, Y: 5}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)
	e := effectOf(t, r, "aura")

	if !r.OK || e.Mode != "damage" || !e.Knock || e.Filter != 59270 || e.Desc == nil || e.Desc.Magic.Max == 0 {
		t.Errorf("sanctuary %+v", e)
	}

	// the same row without the knockback bit does not push
	cf.reg.ByName("Sanctuary").ResultFlags = 3
	cf.p.Start(cf.u, id, tg)

	if e = effectOf(t, cf.p.Do(cf.u, id, tg), "aura"); e.Knock {
		t.Errorf("knock without result flag 8: %+v", e)
	}
}

func TestRedemptionCost(t *testing.T) {
	cf := next2Fixture(map[string]int{"Redemption": 5})
	id := cf.id("Redemption")
	tg := Target{X: 5, Y: 5}
	cf.p.Start(cf.u, id, tg)
	e := effectOf(t, cf.p.Do(cf.u, id, tg), "aura")

	// (lvlmana*(lvl-1) + mana) << manashift = (4 + 3) * 256
	if e.Mode != "redemption" || e.Cost != 7<<8 {
		t.Errorf("redemption %+v, cost want %d", e, 7<<8)
	}
}

func TestSmiteAddsHolyShield(t *testing.T) {
	cf := next2Fixture(map[string]int{"Smite": 10, "Holy Shield": 5})
	z := cf.addFoe("z", 1, 0)
	cf.u.roller = &seq{vals: []uint32{0}}
	id := cf.id("Smite")
	tg := Target{X: 1, Y: 0, Unit: z, UX: 1, UY: 0}
	hs := cf.reg.ByName("Holy Shield")

	hu := holyUnit{shieldUnit: shieldUnit{testUnit: cf.u, lo: 10, hi: 20, has: true}, id: hs.ID, lvl: 5}
	cf.p.Start(hu, id, tg)

	r := cf.p.Do(hu, id, tg)
	if !r.OK || r.Melee == nil {
		t.Fatalf("smite %+v", r)
	}

	// the roll is 0: damage = (shield min + holy shield min at its level) raised by calc1 percent
	env := cf.env("Holy Shield", 5)
	base := int32(10<<8) + hs.PhysMin(env, 5, 0, false)
	pct := int32(cf.env("Smite", 10).eval(cf.reg.ByName("Smite").Calc[1]))
	want := base + base*pct/100

	if hs.PhysMin(env, 5, 0, false) <= 0 {
		t.Fatal("test row has no holy shield damage")
	}

	if r.Melee.Damage.Physical != want {
		t.Errorf("physical %d, want %d", r.Melee.Damage.Physical, want)
	}

	// without the state the shield alone counts
	cf2 := next2Fixture(map[string]int{"Smite": 10, "Holy Shield": 5})
	z2 := cf2.addFoe("z", 1, 0)
	cf2.u.roller = &seq{vals: []uint32{0}}
	su := shieldUnit{testUnit: cf2.u, lo: 10, hi: 20, has: true}
	tg2 := Target{X: 1, Y: 0, Unit: z2, UX: 1, UY: 0}
	cf2.p.Start(su, id, tg2)

	r2 := cf2.p.Do(su, id, tg2)
	base = int32(10 << 8)

	if want = base + base*pct/100; r2.Melee.Damage.Physical != want {
		t.Errorf("without the state: physical %d, want %d", r2.Melee.Damage.Physical, want)
	}
}

type holyUnit struct {
	shieldUnit
	id, lvl int
}

func (h holyUnit) StateSkill(state string) (int, int, bool) {
	return h.id, h.lvl, state == "holyshield"
}

func TestChargeShowsTheBashOverlay(t *testing.T) {
	cf := nextFixture(map[string]int{"Charge": 5})
	z := cf.addFoe("z", 3, 0)
	cf.u.roller = &seq{vals: make([]uint32, 20)}
	id := cf.id("Charge")
	tg := Target{X: 3, Y: 0, Unit: z, UX: 3, UY: 0}
	cf.p.Start(cf.u, id, tg)
	r := cf.p.Do(cf.u, id, tg)
	o := effectOf(t, r, "overlay")

	if o.Overlay != OverlayBash || o.Target == nil || o.Target.ID() != "z" {
		t.Errorf("overlay %+v", o)
	}
}

func TestCorpseExplosionFalloffAndRoll(t *testing.T) {
	cf := newClassFixture(map[string]int{"Corpse Explosion": 5})
	cf.u.roller = &seq{vals: []uint32{30}}
	id := cf.id("Corpse Explosion")
	tg := Target{X: 9, Y: 9, Corpse: true, CX: 9, CY: 9, CorpseID: "c", CorpseHP: 100}
	cf.p.Start(cf.u, id, tg)
	e := effectOf(t, cf.p.Do(cf.u, id, tg), "area_hit")

	// aurarange 12: radius (12+1)/2 = 6, the physical part vanishes beyond (12/2)^2 = 36
	if !e.Falloff || e.FalloffSq != 36 || e.Radius != 6 {
		t.Errorf("falloff %+v", e)
	}

	// the roll is over the life range 70..120: 70 + 30 = 100, half of it fire
	if e.Desc.Fire.Min != 50<<8 || e.Desc.PhysMin != 50<<8 {
		t.Errorf("damage fire %d phys %d, want 50 each", e.Desc.Fire.Min>>8, e.Desc.PhysMin>>8)
	}
}
