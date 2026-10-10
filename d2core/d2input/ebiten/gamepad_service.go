package ebiten

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
)

// GamepadService is the InputService of the game: the keyboard and mouse of
// InputService plus the virtual keys, mouse buttons and cursor that the
// gamepad controller derives from the connected controllers (and from the
// synthetic controller of the autoscript harness). Without a controller it
// behaves exactly like InputService.
type GamepadService struct {
	InputService

	ctl      *d2gamepad.Controller
	profiles *d2gamepad.Profiles
	synth    *d2gamepad.Synthetic
	logger   *d2util.Logger
	reported map[ebiten.GamepadID]bool
}

// NewGamepadService wires the service to the default controller.
func NewGamepadService() *GamepadService {
	l := d2util.NewLogger()
	l.SetPrefix("Gamepad")

	return &GamepadService{
		ctl:      d2gamepad.Default(),
		profiles: d2gamepad.DefaultProfiles(),
		synth:    d2gamepad.DefaultSynthetic(),
		logger:   l,
		reported: map[ebiten.GamepadID]bool{},
	}
}

// Update polls the controllers; the input manager calls it once per frame
// before it asks for any key.
func (s *GamepadService) Update(elapsed float64) {
	var pads []d2gamepad.PadInfo

	for _, id := range ebiten.GamepadIDs() {
		name := ebiten.GamepadName(id)
		prof := s.profiles.For(name, ebiten.GamepadSDLID(id))

		nb, na := ebiten.GamepadButtonNum(id), ebiten.GamepadAxisNum(id)
		buttons := make([]bool, nb)
		axes := make([]float64, na)

		for i := range buttons {
			buttons[i] = ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(i))
		}

		for i := range axes {
			axes[i] = ebiten.GamepadAxis(id, i)
		}

		if !s.reported[id] {
			s.reported[id] = true
			s.logger.Infof("GAMEPAD raw layout id=%d name=%q guid=%s profile=%s buttons=%d axes=%d",
				id, name, ebiten.GamepadSDLID(id), prof.Name, nb, na)
		}

		pads = append(pads, d2gamepad.PadInfo{ID: int(id), Name: name, State: prof.Translate(buttons, axes)})
	}

	for id := range s.reported {
		if !contains(ebiten.GamepadIDs(), id) {
			delete(s.reported, id)
		}
	}

	if p, ok := s.synth.Poll(); ok {
		pads = append(pads, p)
	}

	mx, my := ebiten.CursorPosition()
	sz := d2display.Get() // the virtual cursor and the hero stand on the real screen
	s.ctl.SetScreen(sz.W, sz.H, sz.W/2, sz.H/2-30)
	s.ctl.Update(elapsed, pads, mx, my)

	for _, line := range s.ctl.DrainLog() {
		s.logger.Info(line)
	}
}

func contains(ids []ebiten.GamepadID, id ebiten.GamepadID) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}

	return false
}

// CursorPosition is the gamepad cursor while a controller is in use, the mouse otherwise.
func (s *GamepadService) CursorPosition() (x, y int) {
	if vx, vy, ok := s.ctl.Cursor(); ok {
		return vx, vy
	}

	return s.InputService.CursorPosition()
}

// IsKeyPressed checks the keyboard and the virtual keys.
func (s *GamepadService) IsKeyPressed(key d2enum.Key) bool {
	return s.ctl.KeyDown(key) || s.InputService.IsKeyPressed(key)
}

// IsKeyJustPressed checks the keyboard and the virtual keys.
func (s *GamepadService) IsKeyJustPressed(key d2enum.Key) bool {
	return s.ctl.KeyJustPressed(key) || s.InputService.IsKeyJustPressed(key)
}

// IsKeyJustReleased checks the keyboard and the virtual keys.
func (s *GamepadService) IsKeyJustReleased(key d2enum.Key) bool {
	return s.ctl.KeyJustReleased(key) || s.InputService.IsKeyJustReleased(key)
}

// KeyPressDuration returns the longer of the keyboard and virtual press.
func (s *GamepadService) KeyPressDuration(key d2enum.Key) int {
	if v, k := s.ctl.KeyDuration(key), s.InputService.KeyPressDuration(key); v > k {
		return v
	}

	return s.InputService.KeyPressDuration(key)
}

// IsMouseButtonPressed checks the mouse and the virtual buttons.
func (s *GamepadService) IsMouseButtonPressed(button d2enum.MouseButton) bool {
	return s.ctl.MouseDown(button) || s.InputService.IsMouseButtonPressed(button)
}

// IsMouseButtonJustPressed checks the mouse and the virtual buttons.
func (s *GamepadService) IsMouseButtonJustPressed(button d2enum.MouseButton) bool {
	return s.ctl.MouseJustPressed(button) || s.InputService.IsMouseButtonJustPressed(button)
}

// IsMouseButtonJustReleased checks the mouse and the virtual buttons.
func (s *GamepadService) IsMouseButtonJustReleased(button d2enum.MouseButton) bool {
	return s.ctl.MouseJustReleased(button) || s.InputService.IsMouseButtonJustReleased(button)
}
