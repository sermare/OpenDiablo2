package drlgoutdoor

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"

// ds1GateSteps returns the gate step count of a LvlPrest file name: the
// level-seed steps DRLG_FilterPresetObjects takes when the DS1 is loaded while
// the level is generated (d2drlg.DS1GateSteps, measured with the emulator).
func ds1GateSteps(file string) int {
	return d2drlg.DS1GateSteps(NormalizePrestFile(file))
}

// levelTypeFill is the end of DRLG_BuildOutdoorRoomGrids (0x6802b0): every
// grid-C cell without a floor/wall/orientation payload gets the level type's
// base value (drlg-act23-outdoor.md 3.5; Act 1 / LevelType 2 adds nothing).
func (b *roomBuilder) levelTypeFill() {
	var fill uint32

	switch b.l.LType {
	case 0x10:
		fill = 0x100
	case 0x15:
		fill = 0x120000
	case 0x16:
		fill = 0x100000
	case 0x1b:
		fill = 0xa00000
	case 0x1c:
		fill = 0x1600000
	case 0x1f:
		if b.l.Params.ID == 0x75 {
			fill = 0x600000
		}
	}

	if fill == 0 {
		return
	}

	c := b.g.C
	for i, v := range c.C {
		if v&0x3f0ff80 == 0 {
			c.C[i] = v | fill
		}
	}
}
