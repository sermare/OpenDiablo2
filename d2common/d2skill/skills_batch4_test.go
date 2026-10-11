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

func TestPoisonExplosionLeavesEightClouds(t *testing.T) {
	cf := newClassFixture(map[string]int{"Poison Explosion": 5})
	id := cf.id("Poison Explosion")
	tg := Target{X: 20, Y: 20, Corpse: true, CX: 20, CY: 20, CorpseID: "c", CorpseHP: 100}

	if st := cf.p.Start(cf.u, id, tg); !st.OK {
		t.Fatalf("start: %+v", st)
	}

	r := cf.p.Do(cf.u, id, tg)
	if len(r.Missiles) != 8 {
		t.Fatalf("%d clouds, want 8", len(r.Missiles))
	}

	e := effectOf(t, r, "area_hit")
	if e.CorpseID != "c" || e.Desc != nil {
		t.Errorf("the corpse is only consumed: %+v", e)
	}

	seen := map[[2]int]bool{}

	for _, m := range r.Missiles {
		seen[[2]int{int(m.X), int(m.Y)}] = true
	}

	if len(seen) != 8 {
		t.Errorf("clouds on %d distinct cells", len(seen))
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
