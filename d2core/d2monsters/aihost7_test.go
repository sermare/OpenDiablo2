package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestVitalsOfMonsterTargets(t *testing.T) {
	conv := petUnit(5, 0, 0, "Melee", "")
	conv.m.Vitals.HP, conv.m.Vitals.MaxHP = 30, 120

	merc := petUnit(6, 0, 0, "Melee", "")
	merc.m.Vitals.HP, merc.m.Vitals.MaxHP = 0, 50

	d := testDirector(conv, merc)

	for _, tc := range []struct {
		name string
		id   uint32
		want int
		maxP int
	}{
		{"converted monster", unitTargetBase + 5, 25, 120},
		{"mercenary at zero", mercTargetBase + 6, 0, 50},
		{"unknown unit", unitTargetBase + 99, 100, 0},
	} {
		tg := d2monster.Target{ID: tc.id}

		if got := d.LifePercent(tg); got != tc.want {
			t.Errorf("%s: life %d want %d", tc.name, got, tc.want)
		}

		if got := d.MaxHP(tg); got != tc.maxP {
			t.Errorf("%s: max hp %d want %d", tc.name, got, tc.maxP)
		}

		if got := d.MaxMana(tg); got != 0 {
			t.Errorf("%s: monsters carry no mana, got %d", tc.name, got)
		}
	}

	if d.HasStatListFlag(d2monster.Target{}, 0x20) {
		t.Error("no flagged stat lists are modelled")
	}
}

func TestSlotAuraState(t *testing.T) {
	rec := &d2records.RecordManager{}
	rec.Skill.Details = d2records.SkillDetails{
		1: {Skill: "Battle Cry", Aurastate: "battle_orders"},
		2: {Skill: "Plain Hit"},
	}
	rec.States = d2records.States{
		"battle_orders": {ID: 32, State: "battle_orders"},
	}

	d := &Director{asset: &d2asset.AssetManager{Records: rec}}

	p := &d2monster.Profile{}
	p.Skills[0] = d2monster.SkillSlot{Name: "Battle Cry"}
	p.Skills[1] = d2monster.SkillSlot{Name: "Plain Hit"}
	p.Skills[2] = d2monster.SkillSlot{Name: "Unknown"}
	b := &d2monster.Brain{Profile: p}

	for slot, want := range map[int]int{0: 32, 1: -1, 2: -1, 3: -1, -1: -1, 99: -1} {
		if got := d.SlotAuraState(b, slot); got != want {
			t.Errorf("slot %d: %d want %d", slot, got, want)
		}
	}
}
