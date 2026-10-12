package ebiten

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// static check that we implement our interface
var _ d2interface.Surface = &ebitenSurface{}
var _ d2interface.ShadedSurface = &ebitenSurface{}
var _ d2interface.ShadedGridSurface = &ebitenSurface{}

const (
	maxAlpha       = 0xff
	cacheLimit     = 512
	transparency25 = 0.25
	transparency50 = 0.50
	transparency75 = 0.75
)

type colorMCacheKey uint32

type colorMCacheEntry struct {
	colorMatrix ebiten.ColorM
	atime       int64
}

type ebitenSurface struct {
	renderer       *Renderer
	stateStack     []surfaceState
	stateCurrent   surfaceState
	image          *ebiten.Image
	colorMCache    map[colorMCacheKey]*colorMCacheEntry
	monotonicClock int64
	scratchOpts    ebiten.DrawImageOptions
}

func createEbitenSurface(r *Renderer, img *ebiten.Image, currentState ...surfaceState) *ebitenSurface {
	state := surfaceState{
		effect:     d2enum.DrawEffectNone,
		saturation: defaultSaturation,
		brightness: defaultBrightness,
		skewX:      defaultSkewX,
		skewY:      defaultSkewY,
		scaleX:     defaultScaleX,
		scaleY:     defaultScaleY,
	}
	if len(currentState) > 0 {
		state = currentState[0]
	}

	return &ebitenSurface{
		renderer:     r,
		image:        img,
		stateCurrent: state,
		colorMCache:  make(map[colorMCacheKey]*colorMCacheEntry),
	}
}

// Renderer returns the renderer
func (s *ebitenSurface) Renderer() d2interface.Renderer {
	return s.renderer
}

// PushTranslation pushes an x,y translation to the state stack
func (s *ebitenSurface) PushTranslation(x, y int) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.x += x
	s.stateCurrent.y += y
}

// PushSkew pushes a skew to the state stack
func (s *ebitenSurface) PushSkew(skewX, skewY float64) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.skewX = skewX
	s.stateCurrent.skewY = skewY
}

// PushScale pushes a scale to the state stack
func (s *ebitenSurface) PushScale(scaleX, scaleY float64) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.scaleX = scaleX
	s.stateCurrent.scaleY = scaleY
}

// PushEffect pushes an effect to the state stack
func (s *ebitenSurface) PushEffect(effect d2enum.DrawEffect) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.effect = effect
}

// PushFilter pushes a filter to the state stack
func (s *ebitenSurface) PushFilter(filter d2enum.Filter) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.filter = d2ToEbitenFilter(filter)
}

// PushColor pushes a color to the stat stack
func (s *ebitenSurface) PushColor(c color.Color) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.color = c
}

// PushBrightness pushes a brightness value to the state stack
func (s *ebitenSurface) PushBrightness(brightness float64) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.brightness = brightness
}

// PushSaturation pushes a saturation value to the state stack
func (s *ebitenSurface) PushSaturation(saturation float64) {
	s.stateStack = append(s.stateStack, s.stateCurrent)
	s.stateCurrent.saturation = saturation
}

// Pop pops a state off of the state stack
func (s *ebitenSurface) Pop() {
	count := len(s.stateStack)
	if count == 0 {
		panic("empty stack")
	}

	s.stateCurrent = s.stateStack[count-1]
	s.stateStack = s.stateStack[:count-1]
}

// PopN pops n states off the the state stack
func (s *ebitenSurface) PopN(n int) {
	for i := 0; i < n; i++ {
		s.Pop()
	}
}

func (s *ebitenSurface) RenderSprite(sprite *d2ui.Sprite) {
	opts := s.createDrawImageOptions()

	if s.stateCurrent.brightness != 1 || s.stateCurrent.saturation != 1 {
		s.applyHSV(opts)
	}

	s.handleStateEffect(opts)

	sprite.Render(s)
}

// Render renders the given surface
func (s *ebitenSurface) Render(sfc d2interface.Surface) {
	opts := s.scratchDrawImageOptions()

	s.prepareColorM(opts, s.stateCurrent.brightness != 1 || s.stateCurrent.saturation != 1)

	s.image.DrawImage(sfc.(*ebitenSurface).image, opts)
}

// Renders the section of the surface, given the bounds
func (s *ebitenSurface) RenderSection(sfc d2interface.Surface, bound image.Rectangle) {
	opts := s.scratchDrawImageOptions()

	s.prepareColorM(opts, s.stateCurrent.brightness != 0)

	s.image.DrawImage(sfc.(*ebitenSurface).image.SubImage(bound).(*ebiten.Image), opts)
}

// matrixKey identifies the colour matrix that applyHSV (when hsv) and handleStateEffect build from a
// surface state: it is a pure function of the draw colour, saturation, brightness and effect.
type matrixKey struct {
	r, g, b, a  uint32
	hasColor    bool
	hsv         bool
	sat, bright float64
	effect      d2enum.DrawEffect
}

const matrixCacheMax = 4096

//nolint:gochecknoglobals // render-thread cache, guarded by hsvMu
var matrixCache = map[matrixKey]ebiten.ColorM{}

// effectEditsMatrix reports whether handleStateEffect changes the colour matrix for the effect.
func effectEditsMatrix(e d2enum.DrawEffect) bool {
	switch e {
	case d2enum.DrawEffectPctTransparency25, d2enum.DrawEffectPctTransparency50, d2enum.DrawEffectPctTransparency75,
		d2enum.DrawEffectBurn, d2enum.DrawEffectMod2XTrans, d2enum.DrawEffectMod2X:
		return true
	}

	return false
}

// prepareColorM does applyHSV (when hsv) and handleStateEffect on freshly created opts. The matrix part is
// memoised: a lit scene draws thousands of shadows and tinted sprites per frame from a few dozen distinct
// (colour, brightness, effect) states, and rebuilding the matrices allocated about ten objects per draw.
// The cached matrices are immutable values shared between draws, as ebiten intends.
func (s *ebitenSurface) prepareColorM(opts *ebiten.DrawImageOptions, hsv bool) {
	st := &s.stateCurrent
	if !hsv && !effectEditsMatrix(st.effect) {
		s.handleStateEffect(opts) // nothing to rebuild: only the composite mode
		return
	}

	k := matrixKey{hsv: hsv, effect: st.effect, sat: st.saturation, bright: st.brightness}

	if st.color != nil {
		k.r, k.g, k.b, k.a = st.color.RGBA()
		k.hasColor = true
	}

	hsvMu.Lock()
	m, ok := matrixCache[k]
	hsvMu.Unlock()

	if ok {
		s.handleStateComposite(opts)
		opts.ColorM = m

		return
	}

	if hsv {
		s.applyHSV(opts)
	}

	s.handleStateEffect(opts)

	hsvMu.Lock()
	if len(matrixCache) >= matrixCacheMax {
		matrixCache = map[matrixKey]ebiten.ColorM{}
	}

	matrixCache[k] = opts.ColorM
	hsvMu.Unlock()
}

// handleStateComposite sets the composite mode of the current effect (the part of handleStateEffect that
// does not touch the colour matrix).
func (s *ebitenSurface) handleStateComposite(opts *ebiten.DrawImageOptions) {
	switch s.stateCurrent.effect {
	case d2enum.DrawEffectModulate:
		opts.CompositeMode = ebiten.CompositeModeLighter
	case d2enum.DrawEffectBurn:
		opts.CompositeMode = ebiten.CompositeModeMultiply
	case d2enum.DrawEffectNormal, d2enum.DrawEffectMod2XTrans, d2enum.DrawEffectNone:
		opts.CompositeMode = ebiten.CompositeModeSourceOver
	}
}

// hsvCache holds the colour matrices of ColorM.ChangeHSV(0, saturation, brightness) applied to the
// identity. Building one allocates several matrices; the same few (saturation, brightness) pairs are
// drawn thousands of times per frame (every lit monster and tile). ColorM values are immutable
// (every operation returns a new matrix), so sharing them is safe.
//
//nolint:gochecknoglobals // render-thread cache
var (
	hsvMu    sync.Mutex
	hsvCache = map[[2]float64]ebiten.ColorM{}
)

const hsvCacheMax = 256

// tintHSVKey identifies a (draw colour, saturation, brightness) combination.
type tintHSVKey struct {
	r, g, b, a  uint32
	sat, bright float64
}

const tintHSVCacheMax = 4096

//nolint:gochecknoglobals // render-thread cache, guarded by hsvMu
var tintHSVCache = map[tintHSVKey]ebiten.ColorM{}

// applyHSV applies the surface's saturation and brightness to opts.ColorM.
func (s *ebitenSurface) applyHSV(opts *ebiten.DrawImageOptions) {
	sat, bright := s.stateCurrent.saturation, s.stateCurrent.brightness

	if s.stateCurrent.color != nil { // the matrix already holds the draw colour
		// ChangeHSV on a colour matrix allocates ~10 objects; the matrix is a pure function of
		// (colour, saturation, brightness) and every lit shadow draws with the same few dozen
		// light tints, so the result is memoised (same value, shared immutable matrix)
		cr, cg, cb, ca := s.stateCurrent.color.RGBA()
		ck := tintHSVKey{cr, cg, cb, ca, sat, bright}

		hsvMu.Lock()
		defer hsvMu.Unlock()

		m, ok := tintHSVCache[ck]
		if !ok {
			m = opts.ColorM
			m.ChangeHSV(0, sat, bright)

			if len(tintHSVCache) >= tintHSVCacheMax {
				tintHSVCache = map[tintHSVKey]ebiten.ColorM{}
			}

			tintHSVCache[ck] = m
		}

		opts.ColorM = m

		return
	}

	key := [2]float64{sat, bright}

	hsvMu.Lock()
	defer hsvMu.Unlock()

	m, ok := hsvCache[key]
	if !ok {
		m.ChangeHSV(0, sat, bright)

		if len(hsvCache) >= hsvCacheMax {
			hsvCache = map[[2]float64]ebiten.ColorM{}
		}

		hsvCache[key] = m
	}

	opts.ColorM = m
}

func (s *ebitenSurface) createDrawImageOptions() *ebiten.DrawImageOptions {
	return s.fillDrawImageOptions(&ebiten.DrawImageOptions{})
}

// scratchDrawImageOptions returns the surface's reusable options, reset and filled from the current state.
// For the draw calls that use the options immediately and do not call back into the surface (ebiten reads
// them synchronously); it saves one allocation per draw call.
func (s *ebitenSurface) scratchDrawImageOptions() *ebiten.DrawImageOptions {
	s.scratchOpts = ebiten.DrawImageOptions{}

	return s.fillDrawImageOptions(&s.scratchOpts)
}

func (s *ebitenSurface) fillDrawImageOptions(opts *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {

	if s.stateCurrent.skewX != 0 || s.stateCurrent.skewY != 0 {
		opts.GeoM.Skew(s.stateCurrent.skewX, s.stateCurrent.skewY)
	}

	if s.stateCurrent.scaleX != 1.0 || s.stateCurrent.scaleY != 1.0 {
		opts.GeoM.Scale(s.stateCurrent.scaleX, s.stateCurrent.scaleY)
	}

	opts.GeoM.Translate(float64(s.stateCurrent.x), float64(s.stateCurrent.y))

	opts.Filter = s.stateCurrent.filter

	if s.stateCurrent.color != nil {
		opts.ColorM = s.colorToColorM(s.stateCurrent.color)
	}

	return opts
}

func (s *ebitenSurface) handleStateEffect(opts *ebiten.DrawImageOptions) {
	switch s.stateCurrent.effect {
	case d2enum.DrawEffectPctTransparency25:
		opts.ColorM.Translate(0, 0, 0, -transparency25)
	case d2enum.DrawEffectPctTransparency50:
		opts.ColorM.Translate(0, 0, 0, -transparency50)
	case d2enum.DrawEffectPctTransparency75:
		opts.ColorM.Translate(0, 0, 0, -transparency75)
	case d2enum.DrawEffectModulate:
		opts.CompositeMode = ebiten.CompositeModeLighter
	// Burn is the multiplicative PL2 table E (dst*src). ebiten's multiply is src*dst with no term for the
	// destination under a transparent source, so every transparent pixel of the sprite turned the
	// destination black (the flies of Duriel's Lair were solid 40x45 black boxes, w51): the matrix makes
	// the transparent pixels white, the neutral factor.
	case d2enum.DrawEffectBurn:
		opts.CompositeMode = ebiten.CompositeModeMultiply
		opts.ColorM.Concat(burnFillMatrix())
	case d2enum.DrawEffectNormal:
		opts.CompositeMode = ebiten.CompositeModeSourceOver
	// Mod2XTrans is PL2 table G: the pixel blends over the destination with a weight that rises
	// with its own brightness (d2enum.Mode6Weight). The colour matrix puts the luma in the alpha
	// row; transparent pixels are black so they stay invisible.
	case d2enum.DrawEffectMod2XTrans:
		opts.CompositeMode = ebiten.CompositeModeSourceOver

		var m ebiten.ColorM

		m.SetElement(3, 0, d2enum.Mode6Gain*d2enum.Mode6LumaR)
		m.SetElement(3, 1, d2enum.Mode6Gain*d2enum.Mode6LumaG)
		m.SetElement(3, 2, d2enum.Mode6Gain*d2enum.Mode6LumaB)
		m.SetElement(3, 3, 0)
		m.Concat(opts.ColorM) // luma of the unlit pixel first, then the light tint
		opts.ColorM = m
	// Mod2X is the hover highlight: the palette brightened to 170%.
	case d2enum.DrawEffectMod2X:
		opts.ColorM.Scale(d2enum.Mod2XBrightness, d2enum.Mod2XBrightness, d2enum.Mod2XBrightness, 1)
	case d2enum.DrawEffectNone:
		opts.CompositeMode = ebiten.CompositeModeSourceOver
	}
}

// DrawTextf renders the string to the surface with the given format string and a set of parameters
func (s *ebitenSurface) DrawTextf(format string, params ...interface{}) {
	str := fmt.Sprintf(format, params...)
	s.Renderer().PrintAt(s.image, str, s.stateCurrent.x, s.stateCurrent.y)
}

// DrawLine draws a line
func (s *ebitenSurface) DrawLine(x, y int, fillColor color.Color) {
	ebitenutil.DrawLine(
		s.image,
		float64(s.stateCurrent.x),
		float64(s.stateCurrent.y),
		float64(s.stateCurrent.x+x),
		float64(s.stateCurrent.y+y),
		fillColor,
	)
}

// DrawRect draws a rectangle
func (s *ebitenSurface) DrawRect(width, height int, fillColor color.Color) {
	ebitenutil.DrawRect(
		s.image,
		float64(s.stateCurrent.x),
		float64(s.stateCurrent.y),
		float64(width),
		float64(height),
		fillColor,
	)
}

// Clear clears the entire surface, filling with the given color
func (s *ebitenSurface) Clear(fillColor color.Color) {
	s.image.Fill(fillColor)
}

// GetSize gets the size of the surface
func (s *ebitenSurface) GetSize() (x, y int) {
	return s.image.Size()
}

// GetDepth returns the depth of this surface in the stack
func (s *ebitenSurface) GetDepth() int {
	return len(s.stateStack)
}

// ReplacePixels replaces pixels in the surface with the given pixels
func (s *ebitenSurface) ReplacePixels(pixels []byte) {
	s.image.ReplacePixels(pixels)
}

// Screenshot returns an *image.RGBA of the surface
func (s *ebitenSurface) Screenshot() *image.RGBA {
	width, height := s.GetSize()
	bounds := image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: width, Y: height}}
	rgba := image.NewRGBA(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgba.Set(x, y, s.image.At(x, y))
		}
	}

	return rgba
}

func (s *ebitenSurface) now() int64 {
	s.monotonicClock++
	return s.monotonicClock
}

// colorToColorM converts a normal color to a color matrix
func (s *ebitenSurface) colorToColorM(clr color.Color) ebiten.ColorM {
	// RGBA() is in [0 - 0xffff]. Adjust them in [0 - 0xff].
	cr, cg, cb, ca := clr.RGBA()
	cr >>= 8
	cg >>= 8
	cb >>= 8
	ca >>= 8

	if ca == 0 {
		emptyColorM := ebiten.ColorM{}
		emptyColorM.Scale(0, 0, 0, 0)

		return emptyColorM
	}

	// nolint:gomnd // byte values
	key := colorMCacheKey(cr | (cg << 8) | (cb << 16) | (ca << 24))
	e, ok := s.colorMCache[key]

	if ok {
		e.atime = s.now()
		return e.colorMatrix
	}

	if len(s.colorMCache) > cacheLimit {
		oldest := int64(math.MaxInt64)
		oldestKey := colorMCacheKey(0)

		for key, c := range s.colorMCache {
			if c.atime < oldest {
				oldestKey = key
				oldest = c.atime
			}
		}

		delete(s.colorMCache, oldestKey)
	}

	cm := ebiten.ColorM{}
	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / maxAlpha
	cm.Scale(rf, gf, bf, af)

	e = &colorMCacheEntry{
		colorMatrix: cm,
		atime:       s.now(),
	}

	s.colorMCache[key] = e

	return e.colorMatrix
}

// RenderShaded draws sfc split into cols x rows quads with a colour per vertex
// (d2interface.ShadedSurface). The surface state (translation, scale, colour,
// effect) applies as for Render.
func (s *ebitenSurface) RenderShaded(sfc d2interface.Surface, cols, rows int, shade func(x, y float64) color.RGBA) {
	w, h := sfc.(*ebitenSurface).image.Size()
	vals := make([]color.RGBA, 0, (cols+1)*(rows+1))

	for j := 0; j <= rows; j++ {
		for i := 0; i <= cols; i++ {
			vals = append(vals, shade(float64(i*w)/float64(cols), float64(j*h)/float64(rows)))
		}
	}

	s.RenderShadedGrid(sfc, cols, rows, vals)
}

// shadedScratch is reused by every RenderShadedGrid call (rendering is single threaded; ebiten copies
// the vertices and indices it is given).
//
//nolint:gochecknoglobals // scratch buffers
var shadedScratch struct {
	vertices []ebiten.Vertex
	indices  map[[2]int][]uint16
}

// shadedIndices returns the triangle indices of a cols x rows grid (shared, read only).
func shadedIndices(cols, rows int) []uint16 {
	key := [2]int{cols, rows}
	if idx, ok := shadedScratch.indices[key]; ok {
		return idx
	}

	if shadedScratch.indices == nil {
		shadedScratch.indices = make(map[[2]int][]uint16)
	}

	indices := make([]uint16, 0, cols*rows*6)
	stride := cols + 1

	for j := 0; j < rows; j++ {
		for i := 0; i < cols; i++ {
			a := uint16(j*stride + i)
			b, c, d := a+1, a+uint16(stride), a+uint16(stride)+1
			indices = append(indices, a, b, c, b, d, c)
		}
	}

	shadedScratch.indices[key] = indices

	return indices
}

// RenderShadedGrid is RenderShaded with the vertex colours already computed, row by row
// (d2interface.ShadedGridSurface).
func (s *ebitenSurface) RenderShadedGrid(sfc d2interface.Surface, cols, rows int, vals []color.RGBA) {
	src := sfc.(*ebitenSurface).image
	w, h := src.Size()

	opts := s.createDrawImageOptions()
	s.handleStateEffect(opts)

	vertices := shadedScratch.vertices[:0]

	for j := 0; j <= rows; j++ {
		sy := float64(j*h) / float64(rows)

		for i := 0; i <= cols; i++ {
			sx := float64(i*w) / float64(cols)
			dx, dy := opts.GeoM.Apply(sx, sy)
			c := vals[j*(cols+1)+i]

			vertices = append(vertices, ebiten.Vertex{
				DstX: float32(dx), DstY: float32(dy), SrcX: float32(sx), SrcY: float32(sy),
				ColorR: float32(c.R) / maxAlpha, ColorG: float32(c.G) / maxAlpha,
				ColorB: float32(c.B) / maxAlpha, ColorA: float32(c.A) / maxAlpha,
			})
		}
	}

	shadedScratch.vertices = vertices

	s.image.DrawTriangles(vertices, shadedIndices(cols, rows), src, &ebiten.DrawTrianglesOptions{
		ColorM: opts.ColorM, CompositeMode: opts.CompositeMode, Filter: opts.Filter,
	})
}

// burnFillMatrix maps a pixel to (rgb + (1 - alpha), alpha 1): opaque pixels keep their colour, fully
// transparent ones become opaque white, which is the identity of the multiplicative blend.
func burnFillMatrix() ebiten.ColorM {
	var m ebiten.ColorM

	for c := 0; c < 3; c++ { //nolint:gomnd // r, g, b
		m.SetElement(c, 3, -1)
		m.SetElement(c, 4, 1)
	}

	m.SetElement(3, 3, 0)
	m.SetElement(3, 4, 1)

	return m
}
