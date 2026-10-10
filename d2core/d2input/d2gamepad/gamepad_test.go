package d2gamepad

import (
	"encoding/json"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func pad(buttons ...Button) PadInfo {
	p := PadInfo{ID: 1, Name: "test pad"}
	for _, b := range buttons {
		p.State.Pressed[b] = true
	}

	return p
}

func padStick(lx, ly, rx, ry float64, buttons ...Button) PadInfo {
	p := pad(buttons...)
	p.State.Axes = [4]float64{lx, ly, rx, ry}

	return p
}

func TestDefaultMappingActions(t *testing.T) {
	m := DefaultMapping()
	cases := []struct {
		b Button
		a Action
	}{
		{ButtonRT, ActionLeftSkill}, {ButtonLT, ActionRightSkill}, {ButtonA, ActionClick},
		{ButtonDUp, ActionPotion1}, {ButtonDRight, ActionPotion2}, {ButtonDDown, ActionPotion3}, {ButtonDLeft, ActionPotion4},
		{ButtonX, ActionInventory}, {ButtonY, ActionCharacter}, {ButtonBack, ActionAutomap},
		{ButtonLB, ActionCycleLeftSkill}, {ButtonRB, ActionCycleRightSkill}, {ButtonStart, ActionGameMenu},
	}

	for _, c := range cases {
		if m[c.b] != c.a {
			t.Errorf("%s: got %s, want %s", c.b, m[c.b], c.a)
		}
	}
}

func TestKeyActionsPressKeys(t *testing.T) {
	cases := []struct {
		name string
		b    Button
		key  d2enum.Key
	}{
		{"potion 1", ButtonDUp, d2enum.Key1},
		{"potion 4", ButtonDLeft, d2enum.Key4},
		{"inventory", ButtonX, d2enum.KeyI},
		{"character", ButtonY, d2enum.KeyC},
		{"skill tree", ButtonL3, d2enum.KeyT},
		{"automap", ButtonBack, d2enum.KeyTab},
		{"menu", ButtonStart, d2enum.KeyEscape},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctl := NewController(DefaultConfig())
			ctl.Update(0.016, []PadInfo{pad()}, 0, 0)

			if ctl.KeyDown(c.key) {
				t.Fatal("key down before the press")
			}

			ctl.Update(0.016, []PadInfo{pad(c.b)}, 0, 0)

			if !ctl.KeyDown(c.key) || !ctl.KeyJustPressed(c.key) {
				t.Fatalf("key %v not pressed on the first frame", c.key)
			}

			ctl.Update(0.016, []PadInfo{pad(c.b)}, 0, 0)

			if ctl.KeyJustPressed(c.key) || ctl.KeyDuration(c.key) != 2 {
				t.Fatalf("held: justPressed=%v duration=%d", ctl.KeyJustPressed(c.key), ctl.KeyDuration(c.key))
			}

			ctl.Update(0.016, []PadInfo{pad()}, 0, 0)

			if ctl.KeyDown(c.key) || !ctl.KeyJustReleased(c.key) {
				t.Fatal("key not released")
			}
		})
	}
}

func TestResolverUsesPlayerBinding(t *testing.T) {
	ctl := NewController(DefaultConfig())
	ctl.SetKeyResolver(func(e d2enum.GameEvent) (d2enum.Key, bool) {
		if e == d2enum.ToggleInventoryPanel {
			return d2enum.KeyB, true
		}

		return 0, false
	})
	ctl.Update(0.016, []PadInfo{pad(ButtonX)}, 0, 0)

	if !ctl.KeyDown(d2enum.KeyB) || ctl.KeyDown(d2enum.KeyI) {
		t.Fatal("inventory did not use the bound key")
	}

	ctl.Update(0.016, []PadInfo{pad(ButtonY)}, 0, 0)

	if !ctl.KeyDown(d2enum.KeyC) {
		t.Fatal("unresolved events must fall back to the default key")
	}
}

func TestSkillButtonsAndClicks(t *testing.T) {
	ctl := NewController(DefaultConfig())
	ctl.Update(0.016, []PadInfo{pad(ButtonRT)}, 0, 0)

	if !ctl.MouseJustPressed(d2enum.MouseButtonLeft) || ctl.MouseDown(d2enum.MouseButtonRight) {
		t.Fatal("RT must press the left mouse button only")
	}

	ctl.Update(0.016, []PadInfo{pad(ButtonLT)}, 0, 0)

	if !ctl.MouseDown(d2enum.MouseButtonRight) || !ctl.MouseJustReleased(d2enum.MouseButtonLeft) {
		t.Fatal("LT must press the right mouse button and release the left one")
	}
}

func TestCycleSkillsCallsHandlerOncePerPress(t *testing.T) {
	ctl := NewController(DefaultConfig())

	var got []Action

	ctl.SetActionHandler(func(a Action) { got = append(got, a) })

	for _, held := range []bool{true, true, true, false, true} {
		var p PadInfo
		if held {
			p = pad(ButtonRB)
		} else {
			p = pad()
		}

		ctl.Update(0.016, []PadInfo{p}, 0, 0)
	}

	if len(got) != 2 || got[0] != ActionCycleRightSkill {
		t.Fatalf("handler calls = %v, want two cycle-right presses", got)
	}
}

func TestLeftStickWalksAndStops(t *testing.T) {
	cfg := DefaultConfig()
	ctl := NewController(cfg)
	ctl.SetUIMode(func() bool { return false })
	ctl.Update(0.016, []PadInfo{padStick(1, 0, 0, 0)}, 0, 0)

	x, y, active := ctl.Cursor()
	if !active || !ctl.MouseDown(d2enum.MouseButtonLeft) {
		t.Fatal("the left stick must hold the left mouse button")
	}

	if x <= cfg.HeroX || y < cfg.HeroY-1 {
		t.Fatalf("target (%d,%d) is not right of the hero (%d,%d)", x, y, cfg.HeroX, cfg.HeroY)
	}

	ctl.Update(0.016, []PadInfo{padStick(0.1, 0.1, 0, 0)}, 0, 0) // inside the dead zone

	if ctl.MouseDown(d2enum.MouseButtonLeft) || !ctl.MouseJustReleased(d2enum.MouseButtonLeft) {
		t.Fatal("releasing the stick must release the mouse button")
	}
}

func TestLeftStickMovesCursorInMenus(t *testing.T) {
	ctl := NewController(DefaultConfig())
	ctl.SetUIMode(func() bool { return true })
	x0, y0, _ := ctl.Cursor()
	ctl.Update(0.1, []PadInfo{padStick(0, 1, 0, 0)}, 0, 0)

	x, y, _ := ctl.Cursor()
	if ctl.MouseDown(d2enum.MouseButtonLeft) {
		t.Fatal("moving the cursor in a menu must not click")
	}

	if x != x0 || y <= y0 {
		t.Fatalf("cursor (%d,%d) did not move down from (%d,%d)", x, y, x0, y0)
	}
}

func TestRightStickCursorClampedAndDeadzone(t *testing.T) {
	cfg := DefaultConfig()
	ctl := NewController(cfg)

	ctl.Update(0.1, []PadInfo{padStick(0, 0, 0.2, 0.2)}, 0, 0)

	if x, y, _ := ctl.Cursor(); x != cfg.ScreenW/2 || y != cfg.ScreenH/2 {
		t.Fatalf("drift inside the dead zone: (%d,%d)", x, y)
	}

	for i := 0; i < 200; i++ {
		ctl.Update(0.05, []PadInfo{padStick(0, 0, 1, 1)}, 0, 0)
	}

	if x, y, _ := ctl.Cursor(); x != cfg.ScreenW-1 || y != cfg.ScreenH-1 {
		t.Fatalf("cursor not clamped to the screen: (%d,%d)", x, y)
	}
}

func TestRealMouseTakesCursorBack(t *testing.T) {
	ctl := NewController(DefaultConfig())
	ctl.Update(0.016, []PadInfo{padStick(0, 0, 1, 0)}, 10, 10)

	if _, _, active := ctl.Cursor(); !active {
		t.Fatal("pad should be in charge after stick use")
	}

	ctl.Update(0.016, []PadInfo{pad()}, 50, 60) // mouse moved

	if x, y, active := ctl.Cursor(); active || x != 50 || y != 60 {
		t.Fatalf("mouse did not take over: (%d,%d) active=%v", x, y, active)
	}
}

func TestHotPlug(t *testing.T) {
	ctl := NewController(DefaultConfig())
	ctl.Update(0.016, nil, 0, 0)

	if ctl.Connected() != 0 {
		t.Fatal("no pad expected")
	}

	ctl.Update(0.016, []PadInfo{pad(ButtonX)}, 0, 0)

	if ctl.Connected() != 1 || !ctl.KeyDown(d2enum.KeyI) {
		t.Fatal("pad plugged in while the game runs must work")
	}

	// unplugged with the button still held: everything is released
	ctl.Update(0.016, nil, 0, 0)

	if ctl.Connected() != 0 || ctl.KeyDown(d2enum.KeyI) || !ctl.KeyJustReleased(d2enum.KeyI) {
		t.Fatal("unplugging must release held keys")
	}

	if _, _, active := ctl.Cursor(); active {
		t.Fatal("without a pad the mouse is in charge")
	}

	// plugged back in: the held button is a fresh press, not a stuck state
	ctl.Update(0.016, []PadInfo{pad(ButtonX)}, 0, 0)

	if !ctl.KeyJustPressed(d2enum.KeyI) {
		t.Fatal("re-plugged pad press not seen")
	}

	log := ctl.DrainLog()
	connected, disconnected := 0, 0

	for _, l := range log {
		switch {
		case len(l) > 18 && l[:18] == "GAMEPAD connected ":
			connected++
		case len(l) > 21 && l[:21] == "GAMEPAD disconnected ":
			disconnected++
		}
	}

	if connected != 2 || disconnected != 1 {
		t.Fatalf("log: connected=%d disconnected=%d in %v", connected, disconnected, log)
	}
}

func TestTwoPadsMerge(t *testing.T) {
	ctl := NewController(DefaultConfig())
	a, b := pad(ButtonX), pad(ButtonY)
	b.ID = 2
	ctl.Update(0.016, []PadInfo{a, b}, 0, 0)

	if !ctl.KeyDown(d2enum.KeyI) || !ctl.KeyDown(d2enum.KeyC) || ctl.Connected() != 2 {
		t.Fatal("both pads must work")
	}
}

func TestRemapAndJSON(t *testing.T) {
	m := DefaultMapping()
	m.Set(ButtonA, ActionPotion3)

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	back := DefaultMapping()
	if err := json.Unmarshal(data, &back); err != nil || back != m {
		t.Fatalf("round trip: %v %v", err, back == m)
	}

	partial := DefaultMapping()
	if err := json.Unmarshal([]byte(`{"x":"potion 4","B":"bogus","nope":"CLICK"}`), &partial); err != nil {
		t.Fatal(err)
	}

	if partial[ButtonX] != ActionPotion4 || partial[ButtonB] != DefaultMapping()[ButtonB] {
		t.Fatalf("partial override wrong: X=%s B=%s", partial[ButtonX], partial[ButtonB])
	}

	ctl := NewController(DefaultConfig())
	ctl.SetMapping(m)
	ctl.Update(0.016, []PadInfo{pad(ButtonA)}, 0, 0)

	if !ctl.KeyDown(d2enum.Key3) || ctl.MouseDown(d2enum.MouseButtonLeft) {
		t.Fatal("remapped A must press potion 3, not click")
	}
}

func TestMappingFromChoices(t *testing.T) {
	m := MappingFromChoices(func(k string) string {
		if k == OptionKey(ButtonB) {
			return "AUTOMAP"
		}

		return ""
	})

	if m[ButtonB] != ActionAutomap || m[ButtonA] != ActionClick {
		t.Fatalf("B=%s A=%s", m[ButtonB], m[ButtonA])
	}

	if OptionKey(ButtonDUp) != "pad.dup" {
		t.Fatal(OptionKey(ButtonDUp))
	}
}

func TestActionNamesRoundTrip(t *testing.T) {
	for i, n := range ActionNames() {
		a, ok := ParseAction(n)
		if !ok || int(a) != i {
			t.Errorf("action %d %q does not parse back", i, n)
		}
	}

	if a, ok := ParseAction("next_left_skill"); !ok || a != ActionCycleLeftSkill {
		t.Error("underscore spelling not accepted")
	}
}

func TestProfilesTranslate(t *testing.T) {
	ps := NewProfiles()

	cases := []struct {
		name, want string
	}{
		{"Xbox Wireless Controller", "xbox"},
		{"Wireless Controller", "playstation"},
		{"DualSense Wireless Controller", "playstation"},
		{"Pro Controller", "switch"},
		{"Some Generic Pad", "xbox"},
	}

	for _, c := range cases {
		if got := ps.For(c.name, "").Name; got != c.want {
			t.Errorf("%q -> %s, want %s", c.name, got, c.want)
		}
	}

	xbox := ps.For("Xbox", "")
	buttons := make([]bool, 19)
	buttons[0], buttons[len(buttons)-4] = true, true // A and the hat's "up"
	axes := []float64{0.5, -0.5, 0, 0, 1, -1}        // RT fully pressed (axis 4), LT at rest (axis 5 unused)

	st := xbox.Translate(buttons, axes)
	if !st.Pressed[ButtonA] || !st.Pressed[ButtonDUp] || st.Pressed[ButtonDDown] {
		t.Fatalf("buttons: %+v", st.Pressed)
	}

	if !st.Pressed[ButtonRT] || st.Pressed[ButtonLT] {
		t.Fatalf("triggers: RT=%v LT=%v", st.Pressed[ButtonRT], st.Pressed[ButtonLT])
	}

	if st.Axes[AxisLX] != 0.5 || st.Axes[AxisLY] != -0.5 {
		t.Fatalf("axes: %v", st.Axes)
	}

	// a Switch pad has its A and B swapped against the Xbox layout
	sw := ps.For("Pro Controller", "")
	raw := make([]bool, 20)
	raw[1] = true

	if !sw.Translate(raw, nil).Pressed[ButtonA] {
		t.Fatal("switch: raw button 1 should be the south (A) button")
	}
}

func TestProfileOverride(t *testing.T) {
	ps := NewProfiles()
	if err := ps.Override([]byte(`[{"name":"mine","match":["mypad"],"buttons":{"A":{"button":7}},"axes":[0,1,2,3]}]`)); err != nil {
		t.Fatal(err)
	}

	p := ps.For("MyPad 3000", "")
	raw := make([]bool, 10)
	raw[7] = true

	if p.Name != "mine" || !p.Translate(raw, nil).Pressed[ButtonA] {
		t.Fatalf("override not used: %s", p.Name)
	}

	if ps.Override([]byte(`not json`)) == nil {
		t.Fatal("bad JSON must fail")
	}
}

func TestSyntheticSteps(t *testing.T) {
	s := &Synthetic{}

	if _, ok := s.Poll(); ok {
		t.Fatal("must start disconnected")
	}

	for _, op := range []string{"connect", "press=A", "hold=DUP", "stick=left,1,0", "release=DUP"} {
		if err := s.Do(op); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
	}

	p, ok := s.Poll()
	if !ok || !p.State.Pressed[ButtonA] || p.State.Pressed[ButtonDUp] || p.State.Axes[AxisLX] != 1 {
		t.Fatalf("poll: %+v ok=%v", p.State, ok)
	}

	for i := 0; i < tapFrames; i++ {
		p, _ = s.Poll()
	}

	if p.State.Pressed[ButtonA] {
		t.Fatal("a tap must end by itself")
	}

	for _, bad := range []string{"press=Q", "stick=up,1,0", "stick=left,x,0", "wiggle"} {
		if s.Do(bad) == nil {
			t.Errorf("%q should fail", bad)
		}
	}

	_ = s.Do("disconnect")

	if _, ok := s.Poll(); ok {
		t.Fatal("disconnect")
	}
}
