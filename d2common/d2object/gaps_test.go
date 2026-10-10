package d2object

import (
	"os"
	"path/filepath"
	"testing"
)

// script is a Rand that returns fixed answers (clamped below n), then 0.
type script struct {
	vals []int
	seen []int // the n of every call
}

func (s *script) Roll(n int) int {
	s.seen = append(s.seen, n)

	if len(s.vals) == 0 {
		return 0
	}

	v := s.vals[0]
	s.vals = s.vals[1:]

	if n <= 1 {
		return 0
	}

	if v >= n {
		v = n - 1
	}

	return v
}

func TestWithinRadiusAndDistance(t *testing.T) {
	tests := []struct {
		dx, dy int
		in     bool
	}{
		{0, 0, true}, {3, 0, true}, {0, -3, true}, {2, 2, true}, {3, 1, false}, {2, 3, false}, {-3, 0, true}, {4, 0, false},
	}

	for _, tc := range tests {
		if got := WithinRadius(tc.dx, tc.dy, ExplosionRadius); got != tc.in {
			t.Errorf("WithinRadius(%d,%d,3) = %v want %v", tc.dx, tc.dy, got, tc.in)
		}
	}

	dist := []struct{ dx, dy, sa, sb, want int }{
		{0, 0, 1, 1, 0},
		{3, 0, 1, 1, 3},  // (0 + 3*2) / 2
		{3, 3, 1, 1, 4},  // (3 + 3*2) / 2
		{4, 2, 3, 3, 2},  // both sizes cut 2: (0 + 2*2) / 2
		{-2, 1, 2, 2, 0}, // both axes cut to 0
		{6, -6, 2, 2, 6}, // cut 2: 4,4 -> (4 + 4*2) / 2
	}

	for _, tc := range dist {
		if got := UnitDistance(tc.dx, tc.dy, tc.sa, tc.sb); got != tc.want {
			t.Errorf("UnitDistance(%d,%d,%d,%d) = %d want %d", tc.dx, tc.dy, tc.sa, tc.sb, got, tc.want)
		}
	}
}

func TestExplosionHitChance(t *testing.T) {
	tests := []struct {
		name                  string
		level, dex, def, roll int
		want                  int
	}{
		// 2*(r - 5*(dex/2)) - def + 125, floored at 65; r is a roll below level/4
		{"weak target", 10, 0, 0, 1, 127},
		{"high defense", 10, 0, 100, 0, 65},
		{"dexterity 30", 30, 30, 0, 5, 65},
		{"zero level rolls nothing", 0, 0, 20, 0, 105},
		{"monster no dex", 40, 0, 50, 9, 93},
	}

	for _, tc := range tests {
		s := &script{vals: []int{tc.roll}}
		if got := ExplosionHitChance(tc.level, tc.dex, tc.def, s); got != tc.want {
			t.Errorf("%s: chance %d want %d", tc.name, got, tc.want)
		}

		if len(s.seen) != 1 || s.seen[0] != tc.level>>2 {
			t.Errorf("%s: roll bound %v want level>>2 = %d", tc.name, s.seen, tc.level>>2)
		}
	}
}

func TestExplosionDamage8(t *testing.T) {
	tests := []struct {
		name       string
		life8, pct int
		roll       int
		want       int
		rollSize   int
	}{
		// life 100 points = 25600: lo = 800, hi = 3200, roll in [0, 2400+256)
		{"100 life low", 100 << 8, 100, 0, 800, 3200 - 800 + 256},
		{"100 life high", 100 << 8, 100, 999999, 800 + (3200 - 800 + 255), 3200 - 800 + 256},
		{"1 life", 1 << 8, 100, 0, 8, 32 - 8 + 256},
		// 25 -> 25>>5 = 0 raised to 1; 25>>3 = 3
		{"tiny life", 25, 100, 0, 1, 3 - 1 + 256},
		{"half percent", 100 << 8, 50, 0, 400, 3200 - 800 + 256},
	}

	for _, tc := range tests {
		s := &script{vals: []int{tc.roll}}
		if got := ExplosionDamage8(tc.life8, tc.pct, s); got != tc.want {
			t.Errorf("%s: damage %d want %d", tc.name, got, tc.want)
		}

		if len(s.seen) != 1 || s.seen[0] != tc.rollSize {
			t.Errorf("%s: roll bound %v want %d", tc.name, s.seen, tc.rollSize)
		}
	}
}

func TestRollExplosionOrderAndMiss(t *testing.T) {
	// level 40: first roll(10), then roll(100), then (on a hit) the damage roll
	hit := &script{vals: []int{0, 10, 0}}
	o := RollExplosion(hit, ExplosionTarget{Level: 40, Dex: 0, Defense: 0, Life8: 200 << 8}, 100)

	if !o.Hit || o.Chance != 125 || o.Roll != 10 || o.Damage8 != 200<<8>>5 {
		t.Errorf("hit: %+v", o)
	}

	if len(hit.seen) != 3 || hit.seen[0] != 10 || hit.seen[1] != 100 {
		t.Errorf("hit roll order: %v", hit.seen)
	}

	miss := &script{vals: []int{0, 99, 0}}
	o = RollExplosion(miss, ExplosionTarget{Level: 40, Dex: 0, Defense: 500, Life8: 200 << 8}, 100)

	if o.Hit || o.Damage8 != 0 || o.Chance != ExplosionMinHitChance || len(miss.seen) != 2 {
		t.Errorf("miss: %+v seen %v", o, miss.seen)
	}
}

func TestFixed8Points(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 0}, {-5, 0}, {1, 1}, {255, 1}, {256, 1}, {767, 2}, {1 << 16, 256}} {
		if got := Fixed8Points(tc.in); got != tc.want {
			t.Errorf("Fixed8Points(%d) = %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestChancesByMonsterLevel(t *testing.T) {
	for _, tc := range []struct{ lvl, trap, lock int }{{0, 5, 8}, {1, 5, 8}, {7, 5, 11}, {8, 6, 12}, {30, 8, 23}, {85, 15, 50}} {
		if TrapChancePct(tc.lvl) != tc.trap || LockChancePct(tc.lvl) != tc.lock {
			t.Errorf("lvl %d: trap %d lock %d want %d %d", tc.lvl, TrapChancePct(tc.lvl), LockChancePct(tc.lvl), tc.trap, tc.lock)
		}
	}
}

func TestRollChestInit(t *testing.T) {
	tests := []struct {
		name     string
		initFn   int
		lockable bool
		lvl      int
		vals     []int
		want     ChestInit
		calls    int
	}{
		{"plain init fn", 0, true, 30, []int{0, 0, 0}, ChestInit{}, 0},
		{"no trap no lock", InitVariantLock, true, 30, []int{50, 90}, ChestInit{}, 2},
		{"trap handler 5 not locked", InitVariantLock, true, 30, []int{7, 4, 99}, ChestInit{Handler: 5}, 3},
		{"locked, no trap", InitVariantLock, true, 30, []int{50, 22}, ChestInit{Locked: true}, 2},
		{"trap + lock", InitVariantLock, true, 30, []int{0, 7, 0}, ChestInit{Locked: true, Handler: 8}, 3},
		{"urn init fn 2 never locks", InitVariant, true, 30, []int{0, 0, 0}, ChestInit{Handler: 1}, 2},
		{"not lockable", InitVariantLock, false, 30, []int{0, 1}, ChestInit{Handler: 2}, 2},
		{"lock edge 23 is not locked at level 30", InitVariantLock, true, 30, []int{99, 23}, ChestInit{}, 2},
	}

	for _, tc := range tests {
		s := &script{vals: tc.vals}
		if got := RollChestInit(tc.initFn, tc.lockable, tc.lvl, s); got != tc.want {
			t.Errorf("%s: %+v want %+v", tc.name, got, tc.want)
		}

		if len(s.seen) != tc.calls {
			t.Errorf("%s: %d rolls %v want %d", tc.name, len(s.seen), s.seen, tc.calls)
		}
	}
}

func TestSpawnHandlers(t *testing.T) {
	want := map[int]SpawnHandler{
		0: {}, 1: {SpawnTrap, 330}, 2: {SpawnTrap, 326}, 3: {SpawnTrap, 329}, 4: {SpawnTrap, 369}, 5: {SpawnFire, 0},
		6: {SpawnTrap, 326}, 7: {SpawnFire, 0}, 8: {SpawnMonster, 0}, 9: {SpawnMonster, 0}, 10: {}, -1: {},
	}

	for n, w := range want {
		if got := HandlerFor(n); got != w {
			t.Errorf("handler %d = %+v want %+v", n, got, w)
		}
	}

	// the rolled range is 1..8, never the 9th (the table has nine usable rows only for the exe's range check)
	for i := 0; i < 8; i++ {
		s := &script{vals: []int{0, i}}
		if h := RollChestInit(InitVariant, false, 0, s).Handler; h != i+1 {
			t.Errorf("roll %d -> handler %d", i, h)
		}
	}
}

func TestContainerDropRolls(t *testing.T) {
	tests := []struct {
		name   string
		locked bool
		roll   int
		want   int
	}{
		{"empty at 24", false, 24, 0}, {"drops at 25", false, 25, 1}, {"locked always drops twice", true, 0, 2},
		{"locked high roll", true, 99, 2}, {"top roll", false, 99, 1}, {"empty at 0", false, 0, 0},
	}

	for _, tc := range tests {
		s := &script{vals: []int{tc.roll}}
		if got := ContainerDropRolls(tc.locked, s); got != tc.want {
			t.Errorf("%s: %d want %d", tc.name, got, tc.want)
		}

		if len(s.seen) != 1 || s.seen[0] != 100 {
			t.Errorf("%s: the roll must always be consumed: %v", tc.name, s.seen)
		}
	}
}

func TestSmallContainers(t *testing.T) {
	for _, tc := range []struct {
		roll int
		drop bool
	}{{0, true}, {20, true}, {21, false}, {99, false}} {
		if got := SmallContainerDrops(&script{vals: []int{tc.roll}}); got != tc.drop {
			t.Errorf("small drop roll %d = %v", tc.roll, got)
		}
	}

	for _, tc := range []struct {
		roll  int
		spawn bool
	}{{0, false}, {8191, false}, {8192, true}, {9999, true}} {
		if got := BarrelSpawnsMonster(&script{vals: []int{tc.roll}}); got != tc.spawn {
			t.Errorf("barrel monster roll %d = %v", tc.roll, got)
		}
	}
}

func TestStormAndPotionShrines(t *testing.T) {
	for _, tc := range []struct{ life, pct, want int }{{100, 50, 50}, {1, 50, 0}, {3, 50, 1}, {0, 50, 0}, {250, 50, 125}, {7, 0, 0}} {
		if got := StormLifeLoss(tc.life, tc.pct); got != tc.want {
			t.Errorf("StormLifeLoss(%d,%d) = %d want %d", tc.life, tc.pct, got, tc.want)
		}
	}

	m := StormMissiles()
	if len(m) != 16 || m[0] != (MissileShot{62, 5, 5}) || m[1] != (MissileShot{62, 5, -10}) || m[5] != (MissileShot{62, -10, -10}) ||
		m[15] != (MissileShot{62, -20, -20}) {
		t.Errorf("storm grid %v", m)
	}

	for _, tc := range []struct{ lvl, want int }{{1, 1}, {4, 1}, {5, 1}, {12, 2}, {40, 8}, {94, 8}, {0, 1}} {
		if got := MissileLevel(tc.lvl); got != tc.want {
			t.Errorf("MissileLevel(%d) = %d want %d", tc.lvl, got, tc.want)
		}
	}

	for _, tc := range []struct{ a0, a1, roll, want int }{{5, 10, 0, 5}, {5, 10, 4, 9}, {5, 10, 40, 9}, {5, 5, 3, 5}, {8, 3, 3, 8}} {
		if got := PotionCount(tc.a0, tc.a1, &script{vals: []int{tc.roll}}); got != tc.want {
			t.Errorf("PotionCount(%d,%d) = %d want %d", tc.a0, tc.a1, got, tc.want)
		}
	}

	pm := PoisonShrine.PotionMissiles()
	if len(pm) != 6 || pm[0] != (MissileShot{48, -6, 6}) || pm[5] != (MissileShot{48, 6, -6}) || ExplodingShrine.ItemCode != "opm" ||
		PoisonShrine.ItemCode != "gpm" || ExplodingShrine.MissileID != 45 {
		t.Errorf("potion shrines %+v %v", PoisonShrine, pm)
	}
}

func TestGemUpgrade(t *testing.T) {
	tests := []struct {
		name string
		inv  []GemChoice
		roll int
		want string
		up   bool
	}{
		{"first upgradable", []GemChoice{{"gcv", "gfv"}, {"gsr", "glr"}}, 0, "gfv", true},
		{"skips perfect", []GemChoice{{"gpv", "non"}, {"gsr", "glr"}}, 0, "glr", true},
		{"empty better field", []GemChoice{{"zzz", ""}, {"gfy", "gsy"}}, 0, "gsy", true},
		{"none: chipped diamond", nil, 0, "gcw", false},
		{"none: chipped amethyst", []GemChoice{{"gpv", "non"}}, 5, "gcv", false},
		{"none: ruby", nil, 1, "gcr", false},
	}

	for _, tc := range tests {
		code, up := GemUpgrade(tc.inv, &script{vals: []int{tc.roll}})
		if code != tc.want || up != tc.up {
			t.Errorf("%s: %q %v want %q %v", tc.name, code, up, tc.want, tc.up)
		}
	}
}

func TestWellPulses(t *testing.T) {
	if WellCharges(1) != 2 || WellCharges(3) != 6 || WellRefillFrames(750) != 751 {
		t.Fatal("charges / refill")
	}

	tests := []struct {
		c, parm2 int
		mode     int
		ok       bool
	}{
		{2, 1, 0, true}, {1, 1, 1, true}, {0, 1, 2, true},
		{6, 3, 0, true}, {5, 3, 0, false}, {4, 3, 0, false}, {3, 3, 1, true}, {0, 3, 2, true}, {7, 3, 0, false},
		{1, 0, 0, false}, {-1, 1, 0, false},
	}

	for _, tc := range tests {
		m, ok := WellMode(tc.c, tc.parm2)
		if ok != tc.ok || (ok && m != tc.mode) {
			t.Errorf("WellMode(%d,%d) = %d,%v want %d,%v", tc.c, tc.parm2, m, ok, tc.mode, tc.ok)
		}
	}
}

// TestGapsRealTables ties the constants to the extracted tables when D2_TABLES is set.
func TestGapsRealTables(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "objects", "patch_d2", "Objects.txt"))
	if err != nil {
		t.Skip(err)
	}

	defs, err := ParseObjects(data)
	if err != nil {
		t.Fatal(err)
	}

	locks, variants := 0, 0

	for _, d := range defs {
		if d.Lockable {
			locks++
		}

		if d.InitFn == InitVariantLock || d.InitFn == InitVariant {
			variants++
		}

		switch d.ID {
		case ExplodingBarrelID:
			if d.OperateFn != 7 || d.Damage != 100 {
				t.Errorf("object 11: fn %d damage %d, want 7 and 100", d.OperateFn, d.Damage)
			}
		case 111, 113, 115:
			if d.OperateFn != 22 || d.Parm0 != 750 || d.Parm1 != 128 || d.Parm3 != 3 {
				t.Errorf("well %d: %+v", d.ID, d)
			}
		}
	}

	if locks == 0 || variants == 0 {
		t.Errorf("no lockable (%d) or variant (%d) rows read", locks, variants)
	}

	// the shrine rows the world effects read
	for _, tc := range []struct{ code, a0, a1 int }{{ShrineStorm, 50, 2000}, {ShrineExploding, 5, 10}, {ShrinePoison, 5, 10}} {
		s := shrine(t, tc.code)
		if s.Arg0 != tc.a0 || s.Arg1 != tc.a1 {
			t.Errorf("%s args %d %d want %d %d", s.Name, s.Arg0, s.Arg1, tc.a0, tc.a1)
		}
	}
}
