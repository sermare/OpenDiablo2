package d2object

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

func shrine(t *testing.T, code int) Shrine {
	t.Helper()

	s, ok := ShrineByCode(DefaultShrines, code)
	if !ok {
		t.Fatalf("no shrine %d", code)
	}

	return s
}

func TestShrineTimes(t *testing.T) {
	tests := []struct {
		code      int
		seconds   float64
		resetSecs float64
	}{
		{ShrineArmor, 96, 5*0x4b0 + 1},
		{ShrineCombat, 96, 5*0x4b0 + 1},
		{ShrineResistFire, 144, 5*0x4b0 + 1},
		{ShrineStamina, 192, 5*0x4b0 + 1},
		{ShrineRefill, 0, 2*0x4b0 + 1},
		{ShrineStorm, 0, 0},
	}

	for _, tc := range tests {
		s := shrine(t, tc.code)
		if got := s.DurationSeconds(); got != tc.seconds {
			t.Errorf("%s: duration %v want %v", s.Name, got, tc.seconds)
		}

		if got := s.ResetSeconds() * FramesPerSecond; got != tc.resetSecs {
			t.Errorf("%s: reset frames %v want %v", s.Name, got, tc.resetSecs)
		}
	}
}

func TestInstant(t *testing.T) {
	v := Vitals{Life: 30, MaxLife: 100, Mana: 10, MaxMana: 50}

	tests := []struct {
		code int
		want Vitals
	}{
		{ShrineRefill, Vitals{100, 100, 50, 50}},
		{ShrineHealth, Vitals{60, 100, 10, 50}},
		{ShrineMana, Vitals{30, 100, 20, 50}},
		{ShrineHealthExchange, Vitals{15, 100, 50, 50}}, // 15 taken * 5 = 75 mana, capped at 50
		{ShrineManaExchange, Vitals{100, 100, 5, 50}},   // 5 taken * 5 = 25 -> 55 capped... 30+25
	}

	tests[4].want.Life = 55

	for _, tc := range tests {
		got, ok := ApplyInstant(shrine(t, tc.code), v)
		if !ok || got != tc.want {
			t.Errorf("code %d: got %+v ok=%v want %+v", tc.code, got, ok, tc.want)
		}
	}

	if _, ok := ApplyInstant(shrine(t, ShrineArmor), v); ok {
		t.Error("armor shrine is not an instant effect")
	}
}

func TestBuffsAndOverlay(t *testing.T) {
	tot := &d2statlist.Totals{Defense: 200, AttackRating: 100, DamageMin: 10, DamageMax: 20}
	tot.Resist[d2statlist.ResFire] = 10
	tot.ResistShown[d2statlist.ResFire] = 10
	tot.MaxResist[d2statlist.ResFire] = 75

	var o Overlay

	armor, _ := NewShrineBuff(shrine(t, ShrineArmor), 100)
	fire, _ := NewShrineBuff(shrine(t, ShrineResistFire), 100)
	combat, _ := NewShrineBuff(shrine(t, ShrineCombat), 100)
	o.Buffs.Add(armor)
	o.Buffs.Add(fire)
	o.Buffs.Add(combat)

	o.Apply(tot, 101)

	if tot.Defense != 400 || tot.AttackRating != 300 || tot.DamageMin != 30 || tot.DamageMax != 60 {
		t.Errorf("combat stats %+v", *tot)
	}

	if tot.Resist[0] != 85 || tot.ResistShown[0] != 75 {
		t.Errorf("fire resist %d shown %d", tot.Resist[0], tot.ResistShown[0])
	}

	// applying twice does not stack
	o.Apply(tot, 102)

	if tot.Defense != 400 {
		t.Errorf("stacked: %d", tot.Defense)
	}

	// armor and combat end at 100+96
	gone := o.Apply(tot, 197)
	if len(gone) != 2 || gone[0].State != StateShrineArmor || tot.Defense != 200 || tot.AttackRating != 100 {
		t.Errorf("after armor expiry: gone=%v defense=%d", gone, tot.Defense)
	}

	// a recalculation replaces the totals: buffs follow
	fresh := &d2statlist.Totals{Defense: 300}
	o.Apply(fresh, 198)

	if fresh.Defense != 300 || fresh.AttackRating != 0 || fresh.Resist[0] != 75 {
		t.Errorf("fresh totals %+v", *fresh)
	}

	// refreshing a state replaces it
	armor2, _ := NewShrineBuff(shrine(t, ShrineArmor), 200)
	o.Buffs.Add(armor2)
	o.Buffs.Add(armor2)

	n := 0

	for _, b := range o.Buffs.Active() {
		if b.State == StateShrineArmor {
			n++
		}
	}

	if n != 1 {
		t.Errorf("armor buffs: %d", n)
	}
}

func TestOverlayFlags(t *testing.T) {
	var o Overlay

	for _, c := range []int{ShrineExperience, ShrineStamina, ShrineManaRecharge} {
		b, ok := NewShrineBuff(shrine(t, c), 0)
		if !ok {
			t.Fatalf("no buff for %d", c)
		}

		o.Buffs.Add(b)
	}

	if o.ExperiencePct() != 50 || !o.UnlimitedStamina() || o.ManaRecoveryPct() != 400 {
		t.Errorf("exp=%d stamina=%v mana=%d", o.ExperiencePct(), o.UnlimitedStamina(), o.ManaRecoveryPct())
	}
}

func TestRollShrine(t *testing.T) {
	for seed := uint32(0); seed < 200; seed++ {
		r := NewRoller(seed)

		s, ok := RollShrine(DefaultShrines, r, 1, 3, false)
		if !ok {
			t.Fatal("no shrine at level 1")
		}

		if s.LevelMin > 1 || s.Out || s.EffectClass != 4 {
			t.Fatalf("seed %d: %+v", seed, s)
		}
	}

	// deterministic
	a, _ := RollShrine(DefaultShrines, NewRoller(7), 30, 0, false)
	b, _ := RollShrine(DefaultShrines, NewRoller(7), 30, 0, false)

	if a.Code != b.Code {
		t.Error("roll not deterministic")
	}

	// every non-cut shrine appears at a high level
	seen := map[int]bool{}

	for seed := uint32(0); seed < 3000; seed++ {
		s, _ := RollShrine(DefaultShrines, NewRoller(seed), 40, 0, false)
		seen[s.Code] = true
	}

	for _, s := range DefaultShrines {
		if s.Code != ShrineNone && !s.Out && !seen[s.Code] {
			t.Errorf("%s never rolled", s.Name)
		}
	}
}

func TestLookup(t *testing.T) {
	tests := []struct {
		fn    int
		class Class
	}{
		{2, ClassShrine}, {4, ClassLoot}, {5, ClassLoot}, {15, ClassPortal}, {20, ClassRack}, {22, ClassWell},
		{23, ClassWaypoint}, {10, ClassQuest}, {500, ClassQuest},
	}

	for _, tc := range tests {
		if got := Lookup(tc.fn).Class; got != tc.class {
			t.Errorf("fn %d: %v want %v", tc.fn, got, tc.class)
		}
	}

	if SoundFor(5, "Barrel") != "object_wood_break_1" || SoundFor(4, "chest") != "object_chest_large" {
		t.Error("sound choice")
	}
}

// TestRealShrines compares the embedded table with the extracted file when
// D2_TABLES is set (objects/shrines.txt from d2exp.mpq).
func TestRealShrines(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "objects", "shrines.txt"))
	if err != nil {
		t.Skip(err)
	}

	rows, err := ParseShrines(data)
	if err != nil {
		t.Fatal(err)
	}

	if len(rows) != len(DefaultShrines) {
		t.Fatalf("rows %d, embedded %d", len(rows), len(DefaultShrines))
	}

	for i, r := range rows {
		d := DefaultShrines[i]
		if r.Code != d.Code || r.Arg0 != d.Arg0 || r.Arg1 != d.Arg1 || r.DurationFrame != d.DurationFrame ||
			r.ResetMinutes != d.ResetMinutes || r.Rarity != d.Rarity || r.EffectClass != d.EffectClass ||
			r.LevelMin != d.LevelMin || r.Out != d.Out {
			t.Errorf("row %d differs:\n file %+v\n embed %+v", i, r, d)
		}
	}
}
