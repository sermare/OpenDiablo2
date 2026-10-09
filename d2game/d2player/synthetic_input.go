package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
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

// ParseClickSpec parses a click step argument: "<left|right>[+shift][+ctrl][+alt][@x,y]",
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
		case "alt", "option":
			mod |= d2enum.KeyModAlt
		default:
			return 0, 0, 0, 0, fmt.Errorf("click modifier %q: want shift, ctrl or alt", m)
		}
	}

	return button, mod, x, y, nil
}

// AutoClick sends a synthetic mouse click (down, then up) through the handlers
// a real click reaches, for OD2_AUTOSCRIPT click: steps.
func (g *GameControls) AutoClick(spec string) error {
	button, mod, x, y, err := ParseClickSpec(spec)
	if err != nil {
		return err
	}

	g.OnMouseMove(&synthEvent{x: x, y: y})
	g.lastLeftBtnActionTime, g.lastRightBtnActionTime = 0, 0
	g.OnMouseButtonDown(&synthEvent{button: button, mod: mod, x: x, y: y})
	g.OnMouseButtonUp(&synthEvent{button: button, mod: mod, x: x, y: y})

	return nil
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
