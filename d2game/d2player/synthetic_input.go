package d2player

import (
	"errors"
	"fmt"
	"strings"

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
// e.g. "left", "left+shift@400,300" or "left+ctrl". x,y are screen pixels of the
// 800x600 game screen; without them the click lands near the screen centre.
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
}

// resolveMonsterSpec replaces a "@monster" target of a click spec by the screen position of the living monster
// nearest to the hero, and returns that monster (nil when the spec has no such target).
func (g *GameControls) resolveMonsterSpec(spec string) (string, *d2mapentity.Monster, error) {
	i := strings.Index(spec, "@monster")
	if i < 0 {
		return spec, nil, nil
	}

	hx, hy := g.hero.GetPositionF()

	var best *d2mapentity.Monster

	bestD := 0.0

	for _, e := range g.mapEngine.Entities() {
		m, ok := e.(*d2mapentity.Monster)
		if !ok || !m.Alive() {
			continue
		}

		mx, my := m.GetPositionF()
		if d := (mx-hx)*(mx-hx) + (my-hy)*(my-hy); best == nil || d < bestD {
			best, bestD = m, d
		}
	}

	if best == nil {
		return spec, nil, errors.New("click @monster: no living monster")
	}

	sx, sy := g.mapRenderer.WorldToScreen(best.GetPositionF())

	return fmt.Sprintf("%s@%d,%d", spec[:i], sx, sy), best, nil
}

// AutoClick sends a synthetic mouse click (down, then up) through the handlers
// a real click reaches, for OD2_AUTOSCRIPT click: steps. A target "@monster" clicks the living monster
// nearest to the hero; the monster is made the hovered entity directly (a real hover is only known after
// the next rendered frame).
func (g *GameControls) AutoClick(spec string) error {
	spec, mon, err := g.resolveMonsterSpec(spec)
	if err != nil {
		return err
	}

	button, mod, x, y, err := ParseClickSpec(spec)
	if err != nil {
		return err
	}

	g.OnMouseMove(&synthEvent{x: x, y: y})

	if mon != nil {
		g.hud.hoveredEntity = mon
	}

	g.lastLeftBtnActionTime, g.lastRightBtnActionTime = 0, 0
	g.OnMouseButtonDown(&synthEvent{button: button, mod: mod, x: x, y: y})
	g.OnMouseButtonUp(&synthEvent{button: button, mod: mod, x: x, y: y})

	return nil
}

// AutoHoldStart presses a mouse button (spec as for click:) and keeps it down: AutoHoldTick then repeats it
// like the input manager does for a real held button, until AutoHoldEnd.
func (g *GameControls) AutoHoldStart(spec string) error {
	button, mod, x, y, err := ParseClickSpec(spec)
	if err != nil {
		return err
	}

	ev := synthEvent{button: button, mod: mod, x: x, y: y}
	g.autoHold = &autoHold{ev: ev}

	g.OnMouseMove(&synthEvent{x: x, y: y})
	g.lastLeftBtnActionTime, g.lastRightBtnActionTime = 0, 0
	g.Infof("HOLD start spec=%s", spec)
	g.OnMouseButtonDown(&ev)
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
	g.OnMouseButtonRepeat(&h.ev)

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
	g.OnMouseButtonUp(&h.ev)
	g.Infof("HOLD end seconds=%.1f frames=%d", h.t, h.repeated)

	g.autoHold = nil
}

func (g *GameControls) logHoldPos() {
	p := g.hero.Position.World()
	hover := ""
	if g.hud != nil && g.hud.hoveredEntity != nil {
		hover = g.hud.hoveredEntity.Label()
	}

	g.Infof("HOLD pos t=%.1f (%.2f,%.2f) town=%t hover=%q", g.autoHold.t, p.X(), p.Y(), g.hero.IsInTown(), hover)
}

// AutoKey sends a synthetic key press (down, then up) by key name ("Tab", "I", "Escape").
func (g *GameControls) AutoKey(name string) error {
	k, ok := KeyByName(name)
	if !ok {
		return fmt.Errorf("unknown key %q", name)
	}

	g.OnKeyDown(&synthEvent{key: k})
	g.OnKeyUp(&synthEvent{key: k})
	g.Infof("INPUT panels after %s: inventory=%t character=%t skills=%t quest=%t automap=%t menu=%t", name,
		g.inventory.IsOpen(), g.heroStatsPanel.IsOpen(), g.skilltree.IsOpen(), g.questLog.IsOpen(),
		g.automap.On(), g.escapeMenu.IsOpen())

	return nil
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
