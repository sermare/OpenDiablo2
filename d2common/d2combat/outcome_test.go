package d2combat

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Constants pinned from 0x57b9c0 / 0x6228d0 (see skills-combat notes).
func TestPinnedToHitTable(t *testing.T) {
	tests := []struct {
		name            string
		ar, def, al, dl int
		want            int
	}{
		{"equal", 100, 100, 10, 10, 50},
		{"clamp low", 1, 100000, 10, 10, 5},
		{"clamp high", 100000, 1, 10, 10, 95},
		{"level edge", 100, 100, 20, 10, 66}, // 50*20*2/30
		{"zero sum", 0, 0, 5, 5, 95},
		{"negative def", 100, -50, 10, 10, 95},
	}

	for _, tc := range tests {
		got := ToHitChance(ToHitInput{AttackRating: tc.ar, Defense: tc.def, AttackerLevel: tc.al, DefenderLevel: tc.dl})
		if got != tc.want {
			t.Errorf("%s: %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestPinnedBlockTable(t *testing.T) {
	tests := []struct {
		name                        string
		blk, factor, dex, lvl, want int
	}{
		{"cap 75", 80, 5, 200, 10, 75},
		{"dex formula", 30, 5, 30, 10, 26}, // (30-15)*35/20 = 26
		{"low dex negative", 30, 5, 10, 10, -8},
	}

	for _, tc := range tests {
		got := PlayerBlockChance(PlayerBlockInput{
			HasShield: true, ToBlock: tc.blk, ClassBlockFactor: tc.factor, Dex: tc.dex, Level: tc.lvl, IncludeDex: true,
		})
		if got != tc.want {
			t.Errorf("%s: %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestResolveAttackOrderAndSteps(t *testing.T) {
	hitAll := ToHitInput{AttackRating: 1 << 20, Defense: 1, AttackerLevel: 10, DefenderLevel: 10} // 95%

	tests := []struct {
		name      string
		in        AttackInput
		vals      []uint32
		want      uint32
		wantCalls int
	}{
		{"miss stops", AttackInput{ToHit: hitAll}, []uint32{99}, 0, 1},
		{"plain hit", AttackInput{ToHit: hitAll}, []uint32{0}, ResultHit, 1},
		{"blocked skips avoid", AttackInput{ToHit: hitAll, BlockChance: 50, Avoid: AvoidInput{DodgeChance: 100}},
			[]uint32{0, 10}, ResultBlocked, 2},
		{"block fails then dodge", AttackInput{ToHit: hitAll, BlockChance: 50, Avoid: AvoidInput{DodgeChance: 100}},
			[]uint32{0, 99, 0}, ResultDodged, 3},
		{"missile avoid", AttackInput{ToHit: hitAll, Avoid: AvoidInput{AvoidChance: 10, IsMissile: true}},
			[]uint32{0, 5}, ResultAvoided, 2},
		{"moving block thirds", AttackInput{ToHit: hitAll, BlockChance: 60, DefenderMoving: true},
			[]uint32{0, 20}, ResultHit, 2}, // 60/3=20, roll 20 is not < 20
	}

	for _, tc := range tests {
		f := &fakeRoller{vals: tc.vals}
		got := ResolveAttack(f, tc.in)

		if got.Result != tc.want || f.calls != tc.wantCalls {
			t.Errorf("%s: result %#x calls %d, want %#x / %d", tc.name, got.Result, f.calls, tc.want, tc.wantCalls)
		}
	}
}

func TestResolveAttackSeeded(t *testing.T) {
	in := AttackInput{ToHit: ToHitInput{AttackRating: 300, Defense: 300, AttackerLevel: 20, DefenderLevel: 20}, BlockChance: 30}
	a, b := d2rand.New(7), d2rand.New(7)

	for i := 0; i < 50; i++ {
		if ResolveAttack(a, in) != ResolveAttack(b, in) {
			t.Fatal("not deterministic")
		}
	}
}

func TestReduceHit(t *testing.T) {
	tests := []struct{ dmg, flat, pct, want int }{
		{1000, 100, 50, 450},
		{50, 100, 50, 0},
		{1000, 0, -100, 2000},
		{1000, 0, 100, 0},
	}

	for _, tc := range tests {
		if got := ReduceHit(tc.dmg, tc.flat, tc.pct); got != tc.want {
			t.Errorf("%+v got %d", tc, got)
		}
	}
}
