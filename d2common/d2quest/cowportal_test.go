package d2quest

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestCowPortalCheck(t *testing.T) {
	tests := []struct {
		name   string
		exp    bool
		setup  func(g *Game)
		level  int
		wanted error
	}{
		{"classic, Diablo done, town", false, func(g *Game) { g.Rec.Set(26, FlagRewardGranted) }, 1, nil},
		{"classic, Baal only", false, func(g *Game) { g.Rec.Set(40, FlagRewardGranted) }, 1, ErrCowBossNotDone},
		{"expansion, Diablo only", true, func(g *Game) { g.Rec.Set(26, FlagRewardGranted) }, 1, ErrCowBossNotDone},
		{"expansion, Baal done", true, func(g *Game) { g.Rec.Set(40, FlagRewardGranted) }, 1, nil},
		{"outside town", true, func(g *Game) { g.Rec.Set(40, FlagRewardGranted) }, 40, ErrCowNotInTown},
		{"king killed", true, func(g *Game) { g.Rec.Set(40, FlagRewardGranted); g.Rec.Set(4, FlagCustom6) }, 1, ErrCowKingKilled},
		{"already made", true, func(g *Game) { g.Rec.Set(40, FlagRewardGranted); g.CowPortalMade() }, 1, ErrCowPortalMade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Game{Rec: &d2s.QuestRecord{}, Expansion: tt.exp}
			tt.setup(g)

			if err := g.CowPortalCheck(tt.level); !errors.Is(err, tt.wanted) {
				t.Fatalf("got %v, want %v", err, tt.wanted)
			}
		})
	}
}
