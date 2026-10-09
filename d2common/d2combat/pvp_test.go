package d2combat

import "testing"

func TestPvPScale(t *testing.T) {
	if PvPPercent() != 17 {
		t.Fatalf("player on player scale is %d, want 17", PvPPercent())
	}

	// a monster attacker keeps 100, a player-controlled attacker on a plain monster too
	if got := CombatantScalePercent(false, false, true, false, false); got != 100 {
		t.Fatalf("monster on player: %d", got)
	}

	tests := []struct{ raw, want int }{{0, 0}, {-5, 0}, {1, 0}, {100, 17}, {1000, 170}, {59, 10}}
	for _, tc := range tests {
		if got := PvPDamage(tc.raw); got != tc.want {
			t.Errorf("PvPDamage(%d) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestPvPReceive(t *testing.T) {
	tests := []struct{ scaled, res, reduce, want int }{
		{100, 0, 0, 100},
		{100, 50, 0, 50},
		{100, 0, 30, 70},
		{100, 50, 30, 35},
		{10, 0, 30, 0}, // never negative
		{100, 100, 0, 0},
	}

	for _, tc := range tests {
		if got := PvPReceive(tc.scaled, tc.res, tc.reduce); got != tc.want {
			t.Errorf("PvPReceive(%d,%d,%d) = %d, want %d", tc.scaled, tc.res, tc.reduce, got, tc.want)
		}
	}
}

func TestPvPEar(t *testing.T) {
	if PvPKillGivesEar(false) || !PvPKillGivesEar(true) {
		t.Fatal("ears only for hardcore victims")
	}
}
