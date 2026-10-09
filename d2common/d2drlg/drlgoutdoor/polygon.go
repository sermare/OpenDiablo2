package drlgoutdoor

// buildPolygon is DRLG_BuildOutdoorBoundaryPolygon + DRLG_InsertBoundaryExitNotches
// + DRLG_ScaleBoundaryToCells (drlg3.md section 4). It returns the ring head.
func buildPolygon(rect Rect, nbrs []Neighbor) *Vertex {
	w, h := rect.W-1, rect.H-1
	v0 := &Vertex{X: rect.X, Y: rect.Y + h}
	v1 := &Vertex{X: rect.X, Y: rect.Y}
	v2 := &Vertex{X: rect.X + w, Y: rect.Y}
	v3 := &Vertex{X: rect.X + w, Y: rect.Y + h}
	v0.Next, v1.Next, v2.Next, v3.Next = v1, v2, v3, v0

	for _, n := range nbrs {
		nx, ny, nw, nh := n.Rect.X, n.Rect.Y, n.Rect.W-1, n.Rect.H-1

		var (
			p, q                        *Vertex
			vert                        bool
			s, spanStart, spanEnd, a, e int
		)

		switch n.Dir {
		case 0:
			p, q, vert, s, spanStart, spanEnd = v0, v1, true, -1, ny+nh, ny
			a, e = p.Y, q.Y
		case 1:
			p, q, vert, s, spanStart, spanEnd = v1, v2, false, 1, nx, nx+nw
			a, e = p.X, q.X
		case 2:
			p, q, vert, s, spanStart, spanEnd = v2, v3, true, 1, ny, ny+nh
			a, e = p.Y, q.Y
		case 3:
			p, q, vert, s, spanStart, spanEnd = v3, v0, false, -1, nx+nw, nx
			a, e = p.X, q.X
		default:
			continue
		}

		au, ss, se, eu := a*s, spanStart*s, spanEnd*s, e*s
		cur := p

		mk := func(coord int) *Vertex {
			nv := &Vertex{}
			if vert {
				nv.X, nv.Y = cur.X, coord
			} else {
				nv.Y, nv.X = cur.Y, coord
			}

			nv.Next = cur.Next
			cur.Next = nv

			return nv
		}

		if ss <= au {
			if se < au {
				continue
			}
		} else {
			if ss > eu {
				continue
			}

			cur = mk(spanStart)
		}

		cur.F |= 1
		if n.F8 {
			cur.F |= 2
		}

		if se < eu {
			mk(spanEnd)
		}
	}

	head := v0
	for p := head; ; {
		p.X -= rect.X
		p.Y -= rect.Y
		p = p.Next

		if p == head {
			break
		}
	}

	for p := head; ; {
		p.X, p.Y = sar3(p.X), sar3(p.Y)
		p = p.Next

		if p == head {
			break
		}
	}

	// merge vertices that collapsed onto each other
	for p := head; ; {
		q := p.Next
		if p.X == q.X && p.Y == q.Y {
			if q == head {
				head = p
			}

			p.Next = q.Next
			p.F |= q.F
			p.B = q.B
		}

		p = p.Next
		if p == head {
			break
		}
	}

	return head
}

func (l *Level) verts() []*Vertex {
	var out []*Vertex

	for p := l.Poly; ; {
		out = append(out, p)
		p = p.Next

		if p == l.Poly {
			break
		}
	}

	return out
}

// exitMask is DRLG_GetExitSlotMask (0x676dc0).
func (l *Level) exitMask(v *Vertex) uint32 {
	var side int

	switch {
	case v.X == 0:
		if v.Y != 0 {
			side = 0
		} else {
			side = 1
		}
	case v.Y == 0:
		side = 1
		if v.X == l.W-1 {
			side = 2
		}
	case v.X == l.W-1:
		side = 2
		if v.Y == l.H-1 {
			side = 3
		}
	case v.Y == l.H-1:
		side = 3
	default:
		return 0
	}

	dx := [4]int{-4, 4, 12, 4}
	dy := [4]int{4, -4, 4, 12}
	px := dx[side] + v.X*8 + l.Params.Rect.X
	py := dy[side] + v.Y*8 + l.Params.Rect.Y

	for _, n := range l.Params.Neighbors {
		if n.Dir != side {
			continue
		}

		r := n.Rect
		if r.X <= px && px < r.X+r.W && r.Y <= py && py < r.Y+r.H {
			if n.Flag != 0 {
				return 0
			}

			for k, id := range l.Params.Vis {
				if id == n.Level {
					return 1 << uint(k+4)
				}
			}

			return 0
		}
	}

	return 0
}

// line is DRLG_ApplyGridOpAlongEdge (0x67f700).
func line(g *Grid, v *Vertex, val uint32, op int, includeEnd bool) {
	a, b := v, v.Next

	if a.X == b.X && a.Y == b.Y {
		g.Op(a.X, a.Y, val, op)
		return
	}

	if a.X == b.X {
		lo, hi := a.Y, b.Y
		if hi < lo {
			lo, hi = hi, lo
		}

		for y := lo + 1; y < hi; y++ {
			g.Op(a.X, y, val, op)
		}
	} else {
		lo, hi := a.X, b.X
		if hi < lo {
			lo, hi = hi, lo
		}

		for x := lo + 1; x < hi; x++ {
			g.Op(x, a.Y, val, op)
		}
	}

	g.Op(a.X, a.Y, val, op)

	if includeEnd {
		g.Op(b.X, b.Y, val, op)
	}
}

// markExits is DRLG_MarkExitCells (0x678480).
func (l *Level) markExits() {
	for _, v := range l.verts() {
		if v.F&1 == 0 {
			continue
		}

		m := l.exitMask(v)
		line(l.GridB, v, m, opOr, true)

		side := uint32(1)
		if v.B != 0 {
			side = 3
		}

		line(l.Flag, v, side, opOr, true)
	}
}

// concave is DRLG_MarkConcaveCliffVertices (0x6830b0).
func (l *Level) concave() {
	head := l.Poly
	vs := l.verts()
	p4, p6, local8 := head, vs[len(vs)-1], head
	seen := false

	for {
		p5 := p4
		n4 := p4.Next
		cond := (p4.X < n4.X && p4.Y < p6.Y && p4.F&1 == 0 && p6.F&1 == 0) ||
			(n4.Y < p4.Y && p4.X < p6.X && p4.F&1 == 0 && p6.F&1 == 0)

		if cond {
			p6 = nil

			for {
				if p5 == local8 {
					seen = true
				}

				p2 := p5.Next
				if p5.Y < p2.Y || p2.X < p5.X || p5.F&1 != 0 || p2.F&1 != 0 {
					break
				}

				if (p5.X < p2.X && p2.Y < p2.Next.Y && p2.F&1 == 0) || (p2.Y < p5.Y && p2.X < p2.Next.X && p2.F&1 == 0) {
					p6 = p5
				}

				p5 = p2
				if p2 == p4 {
					break
				}
			}

			if p6 != nil {
				for p4 != p6 {
					p4.B = 1
					p4 = p4.Next
				}

				p4.B = 1
				l.OdFlags |= FlagCliffCorner
			}
		}

		p4 = p5.Next
		if seen {
			return
		}

		local8, p6 = head, p5
		if p4 == local8 {
			return
		}
	}
}
