package d2monster

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func TestPlanGroupSizesAndDeterminism(t *testing.T) {
	info := ClassInfo{Class: 19, MinGrp: 3, MaxGrp: 5, Minion1: -1, Minion2: -1}
	counts := map[int]int{}

	for seed := uint32(1); seed <= 300; seed++ {
		p := PlanGroup(d2rand.New(seed), info)
		n := len(p.Members)

		if n < 3 || n > 5 {
			t.Fatalf("seed %d: group of %d outside 3..5", seed, n)
		}

		counts[n]++

		again := PlanGroup(d2rand.New(seed), info)
		if len(again.Members) != n {
			t.Fatalf("seed %d: not deterministic", seed)
		}
	}

	for n := 3; n <= 5; n++ {
		if counts[n] == 0 {
			t.Errorf("size %d never produced: %v", n, counts)
		}
	}
}

func TestPlanGroupMinimumOne(t *testing.T) {
	p := PlanGroup(d2rand.New(1), ClassInfo{Class: 3, Minion1: -1, Minion2: -1})
	if len(p.Members) != 1 || p.Members[0].Minion {
		t.Fatalf("empty Grp columns must give one monster, got %+v", p.Members)
	}
}

func TestPlanGroupMinions(t *testing.T) {
	// class 0 is a real class, -1 means unset
	info := ClassInfo{Class: 9, MinGrp: 1, MaxGrp: 1, Minion1: 0, Minion2: 4, PartyMin: 2, PartyMax: 4}
	seen := map[int]bool{}

	for seed := uint32(1); seed <= 200; seed++ {
		p := PlanGroup(d2rand.New(seed), info)
		if p.Members[0].Minion || p.Members[0].Class != 9 {
			t.Fatal("leader must come first")
		}

		m := len(p.Members) - 1
		if m < 2 || m > 4 {
			t.Fatalf("party of %d outside 2..4", m)
		}

		for _, x := range p.Members[1:] {
			if !x.Minion || (x.Class != 0 && x.Class != 4) {
				t.Fatalf("bad follower %+v", x)
			}

			seen[x.Class] = true
		}
	}

	if !seen[0] || !seen[4] {
		t.Errorf("both minion classes should occur: %v", seen)
	}

	one := ClassInfo{Class: 9, MinGrp: 1, MaxGrp: 1, Minion1: 7, Minion2: 7, PartyMin: 1, PartyMax: 1}
	if got := minionClasses(one); len(got) != 1 {
		t.Errorf("duplicate minion classes should collapse: %v", got)
	}
}

func TestPlanSuperUniqueCountess(t *testing.T) {
	// The Countess: corruptrogue3 (class 45) with minion1/2 = corruptrogue1/4, 6 followers.
	info := ClassInfo{Class: 45, Minion1: 43, Minion2: 46}
	p := PlanSuperUnique(d2rand.New(5), "The Countess", info, 6, 6)

	if len(p.Members) != 7 || !p.Members[0].Unique || p.Members[0].Class != 45 {
		t.Fatalf("countess pack: %+v", p.Members)
	}

	for _, m := range p.Members[1:] {
		if !m.Minion || (m.Class != 43 && m.Class != 46) {
			t.Errorf("follower %+v", m)
		}
	}

	// no minion columns: followers copy the class
	q := PlanSuperUnique(d2rand.New(5), "x", ClassInfo{Class: 8, Minion1: -1, Minion2: -1}, 2, 2)
	if len(q.Members) != 3 || q.Members[1].Class != 8 {
		t.Errorf("fallback followers: %+v", q.Members)
	}
}

func TestPickLevelTypes(t *testing.T) {
	var list []ClassInfo

	for i := 0; i < 20; i++ {
		list = append(list, ClassInfo{Class: i, IsSpawn: i != 5, Ranged: i == 3 || i == 7})
	}

	got := PickLevelTypes(d2rand.New(9), list, 99, false)
	if len(got) != MaxLevelTypes {
		t.Fatalf("expected the cap of %d, got %d", MaxLevelTypes, len(got))
	}

	seen := map[int]bool{}

	for _, c := range got {
		if c.Class == 5 || seen[c.Class] {
			t.Fatalf("non-spawnable or duplicate class %d", c.Class)
		}

		seen[c.Class] = true
	}

	if n := len(PickLevelTypes(d2rand.New(9), list, 4, false)); n != 4 {
		t.Errorf("nmon limit: %d", n)
	}

	for seed := uint32(1); seed < 50; seed++ {
		first := PickLevelTypes(d2rand.New(seed), list, 3, true)[0]
		if !first.Ranged {
			t.Fatalf("seed %d: first pick %d is not ranged", seed, first.Class)
		}
	}

	if got := PickLevelTypes(d2rand.New(1), nil, 5, false); len(got) != 0 {
		t.Error("empty list")
	}
}

func TestPickByRarityWeights(t *testing.T) {
	types := []ClassInfo{{Class: 1, Rarity: 9}, {Class: 2, Rarity: 1}, {Class: 3, Rarity: 0}}
	r := d2rand.New(77)
	n := map[int]int{}

	for i := 0; i < 3000; i++ {
		c, ok := PickByRarity(r, types)
		if !ok {
			t.Fatal("no pick")
		}

		n[c.Class]++
	}

	if n[1] < n[2]*4 || n[3] == 0 || n[2] == 0 {
		t.Errorf("weights not respected: %v", n)
	}

	if _, ok := PickByRarity(r, nil); ok {
		t.Error("empty must fail")
	}
}

func TestGroupsForRoomAndAverage(t *testing.T) {
	for _, c := range []struct{ tiles, den, avg, want int }{
		{0, 3000, 4, 0}, {400, 0, 4, 0}, {1600, 3000, 4, 12}, {1600, 3000, 0, 48},
	} {
		if got := GroupsForRoom(c.tiles, c.den, c.avg); got != c.want {
			t.Errorf("GroupsForRoom(%d,%d,%d)=%d want %d", c.tiles, c.den, c.avg, got, c.want)
		}
	}

	if got := AvgGroupSize([]ClassInfo{{MinGrp: 2, MaxGrp: 4, Rarity: 1}, {MinGrp: 4, MaxGrp: 6, Rarity: 1}}); got != 4 {
		t.Errorf("avg = %d", got)
	}

	if AvgGroupSize(nil) != 1 {
		t.Error("avg default")
	}
}

func TestResolveLevel(t *testing.T) {
	for _, c := range []struct {
		info          ClassInfo
		stat, area, w int
	}{
		{ClassInfo{}, 5, 30, 30},
		{ClassInfo{NoRatio: true}, 5, 30, 5},
		{ClassInfo{Boss: true}, 5, 30, 5},
		{ClassInfo{}, 5, 0, 5},
	} {
		if got := ResolveLevel(c.info, c.stat, c.area); got != c.w {
			t.Errorf("%+v: %d want %d", c.info, got, c.w)
		}
	}
}

func TestLeaderSuccessor(t *testing.T) {
	if LeaderSuccessor(ClassInfo{}, []bool{true}) != -1 {
		t.Error("no BossXfer, no successor")
	}

	if got := LeaderSuccessor(ClassInfo{BossXfer: true}, []bool{false, true, true}); got != 1 {
		t.Errorf("successor = %d", got)
	}

	if LeaderSuccessor(ClassInfo{BossXfer: true}, []bool{false}) != -1 {
		t.Error("nobody alive")
	}
}
