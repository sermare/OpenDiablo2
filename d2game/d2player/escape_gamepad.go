package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
)

// optPadBase is the first optionID of the gamepad remap rows (one per button).
const optPadBase optionID = 100

func init() {
	optionKeys[optUIScale] = d2config.OptUIScale
	optionKeys[optColorblind] = d2config.OptColorblind
	optionKeys[optSubtitleLog] = d2config.OptSubtitleLog

	for _, b := range d2gamepad.Buttons() {
		optionKeys[optPadBase+optionID(b)] = d2gamepad.OptionKey(b)
	}
}

func (m *EscapeMenu) newAccessibilityLayout() *layout {
	return m.wrapLayout(func(l *layout) {
		m.addTitle(l, "ACCESSIBILITY")
		m.addEnumLabel(l, optUIScale, "WINDOW SCALE")
		m.addEnumLabel(l, optColorblind, "COLOR BLIND ITEMS")
		m.addEnumLabel(l, optSubtitleLog, "SPEECH SUBTITLE LOG")
		m.addPreviousMenuLabel(l)
	})
}

var padButtonTitles = map[d2gamepad.Button]string{
	d2gamepad.ButtonA: "A BUTTON", d2gamepad.ButtonB: "B BUTTON", d2gamepad.ButtonX: "X BUTTON", d2gamepad.ButtonY: "Y BUTTON",
	d2gamepad.ButtonLB: "LEFT BUMPER", d2gamepad.ButtonRB: "RIGHT BUMPER",
	d2gamepad.ButtonLT: "LEFT TRIGGER", d2gamepad.ButtonRT: "RIGHT TRIGGER",
	d2gamepad.ButtonBack: "BACK", d2gamepad.ButtonStart: "START",
	d2gamepad.ButtonL3: "LEFT STICK CLICK", d2gamepad.ButtonR3: "RIGHT STICK CLICK",
	d2gamepad.ButtonDUp: "D-PAD UP", d2gamepad.ButtonDDown: "D-PAD DOWN",
	d2gamepad.ButtonDLeft: "D-PAD LEFT", d2gamepad.ButtonDRight: "D-PAD RIGHT",
}

// newGamepadLayout is one page of gamepad remap rows (buttons [from, to)):
// clicking a row steps through the actions.
func (m *EscapeMenu) newGamepadLayout(title string, from, to int, otherPage layoutID) *layout {
	return m.wrapLayout(func(l *layout) {
		m.addTitle(l, title)

		for i := from; i < to; i++ {
			b := d2gamepad.Button(i)
			m.addEnumLabel(l, optPadBase+optionID(b), padButtonTitles[b])
		}

		m.addSmallLinkLabel(l, "OTHER BUTTONS", otherPage)
		m.addPreviousMenuLabel(l)
	})
}
