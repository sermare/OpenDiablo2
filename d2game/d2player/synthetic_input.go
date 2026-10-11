package d2player

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// synthEvent is an input event made by the OD2_AUTOSCRIPT runner (press: and
// click: steps) instead of the window system, so scenarios can drive the very
// same handlers as a real keyboard and mouse.
type synthEvent struct {
	key    d2enum.Key
	button d2enum.MouseButton
	mod    d2enum.KeyMod
	x, y   int
}

func (e *synthEvent) KeyMod() d2enum.KeyMod            { return e.mod }
func (e *synthEvent) ButtonMod() d2enum.MouseButtonMod { return 0 }
func (e *synthEvent) X() int                           { return e.x }
func (e *synthEvent) Y() int                           { return e.y }
func (e *synthEvent) Key() d2enum.Key                  { return e.key }
func (e *synthEvent) Duration() int                    { return 1 }
func (e *synthEvent) Button() d2enum.MouseButton       { return e.button }

// ParseClickSpec parses a click step argument: "<left|right>[+shift][+ctrl][+cmd][+alt][@x,y]",
// e.g. "left", "left+shift@400,300" or "left+ctrl". x,y are screen pixels (the GameControls
// methods also take @hero:dx,dy and @ui:x,y, see parseClick); without them the click lands near the
// centre of an 800x600 screen.
func ParseClickSpec(spec string) (button d2enum.MouseButton, mod d2enum.KeyMod, x, y int, err error) {
	const centerX, centerY = 400, 280

	x, y = centerX, centerY

	if i := strings.Index(spec, "@"); i >= 0 {
		if _, e := fmt.Sscanf(spec[i+1:], "%d,%d", &x, &y); e != nil {
			return 0, 0, 0, 0, fmt.Errorf("click position %q: want x,y", spec[i+1:])
		}

		spec = spec[:i]
	}

	parts := strings.Split(strings.ToLower(strings.TrimSpace(spec)), "+")

	switch parts[0] {
	case "left":
		button = d2enum.MouseButtonLeft
	case "right":
		button = d2enum.MouseButtonRight
	default:
		return 0, 0, 0, 0, fmt.Errorf("click button %q: want left or right", parts[0])
	}

	for _, m := range parts[1:] {
		switch m {
		case "shift":
			mod |= d2enum.KeyModShift
		case "ctrl", "control":
			mod |= d2enum.KeyModControl
		case "cmd", "command", "super", "meta":
			mod |= d2enum.KeyModSuper
		case "alt", "option":
			mod |= d2enum.KeyModAlt
		default:
			return 0, 0, 0, 0, fmt.Errorf("click modifier %q: want shift, ctrl, cmd or alt", m)
		}
	}

	return button, mod, x, y, nil
}

// autoHold is the state of a hold: step (a mouse button kept down by the script).
type autoHold struct {
	ev       synthEvent
	t        float64 // seconds held
	nextLog  float64
	repeated int
	hover    string // label of the entity under the cursor when last logged
}

// parseClick resolves the position forms of a click spec and returns the event in column space (what the
// handlers receive). Positions: "@x,y" screen pixels; "@hero:dx,dy" relative to the hero's screen position
// (resolution independent); "@ui:x,y" pixels of the 800x600 interface column; "@monster" (see
// resolveMonsterSpec). Without a position the click lands just above the centre of the screen.
func (g *GameControls) parseClick(spec string) (d2enum.MouseButton, d2enum.KeyMod, int, int, error) {
	sx, sy := d2display.W()/2, d2display.H()/2-20 //nolint:gomnd // just above the hero

	if i := strings.Index(spec, "@hero"); i >= 0 {
		dx, dy := 0, 0
		if rest := spec[i+len("@hero"):]; strings.HasPrefix(rest, ":") {
			if _, e := fmt.Sscanf(rest[1:], "%d,%d", &dx, &dy); e != nil {
				return 0, 0, 0, 0, fmt.Errorf("click position %q: want @hero:dx,dy", rest)
			}
		}

		hx, hy := g.mapRenderer.WorldToScreen(g.hero.GetPositionF())
		spec = fmt.Sprintf("%s@%d,%d", spec[:i], hx+dx, hy+dy)
	} else if i := strings.Index(spec, "@ui:"); i >= 0 {
		var ux, uy int
		if _, e := fmt.Sscanf(spec[i+len("@ui:"):], "%d,%d", &ux, &uy); e != nil {
			return 0, 0, 0, 0, fmt.Errorf("click position %q: want @ui:x,y", spec[i:])
		}

		hx, hy := d2display.ToScreen(ux, uy)
		spec = fmt.Sprintf("%s@%d,%d", spec[:i], hx, hy)
	} else if !strings.Contains(spec, "@") {
		spec = fmt.Sprintf("%s@%d,%d", spec, sx, sy)
	}

	button, mod, x, y, err := ParseClickSpec(spec)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	cx, cy := d2display.ToColumn(x, y)

	return button, mod, cx, cy, nil
}

// entityTarget is an entity form of a click position: "@monster", "@npc[:label]", "@object:<id or label>" or
// "@item[:label]", optionally followed by "*f" (f in [0,1]: the point that far from the hero towards the entity on
// the screen, so "@npc:Akara*0.5" is ground half way to Akara and a hold started there walks up to her without
// starting on her). The label is matched case-insensitively as a substring of the entity's label; objects also
// by objects.txt id.
type entityTarget struct {
	kind  string
	name  string
	frac  float64
	start int // index of the "@" in the spec
}

var entityKinds = []string{"monster", "npc", "object", "item"} //nolint:gochecknoglobals // constants

// parseEntityTarget finds an entity target in a click spec. ok is false for the other position forms
// (screen pixels, @hero:, @ui:) and for specs without a position.
func parseEntityTarget(spec string) (t entityTarget, ok bool, err error) {
	i := strings.Index(spec, "@")
	if i < 0 {
		return t, false, nil
	}

	rest := spec[i+1:]
	frac := 1.0

	if j := strings.LastIndex(rest, "*"); j >= 0 {
		f, perr := strconv.ParseFloat(rest[j+1:], 64)
		if perr != nil || f < 0 || f > 1 {
			return t, false, fmt.Errorf("click position %q: want *f with f in 0..1", rest)
		}

		frac, rest = f, rest[:j]
	}

	kind, name := rest, ""
	if j := strings.Index(rest, ":"); j >= 0 {
		kind, name = rest[:j], rest[j+1:]
	}

	found := false

	for _, k := range entityKinds {
		found = found || kind == k
	}

	if !found {
		return t, false, nil
	}

	if kind == "object" && strings.TrimSpace(name) == "" {
		return t, false, errors.New("click @object: needs :<id or label>")
	}

	return entityTarget{kind: kind, name: strings.TrimSpace(name), frac: frac, start: i}, true, nil
}

// matches reports whether the entity is what the target names.
func (t entityTarget) matches(e d2interface.MapEntity) bool {
	switch v := e.(type) {
	case *d2mapentity.Monster:
		return t.kind == "monster" && v.Alive()
	case *d2mapentity.NPC:
		return t.kind == "npc" && labelHas(e, t.name)
	case *d2mapentity.Item:
		return t.kind == "item" && labelHas(e, t.name)
	case *d2mapentity.Object:
		if t.kind != "object" {
			return false
		}

		if id, err := strconv.Atoi(t.name); err == nil {
			return v.Record() != nil && v.Record().Index == id
		}

		return labelHas(e, t.name)
	}

	return false
}

func labelHas(e d2interface.MapEntity, sub string) bool {
	return sub == "" || strings.Contains(strings.ToLower(e.Label()), strings.ToLower(sub))
}

// resolveEntitySpec replaces an entity target of a click spec by a screen position and returns the entity (nil
// when the spec has no entity target, or the position is short of it: *f with f below 1). The entity is the one
// nearest to the hero. Every resolved target is logged (CLICK target ...), the world position included, so a
// scenario can place its expectations.
func (g *GameControls) resolveEntitySpec(spec string) (string, d2interface.MapEntity, error) {
	t, ok, err := parseEntityTarget(spec)
	if err != nil || !ok {
		return spec, nil, err
	}

	hx, hy := g.hero.GetPositionF()

	var best d2interface.MapEntity

	bestD := 0.0

	for _, e := range g.mapEngine.Entities() {
		if !t.matches(e) {
			continue
		}

		mx, my := e.GetPositionF()
		if d := (mx-hx)*(mx-hx) + (my-hy)*(my-hy); best == nil || d < bestD {
			best, bestD = e, d
		}
	}

	if best == nil {
		return spec, nil, fmt.Errorf("click %s: no such entity", spec[t.start:])
	}

	wx, wy := best.GetPositionF()
	ex, ey := g.mapRenderer.WorldToScreen(wx, wy)
	sx, sy := g.mapRenderer.WorldToScreen(hx, hy)

	g.Infof("CLICK target kind=%s label=%q world=(%.2f,%.2f) hero=(%.2f,%.2f) dist=%.2f frac=%.2f", t.kind,
		best.Label(), wx, wy, hx, hy, math.Sqrt(bestD), t.frac)

	if t.frac < 1 {
		ex, ey = sx+int(math.Round(float64(ex-sx)*t.frac)), sy+int(math.Round(float64(ey-sy)*t.frac))
		best = nil
	}

	return fmt.Sprintf("%s@%d,%d", spec[:t.start], ex, ey), best, nil
}

// Pointer events of the script go through the input manager when it can inject them (the handlers of the window
// system see them in priority order, the UI manager's buttons and the escape menu included), else straight to the
// game controls (unit tests, hosts without a manager).

func (g *GameControls) synthMove(x, y int) {
	if g.injector != nil {
		g.injector.InjectMouseMove(x, y, 0)
		return
	}

	g.OnMouseMove(&synthEvent{x: x, y: y})
}

func (g *GameControls) synthButton(down bool, e *synthEvent) {
	switch {
	case g.injector != nil:
		g.injector.InjectMouseButton(down, e.button, e.mod, e.x, e.y)
	case down:
		g.OnMouseButtonDown(e)
	default:
		g.OnMouseButtonUp(e)
	}
}

func (g *GameControls) synthRepeat(e *synthEvent) {
	if g.injector != nil {
		g.injector.InjectMouseRepeat(e.button, e.mod, e.x, e.y)
		return
	}

	g.OnMouseButtonRepeat(e)
}

// SetInputInjector makes the script's clicks and keys go through the input manager (see synthMove).
func (g *GameControls) SetInputInjector(in d2interface.InputInjector) { g.injector = in }

// AutoClick sends a synthetic mouse click (down, then up) through the handlers a real click reaches, for
// OD2_AUTOSCRIPT click: steps. An entity target (@monster, @npc, @object:, @item, see entityTarget) clicks the
// entity nearest to the hero and makes it the hovered entity directly (a real hover is only known after the next
// rendered frame). Any other position finds its hovered entity by the same box test as the renderer, so a click
// on the ground never inherits the hover of the previous position.
func (g *GameControls) AutoClick(spec string) error {
	spec, ent, err := g.resolveEntitySpec(spec)
	if err != nil {
		return err
	}

	button, mod, x, y, err := g.parseClick(spec)
	if err != nil {
		return err
	}

	g.synthMove(x, y)
	g.setHovered(ent, x, y)

	g.lastLeftBtnActionTime, g.lastRightBtnActionTime = 0, 0
	g.synthButton(true, &synthEvent{button: button, mod: mod, x: x, y: y})
	g.synthButton(false, &synthEvent{button: button, mod: mod, x: x, y: y})
	g.logPanelState("click")

	return nil
}

// setHovered fixes the hovered entity of a synthetic click: the entity when there is one, else whatever the
// renderer's test finds at the column position.
func (g *GameControls) setHovered(ent d2interface.MapEntity, cx, cy int) {
	if g.hud == nil {
		return
	}

	if ent == nil {
		ent = g.hud.entityAt(cx, cy)
	}

	g.hud.hoveredEntity = ent
}

// AutoHoldStart presses a mouse button (spec as for click:) and keeps it down: AutoHoldTick then repeats it
// like the input manager does for a real held button, until AutoHoldEnd.
func (g *GameControls) AutoHoldStart(spec string) error {
	spec, ent, err := g.resolveEntitySpec(spec)
	if err != nil {
		return err
	}

	button, mod, x, y, err := g.parseClick(spec)
	if err != nil {
		return err
	}

	ev := synthEvent{button: button, mod: mod, x: x, y: y}
	g.autoHold = &autoHold{ev: ev}

	g.synthMove(x, y)
	g.setHovered(ent, x, y)
	g.autoHold.hover = g.hoveredLabel()

	g.lastLeftBtnActionTime, g.lastRightBtnActionTime = 0, 0
	g.Infof("HOLD start spec=%s hover=%q", spec, g.autoHold.hover)
	g.synthButton(true, &ev)
	g.logHoldPos()

	return nil
}

// AutoHoldTick is one frame of a held button (the input manager's updatePressedButton).
func (g *GameControls) AutoHoldTick(elapsed float64) {
	h := g.autoHold
	if h == nil {
		return
	}

	h.t += elapsed
	h.repeated++
	g.synthRepeat(&h.ev)

	// the entity under the cursor changes as the hero walks past it: log every change (HOLD hover), so a
	// scenario can prove that an NPC or object really was under the cursor while the hold went on
	if cur := g.hoveredLabel(); cur != h.hover {
		h.hover = cur
		g.Infof("HOLD hover t=%.1f %q", h.t, cur)
	}

	const logEvery = 0.5
	if h.t >= h.nextLog+logEvery {
		h.nextLog = h.t
		g.logHoldPos()
	}
}

// AutoHoldEnd releases the held button.
func (g *GameControls) AutoHoldEnd() {
	h := g.autoHold
	if h == nil {
		return
	}

	g.logHoldPos()
	g.synthButton(false, &h.ev)
	g.Infof("HOLD end seconds=%.1f frames=%d", h.t, h.repeated)

	g.autoHold = nil
}

func (g *GameControls) hoveredLabel() string {
	if g.hud != nil && g.hud.hoveredEntity != nil {
		return g.hud.hoveredEntity.Label()
	}

	return ""
}

func (g *GameControls) logHoldPos() {
	p := g.hero.Position.World()
	hover := g.hoveredLabel()

	g.Infof("HOLD pos t=%.1f (%.2f,%.2f) town=%t hover=%q", g.autoHold.t, p.X(), p.Y(), g.hero.IsInTown(), hover)
}

// parseKeySpec reads "<Key>" or "shift+ctrl+<Key>" (modifiers shift, ctrl, cmd, alt).
func parseKeySpec(spec string) (d2enum.Key, d2enum.KeyMod, error) {
	var mod d2enum.KeyMod

	parts := strings.Split(spec, "+")

	for _, m := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(m)) {
		case "shift":
			mod |= d2enum.KeyModShift
		case "ctrl", "control":
			mod |= d2enum.KeyModControl
		case "cmd", "command", "super", "meta":
			mod |= d2enum.KeyModSuper
		case "alt", "option":
			mod |= d2enum.KeyModAlt
		default:
			return 0, 0, fmt.Errorf("key modifier %q: want shift, ctrl, cmd or alt", m)
		}
	}

	name := strings.TrimSpace(parts[len(parts)-1])

	k, ok := KeyByName(name)
	if !ok {
		return 0, 0, fmt.Errorf("unknown key %q", name)
	}

	return k, mod, nil
}

// AutoKey sends a synthetic key press (down, then up) by key name ("Tab", "I", "Escape", "F1"), optionally with
// modifiers ("shift+F1"). It goes through the input manager like a real key.
func (g *GameControls) AutoKey(spec string) error {
	k, mod, err := parseKeySpec(spec)
	if err != nil {
		return err
	}

	if g.injector != nil {
		g.injector.InjectKey(true, k, mod)
		g.injector.InjectKey(false, k, mod)
	} else {
		g.OnKeyDown(&synthEvent{key: k, mod: mod})
		g.OnKeyUp(&synthEvent{key: k, mod: mod})
	}

	g.logPanelState(spec)

	return nil
}

// logPanelState logs which panels are open after a synthetic input, and the other state a click or key can change.
func (g *GameControls) logPanelState(after string) {
	g.Infof("INPUT panels after %s: inventory=%t character=%t skills=%t quest=%t automap=%t menu=%t stash=%t cube=%t "+
		"running=%t cursor_item=%t skillmenu=%t", after,
		g.inventory.IsOpen(), g.heroStatsPanel.IsOpen(), g.skilltree.IsOpen(), g.questLog.IsOpen(),
		g.automap.On(), g.escapeMenu.IsOpen(), g.stash.IsOpen(), g.cube.IsOpen(),
		g.hero.IsRunToggled(), g.inventory.CursorItem() != nil, g.hud.skillSelectMenu.IsOpen())
}

// commandBindKey is the console command "bindkey <event> <key>": it binds the
// key as the event's primary key (what the Configure Controls page does) and
// saves the bindings to the configuration.
func (g *GameControls) commandBindKey(term d2interface.Terminal) func(args []string) error {
	return func(args []string) error {
		var ev d2enum.GameEvent

		for e, n := range gameEventNames {
			if strings.EqualFold(n, args[0]) {
				ev = e
			}
		}

		k, ok := KeyByName(args[1])
		if ev == 0 || !ok {
			term.Errorf("unknown event %q or key %q", args[0], args[1])
			return nil
		}

		g.keyMap.SetPrimaryBinding(ev, k)

		if err := g.keyMap.SaveBindings(); err != nil {
			term.Errorf("could not save: %v", err)
		}

		g.Infof("KEYS bound event=%s key=%s", gameEventNames[ev], KeyName(k))

		return nil
	}
}
