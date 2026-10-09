package d2inventory

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calculation"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestBeltRows(t *testing.T) {
	for boxes, want := range map[int]int{0: 1, 4: 1, 8: 2, 12: 3, 16: 4, 20: 4} {
		if got := BeltRows(boxes); got != want {
			t.Errorf("BeltRows(%d) = %d, want %d", boxes, got, want)
		}
	}
}

func TestFindBeltSlot(t *testing.T) {
	tests := []struct {
		name  string
		kinds map[int]string
		boxes int
		kind  string
		want  int
		ok    bool
	}{
		{"empty belt takes the first front cell", nil, 16, "hpot", 0, true},
		{"new kind goes to the next free front cell", map[int]string{0: "hpot"}, 16, "mpot", 1, true},
		{"same kind stacks up its column", map[int]string{0: "hpot", 1: "mpot"}, 16, "hpot", 4, true},
		{"second of the same kind keeps climbing", map[int]string{0: "hpot", 4: "hpot"}, 16, "hpot", 8, true},
		{"mana column", map[int]string{0: "hpot", 1: "mpot", 4: "hpot"}, 16, "mpot", 5, true},
		{"full matching column falls to a free front cell", map[int]string{0: "hpot", 4: "hpot"}, 8, "hpot", 1, true},
		{"front row full, no match", map[int]string{0: "a", 1: "b", 2: "c", 3: "d"}, 16, "e", 0, false},
		{"belt size limits rows", map[int]string{0: "hpot", 4: "hpot"}, 8, "hpot", 1, true},
		{"no belt: four cells", map[int]string{0: "a", 1: "b", 2: "c", 3: "d"}, 4, "a", 0, false},
	}

	for _, tt := range tests {
		var k BeltKinds
		for c, v := range tt.kinds {
			k[c] = v
		}

		got, ok := FindBeltSlot(&k, tt.boxes, tt.kind)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("%s: got (%d,%v) want (%d,%v)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

// TestBeltBoxesAndSearchVerified pins Belts.txt (7 rows: belt, sash, default,
// girdle, light belt, heavy belt, uber belt) and the 0x63d700 behaviour.
func TestBeltBoxesAndSearchVerified(t *testing.T) {
	want := map[int]int{0: 12, 1: 8, 2: 4, 3: 16, 4: 8, 5: 12, 6: 16, 9: 4, -1: 4}
	for typ, n := range want {
		if got := BeltBoxes(typ); got != n {
			t.Errorf("BeltBoxes(%d)=%d want %d", typ, got, n)
		}
	}

	// a full matching column continues with the next matching column
	var k BeltKinds
	k[0], k[4] = "hpot", "hpot"
	k[1], k[5] = "hpot", ""

	if c, ok := FindBeltSlotOpt(&k, 8, "hpot", false); !ok || c != 5 {
		t.Errorf("second matching column: got (%d,%v)", c, ok)
	}

	// no match and no beltable flag: no slot even though the front row has room
	var e BeltKinds
	e[0] = "hpot"

	if _, ok := FindBeltSlotOpt(&e, 16, "mpot", false); ok {
		t.Error("without the fallback flag a new kind finds no slot")
	}
}

func TestFrontOfColumnAndCompact(t *testing.T) {
	var k BeltKinds
	k[1], k[5], k[13] = "hpot", "hpot", "hpot" // column 1, rows 0, 1, 3

	if c, ok := FrontOfColumn(&k, 16, 1); !ok || c != 1 {
		t.Fatalf("front = %d %v", c, ok)
	}

	if _, ok := FrontOfColumn(&k, 16, 0); ok {
		t.Error("empty column has no front")
	}

	if _, ok := FrontOfColumn(&k, 16, 7); ok {
		t.Error("column 7 does not exist")
	}

	// drink the front one, the others move down
	k[1] = ""
	moves := CompactColumn(&k, 16, 1)

	if k[1] != "hpot" || k[5] != "hpot" || k[13] != "" || len(moves) != 2 || moves[0] != [2]int{5, 1} || moves[1] != [2]int{13, 5} {
		t.Errorf("compact = %v moves %v", k, moves)
	}

	// a belt of 8 ignores cells beyond its size
	var small BeltKinds
	small[9] = "x"

	if _, ok := FrontOfColumn(&small, 8, 1); ok {
		t.Error("cell 9 is outside an 8 box belt")
	}
}

func potion(spell, frames int, stats ...string) *d2records.ItemCommonRecord {
	r := &d2records.ItemCommonRecord{Useable: true, SpellType: spell, EffectLength: frames}
	for i := 0; i+1 < len(stats) && i/2 < 3; i += 2 {
		r.UsageStats[i/2] = d2records.ItemUsageStat{Stat: stats[i], Calc: d2calculation.CalcString(stats[i+1])}
	}

	return r
}

func TestPotionEffects(t *testing.T) {
	heal := PotionEffectOf(potion(3, 256, "hpregen", "320"))
	if heal.HP != 320 || heal.Mana != 0 || math.Abs(heal.Seconds-10.24) > 1e-9 {
		t.Errorf("greater healing = %+v", heal)
	}

	mana := PotionEffectOf(potion(3, 128, "manarecovery", "250"))
	if mana.Mana != 250 || mana.HP != 0 || math.Abs(mana.Seconds-5.12) > 1e-9 {
		t.Errorf("greater mana = %+v", mana)
	}

	rejuv := PotionEffectOf(potion(5, 0, "hitpoints", "35", "mana", "35"))
	if rejuv.InstantHPPercent != 35 || rejuv.InstantManaPercent != 35 || rejuv.HP != 0 {
		t.Errorf("rejuvenation = %+v", rejuv)
	}

	if !PotionEffectOf(potion(1, 0, "hitpoints", "9")).IsEmpty() || !PotionEffectOf(nil).IsEmpty() {
		t.Error("other spells have no effect")
	}

	notUsable := potion(3, 256, "hpregen", "320")
	notUsable.Useable = false

	if !PotionEffectOf(notUsable).IsEmpty() {
		t.Error("an item that is not useable does nothing")
	}
}

func TestRegenSpreadsAndFinishes(t *testing.T) {
	var r Regen

	r.Add(PotionEffect{HP: 100, Seconds: 10})

	if !r.Active() {
		t.Fatal("regen should be active")
	}

	hp, mana := r.Tick(1)
	if math.Abs(hp-10) > 1e-9 || mana != 0 {
		t.Errorf("first second restored %v %v", hp, mana)
	}

	total := hp

	for i := 0; i < 20 && r.Active(); i++ {
		h, _ := r.Tick(1)
		total += h
	}

	if math.Abs(total-100) > 1e-9 || r.Active() {
		t.Errorf("total restored %v, active=%v", total, r.Active())
	}

	if h, _ := r.Tick(5); h != 0 {
		t.Error("finished regen keeps restoring")
	}

	// two potions in a row: both amounts arrive, none is lost
	var two Regen

	two.Add(PotionEffect{HP: 30, Seconds: 3})
	two.Add(PotionEffect{HP: 60, Seconds: 6})

	sum := 0.0

	for i := 0; i < 30; i++ {
		h, _ := two.Tick(0.5)
		sum += h
	}

	if math.Abs(sum-90) > 1e-6 {
		t.Errorf("two potions restored %v of 90", sum)
	}
}

func TestApplyInstant(t *testing.T) {
	if got := ApplyInstant(10, 200, 35); got != 80 {
		t.Errorf("35%% of 200 on 10 = %d", got)
	}

	if got := ApplyInstant(150, 200, 100); got != 200 {
		t.Errorf("clamped = %d", got)
	}
}
