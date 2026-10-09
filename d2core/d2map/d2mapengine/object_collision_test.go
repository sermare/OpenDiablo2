package d2mapengine

import "testing"

func TestObjectCollision(t *testing.T) {
	c := NewObjectCollision()

	c.Set("door", 10, 10, 1, 3)
	c.Set("other", 10, 11, 2, 1) // overlaps the door at (10,11)

	cases := []struct {
		x, y int
		want bool
	}{
		{10, 9, false}, {10, 10, true}, {10, 11, true}, {10, 12, true}, {10, 13, false},
		{11, 11, true}, {11, 10, false}, {9, 11, false},
	}
	for _, tc := range cases {
		if got := c.Blocked(tc.x, tc.y); got != tc.want {
			t.Errorf("Blocked(%d,%d) = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}

	c.Clear("door")

	if c.Blocked(10, 10) || !c.Blocked(10, 11) {
		t.Error("clearing the door must leave the overlapping owner's cells blocked")
	}

	c.Clear("door") // idempotent
	c.Set("other", 50, 50, 1, 1)

	if c.Blocked(10, 11) || !c.Blocked(50, 50) || c.Owners() != 1 {
		t.Error("Set must replace the owner's old rectangle")
	}

	c.Clear("other")

	if c.Owners() != 0 || len(c.count) != 0 {
		t.Error("overlay should be empty")
	}
}
