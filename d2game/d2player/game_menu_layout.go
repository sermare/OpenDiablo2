package d2player

// Layout of the escape ("game") menu of the original (UI_DrawGameMenu 0x47a1d0, UI_HitTestGameMenuItem
// 0x479310, UI_BuildGameMenu 0x479b60). Pure functions; the Go menu is built from gui layouts and is not
// yet driven by them (see d2-re-notes ui-panels-next.md).
//
// Two static menu descriptors exist at 0x7127d8 (main menu: 3 entries, options: 5 entries), both
// with pitch 50, strip y offset 39 and pentagram y offset 51 (verified by reading the bytes).

// Constants of the descriptors.
const (
	gameMenuPitch      = 50
	gameMenuStripDY    = 39
	gameMenuPentDY     = 51
	gameMenuPentDX     = 0xf9 // distance of the pentagrams from the screen centre
	gameMenuHeightCut  = 0x50 // the vertical centre is taken inside H-0x50
	gameMenuPentFrames = 8
)

// GameMenuRowTop returns the y of the visible row number visIdx (0-based among visible rows) of a menu with
// total entries (hidden ones count for centring), as the original computes it:
// pitch*visIdx + (H-0x50)/2 - pitch*total/2.
func (m Mode) GameMenuRowTop(total, visIdx int) int {
	return gameMenuPitch*visIdx + (m.H-gameMenuHeightCut)/2 - (gameMenuPitch*total)/2
}

// GameMenuPentagrams returns the x of the left and right pentagram and their y for the row whose top is
// rowTop. skullW is the widest frame of the pentagram picture (the original measures it at build time).
func (m Mode) GameMenuPentagrams(rowTop, skullW int) (leftX, rightX, y int) {
	return m.W/2 - skullW - gameMenuPentDX, m.W/2 + gameMenuPentDX, rowTop + gameMenuPentDY
}

// GameMenuPentFrames returns the animation frame of the left and right pentagram for a tick counter
// (the original advances the counter every 50 ms and wraps at 8; left plays backwards).
func GameMenuPentFrames(tick int) (left, right int) {
	t := ((tick % gameMenuPentFrames) + gameMenuPentFrames) % gameMenuPentFrames
	if t == 0 {
		return 0, 0
	}

	return gameMenuPentFrames - t, t
}
