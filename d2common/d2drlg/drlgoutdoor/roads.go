package drlgoutdoor

// The dirt-road network, DRLG_PlaceRivers (0x684420) and its helpers
// (drlg3.md section 8).

var (
	jitDX = [4]int{1, 0, -1, 0}
	jitDY = [4]int{0, 1, 0, -1}
	juncY = [4]int{0, 1, -1, 0}
	juncX = [4]int{-1, 0, 0, 1}
)

func clamp2(v int) int { return max(-2, min(2, v)) }

func dirClass(x2, y2, x1, y1 int) int {
	dx, dy := x2-x1, y2-y1
	ax, ay := abs(dx), abs(dy)
	sx, sy := dx, dy

	if ax >= 2*ay {
		if dy < 0 {
			sy = -1
		} else {
			sy = dy & 1
		}
	} else if ay >= 2*ax {
		if dx < 0 {
			sx = -1
		} else {
			sx = dx & 1
		}
	}

	return 5*clamp2(sx) + clamp2(sy) + 12
}

func pathHint(x1, y1, x2, y2 int) int { return t2ac8[3*dirClass(x2, y2, x1, y1)] }

type pathNode struct {
	g, x, y, tries, dirptr, dir int
	parent, child               *pathNode
}

type nodePool struct{ n int }

func (p *nodePool) alloc() *pathNode {
	if p.n == 900 {
		return nil
	}

	p.n++

	return &pathNode{}
}

func (l *Level) passable(nx, ny int, node *pathNode, tx, ty int) bool {
	if nx == tx && ny == ty {
		return true
	}

	if nx < 0 || nx >= l.W || ny < 0 || ny >= l.H {
		return false
	}

	if l.Flag.Get(nx, ny)&cellPreset != 0 {
		return false
	}

	for p := node; p != nil; p = p.parent {
		if p.x == nx && p.y == ny {
			return false
		}
	}

	return true
}

func advance(node, root *pathNode) *pathNode {
	if node.tries < 4 {
		node.dirptr++
		node.dir = (dirRows[node.dirptr] + node.dir) & 3
	}

	node.tries++

	if node.tries == 3 {
		for node != root {
			node = node.parent
			node.dirptr++
			node.tries++
			node.dir = (dirRows[node.dirptr] + node.dir) & 3

			if node.tries != 3 {
				return node
			}
		}

		return nil
	}

	return node
}

func (l *Level) searchPath(root *pathNode, limit int, pool *nodePool, tx, ty int) *pathNode {
	node := root

	for {
		if node.x == tx && node.y == ty {
			return node
		}

		nx := node.x + [4]int{1, 0, -1, 0}[node.dir]
		ny := node.y + [4]int{0, 1, 0, -1}[node.dir]

		if !l.passable(nx, ny, node, tx, ty) {
			if node = advance(node, root); node == nil {
				return nil
			}

			continue
		}

		g := node.g + 2
		dxh, dyh := abs(nx-tx), abs(ny-ty)
		h := min(dxh, dyh) + 2*max(dxh, dyh)

		if g+h > limit {
			if node = advance(node, root); node == nil {
				return nil
			}

			continue
		}

		if node.child == nil {
			c := pool.alloc()
			if c == nil {
				return nil
			}

			node.child, c.parent = c, node
		}

		c := node.child
		c.g, c.tries = g, 0
		a := pathHint(nx, ny, tx, ty) / 2
		base := ((node.dir - a) & 3) * 4
		c.dirptr = base
		c.dir = (dirRows[base] + a) & 3
		c.x, c.y = nx, ny
		node = c
	}
}

// findPath is DRLG_FindRiverPath (0x6847b0); the path is goal first.
func (l *Level) findPath(sx0, sy0, ex0, ey0 int) [][2]int {
	r := l.Params.Rect
	sx, sy := sar3(sx0-r.X), sar3(sy0-r.Y)
	ex, ey := sar3(ex0-r.X), sar3(ey0-r.Y)
	dx, dy := sx-ex, sy-ey

	if abs(dx)+abs(dy) < 2 {
		return [][2]int{{sx, sy}, {ex, ey}}
	}

	hv := pathHint(sx, sy, ex, ey)
	cost0 := min(abs(dx), abs(dy)) + 2*max(abs(dx), abs(dy))
	limit := cost0 + cost0/2
	maxl := limit + 0x23
	pool := &nodePool{}

	for {
		root := &pathNode{x: sx, y: sy, tries: -1, dir: (hv / 2) & 3}
		pool.n = 1
		res := l.searchPath(root, limit, pool, ex, ey)
		limit += 5

		if pool.n >= 900 {
			return nil
		}

		pool.n = 1

		if res != nil {
			var out [][2]int
			for n := res; n != nil; n = n.parent {
				out = append(out, [2]int{n.x, n.y})
			}

			return out
		}

		if !(limit < maxl) {
			return nil
		}
	}
}

func (l *Level) snapPoint(px, py, typ int) (int, int) {
	r := l.Params.Rect
	x, y := px-r.X, py-r.Y

	switch typ {
	case 0:
		x = sar3(x)*8 + 0xb
	case 1:
		y = sar3(y)*8 + 0xb
	case 2:
		x = sar3(x)*8 - 5
	case 3:
		y = sar3(y)*8 - 5
	}

	return x + r.X, y + r.Y
}

func (l *Level) buildEndpoints() [][3]int {
	var pts [][3]int

	for _, n := range l.Params.Neighbors {
		r := n.Rect

		switch n.Level {
		case 1:
			var px, py int

			switch n.Dir {
			case 0:
				px, py = r.X+0x3b, r.Y+0x13
			case 1:
				px, py = r.X+0x1d, r.Y+0x23
			case 2:
				px, py = r.X+4, r.Y+0x16
			case 3:
				px, py = r.X+0x1d, r.Y+3
			}

			pts = append(pts, [3]int{px, py, n.Dir & 0xff})
		case 0x1a:
			pts = append(pts, [3]int{r.X + 0x1b, r.Y + 0xd, 1})
		}
	}

	for x := 0; x < l.W; x++ {
		for y := 0; y < l.H; y++ {
			d := int(l.Def.Get(x, y))
			nib := int(l.Flag.Get(x, y)>>16) & 0xf
			t := 4

			switch {
			case d == 4 && nib == 3:
				t = 3
			case d == 5 && nib == 3:
				t = 0
			case d == 6 && nib == 3:
				t = 1
			case d == 7 && nib == 3:
				t = 2
			case d == 0x18:
				t = 1
			case d == 0x19:
				t = 0
			case d == 0x1c && nib == 1 && x == l.W-2:
				t = 2
			case d == 0x33 || d == 0x34:
				if nib != 0 {
					t = 1
				} else {
					t = 0
				}
			}

			if t != 4 {
				pts = append(pts, [3]int{l.Params.Rect.X + x*8 + 3, l.Params.Rect.Y + y*8 + 3, t})
			}
		}
	}

	return pts
}

func (l *Level) junctions(pts [][3]int) [][3]int {
	n := len(pts)
	j := make([][3]int, n)
	r := l.Params.Rect

	cx, cy := -1, -1

	if l.OdFlags&0x10 != 0 {
		col := cdiv(l.W, 2) - 1

		for y := 1; y < l.W-1; y++ { // the original bounds this loop with W
			if l.Def.Get(col, y) == 0x1c && (l.Flag.Get(col, y)>>16)&0xf == 1 {
				cx, cy = col, y
				break
			}
		}
	}

	if l.OdFlags&0x10 != 0 && cx != -1 {
		bx, by := r.X+cx*8+3, r.Y+cy*8+3

		for i := range pts {
			if pts[i][0] > bx {
				j[i] = [3]int{bx + 8, by, 0}
			} else {
				j[i] = [3]int{bx, by, 2}
			}
		}

		return j
	}

	for i := 0; i < n; i++ {
		if i != 0 {
			j[i] = j[0]
			continue
		}

		var ccx, ccy int

		if n == 1 {
			ccx, ccy = cdiv(l.W, 2), cdiv(l.H, 2)
		} else {
			sx, sy := 0, 0
			for _, p := range pts {
				sx += p[0] - r.X
				sy += p[1] - r.Y
			}

			ccx, ccy = cdiv(sx, n*8), cdiv(sy, n*8)
		}

		fx, fy := 0, 0
		found := false

		for rr := 0; rr < 8 && !found; rr++ {
			for k := 0; k < 4; k++ {
				yy, xx := juncY[k]*rr+ccy, juncX[k]*rr+ccx
				fx, fy = xx, yy

				if xx >= 0 && xx < l.W && yy >= 0 && yy < l.H && l.Flag.Get(xx, yy)&cellBlocked == 0 {
					found = true
					break
				}
			}
		}

		j[0] = [3]int{r.X + fx*8 + 3, r.Y + fy*8 + 3, 4}
	}

	return j
}

// placeRoads is DRLG_PlaceRivers.
func (l *Level) placeRoads() {
	pts := l.buildEndpoints()
	n := len(pts)
	j := l.junctions(pts)
	r := l.Params.Rect

	rv := River{N: n}
	rv.Ends = pts
	rv.Junc = j

	for i := 0; i < n; i++ {
		sx, sy := l.snapPoint(pts[i][0], pts[i][1], pts[i][2])
		ex, ey := l.snapPoint(j[i][0], j[i][1], j[i][2])
		rv.Start = append(rv.Start, [2]int{sx, sy})
		rv.End = append(rv.End, [2]int{ex, ey})
	}

	rv.Paths = make([][][2]int, n)

	for i := 0; i < n; i++ {
		path := l.findPath(rv.Start[i][0], rv.Start[i][1], rv.End[i][0], rv.End[i][1])
		if path == nil {
			continue
		}

		for _, p := range path {
			if l.Flag.In(p[0], p[1]) {
				l.Flag.Op(p[0], p[1], cellRoad, opOr)
			}
		}

		ph := int(l.Seed.Step() & 3)

		var out [][2]int
		if j[i][2] != 4 {
			out = append(out, [2]int{j[i][0], j[i][1]})
		}

		out = append(out, rv.End[i])

		for k := 1; k < len(path); k++ {
			if k != len(path)-1 {
				m1 := int(l.Seed.Step()&1) + 2
				m2 := int(l.Seed.Step()&1) + 2
				out = append(out, [2]int{r.X + path[k][0]*8 + 3 + m1*jitDX[ph], r.Y + path[k][1]*8 + 3 + m2*jitDY[ph]})
				ph = (ph + 1) & 3
			} else {
				out = append(out, rv.Start[i], [2]int{pts[i][0], pts[i][1]})
			}
		}

		rv.Paths[i] = out
	}

	l.River = rv
}
