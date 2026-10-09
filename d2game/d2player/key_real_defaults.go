package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2key"
)

// realCommandEvents maps real default.key command ids to OpenDiablo2 game
// events. Only ids that d2key.Commands names are present; the mapping is as
// good as those names (see d2key.Confidence) and is NOT used to change the
// game's defaults: it exists to compare the two tables.
var realCommandEvents = map[uint32]d2enum.GameEvent{
	0: d2enum.ToggleCharacterPanel, 1: d2enum.ToggleInventoryPanel, 2: d2enum.TogglePartyPanel,
	3: d2enum.ToggleMessageLog, 4: d2enum.ToggleQuestLog, 5: d2enum.ToggleChatBox,
	6: d2enum.ToggleHelpScreen, 7: d2enum.ToggleAutomap, 12: d2enum.ToggleSkillTreePanel,
	13: d2enum.ToggleRightSkillSelector,
	14: d2enum.UseSkill1, 15: d2enum.UseSkill2, 16: d2enum.UseSkill3, 17: d2enum.UseSkill4,
	18: d2enum.UseSkill5, 19: d2enum.UseSkill6, 20: d2enum.UseSkill7, 21: d2enum.UseSkill8,
	22: d2enum.ToggleBelts,
	23: d2enum.UseBeltSlot1, 24: d2enum.UseBeltSlot2, 25: d2enum.UseBeltSlot3, 26: d2enum.UseBeltSlot4,
	27: d2enum.SayHelp, 28: d2enum.SayFollowMe, 29: d2enum.SayThisIsForYou, 30: d2enum.SayThanks,
	31: d2enum.SaySorry, 32: d2enum.SayBye, 33: d2enum.SayNowYouDie, 55: d2enum.SayRetreat,
	34: d2enum.HoldRun, 35: d2enum.ToggleRunWalk, 36: d2enum.HoldStandStill,
	37: d2enum.HoldShowGroundItems, 38: d2enum.ClearScreen,
	39: d2enum.SelectPreviousSkill, 40: d2enum.SelectNextSkill,
	41: d2enum.ClearMessages, 42: d2enum.TakeScreenShot, 43: d2enum.HoldShowPortraits,
	44: d2enum.SwapWeapons, 45: d2enum.ToggleMiniMap, 54: d2enum.ToggleHirelingPanel,
	56: d2enum.ToggleGameMenu,
}

// vkToKey converts a Windows virtual-key code (as stored in default.key) to
// an OpenDiablo2 key; ok is false for unbound or unsupported codes.
func vkToKey(vk uint16) (d2enum.Key, bool) {
	switch {
	case vk >= '0' && vk <= '9':
		return d2enum.Key0 + d2enum.Key(vk-'0'), true
	case vk >= 'A' && vk <= 'Z':
		return d2enum.KeyA + d2enum.Key(vk-'A'), true
	case vk >= 0x70 && vk <= 0x7b: // VK_F1..VK_F12
		return d2enum.KeyF1 + d2enum.Key(vk-0x70), true
	case vk >= 0x60 && vk <= 0x69: // VK_NUMPAD0..9
		return d2enum.KeyKP0 + d2enum.Key(vk-0x60), true
	}

	special := map[uint16]d2enum.Key{
		0x0d: d2enum.KeyEnter, 0x09: d2enum.KeyTab, 0x1b: d2enum.KeyEscape, 0x20: d2enum.KeySpace,
		0x10: d2enum.KeyShift, 0x11: d2enum.KeyControl, 0x12: d2enum.KeyAlt,
		0x2c: d2enum.KeyPrintScreen, 0xc0: d2enum.KeyTilde,
		0x103: d2enum.KeyMouseWheelUp, 0x104: d2enum.KeyMouseWheelDown,
	}

	k, ok := special[vk]

	return k, ok
}

// RealDefaultBindings converts a parsed default.key to OpenDiablo2 bindings
// for every command that has a mapping. Unbound or unknown codes become -1.
func RealDefaultBindings(f *d2key.File) map[d2enum.GameEvent]KeyBinding {
	out := make(map[d2enum.GameEvent]KeyBinding, len(realCommandEvents))

	conv := func(vk uint16) d2enum.Key {
		if k, ok := vkToKey(vk); ok {
			return k
		}

		return -1
	}

	for id, event := range realCommandEvents {
		rec, ok := f.Lookup(id)
		if !ok {
			continue
		}

		out[event] = KeyBinding{Primary: conv(rec.Primary), Secondary: conv(rec.Secondary)}
	}

	return out
}
