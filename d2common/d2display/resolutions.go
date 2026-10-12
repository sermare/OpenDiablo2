package d2display

import (
	"fmt"
	"sort"
	"strings"
)

// presetSizes are the window sizes offered by the VIDEO RESOLUTION row (when the monitor is big enough).
var presetSizes = []Size{ //nolint:gochecknoglobals // constant table
	{800, 600}, {1024, 768}, {1152, 864}, {1280, 720}, {1280, 800}, {1366, 768}, {1440, 900}, {1512, 982},
	{1600, 900}, {1600, 1200}, {1680, 1050}, {1920, 1080}, {1920, 1200}, {2560, 1080}, {2560, 1440}, {3440, 1440},
}

// ResolutionChoices lists the sizes of the resolution row: the presets that fit the monitor (monitor.W == 0
// means unknown), the monitor size itself and the current size, sorted by area, without duplicates.
func ResolutionChoices(monitor, current Size) []Size {
	set := map[Size]bool{}

	for _, s := range presetSizes {
		if monitor.W == 0 || (s.W <= monitor.W && s.H <= monitor.H) {
			set[s] = true
		}
	}

	if monitor.W > 0 && monitor.H > 0 {
		set[Clamp(monitor)] = true
	}

	if current.W > 0 && current.H > 0 {
		set[Clamp(current)] = true
	}

	out := make([]Size, 0, len(set))
	for s := range set {
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.W*a.H != b.W*b.H {
			return a.W*a.H < b.W*b.H
		}

		return a.W < b.W
	})

	return out
}

// String is the text of the resolution row, "1920X1080".
func (s Size) String() string { return fmt.Sprintf("%dX%d", s.W, s.H) }

// ParseRow reads the text of the resolution row back.
func ParseRow(v string) (Size, error) { return Parse(strings.ToLower(v)) }
