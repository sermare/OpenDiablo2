// Package d2lightmap implements the real game's 48x48 subtile light map
// (Game.exe 1.14b, LIGHT_RebuildLightMap 0x4714e0 and friends; notes in
// d2-re-notes/renderer.md section b4). The map is rebuilt once per game tick
// (25 Hz): cleared to the base light of the level, then every dynamic light
// source paints a radial falloff into it.
//
// Verified in the notes: 48x48 cells of one subtile, origin = hero subtile
// minus 24, positions with 3 fractional bits, octagon distance
// (max*0x3d7 + min*0x197)>>10, linear falloff painter, intensity-weighted
// colour blend in AddToCell, radius ramp of 8 units per tick capped at 0xf8.
// Not implemented: the line-of-sight painter used at the highest quality
// setting (walls do not block light) and the 0x470250 tinting of other levels.
// The reciprocal table at 0x7a8af0 is assumed to be 65536/n (UNVERIFIED).
package d2lightmap

// Size is the width and height of the light map in cells (subtiles).
const Size = 48

const (
	// half is the offset from the hero's subtile to the map origin.
	half = 24

	// MaxRadius is the radius cap in 1/8 subtile units (0xf8).
	MaxRadius = 0xf8

	// RampPerTick is how much a source radius moves toward its target per tick.
	RampPerTick = 8

	// HeroRadiusSubtiles is the radius the hero's light is created with
	// (PLAYER_InitClientPlayer passes 0xd to the light-source constructor).
	HeroRadiusSubtiles = 13

	fixedShift = 3 // positions are in 1/8 subtile units
)

// Cell is one light-map cell: an intensity 0..255 and a colour.
type Cell struct {
	Intensity uint8
	R, G, B   uint8
}

// Source is a dynamic light.
type Source struct {
	// X and Y are the centre in 1/8 subtile units (use SetPosition).
	X, Y int
	// Radius is the current radius in 1/8 subtile units; Target is where it
	// ramps to.
	Radius, Target int
	Intensity      uint8
	R, G, B        uint8
}

// NewSource creates a light of the given radius (in subtiles), fully grown.
func NewSource(radiusSubtiles int, intensity, r, g, b uint8) *Source {
	rad := radiusSubtiles << fixedShift
	if rad > MaxRadius {
		rad = MaxRadius
	}

	return &Source{Radius: rad, Target: rad, Intensity: intensity, R: r, G: g, B: b}
}

// SetPosition places the source at a (fractional) subtile position. The
// original adds 4 (half a subtile) to the 3-fraction-bit position.
func (s *Source) SetPosition(subX, subY float64) {
	s.X = int(subX*(1<<fixedShift)) + 4
	s.Y = int(subY*(1<<fixedShift)) + 4
}

// SetTargetRadius sets the radius the source ramps toward, in subtiles.
func (s *Source) SetTargetRadius(subtiles int) {
	s.Target = subtiles << fixedShift
	if s.Target > MaxRadius {
		s.Target = MaxRadius
	}

	if s.Target < 0 {
		s.Target = 0
	}
}

// Step ramps the radius toward the target by 8 units (one game tick).
func (s *Source) Step() {
	switch {
	case s.Radius < s.Target:
		s.Radius += RampPerTick
	case s.Radius > s.Target:
		s.Radius -= RampPerTick
	}
}

// Map is the light map.
type Map struct {
	// OriginX and OriginY are the subtile coordinates of cell (0,0).
	OriginX, OriginY int
	cells            [Size * Size]Cell
	recip            [256]int
}

// New returns a map filled with black.
func New() *Map {
	m := &Map{}

	for i := 1; i < len(m.recip); i++ {
		m.recip[i] = 65536 / i
	}

	return m
}

// ApproxDistance is the octagon distance used by the painters.
func ApproxDistance(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	hi, lo := dx, dy
	if lo > hi {
		hi, lo = lo, hi
	}

	return (hi*0x3d7 + lo*0x197) >> 10
}

// Rebuild clears the map to base and paints all sources. heroSubX/Y is the
// hero's integer subtile position.
func (m *Map) Rebuild(heroSubX, heroSubY int, base Cell, sources []*Source) {
	m.OriginX, m.OriginY = heroSubX-half, heroSubY-half

	for i := range m.cells {
		m.cells[i] = base
	}

	for _, s := range sources {
		m.paintRadial(s)
	}
}

func (m *Map) paintRadial(s *Source) {
	r := s.Radius
	if r <= 0 || s.Intensity == 0 {
		return
	}

	step := (int(s.Intensity) << 16) / r
	minX, maxX := (s.X-r)>>fixedShift-m.OriginX, (s.X+r)>>fixedShift-m.OriginX
	minY, maxY := (s.Y-r)>>fixedShift-m.OriginY, (s.Y+r)>>fixedShift-m.OriginY

	for cy := max(minY, 0); cy <= min(maxY, Size-1); cy++ {
		for cx := max(minX, 0); cx <= min(maxX, Size-1); cx++ {
			// distance from the source to the cell centre
			dx := (cx+m.OriginX)<<fixedShift + 4 - s.X
			dy := (cy+m.OriginY)<<fixedShift + 4 - s.Y
			d := ApproxDistance(dx, dy)

			v := ((r - d) * step) >> 16
			if v > 0 {
				m.addToCell(cx, cy, v, s.R, s.G, s.B)
			}
		}
	}
}

// addToCell is AddToCell (0x470410): intensity adds with saturation, colour
// is the intensity-weighted mean.
func (m *Map) addToCell(cx, cy, v int, r, g, b uint8) {
	c := &m.cells[cy*Size+cx]
	oldI := int(c.Intensity)

	newI := oldI + v
	if newI > 255 {
		newI = 255
	}

	if newI == 0 {
		return
	}

	mix := func(old, add uint8) uint8 {
		n := (int(old)*oldI + int(add)*v) * m.recip[newI] >> 16
		if n > 255 {
			n = 255
		}

		return uint8(n)
	}

	c.R, c.G, c.B = mix(c.R, r), mix(c.G, g), mix(c.B, b)
	c.Intensity = uint8(newI)
}

// At returns the cell at a map index, clamped like LIGHT_SampleLightMapCell.
func (m *Map) At(cx, cy int) Cell {
	cx, cy = clamp(cx), clamp(cy)
	return m.cells[cy*Size+cx]
}

// Sample returns the light at a (fractional) absolute subtile position.
func (m *Map) Sample(subX, subY float64) Cell {
	return m.At(floorInt(subX)-m.OriginX, floorInt(subY)-m.OriginY)
}

func floorInt(f float64) int {
	i := int(f)
	if float64(i) > f {
		i--
	}

	return i
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}

	if v >= Size {
		return Size - 1
	}

	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
