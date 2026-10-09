package d2player

import (
	"errors"
	"testing"
)

func TestSettle(t *testing.T) {
	tests := []struct {
		name        string
		gold, price int
		want        int
		err         error
	}{
		{"exact", 100, 100, 0, nil},
		{"change", 500, 35, 465, nil},
		{"too expensive", 34, 35, 34, ErrNotEnoughGold},
		{"free", 0, 0, 0, nil},
		{"negative price rejected", 10, -5, 10, ErrNotEnoughGold},
	}

	for _, tt := range tests {
		got, err := settle(tt.gold, tt.price)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%s: settle(%d,%d)=(%d,%v) want (%d,%v)", tt.name, tt.gold, tt.price, got, err, tt.want, tt.err)
		}
	}
}
