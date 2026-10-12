package d2hero

import (
	"os"
	"sort"
	"time"
)

// LastPlayed is when the character was last played: the write time of the .d2s it lives in (the import writes a
// fresh .od2 every time, so that file's time says nothing), else of the .od2; zero when neither can be read.
func (h *HeroState) LastPlayed() time.Time {
	paths := []string{h.FilePath}
	if h.Imported != nil && h.Imported.Source != "" {
		paths = []string{h.Imported.Source, h.FilePath}
	}

	for _, p := range paths {
		if p == "" {
			continue
		}

		if fi, err := os.Stat(p); err == nil {
			return fi.ModTime()
		}
	}

	return time.Time{}
}

// SortByLastPlayed orders the character list like the expansion character select does
// (CHARSEL_AddCharacterEntry 0x434060, VERIFIED: entries are inserted by the file time of the save, newest first;
// only the classic game keeps the file order). The sort is stable, so equal times keep the directory order.
func SortByLastPlayed(states []*HeroState) {
	times := make(map[*HeroState]time.Time, len(states))
	for _, s := range states {
		times[s] = s.LastPlayed()
	}

	sort.SliceStable(states, func(i, j int) bool { return times[states[i]].After(times[states[j]]) })
}
