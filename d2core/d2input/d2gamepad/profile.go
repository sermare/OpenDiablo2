package d2gamepad

import (
	"encoding/json"
	"strings"
)

// Input says where a logical button comes from in the raw data ebiten 2.0.2
// reports (it has no standard-layout API; the indexes are those of GLFW's
// joystick API, in which a D-pad hat is appended to the buttons as four buttons
// up, right, down, left).
type Input struct {
	// Button is a raw button index; a negative index counts from the end (-1 is
	// the last raw button), which is how the appended D-pad hat is addressed.
	Button *int `json:"button,omitempty"`
	// Axis is a raw axis index for analog triggers; Rest is the value at rest
	// (-1 for triggers that run from -1 to 1, 0 for 0 to 1).
	Axis *int    `json:"axis,omitempty"`
	Rest float64 `json:"rest,omitempty"`
}

func btn(i int) Input { return Input{Button: &i} }
func trig(i int, rest float64) Input {
	return Input{Axis: &i, Rest: rest}
}

// Profile describes the raw layout of one controller family.
type Profile struct {
	Name string `json:"name"`
	// Match lists lower-case fragments of the controller name or GUID that
	// select this profile.
	Match   []string         `json:"match,omitempty"`
	Buttons map[string]Input `json:"buttons"` // keyed by logical button name
	Axes    [4]int           `json:"axes"`    // raw axes of LX, LY, RX, RY
}

// TriggerPressPoint is the normalised trigger value that counts as pressed.
const TriggerPressPoint = 0.5

// Translate converts raw values into the logical state.
func (p Profile) Translate(buttons []bool, axes []float64) State {
	var s State

	for name, in := range p.Buttons {
		b, ok := ParseButton(name)
		if !ok {
			continue
		}

		switch {
		case in.Button != nil:
			i := *in.Button
			if i < 0 {
				i += len(buttons)
			}

			s.Pressed[b] = i >= 0 && i < len(buttons) && buttons[i]
		case in.Axis != nil:
			i := *in.Axis
			if i >= 0 && i < len(axes) {
				s.Pressed[b] = normaliseTrigger(axes[i], in.Rest) >= TriggerPressPoint
			}
		}
	}

	for a := range p.Axes {
		if i := p.Axes[a]; i >= 0 && i < len(axes) {
			s.Axes[a] = axes[i]
		}
	}

	return s
}

func normaliseTrigger(v, rest float64) float64 {
	if rest >= 0 {
		return v
	}

	return (v - rest) / (1 - rest)
}

// Built-in profiles. UNVERIFIED: the indexes are recalled from the SDL
// controller database entries for macOS (IOKit) and have not been checked
// against hardware by this project; docs/gamepad.md explains how to override
// a profile with gamepad-profiles.json and the game logs the raw button and
// axis counts of every controller it sees. The ebiten 2.0.2 glfw driver reports
// no hats separately, so D-pads are addressed from the end of the button list.
func builtinProfiles() []Profile {
	dpadEnd := map[string]Input{"DUP": btn(-4), "DRIGHT": btn(-3), "DDOWN": btn(-2), "DLEFT": btn(-1)}

	merge := func(m map[string]Input) map[string]Input {
		for k, v := range dpadEnd {
			if _, set := m[k]; !set {
				m[k] = v
			}
		}

		return m
	}

	return []Profile{
		{
			Name:  "xbox",
			Match: []string{"xbox", "x-box", "xinput", "045e"},
			Buttons: merge(map[string]Input{
				"A": btn(0), "B": btn(1), "X": btn(3), "Y": btn(4),
				"LB": btn(6), "RB": btn(7), "BACK": btn(10), "START": btn(11),
				"L3": btn(13), "R3": btn(14),
				"LT": trig(5, -1), "RT": trig(4, -1),
			}),
			Axes: [4]int{0, 1, 2, 3},
		},
		{
			Name:  "playstation",
			Match: []string{"dualshock", "dualsense", "playstation", "ps4", "ps5", "wireless controller", "054c"},
			Buttons: map[string]Input{
				"A": btn(1), "B": btn(2), "X": btn(0), "Y": btn(3),
				"LB": btn(4), "RB": btn(5), "BACK": btn(8), "START": btn(9),
				"L3": btn(10), "R3": btn(11),
				"DUP": btn(14), "DDOWN": btn(15), "DLEFT": btn(16), "DRIGHT": btn(17),
				"LT": trig(3, -1), "RT": trig(4, -1),
			},
			Axes: [4]int{0, 1, 2, 5},
		},
		{
			Name:  "switch",
			Match: []string{"pro controller", "switch", "nintendo", "057e"},
			Buttons: map[string]Input{
				// Nintendo's A/B and X/Y are swapped against Xbox: map by position.
				"A": btn(1), "B": btn(0), "X": btn(3), "Y": btn(2),
				"LB": btn(4), "RB": btn(5), "LT": btn(6), "RT": btn(7),
				"BACK": btn(9), "START": btn(10), "L3": btn(11), "R3": btn(12),
				"DUP": btn(14), "DDOWN": btn(15), "DLEFT": btn(16), "DRIGHT": btn(17),
			},
			Axes: [4]int{0, 1, 2, 3},
		},
	}
}

// Profiles selects the profile of a controller.
type Profiles struct {
	list []Profile
}

// NewProfiles returns the built-in profiles.
func NewProfiles() *Profiles { return &Profiles{list: builtinProfiles()} }

// Override adds the profiles of a gamepad-profiles.json ([]Profile); they take
// precedence over the built-in ones with the same name or matching fragments.
func (ps *Profiles) Override(data []byte) error {
	var extra []Profile
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}

	ps.list = append(extra, ps.list...)

	return nil
}

// For returns the profile for a controller name / GUID; unknown controllers get
// the Xbox layout (the de-facto standard).
func (ps *Profiles) For(name, guid string) Profile {
	hay := strings.ToLower(name + " " + guid)

	for _, p := range ps.list {
		for _, m := range p.Match {
			if m != "" && strings.Contains(hay, strings.ToLower(m)) {
				return p
			}
		}
	}

	for _, p := range ps.list {
		if p.Name == "xbox" {
			return p
		}
	}

	return ps.list[0]
}
