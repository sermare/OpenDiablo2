package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestEffectiveButton(t *testing.T) {
	const (
		l, r = d2enum.MouseButtonLeft, d2enum.MouseButtonRight
		ctrl = d2enum.KeyModControl
	)

	tests := []struct {
		name string
		goos string
		b    d2enum.MouseButton
		mod  d2enum.KeyMod
		want d2enum.MouseButton
	}{
		{"mac plain left", "darwin", l, 0, l},
		{"mac ctrl+click is right", "darwin", l, ctrl, r},
		{"mac ctrl+shift+click is right", "darwin", l, ctrl | d2enum.KeyModShift, r},
		{"mac two-finger click is right", "darwin", r, 0, r},
		{"mac shift+click stays left", "darwin", l, d2enum.KeyModShift, l},
		{"mac ctrl+right stays right", "darwin", r, ctrl, r},
		{"linux ctrl+click stays left", "linux", l, ctrl, l},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveButton(tt.b, tt.mod, tt.goos); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestResolveWorldClick(t *testing.T) {
	const (
		l, r      = d2enum.MouseButtonLeft, d2enum.MouseButtonRight
		fireBolt  = 36
		shift     = d2enum.KeyModShift
		altOption = d2enum.KeyModAlt
	)

	tests := []struct {
		name string
		in   WorldClickInput
		want WorldAction
	}{
		{"attack skill, ground: walk", WorldClickInput{Button: l, LeftSkillID: skillIDAttack}, WorldMove},
		{"attack skill, monster: attack", WorldClickInput{Button: l, OverMonster: true, LeftSkillID: skillIDAttack}, WorldAttack},
		{"left hand swing walks too", WorldClickInput{Button: l, LeftSkillID: skillIDLeftHandSwing}, WorldMove},
		{"spell, ground: cast it (not walk)", WorldClickInput{Button: l, LeftSkillID: fireBolt}, WorldCastLeft},
		{"spell, monster: cast it", WorldClickInput{Button: l, OverMonster: true, LeftSkillID: fireBolt}, WorldCastLeft},
		{"shift+click, spell: cast standing", WorldClickInput{Button: l, Mod: shift, LeftSkillID: fireBolt}, WorldStandStill},
		{"shift+click, attack on ground: swing in place", WorldClickInput{Button: l, Mod: shift, LeftSkillID: skillIDAttack}, WorldStandStill},
		{"shift+click, attack on monster: swing in place", WorldClickInput{Button: l, Mod: shift, OverMonster: true}, WorldStandStill},
		{"alt+click is a plain click", WorldClickInput{Button: l, Mod: altOption, LeftSkillID: skillIDAttack}, WorldMove},
		{"right click: right skill", WorldClickInput{Button: r, LeftSkillID: skillIDAttack}, WorldCastRight},
		{"right click on monster: right skill", WorldClickInput{Button: r, OverMonster: true}, WorldCastRight},
		{"shift+right: still the right skill", WorldClickInput{Button: r, Mod: shift}, WorldCastRight},
		{"middle button: nothing", WorldClickInput{Button: d2enum.MouseButtonMiddle}, WorldNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveWorldClick(tt.in); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

// the full chain on a Mac: physical click + modifiers -> action
func TestMacClickChain(t *testing.T) {
	tests := []struct {
		name string
		b    d2enum.MouseButton
		mod  d2enum.KeyMod
		want WorldAction
	}{
		{"click", d2enum.MouseButtonLeft, 0, WorldCastLeft},
		{"ctrl+click", d2enum.MouseButtonLeft, d2enum.KeyModControl, WorldCastRight},
		{"two-finger click", d2enum.MouseButtonRight, 0, WorldCastRight},
		{"shift+click", d2enum.MouseButtonLeft, d2enum.KeyModShift, WorldStandStill},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := EffectiveButton(tt.b, tt.mod, "darwin")
			if got := ResolveWorldClick(WorldClickInput{Button: b, Mod: tt.mod, LeftSkillID: 36}); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestParseClickSpec(t *testing.T) {
	tests := []struct {
		spec    string
		button  d2enum.MouseButton
		mod     d2enum.KeyMod
		x, y    int
		wantErr bool
	}{
		{"left", d2enum.MouseButtonLeft, 0, 400, 280, false},
		{"right@10,20", d2enum.MouseButtonRight, 0, 10, 20, false},
		{"Left+Shift@400,300", d2enum.MouseButtonLeft, d2enum.KeyModShift, 400, 300, false},
		{"left+ctrl+alt", d2enum.MouseButtonLeft, d2enum.KeyModControl | d2enum.KeyModAlt, 400, 280, false},
		{"middle", 0, 0, 0, 0, true},
		{"left+meta", 0, 0, 0, 0, true},
		{"left@x", 0, 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			b, m, x, y, err := ParseClickSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}

			if !tt.wantErr && (b != tt.button || m != tt.mod || x != tt.x || y != tt.y) {
				t.Fatalf("got %v %v %d,%d", b, m, x, y)
			}
		})
	}
}

func newTestKeyMap() *KeyMap {
	km := &KeyMap{
		mapping:  make(map[d2enum.Key]d2enum.GameEvent),
		controls: make(map[d2enum.GameEvent]*KeyBinding),
	}
	km.ResetToDefault()

	return km
}

// the keys of the original's default layout that the task names
func TestDefaultKeyLayout(t *testing.T) {
	want := map[d2enum.Key]d2enum.GameEvent{
		d2enum.KeyF1: d2enum.UseSkill1, d2enum.KeyF2: d2enum.UseSkill2, d2enum.KeyF3: d2enum.UseSkill3,
		d2enum.KeyF4: d2enum.UseSkill4, d2enum.KeyF5: d2enum.UseSkill5, d2enum.KeyF6: d2enum.UseSkill6,
		d2enum.KeyF7: d2enum.UseSkill7, d2enum.KeyF8: d2enum.UseSkill8,
		d2enum.KeyTab: d2enum.ToggleAutomap,
		d2enum.KeyI:   d2enum.ToggleInventoryPanel, d2enum.KeyB: d2enum.ToggleInventoryPanel,
		d2enum.KeyC: d2enum.ToggleCharacterPanel, d2enum.KeyA: d2enum.ToggleCharacterPanel,
		d2enum.KeyT: d2enum.ToggleSkillTreePanel, d2enum.KeyS: d2enum.ToggleRightSkillSelector,
		d2enum.KeyQ: d2enum.ToggleQuestLog, d2enum.KeyP: d2enum.TogglePartyPanel, d2enum.KeyH: d2enum.ToggleHelpScreen,
		d2enum.KeyAlt:   d2enum.HoldShowGroundItems,
		d2enum.KeyShift: d2enum.HoldStandStill,
	}

	km := newTestKeyMap()

	for k, ev := range want {
		if got := km.getGameEvent(k); got != ev {
			t.Errorf("key %s: event %d, want %d", KeyName(k), got, ev)
		}
	}
}

func TestKeyNamesRoundTrip(t *testing.T) {
	for k, n := range keyNames {
		if n == "" {
			t.Fatalf("key %d has an empty name", k)
		}

		if got, ok := KeyByName(n); !ok || got != k {
			t.Fatalf("KeyByName(%q) = %d,%v want %d", n, got, ok, k)
		}
	}

	if _, ok := KeyByName("tab"); !ok {
		t.Fatal("names are case-insensitive")
	}

	if _, ok := KeyByName(""); ok {
		t.Fatal("empty name is no key")
	}
}

func TestBindingsPersistRoundTrip(t *testing.T) {
	km := newTestKeyMap()

	// rebind: inventory to X (secondary cleared), automap to M, which the message log had
	km.SetPrimaryBinding(d2enum.ToggleInventoryPanel, d2enum.KeyX)
	km.SetSecondaryBinding(d2enum.ToggleInventoryPanel, -1)
	km.SetPrimaryBinding(d2enum.ToggleAutomap, d2enum.KeyM)

	saved := km.ExportBindings()
	if got := saved["ToggleInventoryPanel"]; got[0] != "X" || got[1] != "" {
		t.Fatalf("exported inventory = %q", got)
	}

	fresh := newTestKeyMap()
	if n := fresh.ApplyBindings(saved); n != len(saved) {
		t.Fatalf("applied %d of %d", n, len(saved))
	}

	for ev, name := range gameEventNames {
		a, b := km.GetKeysForGameEvent(ev), fresh.GetKeysForGameEvent(ev)
		if a == nil || b == nil {
			if a != b {
				t.Fatalf("%s: %v vs %v", name, a, b)
			}

			continue
		}

		if *a != *b {
			t.Fatalf("%s: saved %+v, reloaded %+v", name, *a, *b)
		}
	}

	if fresh.getGameEvent(d2enum.KeyX) != d2enum.ToggleInventoryPanel || fresh.getGameEvent(d2enum.KeyM) != d2enum.ToggleAutomap {
		t.Fatal("reverse lookup not rebuilt")
	}
}

func TestApplyBindingsSkipsJunk(t *testing.T) {
	km := newTestKeyMap()
	before := km.ExportBindings()

	n := km.ApplyBindings(map[string][]string{
		"NoSuchEvent":          {"A", ""},
		"ToggleInventoryPanel": {"NoSuchKey", ""},
		"ToggleQuestLog":       {"Q"}, // wrong shape
	})
	if n != 0 {
		t.Fatalf("applied %d junk entries", n)
	}

	after := km.ExportBindings()
	for k, v := range before {
		if after[k][0] != v[0] || after[k][1] != v[1] {
			t.Fatalf("%s changed: %v -> %v", k, v, after[k])
		}
	}
}

func TestEveryEventHasAName(t *testing.T) {
	for ev := d2enum.ToggleGameMenu; ev <= d2enum.ClearMessages; ev++ {
		if gameEventNames[ev] == "" {
			t.Errorf("event %d has no config name", ev)
		}
	}
}
