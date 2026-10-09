package d2maprenderer

import (
	"image/color"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2pl2"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2lightmap"
)

const (
	lightTicksPerSecond = 25.0
	maxCatchUpTicks     = 5

	floorShadeCols = 5 // 6x6 = 36 vertex samples per floor tile
	floorShadeRows = 5
	wallShadeCols  = 5 // 6 column samples per wall tile
	wallShadeRows  = 1

	wallHalfWidth = 80 // a tile is 160 px wide; its centre column is the bottom corner

	// fade of walls covering the hero (renderer.md b1): 500 ms linear. The
	// target alpha is not known from the notes; 0.4 is a visual choice (U).
	wallFadeSeconds = 0.5
	wallFadeTarget  = 0.4
)

// LightInput is what the game tells the renderer about the lighting each frame.
type LightInput struct {
	// HeroX and HeroY are the hero's world position in tile units.
	HeroX, HeroY float64
	// Base is the base light of the hero's level (Levels.txt Intensity/R/G/B,
	// or the day/night ambient when the level has none).
	Base d2lightmap.Cell
	// RadiusSubtiles is the hero's light radius target in subtiles (0 = the
	// default d2lightmap.HeroRadiusSubtiles).
	RadiusSubtiles int
}

// lighting holds the light map and the PL2 derived shade table.
type lighting struct {
	enabled  bool
	hasInput bool
	built    bool
	input    LightInput
	lm       *d2lightmap.Map
	hero     *d2lightmap.Source
	acc      float64
	shade    [d2pl2.ShadeRows]float64
	fades    map[wallKey]*wallFade
	elapsed  float64
}

type wallKey struct{ x, y, idx int }

type wallFade struct{ alpha float64 }

func newLighting() *lighting {
	enabled := strings.TrimSpace(os.Getenv("OD2_LIGHTING")) != "0"

	return &lighting{
		enabled: enabled,
		lm:      d2lightmap.New(),
		hero:    d2lightmap.NewSource(d2lightmap.HeroRadiusSubtiles, 255, 255, 255, 255),
		shade:   d2pl2.LinearShadeFactors(),
		fades:   make(map[wallKey]*wallFade),
	}
}

// SetLightingEnabled switches the light map rendering on or off (it defaults
// to on; OD2_LIGHTING=0 turns it off).
func (mr *MapRenderer) SetLightingEnabled(on bool) { mr.light.enabled = on }

// LightingEnabled reports whether lighting is on.
func (mr *MapRenderer) LightingEnabled() bool { return mr.light.enabled }

// SetLightInput gives the renderer this frame's hero position and base light.
func (mr *MapRenderer) SetLightInput(in LightInput) {
	l := mr.light
	l.input, l.hasInput = in, true

	radius := in.RadiusSubtiles
	if radius <= 0 {
		radius = d2lightmap.HeroRadiusSubtiles
	}

	l.hero.SetTargetRadius(radius)
}

// loadShadeTable derives the shade factors from the act's PL2.
func (mr *MapRenderer) loadShadeTable(dat string) {
	pl2Path := strings.TrimSuffix(dat, ".dat") + ".pl2"

	data, err := mr.asset.LoadFile(pl2Path)
	if err != nil {
		mr.Warningf("no PL2 for lighting (%s), using linear shading: %v", pl2Path, err)
		return
	}

	pl2, err := d2pl2.Load(data)
	if err != nil {
		mr.Warningf("bad PL2 %s: %v", pl2Path, err)
		return
	}

	mr.light.shade = pl2.ShadeFactors()
}

// advanceLighting runs the 25 Hz light map rebuild.
func (mr *MapRenderer) advanceLighting(elapsed float64) {
	l := mr.light
	l.elapsed = elapsed

	if !l.enabled || !l.hasInput {
		return
	}

	l.acc += elapsed * lightTicksPerSecond

	ticks := int(l.acc)
	if ticks > maxCatchUpTicks {
		ticks = maxCatchUpTicks
		l.acc = 0
	}

	l.acc -= float64(ticks)

	if ticks == 0 && l.built {
		return
	}

	for i := 0; i < ticks; i++ {
		l.hero.Step()
	}

	subX, subY := l.input.HeroX*subtilesPerTile, l.input.HeroY*subtilesPerTile
	l.built = true
	l.hero.SetPosition(subX, subY)
	l.lm.Rebuild(int(subX), int(subY), l.input.Base, []*d2lightmap.Source{l.hero})
}

func (l *lighting) active() bool { return l.enabled && l.hasInput }

// tint converts a light cell into a multiply colour: RGB scaled by the PL2
// shade row of the intensity.
func (l *lighting) tint(c d2lightmap.Cell) color.RGBA {
	f := l.shade[d2pl2.ShadeRow(int(c.Intensity))]

	return color.RGBA{R: uint8(float64(c.R) * f), G: uint8(float64(c.G) * f), B: uint8(float64(c.B) * f), A: 255}
}

// tintAt is the tint at an absolute subtile position.
func (l *lighting) tintAt(subX, subY float64) color.RGBA {
	return l.tint(l.lm.Sample(subX, subY))
}

// ambientTint is the flat frame ambient used for roofs.
func (l *lighting) ambientTint() color.RGBA { return l.tint(l.input.Base) }

// withAlpha premultiplies a tint with an alpha.
func withAlpha(c color.RGBA, a float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * a), G: uint8(float64(c.G) * a), B: uint8(float64(c.B) * a), A: uint8(a * 255),
	}
}

func clampSub(v float64) float64 {
	if v < 0 {
		return 0
	}

	if v > subtilesPerTile {
		return subtilesPerTile
	}

	return v
}

// floorSubtile converts a pixel position in a floor image (apex at x=80, y=0)
// into subtile offsets inside the tile.
func floorSubtile(x, y float64) (u, v float64) {
	a := (x - wallHalfWidth) / orthoSubTileWidth // u - v
	b := y / orthoSubTileHeight                  // u + v

	return clampSub((b + a) / 2), clampSub((b - a) / 2)
}

// wallBaseSubtile maps a pixel column of a wall image to a subtile position on
// the tile's south edges (left corner -> bottom corner -> right corner).
func wallBaseSubtile(x float64) (u, v float64) {
	if x < wallHalfWidth {
		return clampSub(x / wallHalfWidth * subtilesPerTile), subtilesPerTile
	}

	return subtilesPerTile, clampSub(subtilesPerTile - (x-wallHalfWidth)/wallHalfWidth*subtilesPerTile)
}

// renderShadedImage draws img with per-vertex light colours. at(x,y) gives the
// tint for a pixel of the image. Uniformly lit images take the plain path.
func (mr *MapRenderer) renderShadedImage(target d2interface.Surface, img d2interface.Surface,
	cols, rows int, alpha float64, at func(x, y float64) color.RGBA) {
	w, h := img.GetSize()

	vals := make([]color.RGBA, 0, (cols+1)*(rows+1))
	uniform := true

	for j := 0; j <= rows; j++ {
		for i := 0; i <= cols; i++ {
			c := withAlpha(at(float64(i*w)/float64(cols), float64(j*h)/float64(rows)), alpha)
			if len(vals) > 0 && c != vals[0] {
				uniform = false
			}

			vals = append(vals, c)
		}
	}

	ss, ok := target.(d2interface.ShadedSurface)
	if uniform || !ok {
		target.PushColor(vals[0])
		target.Render(img)
		target.Pop()

		return
	}

	idx := func(x, y float64) int {
		i := int(x/float64(w)*float64(cols) + 0.5)
		j := int(y/float64(h)*float64(rows) + 0.5)

		return j*(cols+1) + i
	}

	ss.RenderShaded(img, cols, rows, func(x, y float64) color.RGBA { return vals[idx(x, y)] })
}

// fadeFor advances and returns the alpha of a wall tile fading toward target.
func (l *lighting) fadeFor(key wallKey, covering bool) float64 {
	f := l.fades[key]
	if f == nil {
		if !covering {
			return 1
		}

		f = &wallFade{alpha: 1}
		l.fades[key] = f
	}

	target := 1.0
	if covering {
		target = wallFadeTarget
	}

	step := (1 - wallFadeTarget) / wallFadeSeconds * l.elapsed
	switch {
	case f.alpha > target:
		f.alpha -= step
		if f.alpha < target {
			f.alpha = target
		}
	case f.alpha < target:
		f.alpha += step
		if f.alpha > target {
			f.alpha = target
		}
	}

	if f.alpha >= 1 && !covering {
		delete(l.fades, key)
		return 1
	}

	return f.alpha
}
