package d2gamepad

import (
	"fmt"
	"math"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// Config tunes the virtual input.
type Config struct {
	Deadzone    float64 // radial stick dead zone (0..1)
	CursorSpeed float64 // pixels per second at full right stick deflection
	MoveRadius  float64 // pixels from the hero at which the left stick places the move target
	ScreenW     int
	ScreenH     int
	HeroX       int // where the hero stands on the screen
	HeroY       int
}

// DefaultConfig fits the 800x600 game screen.
func DefaultConfig() Config {
	return Config{Deadzone: 0.25, CursorSpeed: 520, MoveRadius: 150, ScreenW: 800, ScreenH: 600, HeroX: 400, HeroY: 270}
}

// PadInfo identifies a connected controller (for hot-plug events).
type PadInfo struct {
	ID    int
	Name  string
	State State
}

// KeyResolver finds the key bound to a game event; ok=false falls back to the
// default key of the action.
type KeyResolver func(e d2enum.GameEvent) (d2enum.Key, bool)

// Controller produces the virtual keyboard and mouse input of each frame from
// the gamepad state. It is safe for concurrent use.
type Controller struct {
	mu sync.Mutex

	cfg     Config
	mapping Mapping
	resolve KeyResolver
	handler func(Action)
	uiMode  func() bool

	// frame state
	pads           map[int]string // connected controllers
	pendingHandler []Action
	prevPress      [NumButtons]bool
	keys, prev     map[d2enum.Key]bool
	keyFrames      map[d2enum.Key]int
	mouse          [3]bool
	prevMouse      [3]bool
	cx, cy         float64
	active         bool // the pad drives the cursor (until the real mouse moves)
	lastMX         int
	lastMY         int
	haveMouse      bool
	moving         bool
	cursorMoving   bool
	log            []string
}

// NewController creates a controller with the default mapping.
func NewController(cfg Config) *Controller {
	return &Controller{
		cfg:       cfg,
		mapping:   DefaultMapping(),
		pads:      map[int]string{},
		keys:      map[d2enum.Key]bool{},
		prev:      map[d2enum.Key]bool{},
		keyFrames: map[d2enum.Key]int{},
		cx:        float64(cfg.ScreenW) / 2,
		cy:        float64(cfg.ScreenH) / 2,
	}
}

// SetMapping replaces the button mapping.
func (c *Controller) SetMapping(m Mapping) {
	c.mu.Lock()
	c.mapping = m
	c.mu.Unlock()
}

// Mapping returns the current mapping.
func (c *Controller) Mapping() Mapping {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.mapping
}

// SetKeyResolver registers how key actions find the player's bound key.
func (c *Controller) SetKeyResolver(r KeyResolver) {
	c.mu.Lock()
	c.resolve = r
	c.mu.Unlock()
}

// SetActionHandler registers the game's handler of the non-key actions
// (skill cycling); it is called once per button press, on the frame it happens.
func (c *Controller) SetActionHandler(h func(Action)) {
	c.mu.Lock()
	c.handler = h
	c.mu.Unlock()
}

// SetUIMode registers a probe telling whether the player is in a menu or has a
// panel open; then the left stick moves the cursor instead of walking. Without
// a probe the left stick always walks.
func (c *Controller) SetUIMode(probe func() bool) {
	c.mu.Lock()
	c.uiMode = probe
	c.mu.Unlock()
}

// SetScreen sets the logical screen size and the hero's screen position.
func (c *Controller) SetScreen(w, h, heroX, heroY int) {
	c.mu.Lock()
	c.cfg.ScreenW, c.cfg.ScreenH, c.cfg.HeroX, c.cfg.HeroY = w, h, heroX, heroY
	c.mu.Unlock()
}

// DrainLog returns and clears the pending log lines (connect, disconnect,
// actions) for the caller to write to the game log.
func (c *Controller) DrainLog() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	r := c.log
	c.log = nil

	return r
}

func (c *Controller) logf(format string, a ...interface{}) {
	c.log = append(c.log, fmt.Sprintf(format, a...))
}

// radial applies a radial dead zone and rescales the remaining range to 0..1.
func radial(x, y, dead float64) (float64, float64, float64) {
	m := math.Hypot(x, y)
	if m < dead || m == 0 {
		return 0, 0, 0
	}

	if m > 1 {
		x, y, m = x/m, y/m, 1
	}

	scale := (m - dead) / (1 - dead) / m

	return x * scale, y * scale, (m - dead) / (1 - dead)
}

// Update advances one frame. pads are the connected controllers (empty when
// none), mouseX/mouseY the real mouse position, dt the elapsed seconds.
func (c *Controller) Update(dt float64, pads []PadInfo, mouseX, mouseY int) {
	c.mu.Lock()

	if dt <= 0 || dt > 0.25 {
		dt = 1.0 / 60
	}

	c.syncPads(pads)

	var st State
	for _, p := range pads {
		st = st.Merge(p.State)
	}

	padBusy := false
	for i, down := range st.Pressed {
		padBusy = padBusy || (down && c.mapping[i] != ActionNone)
	}

	for _, v := range st.Axes {
		padBusy = padBusy || abs(v) >= c.cfg.Deadzone
	}

	// real mouse movement takes the cursor back (unless the pad is being used at that moment)
	if c.haveMouse && !padBusy && (mouseX != c.lastMX || mouseY != c.lastMY) {
		if c.active {
			c.logf("GAMEPAD cursor handed back to the mouse")
		}

		c.active = false
		c.cx, c.cy = float64(mouseX), float64(mouseY)
	}

	c.lastMX, c.lastMY, c.haveMouse = mouseX, mouseY, true

	uiMode := c.uiMode == nil || c.uiMode()

	// sticks
	lx, ly, lm := radial(st.Axes[AxisLX], st.Axes[AxisLY], c.cfg.Deadzone)
	rx, ry, rm := radial(st.Axes[AxisRX], st.Axes[AxisRY], c.cfg.Deadzone)

	if lm > 0 || rm > 0 {
		c.active = true
	}

	moveLeft := false

	switch {
	case lm > 0 && !uiMode:
		// walking: the move target is a point in the stick direction from the hero
		c.cx = clampF(float64(c.cfg.HeroX)+lx/lm*c.cfg.MoveRadius*math.Max(lm, 0.4), 0, float64(c.cfg.ScreenW-1))
		c.cy = clampF(float64(c.cfg.HeroY)+ly/lm*c.cfg.MoveRadius*math.Max(lm, 0.4)*0.75, 0, float64(c.cfg.ScreenH-1))
		moveLeft = true
	case lm > 0:
		c.cx += lx * lm * c.cfg.CursorSpeed * dt
		c.cy += ly * lm * c.cfg.CursorSpeed * dt
	}

	if cursorStick := rm > 0 || (lm > 0 && uiMode); cursorStick != c.cursorMoving {
		c.cursorMoving = cursorStick
		if !cursorStick {
			c.logf("GAMEPAD cursor stopped at (%d,%d)", int(c.cx), int(c.cy))
		}
	}

	if rm > 0 {
		// quadratic response: fine control near the centre, fast at the edge
		c.cx += rx * rm * c.cfg.CursorSpeed * dt
		c.cy += ry * rm * c.cfg.CursorSpeed * dt
	}

	c.cx = clampF(c.cx, 0, float64(c.cfg.ScreenW-1))
	c.cy = clampF(c.cy, 0, float64(c.cfg.ScreenH-1))

	if moveLeft != c.moving {
		c.moving = moveLeft
		if !moveLeft { // the walk target was a point beside the hero: the cursor goes back to the hero
			c.cx, c.cy = float64(c.cfg.HeroX), float64(c.cfg.HeroY)
		}

		c.logf("GAMEPAD walk=%v target=(%d,%d)", moveLeft, int(c.cx), int(c.cy))
	}

	// buttons -> actions
	c.prev, c.keys = c.keys, map[d2enum.Key]bool{}
	c.prevMouse = c.mouse
	c.mouse = [3]bool{moveLeft, false, false}

	for i := range st.Pressed {
		down := st.Pressed[i]
		if down {
			c.active = true
		}

		act := c.mapping[i]
		justPressed := down && !c.prevPress[i]

		if justPressed && act != ActionNone {
			c.logf("GAMEPAD button=%s action=%s", Button(i), act)
		}

		c.applyAction(act, down, justPressed)
		c.prevPress[i] = down
	}

	for k := range c.keys {
		if c.prev[k] {
			c.keyFrames[k]++
		} else {
			c.keyFrames[k] = 1
		}
	}

	for k := range c.prev {
		if !c.keys[k] {
			delete(c.keyFrames, k)
		}
	}

	handler := c.handler
	pending := c.pendingHandler
	c.pendingHandler = nil

	c.mu.Unlock()

	if handler != nil {
		for _, a := range pending {
			handler(a)
		}
	}
}

func (c *Controller) applyAction(a Action, down, justPressed bool) {
	switch {
	case a.isMouse():
		if !down {
			return
		}

		switch a {
		case ActionClick, ActionLeftSkill:
			c.mouse[0] = true
		default:
			c.mouse[2] = true
		}
	case a.isKey():
		if !down {
			return
		}

		if k, ok := c.keyFor(a); ok {
			c.keys[k] = true
		}
	case a.isHandler():
		if justPressed {
			c.pendingHandler = append(c.pendingHandler, a)
		}
	}
}

func (c *Controller) keyFor(a Action) (d2enum.Key, bool) {
	info := actionInfos[a]

	if info.event != 0 && c.resolve != nil {
		if k, ok := c.resolve(info.event); ok {
			return k, true
		}
	}

	return info.key, true
}

func (c *Controller) syncPads(pads []PadInfo) {
	seen := map[int]bool{}

	for _, p := range pads {
		seen[p.ID] = true

		if _, ok := c.pads[p.ID]; !ok {
			c.pads[p.ID] = p.Name
			c.logf("GAMEPAD connected id=%d name=%q", p.ID, p.Name)
		}
	}

	for id, name := range c.pads {
		if !seen[id] {
			delete(c.pads, id)
			c.logf("GAMEPAD disconnected id=%d name=%q", id, name)
			// a pulled controller must not leave buttons held
			c.prevPress = [NumButtons]bool{}
		}
	}

	if len(c.pads) == 0 {
		c.active = false
	}
}

// Connected reports how many controllers are connected.
func (c *Controller) Connected() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.pads)
}

// KeyDown reports whether a virtual key is held this frame.
func (c *Controller) KeyDown(k d2enum.Key) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.keys[k]
}

// KeyJustPressed reports a virtual key that went down this frame.
func (c *Controller) KeyJustPressed(k d2enum.Key) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.keys[k] && !c.prev[k]
}

// KeyJustReleased reports a virtual key that went up this frame.
func (c *Controller) KeyJustReleased(k d2enum.Key) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.keys[k] && c.prev[k]
}

// KeyDuration is how many frames a virtual key has been held.
func (c *Controller) KeyDuration(k d2enum.Key) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.keyFrames[k]
}

func mouseIndex(b d2enum.MouseButton) int {
	switch b {
	case d2enum.MouseButtonLeft:
		return 0
	case d2enum.MouseButtonMiddle:
		return 1
	case d2enum.MouseButtonRight:
		return 2
	}

	return -1
}

// MouseDown reports whether a virtual mouse button is held.
func (c *Controller) MouseDown(b d2enum.MouseButton) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	i := mouseIndex(b)

	return i >= 0 && c.mouse[i]
}

// MouseJustPressed reports a virtual mouse button that went down this frame.
func (c *Controller) MouseJustPressed(b d2enum.MouseButton) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	i := mouseIndex(b)

	return i >= 0 && c.mouse[i] && !c.prevMouse[i]
}

// MouseJustReleased reports a virtual mouse button that went up this frame.
func (c *Controller) MouseJustReleased(b d2enum.MouseButton) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	i := mouseIndex(b)

	return i >= 0 && !c.mouse[i] && c.prevMouse[i]
}

// Cursor returns the virtual cursor and whether it is in charge (a controller
// is connected and was used more recently than the mouse).
func (c *Controller) Cursor() (x, y int, active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return int(c.cx), int(c.cy), c.active
}

// Walking reports that the left stick is walking the hero (its click is a move order, whatever the left skill is).
func (c *Controller) Walking() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.moving
}

func clampF(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
