package d2missile

import "math"

// Path type 0xe, the spiral of Blessed Hammer (SRVDO_073 0x5ce5a0 switches the
// new missile to it; the stepper is PATH_ApplyAngleStepVelocity 0x67d050,
// VERIFIED). The path is a list of up to 77 nodes: iteration n (n = 1, 2, ...)
// puts a point at start + (cos a, sin a) * r with a = 2*pi*16n/512 (the table
// index grows by 16 of 512 per iteration, 11.25 degrees) and r = n * 0x2580 /
// 65536 subtiles (0.1465 per iteration); a point is only kept as a node when
// it lies in a different subtile than the last kept node. The spiral does not
// depend on the heading at all: it always starts towards +x.

const (
	spiralNodes   = 0x4d
	spiralRadStep = 0x2580 / 65536.0
	spiralAngStep = 2 * math.Pi * 16 / 512
)

// SpiralNodes returns the nodes of the spiral that starts at (x, y).
func SpiralNodes(x, y float64) [][2]float64 {
	out := make([][2]float64, 0, spiralNodes)
	lastX, lastY := math.Floor(x), math.Floor(y)

	for n := 1; len(out) < spiralNodes; n++ {
		r := float64(n) * spiralRadStep
		a := float64(n) * spiralAngStep
		px, py := x+math.Cos(a)*r, y+math.Sin(a)*r

		if math.Floor(px) != lastX || math.Floor(py) != lastY {
			out = append(out, [2]float64{px, py})
			lastX, lastY = math.Floor(px), math.Floor(py)
		}
	}

	return out
}

// spiral is a walkable polyline through the nodes.
type spiral struct {
	pts   [][2]float64 // start point first
	cum   []float64    // cumulative length at each point
	total float64
}

func newSpiral(x, y float64) *spiral {
	s := &spiral{pts: append([][2]float64{{x, y}}, SpiralNodes(x, y)...)}
	s.cum = make([]float64, len(s.pts))

	for i := 1; i < len(s.pts); i++ {
		s.cum[i] = s.cum[i-1] + math.Hypot(s.pts[i][0]-s.pts[i-1][0], s.pts[i][1]-s.pts[i-1][1])
	}

	s.total = s.cum[len(s.cum)-1]

	return s
}

// at is the position after walking dist along the polyline and the unit
// direction of the segment it is on.
func (s *spiral) at(dist float64) (x, y, dx, dy float64) {
	i := 1
	for i < len(s.pts)-1 && s.cum[i] < dist {
		i++
	}

	a, b := s.pts[i-1], s.pts[i]
	seg := s.cum[i] - s.cum[i-1]

	if seg <= 0 {
		return b[0], b[1], 1, 0
	}

	t := math.Min(math.Max((dist-s.cum[i-1])/seg, 0), 1)

	return a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, (b[0] - a[0]) / seg, (b[1] - a[1]) / seg
}
