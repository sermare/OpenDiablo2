package d2player

// uiSoundHook plays a Sounds.txt handle for a UI interaction through the game's
// sound engine (so it is logged and mixed like every other sound). Without a
// hook the old direct provider click is used.
var uiSoundHook func(handle, kind string)

// SetUISoundHook sets the function that plays UI sounds.
func SetUISoundHook(fn func(handle, kind string)) { uiSoundHook = fn }

// playUISound plays the named UI sound and reports whether the engine took it.
// Rows (Game.exe 1.14b): cursor_button_click (4) for buttons pressed in the mini
// panel, inventory, stat and skill buttons (UI_HandleMiniPanelMouseDown
// 0x47ad40, INV_HandleMouseDown 0x48d6c0, UI_HandleStatButtonMouseDown
// 0x4a2e12, UI_HandleSkillButtonMouseDown 0x4a2eb2); cursor_pass (1) when a
// game menu selection moves; cursor_switch (6) on a skill tree tab change.
// There is no panel open/close sound: UI_SetPanelState 0x4515f0, UI_OnPanelOpened 0x450de0 and UI_OnPanelClosed
// 0x4511a0 (the only code that runs when a panel opens or closes) make no sound call (VERIFIED, w44).
func playUISound(handle, kind string) bool {
	if uiSoundHook == nil {
		return false
	}

	uiSoundHook(handle, kind)

	return true
}
