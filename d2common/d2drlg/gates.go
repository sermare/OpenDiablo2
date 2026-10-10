package d2drlg

import "strings"

// DS1GateSteps returns how many level-seed steps DRLG_FilterPresetObjects
// consumes when the preset DS1 file is loaded during level generation (the
// gated monster and object ids of the file, see the drlgpop package). file is
// a LvlPrest file name in any spelling (case, slashes, a leading
// data/global/tiles).
func DS1GateSteps(file string) int {
	f := strings.ToLower(strings.ReplaceAll(file, "\\", "/"))
	f = strings.TrimPrefix(f, "/")
	f = strings.TrimPrefix(f, "data/global/tiles/")

	return ds1GateTable[f]
}
