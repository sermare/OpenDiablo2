package d2hero

import "testing"

func TestExpPenalty(t *testing.T) {
	tests := []struct {
		name                             string
		percent, level, exp, start, next int
		wantLost, wantNew                int
	}{
		{"normal has none", 0, 10, 5000, 4000, 6000, 0, 5000},
		{"level 1 never loses", 10, 1, 100, 0, 500, 0, 100},
		{"nightmare 5%", 5, 10, 5000, 4000, 6000, 100, 4900},
		{"hell 10%", 10, 10, 5000, 4000, 6000, 200, 4800},
		{"clamped above start", 10, 10, 4050, 4000, 6000, 49, 4001},
		{"at start of level", 10, 10, 4000, 4000, 6000, 0, 4000},
		{"degenerate span", 10, 10, 5000, 6000, 6000, 0, 5000},
	}

	for _, tc := range tests {
		lost, n := ExpPenalty(tc.percent, tc.level, tc.exp, tc.start, tc.next)
		if lost != tc.wantLost || n != tc.wantNew {
			t.Errorf("%s: lost=%d new=%d, want %d %d", tc.name, lost, n, tc.wantLost, tc.wantNew)
		}
	}
}

func TestDeathAndRespawn(t *testing.T) {
	var s DeathState

	out := s.Die(DeathInput{Difficulty: 1, Level: 10, Experience: 5000, Gold: 1234, ExpStart: 4000, ExpNext: 6000, X: 7, Y: 9})
	if out.GoldDropped != 1234 || out.ExpLost != 100 || out.NewExperience != 4900 || out.CharacterDead {
		t.Fatalf("outcome %+v", out)
	}

	if !s.Died || s.Deaths != 1 || s.Corpse == nil || s.Corpse.X != 7 || s.Corpse.Y != 9 {
		t.Fatalf("state %+v", s)
	}

	if !s.Respawn(false) || s.Died {
		t.Fatal("softcore must respawn and clear the died flag")
	}

	if s.Corpse == nil {
		t.Fatal("the corpse stays after the respawn")
	}

	if s.Recover() == nil || s.Corpse != nil || s.Recover() != nil {
		t.Fatal("recover once")
	}
}

func TestHardcoreDeath(t *testing.T) {
	var s DeathState

	out := s.Die(DeathInput{Hardcore: true, Level: 5, Experience: 10, ExpStart: 0, ExpNext: 100})
	if !out.CharacterDead || !s.Died {
		t.Fatalf("hardcore death must be final: %+v %+v", out, s)
	}

	if s.Respawn(true) || !s.Died {
		t.Fatal("hardcore cannot respawn")
	}
}

func TestRespawnLife(t *testing.T) {
	for in, want := range map[int]int{0: 1, 1: 1, 100: 50, 3: 1, 4: 2} {
		if got := RespawnLife(in); got != want {
			t.Errorf("RespawnLife(%d) = %d, want %d", in, got, want)
		}
	}
}
