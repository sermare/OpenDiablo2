package drlgoutdoor

// Act 4 outdoor levels 104 Outer Steppes, 105 Plains of Despair, 106 City of
// the Damned and 108 Chaos Sanctuary: DRLG_GenerateAct4Outdoors (0x6818d0),
// drlg-act45-outdoor.md section 4.1. Everything else (boundary polygon, edge
// presets, LvlSub matcher, room pass) is the Act 1 code with Act 4 tables.

// Tables of the exe (0x6f3854, 0x6f3860, 0x6f3870).
var (
	act4PitDef  = [3]int{828, 832, 832} // "Pits" base Def per level 104..106
	act4MesaDef = [3]int{812, 817, 823} // "Mesa" base Def per level
	chaosGrid   = [25]int{
		836, 836, 836, 836, 836,
		836, 836, 861, 836, 836,
		836, 858, 862, 859, 836,
		836, 836, 860, 836, 836,
		836, 836, 857, 836, 836,
	}
)

func (l *Level) generateAct4() {
	id := l.Params.ID

	l.markExits()

	if id != 108 {
		l.drawBoundaryEdges()
	}

	switch {
	case id >= 104 && id <= 106:
		// DRLG_PlaceAct4CornerPresets (0x6816a0): the Fortress transition piece
		if l.OdFlags&0x400000 != 0 {
			l.placePreset(0, 1, 0x31e, -1, 0)
		}

		if l.OdFlags&0x800000 != 0 {
			l.placePreset(0, 4, 0x31e, -1, 0)
		}

		for t := 1; t <= 3; t++ {
			l.applyLvlSub(t, 0x31f)
		}

		l.scatterAct4()
	case id == 108:
		// DRLG_PlaceChaosSanctuaryGrid (0x681880)
		for i, def := range chaosGrid {
			l.placePreset((i%5)*3, (i/5)*3, def, -1, 0)
		}
	}
}

// scatterAct4 is DRLG_ScatterAct4RandomPresets (0x6816e0).
func (l *Level) scatterAct4() {
	id := l.Params.ID
	rnd := func(def int) { l.placeRandom(def, -1, 0, 0xf) }
	d, b := act4MesaDef[id-104], act4PitDef[id-104]

	if id == 106 {
		rnd(0x32b)
	}

	rnd(d)

	for _, k := range []int{1, 1, 2, 2, 3, 3} {
		rnd(d + k)
	}

	if id == 105 {
		rnd(0x336)
	}

	for i := 0; i < 4; i++ {
		rnd(d + 4)
	}

	rnd(b)

	for _, k := range []int{1, 1, 2, 2, 3, 3, 3, 3} {
		rnd(b + k)
	}
}
