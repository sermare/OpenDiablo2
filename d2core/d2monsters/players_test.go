package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

func TestPlayerCount(t *testing.T) {
	for _, c := range []struct {
		static, live, forced, want int
		useLive                    bool
	}{
		{0, 0, 0, 1, false},
		{1, 0, 0, 1, false},
		{0, 3, 0, 3, true},
		{0, 0, 0, 1, true},  // a live count of 0 is still one player
		{0, 2, 5, 5, true},  // forced wins when larger
		{0, 4, 2, 4, true},  // real count wins when larger
		{0, 0, 8, 8, false}, // single player with players 8
	} {
		d := difficultyDirector(d2monster.Normal, 1)
		d.opt.Players, d.opt.ForcedPlayers = c.static, c.forced

		if c.useLive {
			n := c.live
			d.opt.PlayersFunc = func() int { return n }
		}

		if got := d.PlayerCount(); got != c.want {
			t.Errorf("%+v: got %d", c, got)
		}
	}
}

func TestForcedPlayersFromEnv(t *testing.T) {
	for in, want := range map[string]int{"": 0, "x": 0, "-2": 0, "0": 0, "3": 3, " 5 ": 5, "99": 8} {
		if got := ForcedPlayersFromEnv(in); got != want {
			t.Errorf("%q: %d want %d", in, got, want)
		}
	}
}

// Players=1 (and the unset default) reproduces the unscaled values, and a
// count that changes later only affects monsters computed afterwards.
func TestPlayersOnlyAffectLaterSpawns(t *testing.T) {
	roll := func(d *Director) (int, int) {
		b := d2monster.NewBrain(1, 1, d2monster.Normal, &d2monster.Profile{}, 1)
		v := d.computeVitals(ratioStat(), b)

		return v.MaxHP, v.Experience
	}

	base := difficultyDirector(d2monster.Normal, 1)
	hp0, xp0 := roll(base)

	n := 1
	d := difficultyDirector(d2monster.Normal, 1)
	d.opt.PlayersFunc = func() int { return n }

	if hp, xp := roll(d); hp != hp0 || xp != xp0 || hp != 7 || xp != 30 {
		t.Fatalf("players=1: %d %d, want %d %d", hp, xp, hp0, xp0)
	}

	n = 3
	if hp, xp := roll(d); hp != 14 || xp != 60 {
		t.Errorf("players=3: %d %d", hp, xp)
	}

	n = 1
	if hp, xp := roll(d); hp != hp0 || xp != xp0 {
		t.Errorf("back to 1: %d %d", hp, xp)
	}

	d.opt.ForcedPlayers = 3
	if hp, xp := roll(d); hp != 14 || xp != 60 {
		t.Errorf("forced 3: %d %d", hp, xp)
	}
}
