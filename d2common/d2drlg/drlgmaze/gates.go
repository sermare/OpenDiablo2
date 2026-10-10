package drlgmaze

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"

// defaultGateSteps is the number of level-seed steps DRLG_FilterPresetObjects
// (0x66a230) consumes each time a preset room is created from a file; see
// d2drlg.DS1GateSteps (measured by emulating the real game, one step per
// gated monster or object record of the file).
func defaultGateSteps(file string) int {
	return d2drlg.DS1GateSteps(file)
}
