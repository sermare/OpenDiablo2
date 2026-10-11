package d2interface

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"

// InputManager manages an InputService
type InputManager interface {
	Advance(elapsedTime, currentTime float64) error
	BindHandlerWithPriority(InputEventHandler, d2enum.Priority) error
	BindHandler(h InputEventHandler) error
	UnbindHandler(handler InputEventHandler) error
}

// InputInjector is implemented by an input manager that can replay synthetic events (the OD2_AUTOSCRIPT runner)
// through the same handler chain as the events polled from the window. Positions are in column space, like the
// positions handlers receive.
type InputInjector interface {
	InjectMouseMove(x, y int, mod d2enum.KeyMod)
	InjectMouseButton(down bool, b d2enum.MouseButton, mod d2enum.KeyMod, x, y int)
	InjectMouseRepeat(b d2enum.MouseButton, mod d2enum.KeyMod, x, y int)
	InjectKey(down bool, k d2enum.Key, mod d2enum.KeyMod)
}
