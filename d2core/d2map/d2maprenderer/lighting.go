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
	// target alpha is 0x80/255 (verified, DRAWLIST_AddTimedRoomEntry 0x4d9f60).
	wallFadeSeconds = 0.5
	wallFadeTarget  = 128.0 / 255.0
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
	// Extra are the static lights near the hero (objects such as torches and fires).
	Extra []LightPoint
}

// LightPoint is a static light: a position in tile units, a radius in subtiles and a colour.
type LightPoint struct {
	X, Y    float64
	Radius  int
	R, G, B uint8
}

// staticLightIntensity is the peak intensity of object lights (U: the object light intensity is not in the notes).
const staticLightIntensity = 255

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
	sources  []*d2lightmap.Source
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
	l.sources = append(l.sources[:0], l.hero)

	for _, p := range l.input.Extra {
		if p.Radius <= 0 {
			continue
		}

		src := d2lightmap.NewSource(p.Radius, staticLightIntensity, p.R, p.G, p.B)
		src.SetPosition(p.X*subtilesPerTile, p.Y*subtilesPerTile)
		l.sources = append(l.sources, src)
	}

	l.lm.Rebuild(int(subX), int(subY), l.input.Base, l.sources)
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

// shadeKind selects how the vertex colours of renderShadedImage are sampled from the light map.
type shadeKind int

const (
	shadeFloor shadeKind = iota // a floor image: the colour follows the position inside the tile
	shadeWall                   // a wall image: only the columns matter, sampled along the tile's south edges
)

// renderShadedImage draws img with per-vertex light colours sampled at tile (tileX, tileY).
// Uniformly lit images take the plain path. The vertex colours live in a scratch slice of the
// renderer, so a draw allocates nothing.
func (mr *MapRenderer) renderShadedImage(target d2interface.Surface, img d2interface.Surface,
	cols, rows int, alpha float64, kind shadeKind, tileX, tileY int) {
	w, h := img.GetSize()

	vals := mr.shadeVals[:0]
	uniform := true
	baseX, baseY := float64(tileX*subtilesPerTile), float64(tileY*subtilesPerTile)

	for j := 0; j <= rows; j++ {
		for i := 0; i <= cols; i++ {
			x, y := float64(i*w)/float64(cols), float64(j*h)/float64(rows)

			var u, v float64
			if kind == shadeFloor {
				u, v = floorSubtile(x, y)
			} else {
				u, v = wallBaseSubtile(x)
			}

			c := withAlpha(mr.light.tintAt(baseX+u, baseY+v), alpha)
			if len(vals) > 0 && c != vals[0] {
				uniform = false
			}

			vals = append(vals, c)
		}
	}

	mr.shadeVals = vals

	if uniform {
		target.PushColor(vals[0])
		target.Render(img)
		target.Pop()

		return
	}

	if gs, ok := target.(d2interface.ShadedGridSurface); ok {
		gs.RenderShadedGrid(img, cols, rows, vals)
		return
	}

	if ss, ok := target.(d2interface.ShadedSurface); ok {
		stride := cols + 1
		ss.RenderShaded(img, cols, rows, func(x, y float64) color.RGBA {
			i := int(x/float64(w)*float64(cols) + 0.5)
			j := int(y/float64(h)*float64(rows) + 0.5)

			return vals[j*stride+i]
		})

		return
	}

	target.PushColor(vals[0])
	target.Render(img)
	target.Pop()
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

// resetFades forgets the wall fades (on a level change).
func (l *lighting) resetFades() {
	if l == nil {
		return
	}

	for k := range l.fades {
		delete(l.fades, k)
	}
}
