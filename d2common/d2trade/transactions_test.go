package d2trade

import "testing"

func TestResultMessage(t *testing.T) {
	tests := []struct {
		code int
		key  string
	}{
		{ResultBought, ""}, {ResultSold, ""}, {ResultDone, ""}, {ResultStacked, ""}, {6, ""},
		{7, "TransactionResults6"}, {8, "TransactionResults6"}, {ResultInvalid, "TransactionResults9"},
		{ResultNoRoom, "TransactionResults4"}, {ResultNoItem, "TransactionResults8"},
		{ResultNoGold, "TransactionResults10"}, {ResultMercBusy, "TransactionResults11"},
		{ResultMercMax, "TransactionResults5"}, {0x20, "TransactionResults9"},
	}

	for _, tt := range tests {
		key, text := ResultMessage(tt.code)
		if key != tt.key || (key != "") != (text != "") {
			t.Errorf("code %#x: key %q text %q, want key %q", tt.code, key, text, tt.key)
		}
	}
}

func TestStackBuy(t *testing.T) {
	tests := []struct {
		name             string
		unit, gold, room int
		topUp            bool
		qty, cost        int
		ok               bool
	}{
		{"gold covers a full new stack", 2, 1000, 350, false, 350, 700, true},
		{"gold limits the new stack", 2, 101, 350, false, 50, 100, true},
		{"top up limited by the missing quantity", 2, 1000, 30, true, 30, 60, true},
		{"top up cannot afford one", 5, 4, 30, true, 0, 0, false},
		{"new stack cannot afford one still asks for one", 5, 4, 350, false, 1, 5, true},
		{"zero unit price is treated as 1", 0, 10, 100, false, 10, 10, true},
	}

	for _, tt := range tests {
		qty, cost, ok := StackBuy(tt.unit, tt.gold, tt.room, tt.topUp)
		if qty != tt.qty || cost != tt.cost || ok != tt.ok {
			t.Errorf("%s: got (%d,%d,%v) want (%d,%d,%v)", tt.name, qty, cost, ok, tt.qty, tt.cost, tt.ok)
		}
	}
}

func TestMultiBuy(t *testing.T) {
	for _, shift := range []bool{false, true} {
		for _, col := range []bool{false, true} {
			for _, sells := range []bool{false, true} {
				if got, want := MultiBuy(shift, col, sells), shift && col && sells; got != want {
					t.Errorf("MultiBuy(%v,%v,%v)=%v", shift, col, sells, got)
				}
			}
		}
	}
}
