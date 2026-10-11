package d2display

import "strings"

// BarFill says how the area beside the 800 pixel wide bottom bar is filled on a wider screen
// (OD2_BAR_FILL, docs/DISPLAY.md).
type BarFill int

const (
	// BarFillNone leaves the world visible on both sides of the bar (default, as before).
	BarFillNone BarFill = iota
	// BarFillBlack draws a black strip of the bar's height on both sides.
	BarFillBlack
	// BarFillTile repeats the outermost columns of the bar art on both sides.
	BarFillTile
)

// ParseBarFill reads "none", "black" or "tile" (case-insensitive). Anything else, including the empty
// string, is BarFillNone and ok is false for the unknown values (the empty string is ok).
func ParseBarFill(v string) (mode BarFill, ok bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "none":
		return BarFillNone, true
	case "black":
		return BarFillBlack, true
	case "tile":
		return BarFillTile, true
	}

	return BarFillNone, false
}

// BarSpan is one piece of the fill, in column space (x = 0 is the left edge of the 800 pixel bar):
// draw columns [SrcOff, SrcOff+W) of the tile at x = X.
type BarSpan struct {
	X, W   int
	SrcOff int
	Left   bool // the piece belongs to the left side (its tile is the left edge of the bar art)
}

// BarFillSpans lays out the tiles that fill the screen beside the bar. screen is the logical size, tileW the
// width of one tile (the outer columns of the bar art). The spans on each side start at the bar edge and go
// outwards in whole tiles; the last (outermost) one is clipped to the screen edge. The left spans are
// ordered from the bar outwards, then the right ones. At screen widths of 800 or less there is nothing to
// fill.
func BarFillSpans(screen Size, tileW int) []BarSpan {
	screen = Clamp(screen)

	ox, _ := Origin(screen, AnchorBottom)
	if ox <= 0 || tileW <= 0 {
		return nil
	}

	rightExtra := screen.W - ox - BaseW // differs from ox by one pixel when W-800 is odd

	var spans []BarSpan

	for covered := 0; covered < ox; covered += tileW {
		w := tileW
		if covered+w > ox {
			w = ox - covered
		}

		// the tile nearest to the bar edge is whole; the clipped outermost one keeps the columns nearest
		// to the bar so that the seam stays continuous
		spans = append(spans, BarSpan{X: -covered - w, W: w, SrcOff: tileW - w, Left: true})
	}

	for covered := 0; covered < rightExtra; covered += tileW {
		w := tileW
		if covered+w > rightExtra {
			w = rightExtra - covered
		}

		spans = append(spans, BarSpan{X: BaseW + covered, W: w, SrcOff: 0})
	}

	return spans
}

// VideoRect is where a cinematic of vw x vh pixels is drawn on a screen: scaled to the largest size that fits the
// whole screen at the video's own aspect ratio (letterbox or pillarbox) and centred; the rest of the screen is
// black (the caller clears it). Sizes are in screen pixels. An empty video gives the empty rectangle.
func VideoRect(screen Size, vw, vh int) (x, y, w, h int) {
	if vw <= 0 || vh <= 0 || screen.W <= 0 || screen.H <= 0 {
		return 0, 0, 0, 0
	}

	// compare the aspect ratios with integers: screen.W/screen.H >= vw/vh means the height is the limit
	if screen.W*vh >= vw*screen.H {
		h = screen.H
		w = (vw*screen.H + vh/2) / vh
	} else {
		w = screen.W
		h = (vh*screen.W + vw/2) / vw
	}

	return (screen.W - w) / 2, (screen.H - h) / 2, w, h
}
