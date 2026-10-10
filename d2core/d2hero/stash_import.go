package d2hero

import (
	"fmt"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Size of the stash grid the game shows (the expansion's Big Bank).
const (
	stashCols = 6
	stashRows = 8
)

// StashEnvVar names a PlugY stash file (.sss/.hss/.d2x) whose items are added
// read-only to the stash of every imported hero; the file is never written.
const StashEnvVar = "OD2_STASH_IMPORT"

// MergeStash adds the items of a stash file to the stash page of c. An item
// keeps its stash position when the cells are free, otherwise it goes to the
// first free place; items that do not fit, are ears or have no OpenDiablo2 record
// are skipped and returned with a reason. size gives an item's grid size.
func MergeStash(c *HeroContainers, items []d2s.Item, known func(code string) bool,
	size func(code string) (w, h int)) (added int, skipped []string) {
	var used [stashRows][stashCols]bool

	mark := func(x, y, w, h int, v bool) {
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				used[y+dy][x+dx] = v
			}
		}
	}

	fits := func(x, y, w, h int) bool {
		if x < 0 || y < 0 || x+w > stashCols || y+h > stashRows {
			return false
		}

		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				if used[y+dy][x+dx] {
					return false
				}
			}
		}

		return true
	}

	for _, s := range c.Page(PageStash) {
		w, h := size(s.Code)
		if w < 1 || h < 1 {
			w, h = 1, 1
		}

		if fits(s.X, s.Y, w, h) {
			mark(s.X, s.Y, w, h, true)
		}
	}

	for i := range items {
		it := &items[i]
		// the item's own position is only a hint: force it onto the stash page
		stored, skip := storedFromD2S(it, PageStash, known)
		if skip != "" {
			skipped = append(skipped, fmt.Sprintf("%s: %s", trimCode(it.Code), skip))
			continue
		}

		w, h := size(stored.Code)
		if w < 1 || h < 1 {
			w, h = 1, 1
		}

		x, y, ok := stored.X, stored.Y, fits(stored.X, stored.Y, w, h)

		for yy := 0; !ok && yy < stashRows; yy++ {
			for xx := 0; !ok && xx < stashCols; xx++ {
				if fits(xx, yy, w, h) {
					x, y, ok = xx, yy, true
				}
			}
		}

		if !ok {
			skipped = append(skipped, fmt.Sprintf("%s: stash is full", stored.Code))
			continue
		}

		mark(x, y, w, h, true)

		stored.X, stored.Y = x, y
		c.Items = append(c.Items, stored)
		added++
	}

	return added, skipped
}

// ImportStashFile reads a PlugY stash file (read-only) and merges its items into
// the stash of the hero. Failures are returned and leave the hero unchanged.
func (f *HeroStateFactory) ImportStashFile(state *HeroState, path string) (added int, skipped []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}

	tables, err := f.loadD2SItemTables()
	if err != nil {
		return 0, nil, err
	}

	stash, err := d2s.ParseStash(data, tables)
	if err != nil {
		return 0, nil, err
	}

	if state.Containers == nil {
		state.Containers = &HeroContainers{Items: []StoredItem{}}
	}

	known := func(code string) bool { return f.asset.Records.Item.All[code] != nil }
	size := func(code string) (int, int) {
		if r := f.asset.Records.Item.All[code]; r != nil {
			return r.InventoryWidth, r.InventoryHeight
		}

		return 1, 1
	}

	added, skipped = MergeStash(state.Containers, stash.Items(), known, size)

	return added, skipped, nil
}
