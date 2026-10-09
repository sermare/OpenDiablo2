package d2combat

import (
	"testing"
	"unsafe"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// fakeRoller returns queued values and counts the steps consumed.
type fakeRoller struct {
	vals  []uint32
	calls int
	ns    []int32
}

func (f *fakeRoller) Roll(n int32) uint32 {
	f.ns = append(f.ns, n)
	v := uint32(0)

	if f.calls < len(f.vals) {
		v = f.vals[f.calls]
	}

	f.calls++

	return v
}

func TestManaCost(t *testing.T) {
	tests := []struct {
		name                          string
		mana, lvlmana, minmana, shift int16
		level                         int
		want                          int
	}{
		{"level 1", 3, 1, 1, 8, 1, 768},
		{"level 5", 3, 1, 1, 8, 5, 1792},
		{"level 0 clamps to base", 3, 1, 1, 8, 0, 768},
		{"negative level clamps", 3, 1, 1, 8, -4, 768},
		{"free skill ignores minmana", 0, 0, 5, 8, 10, 0},
		{"minmana floor", 1, 0, 2, 8, 1, 512},
		{"shift 7", 3, 1, 0, 7, 3, 640},
		{"shift masked to 5 bits", 3, 0, 0, 40, 1, 768},
		{"only lvlmana, floor applies", 0, 2, 1, 8, 1, 256},
		{"only lvlmana grows", 0, 2, 1, 8, 4, 6 << 8},
	}

	for _, tt := range tests {
		if got := ManaCost(tt.mana, tt.lvlmana, tt.minmana, tt.shift, tt.level); got != tt.want {
			t.Errorf("%s: ManaCost = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestPayMana(t *testing.T) {
	if rem, ok := PayMana(1000, 768); !ok || rem != 232 {
		t.Errorf("pay: got %d,%v", rem, ok)
	}

	if rem, ok := PayMana(768, 768); !ok || rem != 0 {
		t.Errorf("exact: got %d,%v", rem, ok)
	}

	if rem, ok := PayMana(767, 768); ok || rem != 767 {
		t.Errorf("short: got %d,%v", rem, ok)
	}
}

func TestDefense(t *testing.T) {
	tests := []struct{ ac, dex, pct, want int }{
		{100, 20, 0, 105},
		{100, 20, 50, 157}, // 105 + 52
		{100, 3, 0, 100},   // dex/4 truncates
		{-10, 0, 50, -5}, // verified in 0x6225a0: for base <= 0 the bonus is subtracted
		{-10, 4, 10, -9}, // -9 + trunc(-0.9) = -9
		{0, 0, 100, 0},
	}

	for _, tt := range tests {
		if got := Defense(tt.ac, tt.dex, tt.pct); got != tt.want {
			t.Errorf("Defense(%d,%d,%d) = %d, want %d", tt.ac, tt.dex, tt.pct, got, tt.want)
		}
	}
}

func TestAttackRating(t *testing.T) {
	if got := PlayerAttackRating(0, 7, 0); got != 0 {
		t.Errorf("got %d", got)
	}

	if got := PlayerAttackRating(10, 17, 30); got != 90 {
		t.Errorf("got %d", got)
	}

	if got := MonsterAttackRating(50, 20, 10); got != 120 {
		t.Errorf("got %d", got)
	}
}

func TestToHitChance(t *testing.T) {
	tests := []struct {
		name string
		in   ToHitInput
		want int
	}{
		{"even", ToHitInput{100, 100, 10, 10, 0}, 50},
		{"stronger attacker, higher level", ToHitInput{200, 100, 20, 10, 0}, 88},
		{"AR 300 vs 100, levels 30/10 clamps high", ToHitInput{300, 100, 30, 10, 0}, 95},
		{"AR 300 vs 100, levels 10/30", ToHitInput{300, 100, 10, 30, 0}, 37},
		{"zero AR clamps low", ToHitInput{0, 100, 10, 10, 0}, 5},
		{"zero DEF clamps high", ToHitInput{1000, 0, 10, 10, 0}, 95},
		{"both zero is 100 then 95", ToHitInput{0, 0, 10, 10, 0}, 95},
		{"negative DEF moves into AR", ToHitInput{100, -50, 10, 10, 0}, 95},
		{"negative AR moves into DEF", ToHitInput{-30, 50, 10, 10, 0}, 5},
		{"attacker level 0", ToHitInput{100, 100, 0, 10, 0}, 5},
		{"both levels 0 skip level factor", ToHitInput{100, 100, 0, 0, 0}, 50},
		{"AR percent", ToHitInput{100, 150, 10, 10, 50}, 50},
		{"integer truncation", ToHitInput{1, 2, 1, 1, 0}, 33},
		{"exactly 6 stays 6", ToHitInput{6, 94, 1, 1, 0}, 6},
		{"exactly 94 stays 94", ToHitInput{94, 6, 1, 1, 0}, 94},
		{"95 stays 95", ToHitInput{95, 5, 1, 1, 0}, 95},
	}

	for _, tt := range tests {
		if got := ToHitChance(tt.in); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestRollToHitBoundary(t *testing.T) {
	in := ToHitInput{100, 100, 10, 10, 0} // chance 50

	f := &fakeRoller{vals: []uint32{49}}
	if hit, c, roll := RollToHit(f, in); !hit || c != 50 || roll != 49 || f.calls != 1 || f.ns[0] != 100 {
		t.Errorf("49: hit=%v c=%d roll=%d calls=%d", hit, c, roll, f.calls)
	}

	f = &fakeRoller{vals: []uint32{50}}
	if hit, _, _ := RollToHit(f, in); hit || f.calls != 1 {
		t.Errorf("50 must miss, calls=%d", f.calls)
	}

	// clamped chances still consume a step and respect 5 / 95
	f = &fakeRoller{vals: []uint32{4}}
	if hit, c, _ := RollToHit(f, ToHitInput{0, 100, 1, 1, 0}); !hit || c != 5 || f.calls != 1 {
		t.Errorf("clamp low: hit=%v c=%d", hit, c)
	}

	f = &fakeRoller{vals: []uint32{95}}
	if hit, c, _ := RollToHit(f, ToHitInput{1000, 0, 1, 1, 0}); hit || c != 95 {
		t.Errorf("clamp high: hit=%v c=%d", hit, c)
	}
}

func TestRollToHitSeeded(t *testing.T) {
	in := ToHitInput{100, 100, 10, 10, 0}
	s := d2rand.New(42)
	want := int(d2rand.New(42).Roll(100))

	_, _, roll := RollToHit(s, in)
	if roll != want {
		t.Errorf("roll = %d, want %d", roll, want)
	}

	ref := d2rand.New(42)
	ref.Step()

	if s.Lo != ref.Lo || s.Hi != ref.Hi {
		t.Error("RollToHit must consume exactly one generator step")
	}
}

func TestBlockChance(t *testing.T) {
	tests := []struct {
		name string
		in   PlayerBlockInput
		want int
	}{
		{"no shield", PlayerBlockInput{false, 30, 20, 100, 10, true}, 0},
		{"no dex term", PlayerBlockInput{true, 30, 20, 100, 10, false}, 50},
		{"dex 15 gives zero", PlayerBlockInput{true, 30, 20, 15, 10, true}, 0},
		{"dex 65 lvl 25", PlayerBlockInput{true, 30, 20, 65, 25, true}, 50},
		{"cap 75", PlayerBlockInput{true, 30, 20, 100, 1, true}, 75},
		{"level 0 acts as 1", PlayerBlockInput{true, 30, 20, 100, 0, true}, 75},
		{"cap without dex", PlayerBlockInput{true, 80, 0, 0, 1, false}, 75},
		{"low dex is negative, truncated", PlayerBlockInput{true, 30, 20, 10, 10, true}, -12},
	}

	for _, tt := range tests {
		if got := PlayerBlockChance(tt.in); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}

	if MonsterBlockChance(false, 50) != 0 || MonsterBlockChance(true, 50) != 50 || MonsterBlockChance(true, 90) != 75 {
		t.Error("monster block chance")
	}
}

func TestRollShieldBlock(t *testing.T) {
	f := &fakeRoller{}
	if RollShieldBlock(f, 0, false) || RollShieldBlock(f, -12, false) || f.calls != 0 {
		t.Error("chance < 1 must not block nor consume a step")
	}

	f = &fakeRoller{vals: []uint32{59}}
	if !RollShieldBlock(f, 60, false) {
		t.Error("59 < 60 should block")
	}

	f = &fakeRoller{vals: []uint32{60}}
	if RollShieldBlock(f, 60, false) {
		t.Error("60 should not block")
	}

	// moving: 60/3 = 20
	f = &fakeRoller{vals: []uint32{19}}
	if !RollShieldBlock(f, 60, true) {
		t.Error("19 < 20 should block while moving")
	}

	f = &fakeRoller{vals: []uint32{20}}
	if RollShieldBlock(f, 60, true) {
		t.Error("20 should not block while moving")
	}

	// moving with chance 2 -> 0, still consumes a step, never blocks
	f = &fakeRoller{vals: []uint32{0}}
	if RollShieldBlock(f, 2, true) || f.calls != 1 {
		t.Errorf("chance 2/3=0: calls=%d", f.calls)
	}
}

func TestRollAvoid(t *testing.T) {
	tests := []struct {
		name  string
		in    AvoidInput
		vals  []uint32
		want  AvoidOutcome
		calls int
	}{
		{"moving evade success", AvoidInput{Moving: true, EvadeChance: 30, DodgeChance: 90}, []uint32{29}, AvoidEvaded, 1},
		{"moving evade fail skips dodge", AvoidInput{Moving: true, EvadeChance: 30, DodgeChance: 90}, []uint32{30, 0}, AvoidNone, 1},
		{"moving no evade, no roll", AvoidInput{Moving: true, DodgeChance: 90}, nil, AvoidNone, 0},
		{"melee dodge", AvoidInput{DodgeChance: 40, AvoidChance: 90}, []uint32{39}, AvoidDodged, 1},
		{"melee dodge miss", AvoidInput{DodgeChance: 40}, []uint32{40}, AvoidNone, 1},
		{"missile avoid", AvoidInput{IsMissile: true, DodgeChance: 90, AvoidChance: 40}, []uint32{0}, AvoidAvoided, 1},
		{"missile avoid ignores dodge", AvoidInput{IsMissile: true, DodgeChance: 90}, nil, AvoidNone, 0},
		{"no chances, no rolls", AvoidInput{}, nil, AvoidNone, 0},
		{"anim block", AvoidInput{AnimBlockChance: 50, AnimIsBlock: true, DodgeChance: 90}, []uint32{49}, AvoidMonBlock, 1},
		{"anim block needs block animation", AvoidInput{AnimBlockChance: 50, DodgeChance: 90}, []uint32{0}, AvoidDodged, 1},
		{"anim block fail then dodge", AvoidInput{AnimBlockChance: 50, AnimIsBlock: true, DodgeChance: 90}, []uint32{50, 10}, AvoidDodged, 2},
	}

	for _, tt := range tests {
		f := &fakeRoller{vals: tt.vals}
		if got := RollAvoid(f, tt.in); got != tt.want || f.calls != tt.calls {
			t.Errorf("%s: got %d (calls %d), want %d (calls %d)", tt.name, got, f.calls, tt.want, tt.calls)
		}
	}
}

func TestEffectiveResist(t *testing.T) {
	hell := ResistInput{DifficultyPenalty: -100}

	tests := []struct {
		name string
		in   ResistInput
		want int
	}{
		{"plain", ResistInput{Resist: 50}, 50},
		{"zero", ResistInput{Resist: 0}, 0},
		{"cap 75", ResistInput{Resist: 80}, 75},
		{"max resist bonus 10", ResistInput{Resist: 90, HasMaxResist: true, MaxResistBonus: 10}, 85},
		{"max resist capped at 95", ResistInput{Resist: 120, HasMaxResist: true, MaxResistBonus: 30}, 95},
		{"negative max resist bonus", ResistInput{Resist: 80, HasMaxResist: true, MaxResistBonus: -10}, 65},
		{"physical cap 50", ResistInput{Resist: 80, IsPhysical: true, NoDifficultyPenalty: true}, 50},
		{"pierce", ResistInput{Resist: 75, HasPierce: true, Pierce: 20}, 55},
		{"pierce on immunity", ResistInput{Resist: 100, HasPierce: true, Pierce: 20}, 75},
		{"pierce stat zero", ResistInput{Resist: 75, HasPierce: true}, 75},
		{"no pierce stat", ResistInput{Resist: 75, Pierce: 20}, 75},
		{"ignore flag skips cap, pierce on immunity, penalty", ResistInput{Resist: 100, HasPierce: true, Pierce: 20, DifficultyPenalty: -100, Ignore: true}, 100},
		{"ignore still pierces below 100", ResistInput{Resist: 90, HasPierce: true, Pierce: 20, Ignore: true}, 70},
		{"hell penalty", ResistInput{Resist: 75, DifficultyPenalty: -100}, -25},
		{"floor -100", ResistInput{Resist: -150}, -100},
		{"floor after penalty", ResistInput{Resist: 0, DifficultyPenalty: -100}, -100},
		{"exempt from penalty", ResistInput{Resist: 50, DifficultyPenalty: -100, NoDifficultyPenalty: true}, 50},
		{"classic nightmare", ResistInput{Resist: 50, DifficultyPenalty: ClassicResistPenalty(1)}, 30},
		{"classic hell", ResistInput{Resist: 50, DifficultyPenalty: ClassicResistPenalty(2)}, 0},
		{"classic normal", ResistInput{Resist: 50, DifficultyPenalty: ClassicResistPenalty(0)}, 50},
	}

	_ = hell

	for _, tt := range tests {
		if got := EffectiveResist(tt.in); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestApplyResist(t *testing.T) {
	tests := []struct{ dmg, res, want int }{
		{1000, 50, 500},
		{1000, -100, 2000},
		{1000, 100, 0},
		{1000, 0, 1000},
		{0, 50, 0},
		{-5, 50, 0},
		{255, 33, 170},
	}

	for _, tt := range tests {
		if got := ApplyResist(tt.dmg, tt.res); got != tt.want {
			t.Errorf("ApplyResist(%d,%d) = %d, want %d", tt.dmg, tt.res, got, tt.want)
		}
	}
}

func TestAbsorb(t *testing.T) {
	tests := []struct {
		name                    string
		dmg                     int
		has                     bool
		pct, flat               int
		wantRemaining, wantHeal int
	}{
		{"no absorb stat", 1000, false, 50, 5, 1000, 0},
		{"percent and flat", 1000, true, 10, 2, 388, 612},
		{"flat capped to remaining", 300, true, 0, 2, 0, 300},
		{"percent only", 1000, true, 25, 0, 750, 250},
		{"zero everything", 1000, true, 0, 0, 1000, 0},
		{"negative pct ignored", 1000, true, -5, 0, 1000, 0},
	}

	for _, tt := range tests {
		rem, heal := Absorb(tt.dmg, tt.has, tt.pct, tt.flat)
		if rem != tt.wantRemaining || heal != tt.wantHeal {
			t.Errorf("%s: got %d,%d want %d,%d", tt.name, rem, heal, tt.wantRemaining, tt.wantHeal)
		}
	}
}

func TestDamageLayout(t *testing.T) {
	var d Damage

	if got := unsafe.Sizeof(d); got != DamageSize {
		t.Fatalf("size = %#x, want %#x", got, DamageSize)
	}

	offsets := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Result", unsafe.Offsetof(d.Result), 0x04},
		{"Physical", unsafe.Offsetof(d.Physical), 0x08},
		{"Fire", unsafe.Offsetof(d.Fire), 0x10},
		{"Lightning", unsafe.Offsetof(d.Lightning), 0x1c},
		{"Magic", unsafe.Offsetof(d.Magic), 0x20},
		{"Cold", unsafe.Offsetof(d.Cold), 0x24},
		{"Poison", unsafe.Offsetof(d.Poison), 0x28},
		{"LifeLeech", unsafe.Offsetof(d.LifeLeech), 0x38},
		{"StunLen", unsafe.Offsetof(d.StunLen), 0x44},
		{"Heal", unsafe.Offsetof(d.Heal), 0x48},
		{"Total", unsafe.Offsetof(d.Total), 0x4c},
		{"HitClass", unsafe.Offsetof(d.HitClass), 0x60},
		{"ForcedClass", unsafe.Offsetof(d.ForcedClass), 0x64},
	}

	for _, o := range offsets {
		if o.got != o.want {
			t.Errorf("%s offset = %#x, want %#x", o.name, o.got, o.want)
		}
	}
}

func TestSumTotal(t *testing.T) {
	d := Damage{Physical: 100, Fire: 10, Lightning: 20, Magic: 30, Cold: 40, Poison: 50, LifeLeech: 7, Burn: 999}

	if got := d.SumTotal(false); got != 250 {
		t.Errorf("player target: %d", got)
	}

	if got := d.SumTotal(true); got != 257 {
		t.Errorf("monster target: %d", got)
	}
}

func TestScaleBySrcDam(t *testing.T) {
	tests := []struct {
		v     int32
		scale uint8
		want  int32
	}{
		{1000, 128, 1000},
		{1000, 64, 500},
		{1000, 0, 1000},
		{-1000, 64, -500},
		{1001, 100, 782},
		{-255, 1, -1},
		{127, 1, 0},
		{1000, 255, 1992},
	}

	for _, tt := range tests {
		if got := ScaleBySrcDam(tt.v, tt.scale); got != tt.want {
			t.Errorf("ScaleBySrcDam(%d,%d) = %d, want %d", tt.v, tt.scale, got, tt.want)
		}
	}
}

func TestRollStrike(t *testing.T) {
	in := StrikeInput{WeaponChance: 30, CriticalChance: 10, DeadlyChance: 5}

	tests := []struct {
		name  string
		in    StrikeInput
		vals  []uint32
		want  bool
		calls int
	}{
		{"weapon hits first", in, []uint32{29}, true, 1},
		{"critical hits second", in, []uint32{30, 9}, true, 2},
		{"deadly hits third", in, []uint32{30, 10, 4}, true, 3},
		{"all miss", in, []uint32{30, 10, 5}, false, 3},
		{"skip weapon", StrikeInput{WeaponChance: 30, SkipWeapon: true, CriticalChance: 10}, []uint32{9}, true, 1},
		{"zero chances never roll", StrikeInput{}, nil, false, 0},
		{"only deadly", StrikeInput{DeadlyChance: 5}, []uint32{4}, true, 1},
	}

	for _, tt := range tests {
		f := &fakeRoller{vals: tt.vals}
		if got := RollStrike(f, tt.in); got != tt.want || f.calls != tt.calls {
			t.Errorf("%s: got %v (calls %d), want %v (calls %d)", tt.name, got, f.calls, tt.want, tt.calls)
		}
	}
}

func TestApplyStrike(t *testing.T) {
	d := Damage{Physical: 500, Fire: 100}
	f := &fakeRoller{vals: []uint32{0}}

	if !d.ApplyStrike(f, StrikeInput{DeadlyChance: 50}) || d.Physical != 1000 || d.Fire != 100 || d.Result&ResultCritical == 0 {
		t.Errorf("strike not applied: %+v", d)
	}

	d = Damage{Physical: 500, Flags: DamageFlagNoPhysical}
	f = &fakeRoller{}

	if d.ApplyStrike(f, StrikeInput{DeadlyChance: 100}) || d.Physical != 500 || f.calls != 0 {
		t.Error("NoPhysical must skip the strike without rolling")
	}
}

func TestRollMonsterDouble(t *testing.T) {
	base := Damage{Physical: 1, Fire: 2, Lightning: 3, Magic: 4, Cold: 5, Poison: 6, Burn: 7, LifeLeech: 8}

	d := base
	f := &fakeRoller{}

	if d.RollMonsterDouble(f, 0) || f.calls != 0 || d != base {
		t.Error("chance 0: no roll, no change")
	}

	d = base
	f = &fakeRoller{vals: []uint32{49}}

	if !d.RollMonsterDouble(f, 50) {
		t.Fatal("49 < 50 must double")
	}

	want := Damage{Physical: 2, Fire: 4, Lightning: 6, Magic: 8, Cold: 10, Poison: 12, Burn: 7, LifeLeech: 8}
	if d != want {
		t.Errorf("got %+v", d)
	}

	d = base
	f = &fakeRoller{vals: []uint32{50}}

	if d.RollMonsterDouble(f, 50) || d != base || f.calls != 1 {
		t.Error("50 must not double and must consume a step")
	}
}

func TestCombatantScalePercent(t *testing.T) {
	tests := []struct {
		name                                             string
		same, aPlayer, dPlayer, aControlled, dControlled bool
		want                                             int
	}{
		{"same unit", true, true, true, true, true, 100},
		{"player hits player", false, true, true, true, true, 17},
		{"plain monster hits player", false, false, true, false, false, 100},
		{"controlled pet hits player", false, false, true, true, false, 17},
		{"player hits controlled merc", false, true, false, true, true, 25},
		{"player hits monster", false, true, false, true, false, 100},
		{"monster hits monster", false, false, false, false, false, 100},
	}

	for _, tt := range tests {
		got := CombatantScalePercent(tt.same, tt.aPlayer, tt.dPlayer, tt.aControlled, tt.dControlled)
		if got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}

	if ScaleByPercent(1000, 25) != 250 || ScaleByPercent(-1000, 17) != -170 || ScaleByPercent(7, 25) != 1 {
		t.Error("ScaleByPercent")
	}
}

func TestMissile(t *testing.T) {
	tests := []struct {
		name        string
		vel, velLev uint8
		level       int
		slowed      bool
		slowPct     int
		want        int
	}{
		{"leveled", 10, 16, 5, false, 0, 20 << 8},
		{"truncates", 10, 3, 2, false, 0, 10 << 8},
		{"level 0", 10, 16, 0, false, 0, 10 << 8},
		{"zero velocity", 0, 0, 5, false, 0, 0},
		{"slowed half", 10, 16, 5, true, 50, 2560},
	}

	for _, tt := range tests {
		if got := MissileVelocity(tt.vel, tt.velLev, tt.level, tt.slowed, tt.slowPct); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}

	if MissileStep(5120) != 3840 || MissileStep(0) != 0 || MissileStep(256) != 192 {
		t.Error("MissileStep")
	}

	if got := MissileRange(20, 5, 3, false, 0, 0, 0, false); got != 35 {
		t.Errorf("range = %d", got)
	}

	if got := MissileRange(20, -4, 3, false, 0, 0, 0, false); got != 8 {
		t.Errorf("negative levrange = %d", got)
	}

	if got := MissileRange(20, 5, 3, true, 1, 9, 3, true); got != 59 {
		t.Errorf("subloop extension = %d", got)
	}

	if got := MissileRange(20, 5, 3, true, 1, 9, 3, false); got != 35 {
		t.Errorf("subloop without flag = %d", got)
	}

	if got := MissileRange(20, 5, 3, false, 1, 9, 3, true); got != 35 {
		t.Errorf("no subloop = %d", got)
	}
}
