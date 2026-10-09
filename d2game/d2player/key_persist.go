package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// Key bindings are saved in config.json (Configuration.KeyBindings) as
// {"ToggleInventoryPanel": ["B", "I"], ...}: the event's Go name, then the
// primary and secondary key names ("" = unbound). Names are stable, unlike the
// numeric enum values, so reordering the enums never breaks a saved config.

var gameEventNames = map[d2enum.GameEvent]string{
	d2enum.ToggleGameMenu: "ToggleGameMenu", d2enum.ToggleCharacterPanel: "ToggleCharacterPanel",
	d2enum.ToggleInventoryPanel: "ToggleInventoryPanel", d2enum.TogglePartyPanel: "TogglePartyPanel",
	d2enum.ToggleSkillTreePanel: "ToggleSkillTreePanel", d2enum.ToggleHirelingPanel: "ToggleHirelingPanel",
	d2enum.ToggleQuestLog: "ToggleQuestLog", d2enum.ToggleHelpScreen: "ToggleHelpScreen",
	d2enum.ToggleChatOverlay: "ToggleChatOverlay", d2enum.ToggleMessageLog: "ToggleMessageLog",
	d2enum.ToggleRightSkillSelector: "ToggleRightSkillSelector", d2enum.ToggleLeftSkillSelector: "ToggleLeftSkillSelector",
	d2enum.ToggleAutomap: "ToggleAutomap", d2enum.CenterAutomap: "CenterAutomap", d2enum.FadeAutomap: "FadeAutomap",
	d2enum.TogglePartyOnAutomap: "TogglePartyOnAutomap", d2enum.ToggleNamesOnAutomap: "ToggleNamesOnAutomap",
	d2enum.ToggleMiniMap: "ToggleMiniMap",
	d2enum.UseSkill1:     "UseSkill1", d2enum.UseSkill2: "UseSkill2", d2enum.UseSkill3: "UseSkill3",
	d2enum.UseSkill4: "UseSkill4", d2enum.UseSkill5: "UseSkill5", d2enum.UseSkill6: "UseSkill6",
	d2enum.UseSkill7: "UseSkill7", d2enum.UseSkill8: "UseSkill8", d2enum.UseSkill9: "UseSkill9",
	d2enum.UseSkill10: "UseSkill10", d2enum.UseSkill11: "UseSkill11", d2enum.UseSkill12: "UseSkill12",
	d2enum.UseSkill13: "UseSkill13", d2enum.UseSkill14: "UseSkill14", d2enum.UseSkill15: "UseSkill15",
	d2enum.UseSkill16:          "UseSkill16",
	d2enum.SelectPreviousSkill: "SelectPreviousSkill", d2enum.SelectNextSkill: "SelectNextSkill",
	d2enum.ToggleBelts: "ToggleBelts", d2enum.UseBeltSlot1: "UseBeltSlot1", d2enum.UseBeltSlot2: "UseBeltSlot2",
	d2enum.UseBeltSlot3: "UseBeltSlot3", d2enum.UseBeltSlot4: "UseBeltSlot4",
	d2enum.SwapWeapons: "SwapWeapons", d2enum.ToggleChatBox: "ToggleChatBox", d2enum.ToggleRunWalk: "ToggleRunWalk",
	d2enum.SayHelp: "SayHelp", d2enum.SayFollowMe: "SayFollowMe", d2enum.SayThisIsForYou: "SayThisIsForYou",
	d2enum.SayThanks: "SayThanks", d2enum.SaySorry: "SaySorry", d2enum.SayBye: "SayBye",
	d2enum.SayNowYouDie: "SayNowYouDie", d2enum.SayRetreat: "SayRetreat",
	d2enum.HoldRun: "HoldRun", d2enum.HoldStandStill: "HoldStandStill",
	d2enum.HoldShowGroundItems: "HoldShowGroundItems", d2enum.HoldShowPortraits: "HoldShowPortraits",
	d2enum.TakeScreenShot: "TakeScreenShot", d2enum.ClearScreen: "ClearScreen", d2enum.ClearMessages: "ClearMessages",
}

var keyNames = buildKeyNames()

func buildKeyNames() map[d2enum.Key]string {
	m := map[d2enum.Key]string{}

	for i := 0; i < 10; i++ {
		m[d2enum.Key0+d2enum.Key(i)] = fmt.Sprint(i)
		m[d2enum.KeyKP0+d2enum.Key(i)] = fmt.Sprintf("KP%d", i)
	}

	for i := 0; i < 26; i++ {
		m[d2enum.KeyA+d2enum.Key(i)] = string(rune('A' + i))
	}

	for i := 0; i < 12; i++ {
		m[d2enum.KeyF1+d2enum.Key(i)] = fmt.Sprintf("F%d", i+1)
	}

	for k, v := range map[d2enum.Key]string{
		d2enum.KeyApostrophe: "Apostrophe", d2enum.KeyBackslash: "Backslash", d2enum.KeyBackspace: "Backspace",
		d2enum.KeyCapsLock: "CapsLock", d2enum.KeyComma: "Comma", d2enum.KeyDelete: "Delete", d2enum.KeyDown: "Down",
		d2enum.KeyEnd: "End", d2enum.KeyEnter: "Enter", d2enum.KeyEqual: "Equal", d2enum.KeyEscape: "Escape",
		d2enum.KeyGraveAccent: "GraveAccent", d2enum.KeyHome: "Home", d2enum.KeyInsert: "Insert",
		d2enum.KeyKPAdd: "KPAdd", d2enum.KeyKPDecimal: "KPDecimal", d2enum.KeyKPDivide: "KPDivide",
		d2enum.KeyKPEnter: "KPEnter", d2enum.KeyKPEqual: "KPEqual", d2enum.KeyKPMultiply: "KPMultiply",
		d2enum.KeyKPSubtract: "KPSubtract", d2enum.KeyLeft: "Left", d2enum.KeyLeftBracket: "LeftBracket",
		d2enum.KeyMenu: "Menu", d2enum.KeyMinus: "Minus", d2enum.KeyNumLock: "NumLock", d2enum.KeyPageDown: "PageDown",
		d2enum.KeyPageUp: "PageUp", d2enum.KeyPause: "Pause", d2enum.KeyPeriod: "Period",
		d2enum.KeyPrintScreen: "PrintScreen", d2enum.KeyRight: "Right", d2enum.KeyRightBracket: "RightBracket",
		d2enum.KeyScrollLock: "ScrollLock", d2enum.KeySemicolon: "Semicolon", d2enum.KeySlash: "Slash",
		d2enum.KeySpace: "Space", d2enum.KeyTab: "Tab", d2enum.KeyUp: "Up", d2enum.KeyAlt: "Alt",
		d2enum.KeyControl: "Control", d2enum.KeyShift: "Shift", d2enum.KeyTilde: "Tilde",
		d2enum.KeyMouse3: "Mouse3", d2enum.KeyMouse4: "Mouse4", d2enum.KeyMouse5: "Mouse5",
		d2enum.KeyMouseWheelUp: "WheelUp", d2enum.KeyMouseWheelDown: "WheelDown",
	} {
		m[k] = v
	}

	return m
}

// KeyName returns the config name of a key ("" for unbound or unknown).
func KeyName(k d2enum.Key) string { return keyNames[k] }

// KeyByName is the inverse of KeyName (case-insensitive); ok is false for "" and unknown names.
func KeyByName(name string) (d2enum.Key, bool) {
	for k, n := range keyNames {
		if strings.EqualFold(n, name) {
			return k, true
		}
	}

	return -1, false
}

// ExportBindings returns the key map in its config form.
func (km *KeyMap) ExportBindings() map[string][]string {
	km.mutex.RLock()
	defer km.mutex.RUnlock()

	out := make(map[string][]string, len(km.controls))

	for ev, b := range km.controls {
		if name, ok := gameEventNames[ev]; ok && b != nil {
			out[name] = []string{KeyName(b.Primary), KeyName(b.Secondary)}
		}
	}

	return out
}

// ApplyBindings loads saved bindings over the current map. Unknown events and
// key names are skipped (a config from another version); a key already taken
// by another event moves to the saved one, as in the options page. It returns
// how many events were applied.
func (km *KeyMap) ApplyBindings(saved map[string][]string) int {
	byName := make(map[string]d2enum.GameEvent, len(gameEventNames))
	for ev, n := range gameEventNames {
		byName[n] = ev
	}

	n := 0

	for name, keys := range saved {
		ev, ok := byName[name]
		if !ok || len(keys) != 2 {
			continue
		}

		p, pok := KeyByName(keys[0])
		s, sok := KeyByName(keys[1])

		if (keys[0] != "" && !pok) || (keys[1] != "" && !sok) {
			continue
		}

		if !pok {
			p = -1
		}

		if !sok {
			s = -1
		}

		km.SetPrimaryBinding(ev, p)
		km.SetSecondaryBinding(ev, s)

		n++
	}

	return n
}

// LoadSavedBindings applies the key bindings saved in the OD2 configuration (if any).
func (km *KeyMap) LoadSavedBindings() {
	if optionsBackend == nil {
		return
	}

	if saved := optionsBackend.Config().KeyBindings; len(saved) > 0 {
		km.ApplyBindings(saved)
	}
}

// SaveBindings writes the key map to config.json (no-op without an options backend).
func (km *KeyMap) SaveBindings() error {
	if optionsBackend == nil {
		return nil
	}

	cfg := optionsBackend.Config()
	cfg.KeyBindings = km.ExportBindings()

	return cfg.Save()
}
