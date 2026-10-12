package d2gamescreen

import "sync"

var (
	hirelingPopupMu   sync.Mutex
	hirelingPopupSeen func() bool
	hirelingPopupMark func()
	hirelingPopupLive bool // the flag of this run when nothing is wired (tests)
)

// SetHirelingPopupFlag wires the registry value PopupHireling of the exe ("Diablo II" key) to the settings: seen reads it, mark
// writes it. The exe reads it when a hire succeeded (UI_InitHirelingPopupSetting 0x48fe60) and, while it is 0, opens the
// mercenary screen by itself once (UI_DrawHirelingPortraitStrip 0x4904f0 calls UI_SetPanelState(0); UNVERIFIED that panel 0 is
// the mercenary screen) and then stores 1.
func SetHirelingPopupFlag(seen func() bool, mark func()) {
	hirelingPopupMu.Lock()
	defer hirelingPopupMu.Unlock()

	hirelingPopupSeen, hirelingPopupMark = seen, mark
}

// takeHirelingPopup says whether the first-hire screen is due now and records that it was shown.
func takeHirelingPopup() bool {
	hirelingPopupMu.Lock()
	defer hirelingPopupMu.Unlock()

	seen := hirelingPopupLive
	if hirelingPopupSeen != nil {
		seen = hirelingPopupSeen()
	}

	if seen {
		return false
	}

	hirelingPopupLive = true

	if hirelingPopupMark != nil {
		hirelingPopupMark()
	}

	return true
}
