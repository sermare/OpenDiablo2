package d2input

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
)

// Synthetic events for the OD2_AUTOSCRIPT runner (click:, hold:, press: steps). They are dispatched to every bound
// handler through the same propagate call as the events polled from the window, so a scripted click reaches what a
// real one reaches: the UI manager's buttons, the escape menu, the game controls, in priority order. Before this the
// runner called the game controls directly and the scenarios could not see a bug in the other handlers or in the
// order between them. Positions are in column space like every handler event (d2display.ToColumn).
//
// The injected position is NOT stored as the cursor of the manager: the polled cursor only fires a move event when the
// real mouse moves, and the script's position must not be undone by a stale comparison on the next frame.

var _ d2interface.InputInjector = (*inputManager)(nil)

func (im *inputManager) syntheticBase(mod d2enum.KeyMod, x, y int) HandlerEvent {
	return HandlerEvent{keyMod: mod, buttonMod: im.buttonMod, x: x, y: y}
}

// InjectMouseMove sends a mouse move event.
func (im *inputManager) InjectMouseMove(x, y int, mod d2enum.KeyMod) {
	event := MouseMoveEvent{im.syntheticBase(mod, x, y)}

	im.propagate(func(h d2interface.InputEventHandler) bool {
		if l, ok := h.(d2interface.MouseMoveHandler); ok {
			return l.OnMouseMove(&event)
		}

		return false
	})
}

// InjectMouseButton sends a button press (down) or release (not down) event.
func (im *inputManager) InjectMouseButton(down bool, b d2enum.MouseButton, mod d2enum.KeyMod, x, y int) {
	event := MouseEvent{im.syntheticBase(mod, x, y), b}

	im.propagate(func(h d2interface.InputEventHandler) bool {
		if down {
			if l, ok := h.(d2interface.MouseButtonDownHandler); ok {
				return l.OnMouseButtonDown(&event)
			}
		} else if l, ok := h.(d2interface.MouseButtonUpHandler); ok {
			return l.OnMouseButtonUp(&event)
		}

		return false
	})
}

// InjectMouseRepeat sends one frame of a held button.
func (im *inputManager) InjectMouseRepeat(b d2enum.MouseButton, mod d2enum.KeyMod, x, y int) {
	event := MouseEvent{im.syntheticBase(mod, x, y), b}

	im.propagate(func(h d2interface.InputEventHandler) bool {
		if l, ok := h.(d2interface.MouseButtonRepeatHandler); ok {
			return l.OnMouseButtonRepeat(&event)
		}

		return false
	})
}

// InjectKey sends a key press (down) or release (not down) event.
func (im *inputManager) InjectKey(down bool, k d2enum.Key, mod d2enum.KeyMod) {
	event := KeyEvent{HandlerEvent: im.syntheticBase(mod, im.cursorX, im.cursorY), key: k, duration: 1}

	im.propagate(func(h d2interface.InputEventHandler) bool {
		if down {
			if l, ok := h.(d2interface.KeyDownHandler); ok {
				return l.OnKeyDown(&event)
			}
		} else if l, ok := h.(d2interface.KeyUpHandler); ok {
			return l.OnKeyUp(&event)
		}

		return false
	})
}
