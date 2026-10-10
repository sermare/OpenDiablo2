package d2gamepad

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// SyntheticID is the pad id of the virtual controller the autoscript harness
// drives (real ids are small non-negative numbers).
const SyntheticID = 1000

// tapFrames is how long a tapped button stays down: long enough for the game
// to see the press on at least one update.
const tapFrames = 3

// Synthetic is a virtual controller for tests and the autoscript harness
// (pad: steps). It is disconnected until Connect is called.
type Synthetic struct {
	mu        sync.Mutex
	connected bool
	state     State
	taps      [NumButtons]int
}

// Connect plugs the virtual controller in.
func (s *Synthetic) Connect() { s.mu.Lock(); s.connected = true; s.mu.Unlock() }

// Disconnect unplugs it and releases everything.
func (s *Synthetic) Disconnect() {
	s.mu.Lock()
	s.connected = false
	s.state = State{}
	s.taps = [NumButtons]int{}
	s.mu.Unlock()
}

// Hold presses a button until Release.
func (s *Synthetic) Hold(b Button) {
	s.mu.Lock()
	s.connected = true
	s.state.Pressed[b] = true
	s.mu.Unlock()
}

// Release lets go of a button.
func (s *Synthetic) Release(b Button) {
	s.mu.Lock()
	s.state.Pressed[b] = false
	s.taps[b] = 0
	s.mu.Unlock()
}

// Tap presses a button for a few frames.
func (s *Synthetic) Tap(b Button) {
	s.mu.Lock()
	s.connected = true
	s.taps[b] = tapFrames
	s.mu.Unlock()
}

// SetStick sets a stick ("left" or "right") to x,y (-1..1, y down).
func (s *Synthetic) SetStick(right bool, x, y float64) {
	s.mu.Lock()
	s.connected = true

	if right {
		s.state.Axes[AxisRX], s.state.Axes[AxisRY] = x, y
	} else {
		s.state.Axes[AxisLX], s.state.Axes[AxisLY] = x, y
	}

	s.mu.Unlock()
}

// Poll returns the pad for this frame (ok=false when disconnected) and ages the taps.
func (s *Synthetic) Poll() (PadInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return PadInfo{}, false
	}

	st := s.state

	for i := range s.taps {
		if s.taps[i] > 0 {
			st.Pressed[i] = true
			s.taps[i]--
		}
	}

	return PadInfo{ID: SyntheticID, Name: "Synthetic Gamepad", State: st}, true
}

// Do runs one pad: step of the autoscript harness: "connect", "disconnect",
// "press=A" (tap), "hold=A", "release=A", "stick=left,0.8,0" or "stick=right,x,y".
func (s *Synthetic) Do(op string) error {
	name, arg := op, ""
	if i := strings.Index(op, "="); i >= 0 {
		name, arg = op[:i], op[i+1:]
	}

	switch strings.ToLower(strings.TrimSpace(name)) {
	case "connect":
		s.Connect()
	case "disconnect":
		s.Disconnect()
	case "press", "hold", "release":
		b, ok := ParseButton(arg)
		if !ok {
			return fmt.Errorf("unknown gamepad button %q", arg)
		}

		switch strings.ToLower(strings.TrimSpace(name)) {
		case "press":
			s.Tap(b)
		case "hold":
			s.Hold(b)
		default:
			s.Release(b)
		}
	case "stick":
		f := strings.Split(arg, ",")
		if len(f) != 3 {
			return fmt.Errorf("stick needs left|right,x,y")
		}

		x, err1 := strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
		y, err2 := strconv.ParseFloat(strings.TrimSpace(f[2]), 64)

		if err1 != nil || err2 != nil {
			return fmt.Errorf("stick x and y must be numbers")
		}

		switch strings.ToLower(strings.TrimSpace(f[0])) {
		case "left":
			s.SetStick(false, x, y)
		case "right":
			s.SetStick(true, x, y)
		default:
			return fmt.Errorf("stick needs left or right")
		}
	default:
		return fmt.Errorf("unknown pad op %q (want connect|disconnect|press=|hold=|release=|stick=)", name)
	}

	return nil
}

var (
	defaultController = NewController(DefaultConfig())
	defaultSynthetic  = &Synthetic{}
)

// Default is the controller the game's input service uses.
func Default() *Controller { return defaultController }

// DefaultSynthetic is the virtual controller driven by the autoscript harness.
func DefaultSynthetic() *Synthetic { return defaultSynthetic }

var defaultProfiles = NewProfiles()

// DefaultProfiles are the controller profiles the game's input service uses
// (the application may Override them from gamepad-profiles.json).
func DefaultProfiles() *Profiles { return defaultProfiles }
