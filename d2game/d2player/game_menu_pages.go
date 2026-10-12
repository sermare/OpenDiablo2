package d2player

// The pages of the escape menu of the original. UI_BuildGameMenu 0x479b60 points DAT_007b49c4 at one of a family
// of 20 byte headers {count, pitch, stripDY, pentDY, titleDY} (read from the exe file):
//
//	main        0x7127d8  3 entries, pitch 50, strip 39, pentagram 51, title 0
//	options     0x7127ec  5 entries, pitch 50, strip 39, pentagram 51, title 0
//	sound       0x715280  8 entries (0x715294: 7), pitch 45, strip 34, pentagram 49, title 36
//	video       0x71a258  8 entries (0x71a26c: 6), same numbers
//	automap     0x71ece0  7 entries, same numbers
//
// The entries of the sound / video / automap pages are the title row, the option rows and the previous menu row.
// The 8 / 8 / 7 pages are the Lord of Destruction ones: UI_BuildGameMenu initialises video 0x71a258 and automap
// 0x71ece0 only when MAIN_CheckExpansionMpq is true, and the 7-entry sound 0x715294 and 6-entry video 0x71a26c
// pages (no automap page) otherwise (VERIFIED, w25-input). The Go pages are the 8 / 8 / 7 variants. The page header values are the same arguments UI_DrawGameMenu 0x47a1d0 reads (count, pitch, strip dy,
// pentagram dy); the fifth int (36) is added to the y of the title row's strip and text (UI_DrawGameMenuTitle 0x47a050 reads header +0x10 and adds it to the row y; VERIFIED, w44).

// GameMenuPage describes one page of the menu.
type GameMenuPage struct {
	Name                                  string
	Total, Pitch, StripDY, PentDY, TitleY int
	// FirstActionRow is the visible row of the first actionable element in the Go page (1 when a title row comes first).
	FirstActionRow int
}

// Pages of the exe (the larger variant where the exe has two).
var (
	GameMenuMain    = GameMenuPage{"main", 3, 50, 39, 51, 0, 0}     //nolint:gochecknoglobals
	GameMenuOptions = GameMenuPage{"options", 5, 50, 39, 51, 0, 0}  //nolint:gochecknoglobals
	GameMenuSound   = GameMenuPage{"sound", 8, 45, 34, 49, 36, 1}   //nolint:gochecknoglobals
	GameMenuVideo   = GameMenuPage{"video", 8, 45, 34, 49, 36, 1}   //nolint:gochecknoglobals
	GameMenuAutomap = GameMenuPage{"automap", 7, 45, 34, 49, 36, 1} //nolint:gochecknoglobals
)

// RowTop is the top of visible row visIdx of the page: pitch*visIdx + (H-0x50)/2 - pitch*total/2.
func (p GameMenuPage) RowTop(m Mode, visIdx int) int {
	return p.Pitch*visIdx + (m.H-gameMenuHeightCut)/2 - (p.Pitch*p.Total)/2
}

// Plan places every row and its pentagrams (skullW: widest frame of the pentagram picture). total overrides
// p.Total when the Go page has other rows than the exe page (OD2 additions); pass 0 to use the exe count.
func (p GameMenuPage) Plan(m Mode, total, skullW int) []GameMenuPlacement {
	if total <= 0 {
		total = p.Total
	}

	q := p
	q.Total = total
	plan := make([]GameMenuPlacement, total)

	for i := range plan {
		top := q.RowTop(m, i)
		l, r, _ := m.GameMenuPentagrams(top, skullW)
		plan[i] = GameMenuPlacement{RowTop: top, LeftX: l, RightX: r, PentY: top + p.PentDY}
	}

	return plan
}

// gameMenuPageFor picks the exe page of a Go layout and the number of rows it really has (title row included).
func gameMenuPageFor(id layoutID, elements int) (GameMenuPage, int) {
	switch id {
	case soundOptionsLayoutID:
		return GameMenuSound, elements + 1
	case videoOptionsLayoutID:
		return GameMenuVideo, elements + 1
	case automapOptionsLayoutID:
		return GameMenuAutomap, elements + 1
	case optionsLayoutID:
		return GameMenuOptions, elements
	}

	return GameMenuMain, elements
}
