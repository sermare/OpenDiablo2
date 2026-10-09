package drlgmaze

import "strings"

// gateSteps maps a preset file (lower case, forward slashes) to the number of
// level-seed steps DRLG_FilterPresetObjects (0x66a230) consumes each time a
// preset room is created from it. Measured by emulating the real game over many
// seeds (constant per file); files not listed consume none.
var gateSteps = map[string]int{
	"act2/tomb/tombnsewarpprev2.ds1": 4,
	"act2/tomb/tombnswwarpprev.ds1":  13,
	"act3/travincal/mephnwarpd.ds1":  2,
}

func defaultGateSteps(file string) int {
	return gateSteps[strings.ToLower(strings.ReplaceAll(file, "\\", "/"))]
}
