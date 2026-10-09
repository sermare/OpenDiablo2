package d2mapengine

import "testing"

// wallWorld blocks a vertical wall at x=10 for y in [0,wallTo], and everything
// outside 0..40.
func wallWorld(wallTo int) func(x, y int) bool {
	return func(x, y int) bool {
		if x < 0 || y < 0 || x > 40 || y > 40 {
			return true
		}

		return x == 10 && y <= wallTo
	}
}

func TestFindPathAroundWall(t *testing.T) {
	blocked := wallWorld(20)

	path, reached := findPath(blocked, pt{5, 5}, pt{15, 5})
	if !reached || len(path) == 0 {
		t.Fatalf("no route: reached=%v path=%v", reached, path)
	}

	if path[len(path)-1] != (pt{15, 5}) {
		t.Errorf("route ends at %v, want the goal", path[len(path)-1])
	}

	// every leg of the smoothed route must be a clear line
	prev := pt{5, 5}
	for _, p := range path {
		if !lineClear(blocked, prev, p) {
			t.Errorf("leg %v -> %v crosses the wall", prev, p)
		}

		prev = p
	}

	// it has to go around the end of the wall, so it cannot be a single leg
	if len(path) < 2 {
		t.Errorf("route %v should bend around the wall", path)
	}
}

func TestFindPathOpenFieldIsOneLeg(t *testing.T) {
	path, reached := findPath(wallWorld(-1), pt{2, 2}, pt{30, 25})
	if !reached || len(path) != 1 || path[0] != (pt{30, 25}) {
		t.Errorf("open field: reached=%v path=%v", reached, path)
	}
}

func TestFindPathBlockedGoalGetsNearest(t *testing.T) {
	blocked := wallWorld(20)

	// the goal is inside the wall: the route ends next to it
	path, reached := findPath(blocked, pt{5, 5}, pt{10, 5})
	if reached || len(path) == 0 {
		t.Fatalf("a blocked goal is not reached but must still get a route: reached=%v path=%v", reached, path)
	}

	end := path[len(path)-1]
	if blocked(end.x, end.y) || abs(end.x-10) > 1 || abs(end.y-5) > 1 {
		t.Errorf("route ends at %v, want a free cell beside (10,5)", end)
	}
}

func TestFindPathNoCornerCutting(t *testing.T) {
	// two blocked cells touching diagonally: a route must not squeeze between them
	blocked := func(x, y int) bool {
		if x < 0 || y < 0 || x > 20 || y > 20 {
			return true
		}

		return (x == 10 && y == 10) || (x == 11 && y == 11)
	}

	path, _ := findPath(blocked, pt{10, 11}, pt{11, 10})
	prev := pt{10, 11}

	for _, p := range path {
		if !lineClear(blocked, prev, p) {
			t.Errorf("leg %v -> %v cuts the corner", prev, p)
		}

		prev = p
	}
}

func TestFindPathEnclosed(t *testing.T) {
	// start in a closed 3x3 room: the goal outside cannot be reached
	blocked := func(x, y int) bool {
		in := x >= 5 && x <= 7 && y >= 5 && y <= 7
		ring := x >= 4 && x <= 8 && y >= 4 && y <= 8

		return x < 0 || y < 0 || x > 30 || y > 30 || (ring && !in)
	}

	path, reached := findPath(blocked, pt{6, 6}, pt{20, 20})
	if reached {
		t.Fatal("goal outside a closed room must not be reached")
	}

	for _, p := range path {
		if blocked(p.x, p.y) {
			t.Errorf("route enters blocked cell %v", p)
		}
	}
}
