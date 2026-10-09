package d2hireling

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

func gearBase() Stats {
	return Stats{Level: 20, MaxHP: 300, Str: 60, Dex: 50, Defense: 100, AR: 200, DmgMin: 5, DmgMax: 9, Resist: 20}
}

func TestApplyGearNoItemsIsIdentity(t *testing.T) {
	s := gearBase()
	g := ApplyGear(s, nil)

	want := Gear{Str: 60, Dex: 50, MaxHP: 300, Defense: 100, AR: 200, DmgMin: 5, DmgMax: 9, Resist: [4]int{20, 20, 20, 20},
		RawResist: [4]int{20, 20, 20, 20}}
	if g != want {
		t.Errorf("no gear: %+v, want %+v", g, want)
	}
}

func TestApplyGear(t *testing.T) {
	armor := d2statlist.Item{Slot: d2statlist.SlotTorso, Defense: 50, Props: []d2statlist.Prop{
		{ID: d2statlist.StatArmorPct, Value: 100}, {ID: d2statlist.StatFireResist, Value: 30}, {ID: d2statlist.StatStrength, Value: 5},
	}}
	helm := d2statlist.Item{Slot: d2statlist.SlotHead, Defense: 20, Props: []d2statlist.Prop{
		{ID: d2statlist.StatFireResist, Value: 30}, {ID: d2statlist.StatMaxHP, Value: 40},
	}}
	sword := d2statlist.Item{Slot: d2statlist.SlotRightHand, Weapon: &d2statlist.WeaponBase{Min: 10, Max: 20}, Props: []d2statlist.Prop{
		{ID: d2statlist.StatMinDmgPct, Value: 50}, {ID: d2statlist.StatMaxDmgPct, Value: 50}, {ID: d2statlist.StatMaxDamage, Value: 4},
	}}
	ring := d2statlist.Item{Slot: 6, Defense: 999, Props: []d2statlist.Prop{{ID: d2statlist.StatStrength, Value: 99}}}
	broken := d2statlist.Item{Slot: d2statlist.SlotLeftHand, Broken: true, Defense: 999}

	g := ApplyGear(gearBase(), []d2statlist.Item{armor, helm, sword, ring, broken})

	if g.Defense != 100+100+20 {
		t.Errorf("defense = %d, want 220 (a ring slot and a broken shield give nothing)", g.Defense)
	}

	if g.Str != 65 || g.MaxHP != 340 {
		t.Errorf("str=%d hp=%d", g.Str, g.MaxHP)
	}

	if g.DmgMin != 5+15 || g.DmgMax != 9+30+4 {
		t.Errorf("damage = %d-%d, want 20-43", g.DmgMin, g.DmgMax)
	}

	if g.Resist[d2statlist.ResFire] != 75 || g.Resist[d2statlist.ResCold] != 20 {
		t.Errorf("resists = %v (fire 20+60 capped at 75)", g.Resist)
	}
}
