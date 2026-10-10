package d2combat

import "testing"

func TestPvPDefendCarryAddsUpSmallTicks(t *testing.T) {
	var c PvPDefendCarry

	got := 0
	for i := 0; i < 8; i++ {
		got += c.ReceiveParts(PvPParts{Fire: 1}, PvPDefender{FireResist: 75})
	}

	if got != 2 {
		t.Fatalf("8 one point ticks at 75 percent resist took %d, want 2", got)
	}

	if n := (&PvPDefendCarry{}).ReceiveParts(PvPParts{Fire: 100}, PvPDefender{FireResist: 75}); n != 25 {
		t.Fatalf("a big hit took %d, want 25", n)
	}
}
