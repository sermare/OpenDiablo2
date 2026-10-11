package d2skill

import "testing"

func TestNextAfterRotation(t *testing.T) {
	ids := []string{"a", "c", "e"}
	cases := []struct {
		last string
		want int
	}{{"", 0}, {"a", 1}, {"b", 1}, {"c", 2}, {"e", 0}, {"z", 0}}

	for _, c := range cases {
		if got := NextAfter(ids, c.last); got != c.want {
			t.Errorf("after %q = %d, want %d", c.last, got, c.want)
		}
	}

	if NextAfter(nil, "a") != -1 {
		t.Error("empty must give -1")
	}
}

func TestStaticFieldDamage(t *testing.T) {
	cases := []struct {
		name                          string
		hp, maxHP, pct, minRaw, floor int
		want                          int
	}{
		{"quarter of current", 400, 400, 25, 0, 0, 100},
		{"at the floor does nothing", 132, 400, 25, 0, 33, 0},
		{"just above the floor may cross it", 133, 400, 25, 0, 33, 33},
		{"never kills", 10, 400, 100, 0, 0, 9},
		{"minimum in 8.8", 8, 400, 10, 2 << 8, 0, 2},
		{"dead input", 0, 400, 25, 0, 0, 0},
	}
	for _, c := range cases {
		if got := StaticFieldDamage(c.hp, c.maxHP, c.pct, c.minRaw, c.floor); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
