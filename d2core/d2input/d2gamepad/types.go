// Package d2gamepad turns gamepad state into the keyboard and mouse input the
// game already understands: the left stick moves or aims, the right stick
// drives a cursor for menus and the inventory, buttons trigger skills, potion
// belt slots, panels, the automap and skill cycling. The package is pure (no
// ebiten): a Source supplies raw button and axis values, a Profile translates
// the layout of a controller family, a Mapping assigns actions to buttons and a
// Controller produces the virtual input of one frame. It is unit tested without
// a controller; the ebiten glue lives in d2core/d2input/ebiten.
package d2gamepad

import "strings"

// Button is a logical button of a standard controller (Xbox naming; PlayStation
// Cross/Circle/Square/Triangle and Switch B/A/Y/X sit at the same positions).
type Button int

// Logical buttons.
const (
	ButtonA Button = iota // south
	ButtonB               // east
	ButtonX               // west
	ButtonY               // north
	ButtonLB
	ButtonRB
	ButtonLT
	ButtonRT
	ButtonBack
	ButtonStart
	ButtonL3
	ButtonR3
	ButtonDUp
	ButtonDDown
	ButtonDLeft
	ButtonDRight
	buttonCount
)

// NumButtons is the number of logical buttons.
const NumButtons = int(buttonCount)

var buttonNames = [...]string{"A", "B", "X", "Y", "LB", "RB", "LT", "RT", "BACK", "START", "L3", "R3", "DUP", "DDOWN", "DLEFT", "DRIGHT"}

// String returns the short name of the button ("A", "LB", "DUP").
func (b Button) String() string {
	if b < 0 || int(b) >= len(buttonNames) {
		return "?"
	}

	return buttonNames[b]
}

// ParseButton finds a button by name (case-insensitive).
func ParseButton(s string) (Button, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	for i, n := range buttonNames {
		if n == s {
			return Button(i), true
		}
	}

	return 0, false
}

// Buttons lists every logical button in order.
func Buttons() []Button {
	r := make([]Button, NumButtons)
	for i := range r {
		r[i] = Button(i)
	}

	return r
}

// Axis is a logical stick axis.
type Axis int

// Logical axes. Positive Y points down the screen.
const (
	AxisLX Axis = iota
	AxisLY
	AxisRX
	AxisRY
	axisCount
)

// State is the logical state of one controller (or of all merged).
type State struct {
	Pressed [NumButtons]bool
	Axes    [axisCount]float64
}

// Merge combines two states: buttons are ORed, stick values with the larger
// magnitude win.
func (s State) Merge(o State) State {
	for i := range s.Pressed {
		s.Pressed[i] = s.Pressed[i] || o.Pressed[i]
	}

	for i := range s.Axes {
		if abs(o.Axes[i]) > abs(s.Axes[i]) {
			s.Axes[i] = o.Axes[i]
		}
	}

	return s
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}

	return v
}
