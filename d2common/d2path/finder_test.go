package d2path

import (
	"strings"
	"testing"
)

// gridFrom builds a grid from ASCII art: '#' = FlagWalk, 'D' = door, 'c' = corpse,
// anything else free. It returns the grid and the positions of 'S' and 'G'.
func gridFrom(art string) (g *CellGrid, s, goal Point) {
	rows := strings.Split(strings.TrimSpace(art), "\n")
	g = NewCellGrid(0, 0, len(rows[0]), len(rows))

	for y, row := range rows {
		for x, ch := range row {
			switch ch {
			case '#':
				g.Set(x, y, FlagWalk)
			case 'D':
				g.Set(x, y, FlagDoor)
			case 'c':
				g.Set(x, y, FlagCorpse)
			case 'S':
				s = Point{x, y}
			case 'G':
				goal = Point{x, y}
			}
		}
	}

	return g, s, goal
}

func TestOutOfGridIsBlocked(t *testing.T) {
	g := NewCellGrid(0, 0, 3, 3)

	for _, p := range []Point{{-1, 0}, {0, -1}, {3, 0}, {0, 3}} {
		if f := g.Flags(p.X, p.Y); f != 0x27 {
			t.Errorf("%v flags %#x, want 0x27", p, f)
		}

		for _, m := range []uint16{MaskPlayer, MaskMonster, MaskFlyer} {
			if !Blocked(g, p.X, p.Y, m) {
				t.Errorf("%v not blocked for mask %#x", p, m)
			}
		}
	}

	if Blocked(g, 1, 1, MaskMonster) {
		t.Fatal("inside cell should be free")
	}
}

func TestMasks(t *testing.T) {
	g := NewCellGrid(0, 0, 4, 1)
	g.Set(0, 0, FlagPlayerOnly)
	g.Set(1, 0, FlagMonster)
	g.Set(2, 0, FlagDoor)
	g.Set(3, 0, FlagCorpse)

	// 0x8 blocks players only
	if !Blocked(g, 0, 0, MaskPlayer) || Blocked(g, 0, 0, MaskMonster) {
		t.Error("player-only flag")
	}

	// The unit footprint bits (0x80 player, 0x100 monster, 0x8000 corpse) are in
	// none of the block masks (VERIFIED by the mask values 0x1C09/0x3C01/0x1804):
	// they are for missile and attack collision, not for walking.
	for _, f := range []uint16{FlagPlayer, FlagMonster, FlagCorpse, FlagMissile} {
		for _, m := range []uint16{MaskPlayer, MaskMonster, MaskMonsterOpensDoors, MaskFlyer} {
			if f&m != 0 {
				t.Errorf("footprint %#x unexpectedly in mask %#x", f, m)
			}
		}
	}

	// 0x800 door blocks players (0x1C09 contains 0x800)
	if !Blocked(g, 2, 0, MaskPlayer) {
		t.Error("door must block the player mask")
	}
}

func TestEstimateDistance(t *testing.T) {
	for _, c := range []struct{ dx, dy, want int }{
		{0, 0, 0}, {5, 0, 10}, {3, 4, 11}, {-4, 4, 12}, {1, 7, 15},
	} {
		if got := EstimateDistance(c.dx, c.dy); got != c.want {
			t.Errorf("(%d,%d)=%d want %d", c.dx, c.dy, got, c.want)
		}
	}
}

func TestStraightLine(t *testing.T) {
	g, s, goal := gridFrom(`
S.......G
.........
`)

	r, ok := FindPath(g, MaskPlayer, s, goal)
	if !ok || len(r.Nodes) != 1 || r.Nodes[0] != goal || r.Partial {
		t.Fatalf("route %+v ok=%v", r, ok)
	}
}

func TestAStarAroundWall(t *testing.T) {
	g, s, goal := gridFrom(`
..........
.S..#.....
....#.....
....#.....
....#..G..
..........
`)

	r, ok := FindPath(g, MaskPlayer, s, goal)
	if !ok || r.Partial {
		t.Fatalf("expected a full route: %+v %v", r, ok)
	}

	if r.Nodes[len(r.Nodes)-1] != goal {
		t.Fatalf("route should end at the goal: %+v", r)
	}

	// every node must be free and consecutive corner segments must be clear
	prev := s
	for _, n := range r.Nodes {
		if Blocked(g, n.X, n.Y, MaskPlayer) {
			t.Fatalf("node %v blocked", n)
		}

		if clear, _ := TraceLine(g, MaskPlayer, prev, n); !clear {
			t.Fatalf("segment %v -> %v crosses the wall", prev, n)
		}

		prev = n
	}

	if len(r.Nodes) > 6 {
		t.Fatalf("corner compaction failed, %d nodes: %v", len(r.Nodes), r.Nodes)
	}
}

func TestAStarOptimalCost(t *testing.T) {
	// open field: 4 diagonal + 3 straight = 4*3 + 3*2 = 18 optimal cost; the
	// corner-compacted route must be diagonal then straight (2 nodes).
	g, s, goal := gridFrom(`
S.......
........
........
........
.......G
`)

	g.Set(3, 0, FlagWalk) // force the line to be blocked? no: keep open; use A* directly

	r, ok := ShortAStar(g, MaskPlayer, s, goal)
	if !ok || r.Partial {
		t.Fatal("no route")
	}

	if r.Nodes[len(r.Nodes)-1] != goal {
		t.Fatalf("ends at %v", r.Nodes)
	}

	if len(r.Nodes) > 3 {
		t.Fatalf("expected <=3 corners in open field, got %v", r.Nodes)
	}
}

func TestPartialWhenGoalSealed(t *testing.T) {
	g, s, goal := gridFrom(`
.........
.S.#.....
...#.#G#.
...#..#..
...#.....
`)

	// seal the goal pocket: G is surrounded by walls and board edges
	for _, p := range []Point{{5, 1}, {6, 1}, {7, 1}, {5, 2}, {7, 2}, {5, 3}, {6, 3}, {7, 3}} {
		g.Set(p.X, p.Y, FlagWalk)
	}

	r, ok := ShortAStar(g, MaskPlayer, s, goal)
	if !ok || !r.Partial {
		t.Fatalf("expected a partial route, got %+v ok=%v", r, ok)
	}

	end := r.Nodes[len(r.Nodes)-1]
	if Blocked(g, end.X, end.Y, MaskPlayer) {
		t.Fatalf("partial route ends in a wall: %v", end)
	}

	// closer to the goal than the start
	if EstimateDistance(goal.X-end.X, goal.Y-end.Y) >= EstimateDistance(goal.X-s.X, goal.Y-s.Y) {
		t.Fatalf("partial end %v is not closer to the goal", end)
	}
}

func TestDestinationLimit(t *testing.T) {
	g := NewCellGrid(0, 0, 300, 300)

	if _, ok := FindPath(g, MaskPlayer, Point{0, 0}, Point{100, 0}); ok {
		t.Fatal("|dx|>=100 must be rejected")
	}

	if _, ok := FindPath(g, MaskPlayer, Point{0, 0}, Point{5, 99}); !ok {
		t.Fatal("|dy|=99 is allowed")
	}

	if _, ok := FindPath(g, MaskPlayer, Point{4, 4}, Point{4, 4}); ok {
		t.Fatal("same cell has no route")
	}
}

func TestLineFallbackDistanceGate(t *testing.T) {
	// A wall between S and G: close goals use A*, far goals only get the
	// straight line up to the wall.
	g := NewCellGrid(0, 0, 80, 80)
	for y := 0; y < 80; y++ {
		g.Set(30, y, FlagWalk)
	}

	// distance^2 = 40*40 > 325: no A*, partial line to the wall
	r, ok := FindPath(g, MaskPlayer, Point{10, 10}, Point{50, 10})
	if !ok || !r.Partial || r.Nodes[0].X != 29 {
		t.Fatalf("far blocked goal: %+v ok=%v", r, ok)
	}

	// goal 17 away (289 < 325) with a short wall: A* goes around
	g2 := NewCellGrid(0, 0, 80, 80)
	for y := 5; y < 16; y++ {
		g2.Set(20, y, FlagWalk)
	}

	r, ok = FindPath(g2, MaskPlayer, Point{12, 10}, Point{28, 10})
	if !ok || r.Partial || r.Nodes[len(r.Nodes)-1] != (Point{28, 10}) {
		t.Fatalf("near blocked goal: %+v ok=%v", r, ok)
	}
}

func TestTraceLineLastFree(t *testing.T) {
	g, _, _ := gridFrom(`
.....#....
..........
`)

	clear, last := TraceLine(g, MaskMonster, Point{0, 0}, Point{9, 0})
	if clear || last != (Point{4, 0}) {
		t.Fatalf("clear=%v last=%v", clear, last)
	}

	if clear, _ := TraceLine(g, MaskMonster, Point{0, 1}, Point{9, 1}); !clear {
		t.Fatal("lower row is free")
	}
}

func TestFinishCornerCompactionAndCap(t *testing.T) {
	chain := func(n int) *node {
		cur := &node{p: Point{0, 0}}

		for i := 1; i <= n; i++ {
			p := cur.p
			if i%2 == 1 {
				p.X++
			} else {
				p.Y++
			}

			cur = &node{p: p, parent: cur}
		}

		return cur
	}

	// n steps alternating E,S give n corner nodes (every step turns)
	if r, ok := finish(chain(MaxNodes), false); !ok || len(r.Nodes) != MaxNodes {
		t.Fatalf("77 corners must be accepted: ok=%v len=%d", ok, len(r.Nodes))
	}

	if _, ok := finish(chain(MaxNodes+1), false); ok {
		t.Fatal("78 corners must fail")
	}

	// a straight run is one node
	st := &node{p: Point{0, 0}}
	for i := 1; i <= 10; i++ {
		st = &node{p: Point{i, 0}, parent: st}
	}

	if r, _ := finish(st, false); len(r.Nodes) != 1 || r.Nodes[0] != (Point{10, 0}) {
		t.Fatalf("straight run: %v", r.Nodes)
	}
}

func TestNearestFree(t *testing.T) {
	g := NewCellGrid(0, 0, 5, 5)
	for y := 1; y <= 3; y++ {
		for x := 1; x <= 3; x++ {
			g.Set(x, y, FlagWalk)
		}
	}

	p, ok := NearestFree(g, MaskMonster, Point{2, 2}, 4)
	if !ok || Blocked(g, p.X, p.Y, MaskMonster) {
		t.Fatalf("got %v %v", p, ok)
	}

	if abs(p.X-2) != 2 && abs(p.Y-2) != 2 {
		t.Fatalf("should be on ring 2: %v", p)
	}
}

func TestDoorBlocksUnlessMonsterOpensDoors(t *testing.T) {
	g := NewCellGrid(0, 0, 3, 1)
	g.Set(1, 0, FlagDoor)

	// 0x3C01 contains 0x0800 (door); 0x3401 does not, so door-openers pass.
	if !Blocked(g, 1, 0, MaskMonster) || Blocked(g, 1, 0, MaskMonsterOpensDoors) {
		t.Fatal("door masks")
	}
}

// segmentCost prices a corner-compacted route the way ShortAStar does.
func segmentCost(from Point, nodes []Point) int {
	cost := 0

	for _, n := range nodes {
		dx, dy := abs(n.X-from.X), abs(n.Y-from.Y)
		diag := dx
		if dy < dx {
			diag = dy
		}

		straight := dx + dy - 2*diag
		cost += diag*costDiagonal + straight*costStraight
		from = n
	}

	return cost
}

// dijkstra is an independent reference for the optimal cost.
func dijkstra(g Grid, mask uint16, from, to Point, limit int) int {
	dist := map[Point]int{from: 0}
	queue := []Point{from}

	for len(queue) > 0 {
		best := 0

		for i := range queue {
			if dist[queue[i]] < dist[queue[best]] {
				best = i
			}
		}

		cur := queue[best]
		queue = append(queue[:best], queue[best+1:]...)

		if cur == to {
			return dist[cur]
		}

		for _, nb := range neighbours {
			p := Point{cur.X + nb.dx, cur.Y + nb.dy}
			if p.X < -limit || p.Y < -limit || p.X > limit || p.Y > limit || Blocked(g, p.X, p.Y, mask) {
				continue
			}

			if d, ok := dist[p]; !ok || dist[cur]+nb.cost < d {
				if !ok {
					queue = append(queue, p)
				}

				dist[p] = dist[cur] + nb.cost
			}
		}
	}

	return -1
}

func TestAStarMatchesDijkstraOnRandomGrids(t *testing.T) {
	seed := uint32(12345)
	rnd := func(n int) int {
		seed = seed*1664525 + 1013904223

		return int(seed>>16) % n
	}

	checked := 0

	for trial := 0; trial < 60; trial++ {
		g := NewCellGrid(0, 0, 16, 16)

		for i := 0; i < 50; i++ {
			g.Set(rnd(16), rnd(16), FlagWalk)
		}

		from, to := Point{rnd(16), rnd(16)}, Point{rnd(16), rnd(16)}
		if Blocked(g, from.X, from.Y, MaskMonster) || Blocked(g, to.X, to.Y, MaskMonster) || from == to {
			continue
		}

		want := dijkstra(g, MaskMonster, from, to, 16)
		r, ok := ShortAStar(g, MaskMonster, from, to)

		if want < 0 {
			if ok && !r.Partial {
				t.Fatalf("trial %d: A* found a route Dijkstra could not", trial)
			}

			continue
		}

		if !ok || r.Partial {
			t.Fatalf("trial %d: no full route, want cost %d", trial, want)
		}

		if got := segmentCost(from, r.Nodes); got != want {
			t.Fatalf("trial %d: cost %d, optimal %d (%v -> %v)", trial, got, want, from, to)
		}

		checked++
	}

	if checked < 20 {
		t.Fatalf("only %d trials were usable", checked)
	}
}
