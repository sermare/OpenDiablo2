package d2inventory

import "testing"

func TestScoreSlotNeighbours(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int // grid
		fill         [][4]int
		x, y, iw, ih int
		want         int
	}{
		{"empty centre", 10, 4, nil, 4, 1, 1, 1, 0},
		{"top-left corner", 10, 4, nil, 0, 0, 1, 1, 2},
		{"one neighbour", 10, 4, [][4]int{{5, 1, 1, 1}}, 4, 1, 1, 1, 1},
		{"corners do not count", 10, 4, [][4]int{{3, 0, 1, 1}}, 4, 1, 1, 1, 0},
		{"surrounded 1x1 is perfect", 10, 4, [][4]int{{3, 1, 1, 1}, {5, 1, 1, 1}, {4, 0, 1, 1}, {4, 2, 1, 1}}, 4, 1, 1, 1, PerfectScore},
		{"corner cell with one neighbour is perfect", 10, 4, [][4]int{{1, 0, 1, 1}, {0, 1, 1, 1}}, 0, 0, 1, 1, PerfectScore},
		{"2x2 in corner", 10, 4, nil, 0, 0, 2, 2, 4},
		{"2x2 snug", 4, 4, [][4]int{{2, 0, 2, 2}, {0, 2, 2, 2}}, 0, 0, 2, 2, PerfectScore},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			g := NewOccupancyGrid(tt.w, tt.h)
			for _, f := range tt.fill {
				g.Fill(f[0], f[1], f[2], f[3], true)
			}

			if got := g.ScoreSlotNeighbours(tt.x, tt.y, tt.iw, tt.ih); got != tt.want {
				t.Errorf("got %d want %d", got, tt.want)
			}
		})
	}
}

func TestChooseStrategy(t *testing.T) {
	tests := []struct {
		w, h   int
		player bool
		want   SearchStrategy
	}{
		{1, 1, true, BestScoreColumnsDesc},
		{1, 1, false, FirstFitColumnsDesc},
		{2, 2, true, BestScoreRowMajor},
		{1, 3, true, BestScoreRowMajor},
		{2, 4, true, BestScoreRowMajor},
		{2, 4, false, FirstFitColumnMajor},
	}

	for _, tt := range tests {
		if got := ChooseStrategy(tt.w, tt.h, tt.player); got != tt.want {
			t.Errorf("ChooseStrategy(%d,%d,%v)=%d want %d", tt.w, tt.h, tt.player, got, tt.want)
		}
	}
}

func TestFindFreeSlot(t *testing.T) {
	tests := []struct {
		name   string
		fill   [][4]int
		w, h   int
		player bool
		wx, wy int
		wantOK bool
	}{
		// empty 10x4: columns right to left, bottom to top, strictly better wins
		// so the bottom-right corner (score 2) is found first and never beaten.
		{"1x1 goes to bottom right", nil, 1, 1, true, 9, 3, true},
		{"1x1 vendor first fit columns desc", nil, 1, 1, false, 9, 0, true},
		{"2x2 row major corner", nil, 2, 2, true, 0, 0, true},
		{"2x4 vendor column major", nil, 2, 4, false, 0, 0, true},
		{"1x1 snug beats corner", [][4]int{{9, 3, 1, 1}, {9, 1, 1, 1}}, 1, 1, true, 9, 2, true},
		{"too large", nil, 11, 1, true, 0, 0, false},
		{"full", [][4]int{{0, 0, 10, 4}}, 1, 1, true, 0, 0, false},
		{"touching neighbour fits", [][4]int{{0, 0, 9, 4}}, 1, 4, true, 9, 0, true},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			g := NewOccupancyGrid(10, 4)
			for _, f := range tt.fill {
				g.Fill(f[0], f[1], f[2], f[3], true)
			}

			x, y, ok := g.FindFreeSlot(tt.w, tt.h, tt.player)
			if ok != tt.wantOK || (ok && (x != tt.wx || y != tt.wy)) {
				t.Errorf("got (%d,%d,%v) want (%d,%d,%v)", x, y, ok, tt.wx, tt.wy, tt.wantOK)
			}
		})
	}
}

func TestCellsFreeTouching(t *testing.T) {
	g := NewOccupancyGrid(4, 2)
	g.Fill(0, 0, 2, 2, true)

	if !g.CellsFree(2, 0, 2, 2) {
		t.Error("an item touching its neighbour must fit")
	}

	if g.CellsFree(1, 0, 2, 2) {
		t.Error("overlapping rectangle must not fit")
	}
}
