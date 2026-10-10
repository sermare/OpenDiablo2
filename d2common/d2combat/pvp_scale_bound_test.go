package d2combat

import "testing"

// lcg is a small deterministic generator for the sequences below.
type lcg uint32

func (l *lcg) next(n int) int {
	*l = *l*1664525 + 1013904223

	return int((uint32(*l) >> 8) % uint32(n))
}

// TestPvPScaleSingleHitExact: without a carry the scale is floor(raw*17/100) per type and never above the exact
// 17 percent; the table has the raws of 9f-pvp-skills-ear (Fire Ball ~1070, Meteor 2450, Blizzard ~1550, a burning
// ground tick ~5 whole points). No slack: the result is the floor, within one point below the exact value.
func TestPvPScaleSingleHitExact(t *testing.T) {
	for _, raw := range []int{1, 2, 5, 6, 16, 17, 58, 59, 100, 101, 1051, 1070, 1149, 1521, 1639, 2450, 100000} {
		got := (PvPParts{Fire: raw}).Scale(17).Fire
		if want := raw * 17 / 100; got != want {
			t.Errorf("Scale(%d) = %d, want floor(%d*17/100) = %d", raw, got, raw, want)
		}

		if got*100 > raw*17 || raw*17-got*100 >= 100 {
			t.Errorf("Scale(%d) = %d is not within [exact-1, exact]", raw, got)
		}
	}
}

// TestPvPCarryTwoHits: the first hit of a type is the floor of the exact scale (nothing is carried yet) and a second
// identical hit pays at most the fraction the first one left, under one point more.
func TestPvPCarryTwoHits(t *testing.T) {
	for _, raw := range []int{1, 5, 6, 16, 17, 58, 59, 100, 1070, 2450} {
		var c PvPCarry

		v := int32(raw) << 8

		first := c.Scale([5]int32{0, v}, 17).Fire
		if want := int(v*17/100) >> 8; first != want {
			t.Errorf("first hit of %d charged %d, want %d", raw, first, want)
		}

		if first > raw*17/100 {
			t.Errorf("first hit of %d charged %d, above floor(raw*17/100) = %d", raw, first, raw*17/100)
		}

		second := c.Scale([5]int32{0, v}, 17).Fire
		if second > raw*17/100+1 {
			t.Errorf("second hit of %d charged %d, more than floor+1 = %d", raw, second, raw*17/100+1)
		}
	}
}

// TestPvPCarryBoundsHold: for deterministic random sequences of 8.8 hits the charged total of every type is inside
// PvPScaledBounds after every hit.
func TestPvPCarryBoundsHold(t *testing.T) {
	tests := []struct {
		name   string
		lo, hi int // 8.8 value range of one hit
		hits   int
		types  []int // which of the 5 types receive the hits
	}{
		{"burning ground tick 3-5.5 fire", 3 * 256, 5*256 + 128, 700, []int{1}},
		{"fire ball 1050-1150 fire", 1050 * 256, 1150 * 256, 40, []int{1}},
		{"blizzard 1510-1640 cold", 1510 * 256, 1640 * 256, 40, []int{4}},
		{"mixed phys and fire", 100, 20000, 500, []int{0, 1}},
		{"tiny hits of under a point", 1, 255, 1000, []int{0, 1, 2, 3, 4}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := lcg(7)

			var (
				c       PvPCarry
				sum     [5]int64
				tsum    [5]int64
				charged [5]int
			)

			for h := 1; h <= tc.hits; h++ {
				var v [5]int32

				for _, ty := range tc.types {
					v[ty] = int32(tc.lo + r.next(tc.hi-tc.lo+1))
					sum[ty] += int64(v[ty])
				}

				out := c.Scale(v, 17).Slice()
				for ty := range out {
					charged[ty] += out[ty]

					// the awk in 9f-pvp-skills-ear.sh checks this exact form: 256*charged <= t < 256*charged + 256
					tsum[ty] += int64(v[ty]) * 17 / 100
					if int64(charged[ty])*256 > tsum[ty] || int64(charged[ty])*256+256 <= tsum[ty] {
						t.Fatalf("hit %d type %d: charged %d vs t=%d/256 breaks 256*charged <= t < 256*charged+256", h, ty, charged[ty], tsum[ty])
					}

					lo, hi := PvPScaledBounds(17, sum[ty], h)
					if charged[ty] < lo || charged[ty] > hi {
						t.Fatalf("hit %d type %d: charged %d outside exact bounds [%d, %d] (sum88=%d)", h, ty, charged[ty], lo, hi, sum[ty])
					}
				}
			}
		})
	}
}

// TestPvPScaledBoundsTable pins the bound arithmetic on values worked out by hand.
func TestPvPScaledBoundsTable(t *testing.T) {
	tests := []struct {
		pct    int
		sum88  int64
		n      int
		lo, hi int
	}{
		{17, 0, 0, 0, 0},
		{17, 2450 * 256, 1, 416, 416},   // Meteor impact: exact 416.5, nothing carried yet
		{17, 1070 * 256, 1, 181, 181},   // Fire Ball: exact 181.9
		{17, 100 * 256, 1, 16, 17},      // exactly 17.0 (lo is conservative by the n/256 truncation term)
		{17, 628 * 1250, 628, 518, 521}, // 628 ticks of 4.88: exact 521.3
		{100, 256 * 5, 1, 4, 5},
	}

	for _, tc := range tests {
		lo, hi := PvPScaledBounds(tc.pct, tc.sum88, tc.n)
		if lo != tc.lo || hi != tc.hi {
			t.Errorf("PvPScaledBounds(%d,%d,%d) = [%d,%d], want [%d,%d]", tc.pct, tc.sum88, tc.n, lo, hi, tc.lo, tc.hi)
		}
	}
}
