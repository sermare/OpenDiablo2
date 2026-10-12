package ebiten

import (
	"errors"
	"image"
	"os"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
)

const (
	defaultSaturation = 1.0
	defaultBrightness = 1.0
	defaultSkewX      = 0.0
	defaultSkewY      = 0.0
	defaultScaleX     = 1.0
	defaultScaleY     = 1.0
)

type renderCallback = func(surface d2interface.Surface) error

type updateCallback = func() error

// static check that we implement our renderer interface
var _ d2interface.Renderer = &Renderer{}

// Renderer is an implementation of a renderer
type Renderer struct {
	updateCallback
	renderCallback
	*d2util.GlyphPrinter
	lastRenderError error
	fullscreenHook  func(bool)
	turboFrame      int
	quietFrames     int
}

// Update calls the game's logical update function (the `Advance` method)
func (r *Renderer) Update() error {
	// OD2_QUIET: send the window behind the others, a few times while it opens (see quiet_darwin.go)
	if d2util.QuietEnabled() {
		if r.quietFrames < 30 || r.quietFrames%250 == 0 {
			quietWindow()
		}

		r.quietFrames++
	}

	if err := r.handleWindowShortcuts(); err != nil {
		return err
	}

	if r.updateCallback == nil {
		return errors.New("no update callback defined for ebiten renderer")
	}

	if d2util.TurboEnabled() {
		return r.turboUpdate()
	}

	return r.updateCallback()
}

// turboUpdate (OD2_TURBO) runs many 25 Hz virtual ticks in one frame, until the wall-clock budget is spent
// or a draw was requested (a capture). Every tick is the same one an unhurried run would take.
func (r *Renderer) turboUpdate() error {
	start := time.Now()
	budget := d2util.TurboBudgetSeconds()

	for n := 1; ; n++ {
		d2util.TurboAdvance()

		if err := r.updateCallback(); err != nil {
			return err
		}

		if !d2util.TurboContinue(n, time.Since(start).Seconds(), budget, d2util.TurboPeekDraw()) {
			return nil
		}
	}
}

const drawError = "no render callback defined for ebiten renderer"

// Draw updates the screen with the given *ebiten.Image
func (r *Renderer) Draw(screen *ebiten.Image) {
	r.lastRenderError = nil

	if d2util.TurboEnabled() {
		r.turboFrame++

		if !d2util.TurboShouldDraw(r.turboFrame-1, d2util.TurboDrawEvery(), d2util.TurboTakeDraw()) {
			return
		}
	}

	if r.renderCallback == nil {
		r.lastRenderError = errors.New(drawError)
		return
	}

	r.lastRenderError = r.renderCallback(createEbitenSurface(r, screen))
}

// Layout returns the logical screen size: the window (or full screen) size divided by the integer UI
// scale, never below 800x600 (see d2display). At an 800x600 window it is the original fixed size.
func (r *Renderer) Layout(outsideWidth, outsideHeight int) (width, height int) {
	d2display.SetOutside(d2display.Size{W: outsideWidth, H: outsideHeight})

	// Menus (main menu, character select, difficulty, loading) are the original 800x600 interface: they use the
	// 800x600 logical size, which Ebiten scales up to fit the window (a 4:3 picture with bars on the sides), instead
	// of standing as a small 800x600 column in the middle of a large window. In the game the logical size is the
	// whole window (the world fills it), or the size OD2_DISPLAY pinned.
	if d2display.CurrentAnchor() == d2display.AnchorCenter {
		d2display.Set(d2display.Base)

		return d2display.BaseW, d2display.BaseH
	}

	if d2display.Fixed() {
		s := d2display.Pinned()
		d2display.Set(s)

		return s.W, s.H
	}

	s := d2display.Logical(outsideWidth, outsideHeight, d2display.Scale())
	d2display.Set(s)

	return s.W, s.H
}

// ScreenSize returns the size of the monitor in full screen mode (0, 0 if unknown).
func (r *Renderer) ScreenSize() (width, height int) {
	return ebiten.ScreenSizeInFullscreen()
}

// CreateRenderer creates an ebiten renderer instance
func CreateRenderer(cfg *d2config.Configuration) (*Renderer, error) {
	result := &Renderer{
		GlyphPrinter: d2util.NewDebugPrinter(),
	}

	if cfg != nil {
		config := cfg

		ebiten.SetCursorMode(ebiten.CursorModeHidden)
		ebiten.SetFullscreen(startFullscreen(config, alwaysFullscreen()))
		ebiten.SetRunnableOnUnfocused(config.RunInBackground || d2util.QuietEnabled()) // a quiet window is never focused
		ebiten.SetVsyncEnabled(config.VsyncEnabled)
		ebiten.SetMaxTPS(config.TicksPerSecond)

		if d2util.TurboEnabled() {
			// OD2_TURBO: no display-rate cap; the update loop of turboUpdate sets the pace
			ebiten.SetVsyncEnabled(false)
			ebiten.SetMaxTPS(ebiten.UncappedTPS)
		}
	}

	return result, nil
}

// GetRendererName returns the name of the renderer
func (*Renderer) GetRendererName() string {
	return "Ebiten"
}

// SetWindowIcon sets the icon for the window, visible in the chrome of the window
func (*Renderer) SetWindowIcon(fileName string) {
	_, iconImage, err := ebitenutil.NewImageFromFile(fileName)
	if err == nil {
		ebiten.SetWindowIcon([]image.Image{iconImage})
	}
}

// IsDrawingSkipped returns a bool for whether or not the drawing has been skipped
func (r *Renderer) IsDrawingSkipped() bool {
	return r.lastRenderError != nil
}

// Run initializes the renderer
func (r *Renderer) Run(f renderCallback, u updateCallback, width, height int, title string) error {
	r.renderCallback = f
	r.updateCallback = u

	ebiten.SetWindowTitle(title)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(width, height)

	err := ebiten.RunGame(r)
	if errors.Is(err, errQuit) {
		return nil
	}

	return err
}

// CreateSurface creates a renderer surface from an existing surface
func (r *Renderer) CreateSurface(surface d2interface.Surface) (d2interface.Surface, error) {
	img := surface.(*ebitenSurface).image
	sfcState := surfaceState{
		filter:     ebiten.FilterNearest,
		effect:     d2enum.DrawEffectNone,
		saturation: defaultSaturation,
		brightness: defaultBrightness,
		skewX:      defaultSkewX,
		skewY:      defaultSkewY,
		scaleX:     defaultScaleX,
		scaleY:     defaultScaleY,
	}
	result := createEbitenSurface(r, img, sfcState)

	return result, nil
}

// NewSurface creates a new surface
func (r *Renderer) NewSurface(width, height int) d2interface.Surface {
	img := ebiten.NewImage(width, height)

	return createEbitenSurface(r, img)
}

// IsFullScreen returns a boolean for whether or not the renderer is currently set to fullscreen
func (r *Renderer) IsFullScreen() bool {
	return ebiten.IsFullscreen()
}

// SetFullScreen sets the renderer to fullscreen, given a boolean
func (r *Renderer) SetFullScreen(fullScreen bool) {
	ebiten.SetFullscreen(fullScreen)
}

// SetVSyncEnabled enables vsync, given a boolean
func (r *Renderer) SetVSyncEnabled(vsync bool) {
	ebiten.SetVsyncEnabled(vsync)
}

// GetVSyncEnabled returns a boolean for whether or not vsync is enabled
func (r *Renderer) GetVSyncEnabled() bool {
	return ebiten.IsVsyncEnabled()
}

// GetCursorPos returns the current cursor position x,y coordinates
func (r *Renderer) GetCursorPos() (x, y int) {
	if vx, vy, ok := d2gamepad.Default().Cursor(); ok {
		return vx, vy // a gamepad is in use: its virtual cursor
	}

	return ebiten.CursorPosition()
}

// CurrentFPS returns the current frames per second of the renderer
func (r *Renderer) CurrentFPS() float64 {
	return ebiten.CurrentFPS()
}

// ShowPanicScreen shows a panic message in a forever loop
func (r *Renderer) ShowPanicScreen(message string) {
	errorScreen := CreatePanicScreen(message)

	err := ebiten.RunGame(errorScreen)
	if err != nil {
		panic(err)
	}
}

// SetWindowScale sets the integer UI scale: the window becomes scale times the logical screen size
// and keeps that logical size (the interface grows with the window).
func (r *Renderer) SetWindowScale(scale int) {
	if scale < 1 {
		return
	}

	s := d2display.Get()
	d2display.SetScale(scale)

	if ebiten.IsFullscreen() {
		return
	}

	// grow the window only when the monitor has room for it (otherwise the scale is lowered by d2display.Logical)
	if mw, mh := ebiten.ScreenSizeInFullscreen(); mw > 0 && (s.W*scale > mw || s.H*scale > mh-80) {
		return
	}

	ebiten.SetWindowSize(s.W*scale, s.H*scale)
}

// SetWindowPixels resizes the window to w x h device-independent pixels (a user dragging the window edge; used by the
// OD2_AUTOFLOW resize: step of the display QA). It is a no-op in full screen.
func (r *Renderer) SetWindowPixels(w, h int) {
	if w < 1 || h < 1 || ebiten.IsFullscreen() {
		return
	}

	ebiten.SetWindowSize(w, h)
}

// SetWindowedSize leaves full screen and sizes the window to w x h device-independent pixels.
func (r *Renderer) SetWindowedSize(w, h int) {
	if w < 1 || h < 1 {
		return
	}

	if ebiten.IsFullscreen() {
		ebiten.SetFullscreen(false)
	}

	ebiten.SetWindowSize(w, h)
}

// startFullscreen decides the start state: scripted runs keep config.FullScreen (off by default); a normal start is
// full screen unless the player chose a window (the FULL SCREEN video option, Cmd+Enter, Cmd+F or the console).
func startFullscreen(cfg *d2config.Configuration, always bool) bool {
	if always {
		return cfg.WantFullscreen()
	}

	return cfg.FullScreen
}

// alwaysFullscreen is the owner's choice: the game starts full screen every time, whatever config.json says. Test and
// scripted runs (any OD2_AUTO* variable, OD2_TURBO, OD2_DISPLAY) and OD2_WINDOWED=1 keep the window.
func alwaysFullscreen() bool {
	for _, name := range []string{"OD2_WINDOWED", "OD2_DISPLAY", "OD2_TURBO", "OD2_AUTOSCRIPT", "OD2_AUTOGAME", "OD2_AUTOFLOW",
		"OD2_AUTOLEVEL", "OD2_AUTOCAST", "OD2_AUTOMONSTER", "OD2_REALMAPS"} {
		if os.Getenv(name) != "" {
			return false
		}
	}

	return true
}
