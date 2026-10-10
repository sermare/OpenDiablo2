package d2drlg

// This file is generated from the emulator measurements (gen_gates4.py): the number of level-seed steps
// DRLG_FilterPresetObjects (0x66a230) consumes when the preset DS1 is loaded while the level is generated,
// per DS1 file. Numbers only. The count is a property of the file (one step per gated monster / object
// record, see drlgpop.Filter); files not listed consume none.

var ds1GateTable = map[string]int{
	"act1/barracks/jailsetheme.ds1":   1,
	"act1/crypt/cryptcountess1.ds1":   1,
	"act1/crypt/cryptcountess2.ds1":   2,
	"act1/outdoors/cott4a.ds1":        1,
	"act2/outdoors/ruindarkelder.ds1": 4,
	"act2/outdoors/viper1.ds1":        2,
	"act2/tomb/tombnsewarpprev2.ds1":  4,
	"act2/tomb/tombnswwarpprev.ds1":   13,
	"act2/town/lutn.ds1":              2,
	"act2/town/lutw.ds1":              2,
	"act3/kurast/burbs08x08_1.ds1":    2,
	"act3/kurast/burbs08x16_2.ds1":    1,
	"act3/kurast/burbs16x08_2.ds1":    2,
	"act3/kurast/burbs16x16_2.ds1":    1,
	"act3/kurast/burbs16x16_3.ds1":    1,
	"act3/kurast/metro16x16_3.ds1":    1,
	"act3/kurast/slums08x08_2.ds1":    1,
	"act3/kurast/slums16x16_0.ds1":    1,
	"act3/kurast/slums16x16_2.ds1":    1,
	"act3/sewer/sewertreasure1.ds1":   9,
	"act3/temple/temple1.ds1":         4,
	"act3/temple/temple11.ds1":        1,
	"act3/temple/temple2.ds1":         22,
	"act3/temple/temple4.ds1":         3,
	"act3/temple/temple5.ds1":         13,
	"act3/temple/temple6.ds1":         8,
	"act3/temple/temple7.ds1":         6,
	"act3/temple/temple8.ds1":         10,
	"act3/temple/temple9.ds1":         7,
	"act3/travincal/mephnwarpd.ds1":   3,
}
