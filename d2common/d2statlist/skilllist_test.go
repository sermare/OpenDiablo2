package d2statlist

import "testing"

func TestComputeSkillList(t *testing.T) {
	h := hero()
	armor := Item{Slot: SlotTorso, Defense: 100}
	sk := NewList()
	sk.Add(StatSkillArmorPct, 0, 100) // Shout L1
	sk.Add(StatMaxHPPct, 0, 35)       // Battle Orders L1
	sk.Add(StatSkillStaminaPct, 0, 35)
	sk.Add(StatFireResist, 0, 15)

	base := Compute(h, []Item{armor}, nil)
	got := Compute(h, []Item{armor}, &Env{Skill: sk})

	if want := 2 * base.Defense; got.Defense != want {
		t.Errorf("defense %d, want %d", got.Defense, want)
	}

	if want := base.MaxLife * 135 / 100; got.MaxLife < want-1 || got.MaxLife > want+1 {
		t.Errorf("life %d, want about %d", got.MaxLife, want)
	}

	if want := base.MaxStamina * 135 / 100; got.MaxStamina < want-1 || got.MaxStamina > want+1 {
		t.Errorf("stamina %d, want about %d", got.MaxStamina, want)
	}

	if got.Resist[ResFire] != 15 {
		t.Errorf("fire resist %d", got.Resist[ResFire])
	}
}
