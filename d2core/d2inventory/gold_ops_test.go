package d2inventory

import "testing"

func TestDropGold(t *testing.T) {
	tests := []struct {
		name                  string
		carried, amount, max  int
		wantCarried, wantDrop int
		wantOK                bool
	}{
		{"some", 5000, 1200, 100000, 3800, 1200, true},
		{"all", 5000, 5000, 100000, 0, 5000, true},
		{"zero is accepted", 5000, 0, 100000, 5000, 0, true},
		{"more than carried", 5000, 5001, 100000, 5000, 0, false},
		{"negative", 5000, -1, 100000, 5000, 0, false},
		{"above the carry limit", 900000, 800000, 10000 * 10, 900000, 0, false},
	}

	for _, tt := range tests {
		c, d, ok := DropGold(tt.carried, tt.amount, tt.max)
		if c != tt.wantCarried || d != tt.wantDrop || ok != tt.wantOK {
			t.Errorf("%s: got %d/%d/%v, want %d/%d/%v", tt.name, c, d, ok, tt.wantCarried, tt.wantDrop, tt.wantOK)
		}
	}
}

func TestDepositGold(t *testing.T) {
	const lim = StashGoldLimit

	tests := []struct {
		name                string
		carried, stash, amt int
		wantC, wantS        int
		wantOK              bool
	}{
		{"plain", 9000, 100, 4000, 5000, 4100, true},
		{"all", 9000, 0, 9000, 0, 9000, true},
		{"more than carried", 9000, 0, 9001, 9000, 0, false},
		{"zero", 9000, 0, 0, 9000, 0, false},
		{"exactly full", 5000, lim - 5000, 5000, 0, lim, true},
		{"partial fill", 5000, lim - 2000, 5000, 3000, lim, true},
		{"stash full", 5000, lim, 100, 5000, lim, false},
	}

	for _, tt := range tests {
		c, s, ok := DepositGold(tt.carried, tt.stash, tt.amt, lim)
		if c != tt.wantC || s != tt.wantS || ok != tt.wantOK {
			t.Errorf("%s: got %d/%d/%v, want %d/%d/%v", tt.name, c, s, ok, tt.wantC, tt.wantS, tt.wantOK)
		}
	}
}

func TestWithdrawGold(t *testing.T) {
	tests := []struct {
		name                string
		carried, stash, amt int
		max                 int
		wantC, wantS        int
		wantOK              bool
	}{
		{"plain", 100, 5000, 2000, 940000, 2100, 3000, true},
		{"all", 0, 5000, 5000, 940000, 5000, 0, true},
		{"more than stash", 0, 5000, 5001, 940000, 0, 5000, false},
		{"zero", 0, 5000, 0, 940000, 0, 5000, false},
		{"fills the purse exactly", 8000, 5000, 2000, 10000, 10000, 3000, true},
		{"over the carry limit is refused whole", 8000, 5000, 2001, 10000, 8000, 5000, false},
	}

	for _, tt := range tests {
		c, s, ok := WithdrawGold(tt.carried, tt.stash, tt.amt, tt.max)
		if c != tt.wantC || s != tt.wantS || ok != tt.wantOK {
			t.Errorf("%s: got %d/%d/%v, want %d/%d/%v", tt.name, c, s, ok, tt.wantC, tt.wantS, tt.wantOK)
		}
	}
}

func TestGoldPickup(t *testing.T) {
	max := InventoryGoldLimit(10)

	tests := []struct {
		name                string
		carried, pile       int
		wantTaken, wantLeft int
	}{
		{"fits", 1000, 500, 500, 0},
		{"overflow stays", max - 1000, 5000, 1000, 4000},
		{"full purse takes nothing", max, 300, 0, 300},
	}

	for _, tt := range tests {
		taken, left := GoldPickup(tt.carried, tt.pile, max)
		if taken != tt.wantTaken || left != tt.wantLeft {
			t.Errorf("%s: got %d/%d, want %d/%d", tt.name, taken, left, tt.wantTaken, tt.wantLeft)
		}
	}
}

func TestGoldDialogRules(t *testing.T) {
	if GoldDialogLimit(GoldDialogDrop, 500, 900) != 500 || GoldDialogLimit(GoldDialogDeposit, 500, 900) != 500 ||
		GoldDialogLimit(GoldDialogWithdraw, 500, 900) != 900 {
		t.Error("limits: carried for drop and deposit, stash for withdraw")
	}

	if !GoldDialogPrefill(GoldDialogDeposit) || GoldDialogPrefill(GoldDialogDrop) {
		t.Error("only the deposit opens prefilled")
	}

	tests := []struct {
		in    string
		limit int
		want  int
	}{
		{"1234", 5000, 1234},
		{"9999", 5000, 5000},
		{"", 5000, 0},
		{"12ab3", 5000, 123},
		{"123456789012", 2000000000, 1234567890},
	}

	for _, tt := range tests {
		if got := ClampGoldInput(tt.in, tt.limit); got != tt.want {
			t.Errorf("ClampGoldInput(%q,%d) = %d, want %d", tt.in, tt.limit, got, tt.want)
		}
	}

	if GoldDialogRequest(GoldDialogDrop, 0) || !GoldDialogRequest(GoldDialogDrop, 1) || !GoldDialogRequest(GoldDialogTrade, 0) {
		t.Error("OK with 0 only closes, except in the trade dialog")
	}
}
