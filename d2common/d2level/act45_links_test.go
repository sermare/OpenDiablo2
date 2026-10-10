package d2level

import "testing"

// The levels of the Act 4 / Act 5 playthrough (scripts/verify.d/9h-act45-playthrough.sh): bugs found by
// playing them are pinned here (docs/PLAYTEST.md, "Act 4 and 5").

func TestAct45EdgeNeighbours(t *testing.T) {
	cases := []struct {
		level int
		want  []int
	}{
		{103, []int{104}}, {104, []int{103, 105}}, {105, []int{104, 106}}, {106, []int{105}},
		{109, []int{110}}, {110, []int{109, 111}}, {111, []int{110, 112}}, {112, []int{111}},
		{117, nil}, {107, nil}, {108, nil},
	}

	for _, c := range cases {
		got := map[int]bool{}
		for _, n := range EdgeNeighbors(c.level) {
			got[n] = true
		}

		if len(got) != len(c.want) {
			t.Errorf("level %d: edge neighbours %v, want %v", c.level, got, c.want)
		}

		for _, w := range c.want {
			if !got[w] {
				t.Errorf("level %d: no edge to %d", c.level, w)
			}
		}
	}
}

func TestSlotDestination(t *testing.T) {
	cases := []struct{ level, style, want int }{
		{107, 0, 106},                               // River of Flame: the south room goes back to the City of the Damned
		{113, 0, 112}, {113, 1, 115}, {113, 2, 114}, // Crystalline Passage: up / ahead / down floor
		{115, 0, 113}, {115, 1, 117}, {115, 2, 116}, // Glacial Trail
		{118, 0, 117}, {118, 1, 120}, {118, 2, 119}, // Ancients' Way
		{114, 0, 113}, {116, 0, 115}, {119, 0, 118}, // the single room caves
		{120, 0, 118}, {120, 1, 128}, // Arreat Summit: to the ice caves, to the Worldstone Keep
		{128, 0, 120}, {128, 1, 129}, {129, 0, 128}, {129, 1, 130}, {130, 0, 129}, {130, 1, 131}, {131, 0, 130},
	}

	for _, c := range cases {
		got, ok := SlotDestination(c.level, c.style)
		if !ok || got != c.want {
			t.Errorf("SlotDestination(%d, %d) = %d, %v; want %d", c.level, c.style, got, ok, c.want)
		}

		if via, ok := TileDestination(c.level, c.style); !ok || via != c.want {
			t.Errorf("TileDestination(%d, %d) = %d, %v; want %d (Act 4/5 dungeons use the slot rule)", c.level, c.style, via, ok, c.want)
		}
	}

	if _, ok := SlotDestination(113, 3); ok {
		t.Error("a style beyond the last slot must not resolve")
	}
}

func TestFrozenTundraExits(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"Expansion/IceCave/WestEntrance_Snow.ds1", 115},
		{"Expansion/IceCave/WestExit_Snow.ds1", 118},
	}

	for _, c := range cases {
		if got, ok := OutdoorExitByPreset(117, c.path); !ok || got != c.want {
			t.Errorf("%s: got %d, %v; want %d", c.path, got, ok, c.want)
		}
	}

	if _, ok := OutdoorExitByPreset(112, "Expansion/IceCave/WestEntrance_Dirt.ds1"); ok {
		t.Error("only Frozen Tundra has two cave exits; Arreat Plateau uses its single link")
	}
}

func TestAct45RouteReachable(t *testing.T) {
	// Act 4: Fortress -> Outer Steppes -> Plains -> City of the Damned -> River of Flame -> Chaos Sanctuary
	// Act 5: Harrogath -> Foothills -> Highlands -> Plateau -> Crystalline Passage -> Glacial Trail -> Tundra ->
	// Ancients' Way -> Arreat Summit -> Worldstone Keep 1..3 -> Throne of Destruction
	for _, route := range [][]int{
		{103, 104, 105, 106, 107, 108},
		{109, 110, 111, 112, 113, 115, 117, 118, 120, 128, 129, 130, 131},
	} {
		for i := 0; i+1 < len(route); i++ {
			from, to := route[i], route[i+1]
			if !Reachable(from, to) {
				t.Errorf("no link from %d to %d", from, to)
			}

			if !Reachable(to, from) {
				t.Errorf("no way back from %d to %d", to, from)
			}
		}
	}
}

func TestAct45Waypoints(t *testing.T) {
	for _, l := range []int{103, 106, 107, 109, 111, 112, 113, 115, 117, 118, 129} {
		if _, ok := WaypointBit(l); !ok {
			t.Errorf("level %d has no waypoint bit", l)
		}
	}
}
