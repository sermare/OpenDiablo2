package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func armorProps(props ...d2statlist.Prop) []d2statlist.Item {
	return []d2statlist.Item{{Slot: d2statlist.SlotTorso, Defense: 0, Props: props}}
}

// hitStats runs n monster attacks (fixed seed) at the owner's merc and sums what it takes.
func hitStats(t *testing.T, d *Director, owner *d2mapentity.Player, n int) (hits, total int) {
	t.Helper()

	tu := d.mercs[owner]
	att := &unit{m: &d2mapentity.Monster{}, b: d2monster.NewBrain(7, 0, d2monster.Normal, &d2monster.Profile{}, 1234)}
	att.m.Vitals.Level = 30

	atk := d2mapentity.MonsterAttack{ToHit: 120, Min: 20, Max: 40}

	for i := 0; i < n; i++ {
		hit, _, _, dmg := d.rollMercHit(att, tu, atk)
		if hit {
			hits++
			total += dmg
		}
	}

	return hits, total
}

func TestMercWithGearTakesLessDamage(t *testing.T) {
	d, owner, _ := gearDirector(t)

	plainHits, plainTotal := hitStats(t, d, owner, 300)

	d.SetMercItems(owner, append(testGear(), armorProps(
		d2statlist.Prop{ID: d2statlist.StatDamageResist, Value: 40},
		d2statlist.Prop{ID: d2statlist.StatNormalReduce, Value: 5})...))

	hits, total := hitStats(t, d, owner, 300)

	if plainHits == 0 {
		t.Fatal("the plain merc was never hit")
	}

	if hits > plainHits {
		t.Errorf("more defense must not give more hits: %d vs %d", hits, plainHits)
	}

	if total >= plainTotal || total*100 > plainTotal*65 {
		t.Errorf("total damage with gear %d, without %d: want at most 65%%", total, plainTotal)
	}
}

// A merc without gear (and one whose gear was removed) takes exactly the monster's roll.
func TestMercWithoutGearUnchanged(t *testing.T) {
	d, owner, _ := gearDirector(t)
	h0, t0 := hitStats(t, d, owner, 200)

	d.SetMercItems(owner, testGear())
	d.SetMercItems(owner, nil)

	h1, t1 := hitStats(t, d, owner, 200)
	if h0 != h1 || t0 != t1 {
		t.Errorf("gear removed: %d hits %d dmg, before %d hits %d dmg", h1, t1, h0, t0)
	}

	tu := d.mercs[owner]
	for _, v := range []int{1, 7, 33, 250} {
		if got := d.takenByMerc(tu, v); got != v {
			t.Errorf("physical %d became %d without gear", v, got)
		}
	}
}

func TestMercResistMatrix(t *testing.T) {
	// the table resist of the test merc at level 3 is 10 (Resist 10, no growth at the row level)
	fire30 := armorProps(d2statlist.Prop{ID: d2statlist.StatFireResist, Value: 30})
	fire100 := armorProps(d2statlist.Prop{ID: d2statlist.StatFireResist, Value: 100})
	fire100max := armorProps(d2statlist.Prop{ID: d2statlist.StatFireResist, Value: 100},
		d2statlist.Prop{ID: d2statlist.StatMaxFireRes, Value: 10})
	phys80 := armorProps(d2statlist.Prop{ID: d2statlist.StatDamageResist, Value: 80})
	reduce := armorProps(d2statlist.Prop{ID: d2statlist.StatNormalReduce, Value: 15})
	pois := armorProps(d2statlist.Prop{ID: d2statlist.StatPoisonResist, Value: 20})

	tests := []struct {
		name      string
		items     []d2statlist.Item
		expansion bool
		diff      d2monster.Difficulty
		typ       MercDamageType
		dmg, want int
	}{
		{"no gear fire normal uses table resist", nil, true, d2monster.Normal, MercFire, 100, 90},
		{"fire 40 normal", fire30, true, d2monster.Normal, MercFire, 100, 60},
		{"cold untouched by fire gear", fire30, true, d2monster.Normal, MercCold, 100, 90},
		{"poison 30", pois, true, d2monster.Normal, MercPoison, 100, 70},
		{"lightning table only", fire30, true, d2monster.Normal, MercLightning, 100, 90},
		{"cap 75", fire100, true, d2monster.Normal, MercFire, 100, 25},
		{"max resist +10 raises cap to 85", fire100max, true, d2monster.Normal, MercFire, 100, 15},
		{"nightmare LoD -40 (40 -> 0)", fire30, true, d2monster.Nightmare, MercFire, 100, 100},
		{"hell LoD -100 (40 -> -60)", fire30, true, d2monster.Hell, MercFire, 100, 160},
		{"hell classic -50 (40 -> -10)", fire30, false, d2monster.Hell, MercFire, 100, 110},
		{"penalty applies before the cap (110 - 40 = 70)", fire100, true, d2monster.Nightmare, MercFire, 100, 30},
		{"physical 80 capped at 50", phys80, true, d2monster.Normal, MercPhysical, 100, 50},
		{"physical exempt from the penalty", phys80, true, d2monster.Hell, MercPhysical, 100, 50},
		{"magic exempt from the penalty", nil, true, d2monster.Hell, MercMagic, 100, 100},
		{"no gear physical", nil, true, d2monster.Hell, MercPhysical, 100, 100},
		{"flat reduction", reduce, true, d2monster.Normal, MercPhysical, 100, 85},
		{"flat reduction larger than the damage is not floored here", reduce, true, d2monster.Normal, MercPhysical, 10, -5},
		{"flat reduction does not touch fire", reduce, true, d2monster.Normal, MercFire, 100, 90},
		{"zero damage", fire30, true, d2monster.Normal, MercFire, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, owner, _ := gearDirector(t)
			d.opt.Expansion, d.opt.Difficulty = tc.expansion, tc.diff

			if tc.items != nil {
				d.SetMercItems(owner, tc.items)
			}

			if got := d.MercReduce(owner, tc.typ, tc.dmg); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// The merc's weapon damage from its gear reaches the attack it makes.
func TestMercGearDamageReachesTheAttack(t *testing.T) {
	d, owner, u := gearDirector(t)
	plain := u.m.Vitals.A1

	d.SetMercItems(owner, testGear())

	if got := u.m.Vitals.A1; got.Max <= plain.Max || got.Min <= plain.Min {
		t.Errorf("attack %+v with a weapon, %+v without", got, plain)
	}

	if u.merc.stats.DmgMax != u.m.Vitals.A1.Max {
		t.Errorf("merc stats %d and attack %d differ", u.merc.stats.DmgMax, u.m.Vitals.A1.Max)
	}
}

func TestMercPotions(t *testing.T) {
	d, owner, u := gearDirector(t)
	u.m.Vitals.HP = u.m.Vitals.MaxHP / 4
	start := u.m.Vitals.HP

	// rejuvenation: instant percent of the maximum
	if !d.DrinkMerc(owner, d2inventory.PotionEffect{InstantHPPercent: 35, InstantManaPercent: 35}) {
		t.Fatal("rejuvenation did nothing")
	}

	if want := start + u.m.Vitals.MaxHP*35/100; u.m.Vitals.HP != want {
		t.Errorf("rejuvenation: hp %d, want %d", u.m.Vitals.HP, want)
	}

	// capped at the maximum
	d.DrinkMerc(owner, d2inventory.PotionEffect{InstantHPPercent: 100})

	if u.m.Vitals.HP != u.m.Vitals.MaxHP {
		t.Errorf("hp %d above or below the maximum %d", u.m.Vitals.HP, u.m.Vitals.MaxHP)
	}

	// healing over time: restored by the regen frames
	u.m.Vitals.HP = 1
	d.DrinkMerc(owner, d2inventory.PotionEffect{HP: 20, Seconds: 4})

	if u.m.Vitals.HP != 1 {
		t.Errorf("over-time part must not be instant, hp %d", u.m.Vitals.HP)
	}

	for i := 0; i < 25*5; i++ {
		u.merc.stepRegen(&u.m.Vitals)
	}

	if u.m.Vitals.HP != 21 {
		t.Errorf("after the potion hp %d, want 21", u.m.Vitals.HP)
	}

	// antidote / thawing: used up, no effect (the merc has no states)
	before := u.m.Vitals.HP
	if d.DrinkMerc(owner, d2inventory.PotionEffect{}) || u.m.Vitals.HP != before {
		t.Error("a potion with no life in it must do nothing")
	}

	// no merc: nothing happens (a dead merc is refused by the same check; Die needs the
	// animation composite, which a unit test does not have)
	if d.DrinkMerc(&d2mapentity.Player{}, d2inventory.PotionEffect{InstantHPPercent: 100}) {
		t.Error("a potion was given to a missing merc")
	}
}
