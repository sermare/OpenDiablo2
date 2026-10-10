package d2combat

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestReduceComponent(t *testing.T) {
	tests := []struct {
		name              string
		dmg, flat, res    int
		ignore, hasAbs    bool
		absPct, absFlat   int
		wantOut, wantHeal int
	}{
		{"flat before percent", 1000, 100, 50, false, false, 0, 0, 450, 0},
		{"flat exceeds damage is not floored", 50, 100, 50, false, false, 0, 0, -50, 0},
		{"immune", 1000, 0, 100, false, false, 0, 0, 0, 0},
		{"over-immune is clamped to 100", 1000, 0, 130, false, false, 0, 0, 0, 0},
		{"negative resist amplifies", 1000, 0, -100, false, false, 0, 0, 2000, 0},
		{"ignore skips flat, positive resist and absorb", 1000, 100, 50, true, true, 40, 0, 1000, 0},
		{"ignore keeps negative resist", 1000, 100, -50, true, false, 0, 0, 1500, 0},
		{"absorb after resist", 1000, 0, 50, false, true, 10, 0, 450, 50},
		{"zero damage", 0, 100, 50, false, false, 0, 0, 0, 0},
	}

	for _, tt := range tests {
		out, heal := ReduceComponent(tt.dmg, tt.flat, tt.res, tt.ignore, tt.hasAbs, tt.absPct, tt.absFlat)
		if out != tt.wantOut || heal != tt.wantHeal {
			t.Errorf("%s: got %d,%d want %d,%d", tt.name, out, heal, tt.wantOut, tt.wantHeal)
		}
	}
}

func TestAbsorbPercentCap(t *testing.T) {
	rem, heal := Absorb(1000, true, 90, 0)
	if rem != 600 || heal != 400 {
		t.Fatalf("got %d,%d want 600,400", rem, heal)
	}
}

func TestFlatReductionSlots(t *testing.T) {
	tests := []struct {
		typ  DamageType
		want int
	}{
		{TypePhysical, 7}, {TypeFire, 3}, {TypeLightning, 3}, {TypeCold, 3}, {TypeMagic, 3}, {TypePoison, 0},
	}

	for _, tt := range tests {
		if got := FlatReduction(tt.typ, 7, 3); got != tt.want {
			t.Errorf("type %d got %d want %d", tt.typ, got, tt.want)
		}
	}

	if got := ScaleFlatReduction(4, 0); got != 1024 {
		t.Errorf("unscaled got %d", got)
	}

	if got := ScaleFlatReduction(4, 0x200); got != 512 {
		t.Errorf("scaled got %d", got)
	}

	if got := ScaleFlatReduction(-4, 0x200); got != -1024 {
		t.Errorf("negative must not scale, got %d", got)
	}
}

func TestPlayerCountBonus(t *testing.T) {
	want := map[int]int{0: 0, 1: 0, 2: 50, 3: 100, 8: 350, 9: 350, 10: 400}
	for p, w := range want {
		if got := PlayerCountBonusPercent(p); got != w {
			t.Errorf("players %d got %d want %d", p, got, w)
		}
	}
}

func TestCrushingBlowDivisor(t *testing.T) {
	tests := []struct {
		def     CrushingBlowDefender
		players int
		missile bool
		want    int
	}{
		{CBNormalMonster, 1, false, 4},
		{CBNormalMonster, 3, false, 8},
		{CBBossMonster, 1, false, 8},
		{CBBossMonster, 2, false, 12},
		{CBSpecial, 8, false, 10},
		{CBSpecial, 1, true, 20},
		{CBNormalMonster, 1, true, 8},
		{CBOther, 5, false, 4},
	}

	for _, tt := range tests {
		if got := CrushingBlowDivisor(tt.def, tt.players, tt.missile); got != tt.want {
			t.Errorf("%+v got %d", tt, got)
		}
	}
}

func TestRollCrushingBlow(t *testing.T) {
	life := ToFixed(400)

	tests := []struct {
		name    string
		roll    uint32
		in      CrushingBlowInput
		want    CrushingBlowResult
		wantNum int
	}{
		{"hit removes a quarter", 10, CrushingBlowInput{Chance: 20, DefenderLife: life, Divisor: 4},
			CrushingBlowResult{Triggered: true, Removed: ToFixed(100), NewLife: ToFixed(300)}, 1},
		{"phys resist reduces it", 10, CrushingBlowInput{Chance: 20, DefenderLife: life, Divisor: 4, DefenderPhysResist: 50},
			CrushingBlowResult{Triggered: true, Removed: ToFixed(50), NewLife: ToFixed(350)}, 1},
		{"negative resist does not amplify", 10, CrushingBlowInput{Chance: 20, DefenderLife: life, Divisor: 4, DefenderPhysResist: -50},
			CrushingBlowResult{Triggered: true, Removed: ToFixed(100), NewLife: ToFixed(300)}, 1},
		{"roll equal to chance fails", 20, CrushingBlowInput{Chance: 20, DefenderLife: life, Divisor: 4},
			CrushingBlowResult{NewLife: life}, 1},
		{"no chance, no roll", 0, CrushingBlowInput{Chance: 0, DefenderLife: life, Divisor: 4},
			CrushingBlowResult{NewLife: life}, 0},
		{"dead defender stays at zero", 0, CrushingBlowInput{Chance: 100, DefenderLife: 0, Divisor: 4},
			CrushingBlowResult{Triggered: true, NewLife: 0, Killed: true}, 1},
	}

	for _, tt := range tests {
		f := &fakeRoller{vals: []uint32{tt.roll}}
		if got := RollCrushingBlow(f, tt.in); got != tt.want {
			t.Errorf("%s: got %+v want %+v", tt.name, got, tt.want)
		}

		if f.calls != tt.wantNum {
			t.Errorf("%s: %d steps, want %d", tt.name, f.calls, tt.wantNum)
		}
	}
}

func TestRollCrushingBlowSeeded(t *testing.T) {
	a, b := d2rand.New(11), d2rand.New(11)
	in := CrushingBlowInput{Chance: 30, DefenderLife: ToFixed(1000), Divisor: 4}

	hits := 0

	for i := 0; i < 200; i++ {
		x, y := RollCrushingBlow(a, in), RollCrushingBlow(b, in)
		if x != y {
			t.Fatal("not deterministic")
		}

		if x.Triggered {
			hits++
		}
	}

	if hits < 30 || hits > 90 {
		t.Fatalf("30%% over 200 rolls gave %d", hits)
	}
}

func TestOpenWoundsLevelTerm(t *testing.T) {
	tests := []struct{ lvl, want int }{
		{0, 0}, {1, 0}, {2, 9}, {15, 126}, {16, 144}, {30, 396}, {31, 423}, {45, 801},
		{46, 837}, {60, 1341}, {61, 1386}, {99, 3096},
	}

	for _, tt := range tests {
		if got := OpenWoundsLevelTerm(tt.lvl); got != tt.want {
			t.Errorf("level %d got %d want %d", tt.lvl, got, tt.want)
		}
	}
}

func TestOpenWoundsValue(t *testing.T) {
	tests := []struct {
		name                    string
		lvl                     int
		player, monster, halved bool
		missile                 bool
		want                    int
	}{
		{"monster level 1", 1, false, true, false, false, 40},
		{"monster level 30", 30, false, true, false, false, 436},
		{"flagged monster halved", 30, false, true, true, false, 218},
		{"flagged monster missile same", 30, false, true, true, true, 218},
		{"plain monster missile not halved", 30, false, true, false, true, 436},
		{"player melee", 30, true, false, false, false, 109},
		{"player missile", 30, true, false, false, true, 54},
	}

	for _, tt := range tests {
		if got := OpenWoundsValue(tt.lvl, tt.player, tt.monster, tt.halved, tt.missile); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}

func TestOpenWoundsRollAndRefresh(t *testing.T) {
	f := &fakeRoller{vals: []uint32{5, 50}}

	e := RollOpenWounds(f, 10, 100, 1000)
	if !e.Triggered || e.RegenStat != -100 || e.Expires != 1200 {
		t.Fatalf("got %+v", e)
	}

	if miss := RollOpenWounds(f, 10, 100, 1100); miss.Triggered {
		t.Fatal("roll 50 vs 10 must miss")
	}

	if f.calls != 2 {
		t.Fatalf("calls %d", f.calls)
	}

	if RollOpenWounds(f, 0, 100, 0).Triggered || f.calls != 2 {
		t.Fatal("chance 0 must not roll")
	}

	e2 := ApplyOpenWounds(e, OpenWoundsEffect{Triggered: true, RegenStat: -500, Expires: 1500})
	if e2.RegenStat != -100 || e2.Expires != 1500 {
		t.Fatalf("refresh must keep the value and extend, got %+v", e2)
	}
}

func TestAdjustAROperands(t *testing.T) {
	tests := []struct {
		name         string
		ar, def      int
		o            AROperands
		wantAR, want int
	}{
		{"nothing", 100, 200, AROperands{}, 100, 200},
		{"ignore defense plain monster", 100, 200, AROperands{IgnoreDefense: true, DefenderIsPlainMonster: true}, 100, 0},
		{"ignore defense on boss does nothing", 100, 200, AROperands{IgnoreDefense: true}, 100, 200},
		{"target ac 40 percent", 100, 200, AROperands{TargetACPct: 40}, 100, 120},
		{"target ac halved for players", 100, 200, AROperands{TargetACPct: 40, HalveTargetAC: true}, 100, 160},
		{"target ac clamped", 100, 200, AROperands{TargetACPct: 250}, 100, 0},
		{"demon ar", 100, 200, AROperands{DemonAR: 50, DefenderDemon: true, UndeadAR: 99}, 150, 200},
		{"undead ar", 100, 200, AROperands{UndeadAR: 70, DefenderUndead: true}, 170, 200},
	}

	for _, tt := range tests {
		ar, def := AdjustAROperands(tt.ar, tt.def, tt.o)
		if ar != tt.wantAR || def != tt.want {
			t.Errorf("%s: got %d,%d want %d,%d", tt.name, ar, def, tt.wantAR, tt.want)
		}
	}
}

func TestBlockRecoveryReady(t *testing.T) {
	tests := []struct {
		since, fbr int
		want       bool
	}{
		{15, 0, false}, {16, 0, true}, {17, 16, false}, {18, 16, true}, {16, -16, true}, {15, -16, true}, {14, -16, true}, {13, -16, false}, {14, -1, false}, {16, -7, true},
	}

	for _, tt := range tests {
		if got := BlockRecoveryReady(tt.since, tt.fbr); got != tt.want {
			t.Errorf("%+v got %v", tt, got)
		}
	}
}

func TestSelectReaction(t *testing.T) {
	ready := ReactionInput{DefenderIsPlayer: true, FramesSinceBlock: 100}

	tests := []struct {
		name string
		in   ReactionInput
		res  uint32
		want Reaction
	}{
		{"player dodge beats block", ready, 0x80 | 0x10, ReactDodge},
		{"player avoid", ready, 0x100, ReactAvoid},
		{"player evade", ready, 0x200, ReactEvade},
		{"player block", ready, 0x10, ReactBlock},
		{"player monster-block bit", ready, 0x8000, ReactBlock},
		{"player block on cooldown", ReactionInput{DefenderIsPlayer: true, FramesSinceBlock: 10}, 0x10, ReactNone},
		{"player block 0x4000 suppressed", ready, 0x10 | 0x4000, ReactNone},
		{"player death beats hit", ready, 0x2 | 0x4 | 0x1, ReactDeath},
		{"player knockback", ready, 0x8 | 0x1, ReactKnockback},
		{"player hit recovery", ReactionInput{DefenderIsPlayer: true}, 0x4 | 0x1, ReactHitRecovery},
		{"player small hit flag only", ReactionInput{DefenderIsPlayer: true, StaggerSuppressed: true}, 0x4 | 0x1, ReactFlagOnly},
		{"state 0x15 forces hit recovery", ReactionInput{DefenderIsPlayer: true, StaggerSuppressed: true, HasState15: true}, 0x4, ReactHitRecovery},
		{"state 0x36 ignores everything but death", ReactionInput{DefenderIsPlayer: true, DefenderHasS36: true}, 0x4 | 0x10, ReactNone},
		{"state 0x36 death", ReactionInput{DefenderHasS36: true}, 0x2, ReactDeath},
		{"monster death first", ReactionInput{}, 0x2 | 0x10 | 0x4, ReactDeath},
		{"monster block", ReactionInput{}, 0x10, ReactBlock},
		{"monster block without anim", ReactionInput{MonsterNoBlockAnim: true}, 0x10, ReactFlagOnly},
		{"monster knockback without mode becomes hit", ReactionInput{}, 0x8, ReactHitRecovery},
		{"monster knockback", ReactionInput{MonsterHasKnockbackMode: true}, 0x8, ReactKnockback},
		{"monster 0x8000 alone does nothing", ReactionInput{}, 0x8000, ReactNone},
	}

	for _, tt := range tests {
		in := tt.in
		in.Result = tt.res

		if got := SelectReaction(in); got != tt.want {
			t.Errorf("%s: got %d want %d", tt.name, got, tt.want)
		}
	}
}

func TestStaggerSuppressed(t *testing.T) {
	maxLife := ToFixed(1000)

	tests := []struct {
		name          string
		frozen        bool
		poison, total int32
		class         int
		noMode        bool
		coin1, coin2  bool
		want          bool
	}{
		{"frozen", true, 0, int32(ToFixed(500)), 0, false, true, true, true},
		{"poison only", false, int32(ToFixed(500)), int32(ToFixed(500)), 0, false, true, true, true},
		{"under one life", false, 0, 255, 0, false, true, true, true},
		{"below 1/16 of max", false, 0, int32(ToFixed(50)), 0, false, true, true, true},
		{"class 2 threshold is 1/8", false, 0, int32(ToFixed(100)), 2, false, true, true, true},
		{"big hit always staggers", false, 0, int32(ToFixed(600)), 0, false, false, false, false},
		{"medium hit needs coin", false, 0, int32(ToFixed(70)), 0, false, false, true, true},
		{"medium hit with coins", false, 0, int32(ToFixed(70)), 0, false, true, true, false},
		{"monster without mode", false, 0, int32(ToFixed(600)), 0, true, true, true, true},
	}

	for _, tt := range tests {
		if got := StaggerSuppressed(tt.frozen, tt.poison, tt.total, tt.class, maxLife, tt.noMode, tt.coin1, tt.coin2); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestEffectiveResistZeroPhysical(t *testing.T) {
	in := ResistInput{Resist: 40, IsPhysical: true, NoDifficultyPenalty: true, ZeroPhysical: true}
	if got := EffectiveResist(in); got != 0 {
		t.Fatalf("got %d want 0", got)
	}

	in.Ignore = true
	if got := EffectiveResist(in); got != 0 {
		t.Fatalf("ignore must not bypass, got %d", got)
	}

	in.Resist = -30
	if got := EffectiveResist(in); got != -30 {
		t.Fatalf("negative resist untouched, got %d", got)
	}

	in = ResistInput{Resist: 40, NoDifficultyPenalty: true, ZeroPhysical: true}
	if got := EffectiveResist(in); got != 40 {
		t.Fatalf("non-physical untouched, got %d", got)
	}
}

func TestResolveAttackAutoHitAndReactBit(t *testing.T) {
	f := &fakeRoller{}
	res := ResolveAttack(f, AttackInput{AutoHit: true, DefenderLacksState36: true})

	if res.Result != ResultHit|ResultHitReact || f.calls != 0 {
		t.Fatalf("got %+v with %d steps", res, f.calls)
	}

	// An evade clears the hit and sets no flag, so it never gets the react bit.
	f = &fakeRoller{vals: []uint32{0, 0}}
	res = ResolveAttack(f, AttackInput{
		AutoHit:              true,
		DefenderLacksState36: true,
		Avoid:                AvoidInput{Moving: true, EvadeChance: 50},
	})

	if res.Result != 0 || f.calls != 1 {
		t.Fatalf("evade: got %+v with %d steps", res, f.calls)
	}
}
