package drlgoutdoor

import "strings"

// ds1Gates are the Act 3 preset files that consume level-seed steps when their
// DS1 is loaded (drlg-act23-outdoor.md 4.6; measured with the emulator over
// every file the Act 2/3 outdoor presets loaded, so UNVERIFIED for files that
// were never drawn: they are assumed to cost 0). Act 2 has none.
var ds1Gates = map[string]int{
	"act3/kurast/burbs08x08_1.ds1": 2,
	"act3/kurast/burbs16x08_2.ds1": 2,
	"act3/kurast/burbs08x16_2.ds1": 1,
	"act3/kurast/burbs16x16_2.ds1": 1,
	"act3/kurast/burbs16x16_3.ds1": 1,
	"act3/kurast/metro16x16_3.ds1": 1,
	"act3/kurast/slums08x08_2.ds1": 1,
	"act3/kurast/slums16x16_0.ds1": 1,
	"act3/kurast/slums16x16_2.ds1": 1,
}

// ds1GateSteps returns the gate step count of a LvlPrest file name.
func ds1GateSteps(file string) int {
	return ds1Gates[strings.ToLower(NormalizePrestFile(file))]
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
