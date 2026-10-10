package d2herostats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManaRegenPerFrame(t *testing.T) {
	cases := []struct {
		name                    string
		maxRaw, secs, pct, flat int
		noRegen                 bool
		want                    int
	}{
		// level 94 sorceress: 477 mana = 122112 raw, ManaRegen 120 -> 3000 frames -> 40 raw per frame
		{"sorceress 477", 477 << 8, 120, 0, 0, false, 40},
		{"regenerate mana 100%", 477 << 8, 120, 100, 0, false, 80},
		{"regenerate mana 30%", 477 << 8, 120, 30, 0, false, 52},
		{"negative bonus", 477 << 8, 120, -50, 0, false, 20},
		{"tiny pool is at least 1 raw", 100, 120, 0, 0, false, 1},
		{"level 1 sorceress 35 mana", 35 << 8, 120, 0, 0, false, 2},
		{"no ManaRegen falls back to 7500 frames", 7500 * 5, 0, 0, 0, false, 5},
		{"flat stat 26 adds on top", 477 << 8, 120, 100, 7, false, 87},
		{"state 85 blocks the base but not the flat part", 477 << 8, 120, 100, 7, true, 7},
		{"state 85 alone", 477 << 8, 120, 50, 0, true, 0},
		{"bonus truncates (1*(100+50)/100)", 100, 120, 50, 0, false, 1},
	}

	for _, c := range cases {
		if got := ManaRegenPerFrame(c.maxRaw, c.secs, c.pct, c.flat, c.noRegen); got != c.want {
			t.Errorf("%s: ManaRegenPerFrame = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestManaRegenFullInManaRegenSeconds(t *testing.T) {
	// with a pool that divides evenly, the whole maximum comes back in
	// ManaRegen seconds (25 frames each)
	max := 3000 * 40
	if per := ManaRegenPerFrame(max, 120, 0, 0, false); per != 40 {
		t.Fatalf("per frame = %d", per)
	}

	cur := 0

	for f := 0; f < 120*RegenFrameHz; f++ {
		cur += ClampVitalDelta(cur, max, ManaRegenPerFrame(max, 120, 0, 0, false))
	}

	if cur != max {
		t.Errorf("after 120 s mana is %d of %d", cur, max)
	}
}

func TestClampVitalDelta(t *testing.T) {
	cases := []struct{ cur, max, delta, want int }{
		{100, 1000, 40, 40},
		{990, 1000, 40, 10},
		{1000, 1000, 40, 0},
		{1200, 1000, 40, -200 + 0}, // above the maximum: clamps down to it
		{30, 1000, -40, -30},
		{0, 1000, -5, 0},
	}

	for _, c := range cases {
		if got := ClampVitalDelta(c.cur, c.max, c.delta); got != c.want {
			t.Errorf("ClampVitalDelta(%d,%d,%d) = %d, want %d", c.cur, c.max, c.delta, got, c.want)
		}
	}
}

func TestLifeRegenStep(t *testing.T) {
	cases := []struct {
		name            string
		cur, max, regen int
		want            int
	}{
		{"no stat, no regeneration", 100 << 8, 1241 << 8, 0, 100 << 8},
		{"replenish life 10", 100 << 8, 1241 << 8, 10, 100<<8 + 10},
		{"capped at the maximum", 1241<<8 - 3, 1241 << 8, 10, 1241 << 8},
		{"below one point is lifted to one", 5, 1241 << 8, 1, 0x100},
		{"degeneration is floored at one point", 0x100, 1241 << 8, -50, 0x100},
		{"degeneration above the floor", 50 << 8, 1241 << 8, -50, 50<<8 - 50},
	}

	for _, c := range cases {
		if got := LifeRegenStep(c.cur, c.max, c.regen); got != c.want {
			t.Errorf("%s: LifeRegenStep = %d, want %d", c.name, got, c.want)
		}
	}

	// 10 points of replenish life are 10*25/256 life per second
	cur := 100 << 8
	for f := 0; f < RegenFrameHz*256; f++ { // 256 s
		cur = LifeRegenStep(cur, 1241<<8, 10)
	}

	if gain := (cur - 100<<8) >> 8; gain != 250 { // 256 s * 25 frames * 10 / 256
		t.Errorf("256 s of replenish life 10 gave %d life, want 250", gain)
	}
}

func TestRegenAdvance(t *testing.T) {
	in := RegenInput{MaxLife: 1241, MaxMana: 477, ManaRegenSeconds: 120}

	t.Run("mana rises by 40 raw per frame", func(t *testing.T) {
		var r Regen

		life, mana := 1241, 464

		if n := r.Advance(2, in, &life, &mana); n != 50 {
			t.Fatalf("frames = %d, want 50", n)
		}

		// 464*256 + 50*40 = 120784 -> 471 points and 208/256
		if mana != 471 || r.manaFrac != 208 || life != 1241 {
			t.Errorf("mana=%d frac=%d life=%d", mana, r.manaFrac, life)
		}
	})

	t.Run("empty to full in 122 s and then it stays", func(t *testing.T) {
		var r Regen

		life, mana := 1241, 0
		for i := 0; i < 130; i++ {
			r.Advance(1, in, &life, &mana)

			if i == 60 && (mana < 230 || mana > 240) { // 61 s * 25 * 40 / 256
				t.Errorf("after 61 s mana = %d", mana)
			}
		}

		if mana != 477 || r.manaFrac != 0 {
			t.Errorf("after 130 s mana = %d frac %d", mana, r.manaFrac)
		}
	})

	t.Run("small time steps add up to whole frames", func(t *testing.T) {
		var r Regen

		life, mana := 1241, 0
		for i := 0; i < 600; i++ { // 10 s at 60 Hz
			r.Advance(1.0/60, in, &life, &mana)
		}

		if r.TotalTicks < 249 || r.TotalTicks > 250 {
			t.Errorf("ticks = %d", r.TotalTicks)
		}

		// 250 frames * 40 raw / 256 = 39 points
		if mana < 38 || mana > 39 {
			t.Errorf("mana = %d after 10 s", mana)
		}
	})

	t.Run("regenerate mana percent speeds it up", func(t *testing.T) {
		var r Regen

		in := in
		in.ManaBonusPct = 100

		life, mana := 1241, 0
		r.Advance(10, in, &life, &mana)

		if mana != 78 { // 250 * 80 / 256 = 78.1
			t.Errorf("mana = %d", mana)
		}
	})

	t.Run("replenish life", func(t *testing.T) {
		var r Regen

		in := in
		in.LifeRegenRaw = 20

		life, mana := 1000, 477
		r.Advance(10, in, &life, &mana)

		if life != 1000+250*20/256 { // 19
			t.Errorf("life = %d", life)
		}
	})

	t.Run("no replenish life, no life", func(t *testing.T) {
		var r Regen

		life, mana := 500, 100
		r.Advance(60, in, &life, &mana)

		if life != 500 {
			t.Errorf("life moved to %d without the stat", life)
		}
	})

	t.Run("a long stall is bounded", func(t *testing.T) {
		var r Regen

		life, mana := 1241, 0
		if n := r.Advance(3600, in, &life, &mana); n != maxCatchUpFrames {
			t.Errorf("frames = %d", n)
		}
	})

	t.Run("no maxima, nothing moves", func(t *testing.T) {
		var r Regen

		life, mana := 7, 3
		r.Advance(5, RegenInput{ManaRegenSeconds: 120, LifeRegenRaw: 10}, &life, &mana)

		if life != 7 || mana != 3 {
			t.Errorf("life=%d mana=%d", life, mana)
		}
	})
}

// TestCharStatsManaRegen checks the table value the formula reads (needs
// D2_TABLES): every class has ManaRegen 120 in 1.14.
func TestCharStatsManaRegen(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CharStats.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(string(data), "\n")
	head := strings.Split(strings.TrimRight(lines[0], "\r"), "\t")
	col := -1

	for i, h := range head {
		if h == "ManaRegen" {
			col = i
		}
	}

	if col < 0 {
		t.Fatal("no ManaRegen column")
	}

	seen := 0

	for _, l := range lines[1:] {
		f := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(f) <= col || f[col] == "" || f[0] == "Expansion" {
			continue
		}

		seen++

		if f[col] != "120" {
			t.Errorf("%s ManaRegen = %q, want 120", f[0], f[col])
		}
	}

	if seen != 7 {
		t.Errorf("%d classes, want 7", seen)
	}
}
