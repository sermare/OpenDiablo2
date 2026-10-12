package d2player

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
)

// The Repair All strip belongs to the regular shop only: the gamble window of a vendor that repairs
// (Jamella, Halbu) has none (w51).
func TestShowsRepair(t *testing.T) {
	tests := []struct {
		name    string
		repairs bool
		gamble  bool
		want    bool
	}{
		{"repairing vendor, shop", true, false, true},
		{"repairing vendor, gamble", true, true, false},
		{"plain vendor, shop", false, false, false},
		{"plain vendor, gamble", false, true, false},
	}

	for _, tc := range tests {
		tw := &TradeWindow{vendor: d2vendor.Vendor{Repairs: tc.repairs}, gamble: tc.gamble}

		if got := tw.showsRepair(); got != tc.want {
			t.Errorf("%s: showsRepair=%v want %v", tc.name, got, tc.want)
		}
	}

	tw := &TradeWindow{vendor: d2vendor.Vendor{Repairs: true}, gamble: true}
	if _, err := tw.RepairAll(); !errors.Is(err, ErrNotRepairer) {
		t.Errorf("RepairAll in a gamble window: %v, want ErrNotRepairer", err)
	}
}
