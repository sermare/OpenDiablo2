package herogen

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// withWorn returns the spec with the item in a worn slot replaced (or added) by a unique.
func withWorn(spec Spec, slot uint8, unique string) Spec {
	items := make([]ItemSpec, 0, len(spec.Items)+1)
	done := false

	for _, it := range spec.Items {
		if it.Place.Location == d2s.LocationEquipped && it.Place.Slot == slot {
			done = true

			if unique != "" {
				it.Unique, it.Code = unique, ""
				items = append(items, it)
			}

			continue
		}

		items = append(items, it)
	}

	if !done && unique != "" {
		items = append(items, ItemSpec{Unique: unique, Place: Worn(slot)})
	}

	spec.Items = items

	return spec
}

// TestGenerateRefusesIllegalGear proves the equip rules reject what a class cannot wear or the attributes
// do not reach: each case is a preset with one wrong item or too few points in a requirement.
func TestGenerateRefusesIllegalGear(t *testing.T) {
	tb := realTables(t)

	more := func(vit int, f func(*Spec)) func(Spec) Spec {
		return func(s Spec) Spec {
			f(&s)
			s.Vitality += vit

			return s
		}
	}

	tests := []struct {
		name   string
		class  d2s.Class
		mutate func(Spec) Spec
		want   string
	}{
		{"primal helm on a Sorceress", d2s.Sorceress, func(s Spec) Spec { return withWorn(s, SlotHead, "Arreat's Face") }, "another class"},
		{"orb on a Barbarian", d2s.Barbarian, func(s Spec) Spec { return withWorn(s, SlotRightHand, "Eschuta's temper") }, "another class"},
		{"auric shield on an Assassin", d2s.Assassin, func(s Spec) Spec { return withWorn(s, SlotLeftHand, "Alma Negra") }, "another class"},
		{"voodoo head on a Paladin", d2s.Paladin, func(s Spec) Spec { return withWorn(s, SlotLeftHand, "Boneflame") }, "another class"},
		{"pelt on an Amazon", d2s.Amazon, func(s Spec) Spec { return withWorn(s, SlotHead, "Jalal's Mane") }, "another class"},
		{"javelin on a Necromancer", d2s.Necromancer, func(s Spec) Spec { return withWorn(s, SlotRightHand, "Titan's Revenge") }, "another class"},
		{"claws on a Druid", d2s.Druid, func(s Spec) Spec { return withWorn(s, SlotRightHand, "Jadetalon") }, "another class"},
		{"a second weapon on a Paladin", d2s.Paladin, func(s Spec) Spec { return withWorn(s, SlotLeftHand, "Zakarum's Hand") }, "off hand"},
		{"a shield next to a two hand staff", d2s.Druid, func(s Spec) Spec { return withWorn(s, SlotLeftHand, "Lidless Wall") }, "pushes out"},
		{"too little dexterity for Titan's Revenge", d2s.Amazon, more(10, func(s *Spec) { s.Dexterity = 100 }), "dexterity"},
		{"too little strength for Alma Negra", d2s.Paladin, more(10, func(s *Spec) { s.Strength = 100 }), "strength"},
		{"belt cells beyond a Sash", d2s.Barbarian, func(s Spec) Spec { return withWorn(s, SlotBelt, "Lenyms Cord") }, "belt cell"},
	}

	for _, tt := range tests {
		spec, err := Preset(tt.class, DefaultName(tt.class))
		if err != nil {
			t.Fatal(err)
		}

		_, err = tb.Generate(tt.mutate(spec), nil)
		if err == nil {
			t.Errorf("%s: accepted", tt.name)
			continue
		}

		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}
}

// TestThrownWeaponIsAStack proves the javelin of the Amazon carries a stack (a thrown weapon is its own ammunition).
func TestThrownWeaponIsAStack(t *testing.T) {
	tb := realTables(t)

	hero, err := tb.Generate(Amazon("NokkaAma"), nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := range hero.Character.Items {
		it := &hero.Character.Items[i]
		if it.Location == d2s.LocationEquipped && it.Equipped == SlotRightHand {
			if strings.TrimSpace(it.Code) != "ama" || it.Quantity < 20 {
				t.Errorf("right hand %q with %d javelins, want Titan's Revenge with a stack of at least 20", it.Code, it.Quantity)
			}

			return
		}
	}

	t.Error("no weapon in the right hand")
}
