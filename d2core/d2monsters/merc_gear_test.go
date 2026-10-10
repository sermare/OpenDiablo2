package d2monsters

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

const gearMini = "Hireling\tSubType\tId\tClass\tAct\tDifficulty\tLevel\tSeller\tNameFirst\tNameLast\tGold\tExp/Lvl\tHP\tHP/Lvl\t" +
	"Defense\tDef/Lvl\tStr\tStr/Lvl\tDex\tDex/Lvl\tAR\tAR/Lvl\tDmg-Min\tDmg-Max\tDmg/Lvl\tResist\tResist/Lvl\tDefaultChance\t" +
	"Skill1\tMode1\tChance1\tChancePerLevel1\tLevel1\tLvlPerLvl1\n" +
	"R\tFire\t0\t271\t1\t1\t3\t150\tmerc01\tmerc41\t100\t100\t45\t8\t20\t3\t30\t8\t40\t8\t50\t4\t2\t5\t8\t10\t4\t75\tInner Sight\t4\t10\t0\t1\t10\n"

// gearDirector is a Director with one merc unit and no map: the gear logic only touches the
// unit's vitals and the hireling table (spawning needs the game data's animations).
func gearDirector(t *testing.T) (*Director, *d2mapentity.Player, *unit) {
	t.Helper()

	tab, err := d2hireling.Parse(strings.NewReader(gearMini))
	if err != nil {
		t.Fatal(err)
	}

	owner := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Level: 40}}
	d := &Director{Logger: d2util.NewLogger(), hire: tab, mercs: map[*d2mapentity.Player]*unit{}}

	st, rec := tab.StatsFor(0, 3)
	if rec == nil {
		t.Fatal("no hireling row")
	}

	m := &d2mapentity.Monster{}
	m.Vitals = d2mapentity.MonsterVitals{Level: 3, HP: st.MaxHP, MaxHP: st.MaxHP, Defense: st.Defense,
		A1: MonsterAttackFrom(st.AR, st.DmgMin, st.DmgMax)}

	mu := &mercUnit{owner: owner, rec: rec, level: 3, stats: st, base: st, save: MercSave{Type: 0}}
	u := &unit{m: m, b: &d2monster.Brain{}, merc: mu}
	d.mercs[owner] = u

	return d, owner, u
}

func testGear() []d2statlist.Item {
	return []d2statlist.Item{
		{Slot: d2statlist.SlotTorso, Defense: 40, Props: []d2statlist.Prop{{ID: d2statlist.StatFireResist, Value: 30}}},
		{Slot: d2statlist.SlotHead, Defense: 10, Props: []d2statlist.Prop{{ID: d2statlist.StatMaxHP, Value: 25}}},
		{Slot: d2statlist.SlotRightHand, Weapon: &d2statlist.WeaponBase{Min: 3, Max: 6}},
	}
}

func TestMercGearShowsInStatsAndSurvivesLevelUp(t *testing.T) {
	d, owner, u := gearDirector(t)

	plain, _ := d.Merc(owner)

	if !d.SetMercItems(owner, nil) {
		t.Fatal("SetMercItems failed")
	}

	if none, _ := d.Merc(owner); none.Stats != plain.Stats || none.HP != plain.HP || none.MaxHP != plain.MaxHP {
		t.Errorf("no gear must change nothing: %+v vs %+v", none, plain)
	}

	d.SetMercItems(owner, testGear())

	got, _ := d.Merc(owner)
	base := plain.Stats

	if got.Stats.Defense != base.Defense+50 || got.MaxHP != base.MaxHP+25 {
		t.Errorf("defense/life: %d/%d, want %d/%d", got.Stats.Defense, got.MaxHP, base.Defense+50, base.MaxHP+25)
	}

	if got.Stats.DmgMin != base.DmgMin+3 || got.Stats.DmgMax != base.DmgMax+6 {
		t.Errorf("damage %d-%d, want %d-%d", got.Stats.DmgMin, got.Stats.DmgMax, base.DmgMin+3, base.DmgMax+6)
	}

	if got.Gear.Resist[d2statlist.ResFire] != base.Resist+30 || got.Gear.Resist[d2statlist.ResCold] != base.Resist {
		t.Errorf("resists %v, want fire %d cold %d", got.Gear.Resist, base.Resist+30, base.Resist)
	}

	if u.m.Vitals.Defense != got.Stats.Defense || u.m.Vitals.MaxHP != got.MaxHP {
		t.Errorf("the unit's vitals did not follow: %+v", u.m.Vitals)
	}

	// a level-up recomputes the table stats: the gear stays on top of the new level
	before := got.Stats.Defense
	d.creditMerc(u.merc, u, 5000) // 10000 exp: level 4 (8000), not 5 (15000)

	up, _ := d.Merc(owner)
	if up.Level != 4 {
		t.Fatalf("level = %d, want 4", up.Level)
	}

	if want := up.Base.Defense + 50; up.Stats.Defense != want || up.Stats.Defense <= before {
		t.Errorf("defense after level-up = %d, want table %d + 50", up.Stats.Defense, up.Base.Defense)
	}

	if up.HP != up.MaxHP || up.MaxHP != up.Base.MaxHP+25 {
		t.Errorf("level-up life %d/%d, want full %d", up.HP, up.MaxHP, up.Base.MaxHP+25)
	}

	// taking the gear off restores the table values of the new level
	d.SetMercItems(owner, nil)

	if off, _ := d.Merc(owner); off.Stats.Defense != off.Base.Defense || off.MaxHP != off.Base.MaxHP {
		t.Errorf("gear off: %+v", off)
	}
}

func TestMercGearKeepsLifeProportion(t *testing.T) {
	d, owner, u := gearDirector(t)

	oldMax := u.m.Vitals.MaxHP
	u.m.Vitals.HP = oldMax / 2
	frac := float64(u.m.Vitals.HP) / float64(oldMax)
	d.SetMercItems(owner, testGear())

	if want := int(frac * float64(u.m.Vitals.MaxHP)); u.m.Vitals.HP != want {
		t.Errorf("life %d/%d, want %d (same proportion)", u.m.Vitals.HP, u.m.Vitals.MaxHP, want)
	}

	// SetMercGear (the raw form) still works and is the same arithmetic
	g := d2hireling.ApplyGear(u.merc.base, testGear())
	if !d.SetMercGear(owner, g) || u.merc.gear.Defense != g.Defense {
		t.Error("SetMercGear")
	}
}
