// Package d2display is the single display model of the game: the logical screen size, the
// integer UI scale and the anchors of the original 800x600 interface inside a larger screen.
//
// Rules (docs/DISPLAY.md):
//   - The world (map, camera, culling, mouse picking) uses the whole W x H logical screen.
//   - The original interface keeps its native 800x600 pixel size and lives in a "column" of
//     800x600 pixels. In the game the column is centred horizontally and stands on the bottom
//     edge of the screen (the bottom bar is centred at the bottom, left panels fill the left half of
//     the column, right panels the right half). Menus and other full screen dialogs centre the
//     column on both axes.
//   - Input handlers receive cursor positions in column space (see ToColumn); the world
//     picking converts back with ToScreen.
//
// This package has no dependencies so that it can be used by every layer.
package d2display

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Native size of the original interface.
const (
	BaseW = 800
	BaseH = 600
)

// DefaultW and DefaultH are the default logical size of the desktop app (MacBook screen).
const (
	DefaultW = 1512
	DefaultH = 982
)

// MaxUIScale is the largest integer UI scale.
const MaxUIScale = 4

// Size is a logical screen size in pixels.
type Size struct{ W, H int }

// Base is the legacy 800x600 size.
var Base = Size{BaseW, BaseH} //nolint:gochecknoglobals // constant

// Anchor says where the 800x600 column of the interface stands inside the screen.
type Anchor int

const (
	// AnchorBottom centres the column horizontally and puts it on the bottom edge (in game).
	AnchorBottom Anchor = iota
	// AnchorCenter centres the column on both axes (menus, loading, character select).
	AnchorCenter
)

// Clamp raises a size to the 800x600 minimum.
func Clamp(s Size) Size {
	if s.W < BaseW {
		s.W = BaseW
	}

	if s.H < BaseH {
		s.H = BaseH
	}

	return s
}

// Logical is the logical screen size for a window (or full screen) of winW x winH pixels drawn at an
// integer UI scale: the window size divided by the scale, never below 800x600.
func Logical(winW, winH, scale int) Size {
	if scale < 1 {
		scale = 1
	}

	if scale > MaxUIScale {
		scale = MaxUIScale
	}

	return Clamp(Size{winW / scale, winH / scale})
}

// Origin is the top-left corner of the 800x600 column inside a screen of size s.
func Origin(s Size, a Anchor) (x, y int) {
	s = Clamp(s)
	x = (s.W - BaseW) / 2

	if a == AnchorCenter {
		return x, (s.H - BaseH) / 2
	}

	return x, s.H - BaseH
}

// Parse reads "WxH" (also "W*H" or "W,H").
func Parse(v string) (Size, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, sep := range []string{"x", "*", ","} {
		if a, b, ok := strings.Cut(v, sep); ok {
			w, err1 := strconv.Atoi(strings.TrimSpace(a))
			h, err2 := strconv.Atoi(strings.TrimSpace(b))

			if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
				break
			}

			return Size{w, h}, nil
		}
	}

	return Size{}, fmt.Errorf("display size %q: want WxH, for example 1920x1080", v)
}

// Resolve picks the logical start size: the override (OD2_DISPLAY) wins over the configured size, which
// wins over the default. The result is never below 800x600 and the configured/default size is limited to
// the screen when one is given (screen.W == 0 means unknown). An unparsable override is reported and
// ignored.
func Resolve(override string, cfg Size, screen Size) (Size, error) {
	var err error

	s := Size{DefaultW, DefaultH}

	if strings.TrimSpace(override) != "" {
		var o Size

		if o, err = Parse(override); err == nil {
			return Clamp(o), nil // an explicit request is honoured as is
		}
	}

	if cfg.W > 0 && cfg.H > 0 {
		s = cfg
	}

	if screen.W > 0 && screen.H > 0 {
		if s.W > screen.W {
			s.W = screen.W
		}

		if s.H > screen.H {
			s.H = screen.H
		}
	}

	return Clamp(s), err
}

var (
	mu       sync.RWMutex
	cur      = Base
	uiScale  = 1
	inputOff [2]int
	anchor   = AnchorBottom
	fixed    bool
)

// SetFixed pins the logical size (OD2_DISPLAY): the window may then have any size, the picture is scaled to
// fit it. Tests rely on this to get the same logical screen on every monitor.
func SetFixed(f bool) {
	mu.Lock()
	fixed = f
	mu.Unlock()
}

// Fixed reports whether the logical size is pinned.
func Fixed() bool {
	mu.RLock()
	defer mu.RUnlock()

	return fixed
}

// Get is the current logical screen size.
func Get() Size {
	mu.RLock()
	defer mu.RUnlock()

	return cur
}

// W is the current logical width.
func W() int { return Get().W }

// H is the current logical height.
func H() int { return Get().H }

// Set stores the logical size (clamped to 800x600 at least) and updates the column origin.
func Set(s Size) {
	mu.Lock()
	cur = Clamp(s)
	inputOff[0], inputOff[1] = Origin(cur, anchor)
	mu.Unlock()
}

// Scale is the integer UI scale (window pixels per logical pixel).
func Scale() int {
	mu.RLock()
	defer mu.RUnlock()

	return uiScale
}

// SetScale sets the integer UI scale (1..MaxUIScale).
func SetScale(s int) {
	if s < 1 {
		s = 1
	}

	if s > MaxUIScale {
		s = MaxUIScale
	}

	mu.Lock()
	uiScale = s
	mu.Unlock()
}

// SetAnchor stores which anchor the column currently uses; the column origin follows it.
func SetAnchor(a Anchor) {
	mu.Lock()
	anchor = a
	inputOff[0], inputOff[1] = Origin(cur, a)
	mu.Unlock()
}

// CurrentAnchor is the anchor in use.
func CurrentAnchor() Anchor {
	mu.RLock()
	defer mu.RUnlock()

	return anchor
}

// ColumnOrigin is the current column origin (the offset between screen and column space).
func ColumnOrigin() (x, y int) {
	mu.RLock()
	defer mu.RUnlock()

	return inputOff[0], inputOff[1]
}

// ToColumn converts screen coordinates to column coordinates (what UI handlers receive).
func ToColumn(x, y int) (int, int) {
	ox, oy := ColumnOrigin()

	return x - ox, y - oy
}

// ToScreen converts column coordinates back to screen coordinates (for world picking and overlays).
func ToScreen(x, y int) (int, int) {
	ox, oy := ColumnOrigin()

	return x + ox, y + oy
}

// ScreenRectInColumn is the whole screen as a rectangle in column space: left, top, width, height.
// Tooltips and text clamp to it so that they may use the space beside the column.
func ScreenRectInColumn() (x, y, w, h int) {
	s := Get()
	ox, oy := ColumnOrigin()

	return -ox, -oy, s.W, s.H
}
