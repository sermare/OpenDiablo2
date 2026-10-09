package drlgoutdoor

// Act 2 (desert) level program: DRLG_GenerateAct2Outdoors (0x6829f0) and its
// helpers, drlg-act23-outdoor.md section 3. Verified against the emulated game
// in the notes; the Go port is diffed against the golden numbers of that run.

// act2Notch places the two Desert Border door pieces at the middle of an exit
// notch (DRLG_DrawBoundaryEdges, act index 1 branch). The edge direction is
// (dx, dy); the notch centre is (mx, my).
func (l *Level) act2Notch(mx, my, dx, dy int) {
	idx := dx + 2 + 2*dy
	l.placePreset(mx, my, t6f26dc[idx][0], -1, 0)
	l.placePreset(mx+abs(dx), my+abs(dy), t6f26dc[idx][1], -1, 0)
}

// variantSet is DRLG_PlacePresetVariantSet (0x6824c0): start = Roll(n), then n
// presets of the list are placed cyclically, either one random file each
// (perFile false) or every file of each Def in order.
func (l *Level) variantSet(list []int, perFile bool) {
	l.trace("variant")

	n := len(list)
	idx := int(l.Seed.Roll(int32(n)))

	for k := 0; k < n; k++ {
		d := list[idx]

		if !perFile {
			l.placeRandom(d, -1, 0, 0xf)
		} else {
			_, _, files := l.prest(d)
			for f := 0; f < files; f++ {
				l.placeRandom(d, f, 0, 0xf)
			}
		}

		idx = (idx + 1) % n
	}
}

// act2TownTransition is DRLG_PlaceAct2TownTransition (0x6825b0).
func (l *Level) act2TownTransition() {
	l.trace("luttrans")

	for _, n := range l.Params.Neighbors {
		if n.Level != 0x28 {
			continue
		}

		if n.Dir == 3 {
			l.placePreset(0, l.H-1, 0x16b, -1, 0)
		} else {
			l.placePreset(l.W-1, 0, 0x16a, -1, 0)
		}

		return
	}
}

// act2DesertCap is DRLG_PlaceAct2DesertCapPresets (0x682610).
func (l *Level) act2DesertCap() {
	l.trace("cap8")

	row := l.Seed.Step() & 7
	for _, r := range cap8[row] {
		l.placePreset(r[2], r[3], r[0], r[1], 0)
	}
}

// act2LvlSubs is DRLG_ApplyAct2LvlSubTypes (0x682680).
func (l *Level) act2LvlSubs() {
	l.trace("lvlsub3")

	for _, t := range [3]int{2, 1, 3} {
		l.trace("lvlsub")
		l.applyLvlSub(t, 0x16c)
	}
}

// act2Center is DRLG_PlaceAct2LevelCenterPreset (0x6826c0).
func (l *Level) act2Center() error {
	l.trace("rand1")

	def := map[int]int{0x29: 0x184, 0x2a: 0x184, 0x2b: 0x186, 0x2c: 0x19c, 0x2d: 0x185}[l.Params.ID]
	if !l.placeRandom(def, -1, 0, 0xf) {
		return GameError{0xaa}
	}

	return nil
}

func (l *Level) waypointStage() {
	l.trace("wpmark")
	l.markWaypoint()
}

func (l *Level) shrineStage() {
	l.trace("scatter")
	l.scatterShrines(5)
}

// generateAct2 is DRLG_GenerateAct2Outdoors (0x6829f0) minus the room pass.
func (l *Level) generateAct2() error {
	l.trace("markexit")
	l.markExits()
	l.trace("edges")
	l.drawBoundaryEdges()

	switch l.Params.ID {
	case 0x29: // Rocky Waste
		l.act2TownTransition()
		l.act2LvlSubs()

		if err := l.act2Center(); err != nil {
			return err
		}

		l.shrineStage()
		l.trace("tail41")
		l.variantSet([]int{0x18b, 0x19b, 0x191, 0x192, 0x18f, 0x18e, 0x193}, false)
	case 0x2a: // Dry Hills
		l.act2DesertCap()
		l.act2LvlSubs()

		if err := l.act2Center(); err != nil {
			return err
		}

		l.waypointStage()
		l.shrineStage()
		l.trace("tail42")
		l.variantSet([]int{0x18b, 0x19b, 0x190, 0x18e}, false)
		l.variantSet([]int{0x194, 0x195, 0x196, 0x197}, true)
	case 0x2b: // Far Oasis
		l.act2DesertCap()
		l.act2LvlSubs()

		if err := l.act2Center(); err != nil {
			return err
		}

		l.variantSet([]int{0x18c, 0x18d}, false)
		l.waypointStage()
		l.shrineStage()
		l.trace("tail43")
		l.variantSet([]int{0x19b, 0x18f, 0x18e, 0x193}, false)
		l.variantSet([]int{0x18b}, true)
		l.variantSet([]int{0x18b}, true)
	case 0x2c: // Lost City
		l.act2DesertCap()
		l.act2LvlSubs()

		if err := l.act2Center(); err != nil {
			return err
		}

		l.trace("pre44")
		l.variantSet([]int{0x19d, 0x198, 0x199, 0x19a}, false)
		l.waypointStage()
		l.shrineStage()
		l.trace("tail44")
		l.variantSet([]int{0x18b, 0x190, 0x18e, 0x194, 0x195}, false)
		l.variantSet([]int{0x19b}, true)
		l.variantSet([]int{0x19b}, true)
	case 0x2d: // Valley of Snakes: only the centre preset
		return l.act2Center()
	case 0x2e: // Canyon of the Magi
		l.trace("pre46")

		for _, r := range pre46 {
			l.placePreset(r[2], r[3], r[0], r[1], 0)
		}

		l.placePreset(4, 4, 0x18a, -1, 0)
		l.act2LvlSubs()
		l.shrineStage()
		l.trace("tail46")
		l.variantSet([]int{0x191, 0x192, 0x196, 0x197, 0x193}, false)
		l.variantSet([]int{0x188, 0x189}, true)
	}

	return nil
}
