package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func goldControls(level, gold, stash int) (*GameControls, *[]int) {
	hero := &d2mapentity.Player{Gold: gold, StashGold: stash, Stats: &d2hero.HeroStatsState{Level: level}}
	g := &GameControls{hero: hero, inventory: &Inventory{moveGoldPanel: &MoveGoldPanel{gold: gold}}}
	g.Logger = d2util.NewLogger()

	var dropped []int

	g.OnDropGold = func(n int) { dropped = append(dropped, n) }

	return g, &dropped
}

func TestGoldMove(t *testing.T) {
	tests := []struct {
		name                string
		kind                d2inventory.GoldDialogKind
		level, gold, stash  int
		amount              int
		wantGold, wantStash int
		wantDropped         int
		wantOK              bool
	}{
		{"drop some", d2inventory.GoldDialogDrop, 10, 5000, 0, 1200, 3800, 0, 1200, true},
		{"drop more than carried", d2inventory.GoldDialogDrop, 10, 5000, 0, 6000, 5000, 0, 0, false},
		{"deposit", d2inventory.GoldDialogDeposit, 10, 5000, 100, 4000, 1000, 4100, 0, true},
		{"deposit into a full stash", d2inventory.GoldDialogDeposit, 10, 5000, d2inventory.StashGoldLimit, 10, 5000, d2inventory.StashGoldLimit, 0, false},
		{"deposit fills the stash", d2inventory.GoldDialogDeposit, 94, 9000, d2inventory.StashGoldLimit - 4000, 9000, 5000, d2inventory.StashGoldLimit, 0, true},
		{"withdraw", d2inventory.GoldDialogWithdraw, 10, 100, 5000, 2000, 2100, 3000, 0, true},
		{"withdraw above the carry limit", d2inventory.GoldDialogWithdraw, 1, 9000, 5000, 2000, 9000, 5000, 0, false},
		{"withdraw more than the stash", d2inventory.GoldDialogWithdraw, 10, 0, 5000, 5001, 0, 5000, 0, false},
	}

	for _, tt := range tests {
		g, dropped := goldControls(tt.level, tt.gold, tt.stash)

		if got := g.GoldMove(tt.kind, tt.amount); got != tt.wantOK {
			t.Errorf("%s: ok %v, want %v", tt.name, got, tt.wantOK)
		}

		if g.hero.Gold != tt.wantGold || g.hero.StashGold != tt.wantStash || g.inventory.Gold() != tt.wantGold {
			t.Errorf("%s: gold %d (panel %d) stash %d, want %d/%d", tt.name, g.hero.Gold, g.inventory.Gold(), g.hero.StashGold,
				tt.wantGold, tt.wantStash)
		}

		sum := 0
		for _, n := range *dropped {
			sum += n
		}

		if sum != tt.wantDropped {
			t.Errorf("%s: dropped %d, want %d", tt.name, sum, tt.wantDropped)
		}
	}
}

func TestGoldDialogText(t *testing.T) {
	none := func(string) string { return "" }
	tests := []struct {
		kind d2inventory.GoldDialogKind
		want string
	}{
		{d2inventory.GoldDialogDrop, "How much Gold would you like to drop?"},
		{d2inventory.GoldDialogDeposit, "How much Gold would you like to deposit?"},
		{d2inventory.GoldDialogWithdraw, "How much Gold would you like to withdraw?"},
		{d2inventory.GoldDialogTrade, "How much Gold would you like to offer?"},
	}

	for _, tt := range tests {
		if got := GoldDialogText(none, tt.kind); got != tt.want {
			t.Errorf("kind %d: %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestMessageDialogLayout(t *testing.T) {
	r := MessageDialog(300, 170, 3, 32, 32)

	if len(r.TextBaselines) != 3 || r.TextBaselines[1]-r.TextBaselines[0] != goldDlgTextRowAdvance {
		t.Fatalf("text rows: %v", r.TextBaselines)
	}

	// the button is centred on the frame and stands where the OK button of the gold dialog does
	if r.Button.X != r.Frame.X+(300-32)/2 || r.Button.Y != goldDlgOkY-32 {
		t.Errorf("button %+v", r.Button)
	}

	if r.TextX != r.Frame.X+300/2+1 {
		t.Errorf("text x %d", r.TextX)
	}
}
