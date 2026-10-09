package drlgoutdoor

// Act 3 (jungle and Kurast) level program: DRLG_GenerateAct3Outdoors (0x6824a0)
// and its helpers, drlg-act23-outdoor.md section 4.

func b2i(b bool) int {
	if b {
		return 1
	}

	return 0
}

func (l *Level) put(x, y, def int) { l.placePreset(x, y, def, -1, 0) }

// lowerKurastWalls is DRLG_DrawLowerKurastWalls (0x681b40).
func (l *Level) lowerKurastWalls(flip int) {
	edi, ebp := l.W-1, l.H-1
	mid := cdiv(edi, 2)

	gap := edi - 1
	if flip != 0 {
		gap = 1
	}

	for x := 1; x < edi; x++ {
		l.put(x, 0, 0x25d+8*b2i(x == gap))
	}

	for x := 1; x < edi; x += 1 + b2i(x == mid) {
		l.put(x, ebp, 0x25e+8*b2i(x == mid))
	}

	for y := 1; y < ebp; y++ {
		l.put(edi, y, 0x25f)
		l.put(0, y, 0x260)
	}

	l.put(0, 0, 0x262)
	l.put(edi, 0, 0x261)
	l.put(0, ebp, 0x264)
	l.put(edi, ebp, 0x263)
}

// bazaarWalls is DRLG_DrawKurastBazaarWalls (0x681ca0).
func (l *Level) bazaarWalls(flip int) {
	ebx, ebp := l.W-1, l.H-1
	g14, g10 := ebx-1, 1

	if flip != 0 {
		g14, g10 = 1, ebx-1
	}

	for x := 1; x < ebx; x++ {
		l.put(x, 0, 0x26b+8*b2i(x == g10))
		l.put(x, ebp, 0x26c+8*b2i(x == g14))
	}

	for y := 1; y < ebp; y++ {
		l.put(ebx, y, 0x26d)
		l.put(0, y, 0x26e)
	}

	l.put(0, 0, 0x270)
	l.put(ebx, 0, 0x26f)
	l.put(0, ebp, 0x272)
	l.put(ebx, ebp, 0x271)
}

// upperKurastWalls is DRLG_DrawUpperKurastWalls (0x681de0).
func (l *Level) upperKurastWalls(flip int) {
	edi, ebp := l.W-1, l.H-1
	mid := cdiv(edi, 2)

	g14 := 1
	if flip != 0 {
		g14 = edi - 1
	}

	for x := 1; x < edi; x += 1 + b2i(x == mid) {
		l.put(x, 0, 0x27c+8*b2i(x == mid))
	}

	for x := 1; x < edi; x++ {
		l.put(x, ebp, 0x27d+8*b2i(x == g14))
	}

	for y := 1; y < ebp; y++ {
		l.put(edi, y, 0x27e)
		l.put(0, y, 0x27f)
	}

	l.put(0, 0, 0x281)
	l.put(edi, 0, 0x280)
	l.put(0, ebp, 0x283)
	l.put(edi, ebp, 0x282)
}

// scatterRange is DRLG_ScatterPresetRange (0x681f40): shuffle ALL cells with 2n
// level-seed rolls, then give every cell a Def in [lo, hi] and place it if it
// fits; stop after max placements (max <= 0: never).
func (l *Level) scatterRange(lo, hi, maxCount int) {
	l.trace("rangescatter")

	n := l.W * l.H
	if n == 0 {
		return
	}

	lst := make([]pt, n)
	for k := range lst {
		lst[k] = pt{k % l.W, k / l.W}
	}

	for i := 0; i < n; i++ {
		a := int(l.Seed.Roll(int32(n)))
		b := int(l.Seed.Roll(int32(n)))
		lst[a], lst[b] = lst[b], lst[a]
	}

	cnt := 0

	for _, c := range lst {
		d := lo + int(l.Seed.Roll(int32(hi-lo+1)))
		if !l.fits(c.x, c.y, d, 0, 0xf) {
			continue
		}

		l.put(c.x, c.y, d)

		if cnt++; maxCount > 0 && cnt >= maxCount {
			return
		}
	}
}

// jungleCells is DRLG_PlaceJungleCells (0x681950): the 2x6 grid of 32x32 pieces
// of a jungle level from the array the world placer computed.
func (l *Level) jungleCells() error {
	arr, cnt := l.Params.Jungle.Arr, l.Params.Jungle.Count
	id := l.Params.ID

	sel := 2
	if cnt == 3 {
		sel = 6
	}

	r := int(l.Seed.Roll(int32(sel)))
	used, idx := 0, 0

	for row := 0; row < 6; row++ {
		switch {
		case id == 76 && row == 5:
			l.placePreset(0, row*4, 0x23d, b2i(arr[11] == 0), 0)

			idx += 2
		case id == 78 && row == 0:
			l.placePreset(0, 0, 0x23e, b2i(arr[1] == 0), 0)

			idx += 2
		default:
			for col := 0; col < 2; col++ {
				v := arr[idx]
				idx++
				file := -1

				if v > 0x23e {
					if used >= 3 {
						return GameError{0x47}
					}

					v += t37f0[id]
					file = t38d8[used+3*r]
					used++
				}

				if v != 0 {
					l.placePreset(col*4, row*4, v, file, 0)
				}
			}
		}
	}

	return nil
}

// kurastSpecials is DRLG_PlaceKurastSpecials (0x6821e0) for levels 76..82
// (the jungle levels place nothing here, but the stage runs).
func (l *Level) kurastSpecials() {
	l.trace("kurast_specials")

	flip := l.Params.Flip
	id := l.Params.ID

	switch id {
	case 79:
		l.lowerKurastWalls(flip)
	case 80:
		l.bazaarWalls(flip)
	case 81:
		l.upperKurastWalls(flip)
	}

	ebx, edi := l.W-4, l.H-4

	switch id {
	case 79:
		l.placeRandom(0x277, 0, 0, 0xf)
		l.scatterRange(0x26a, 0x26a, 4)
		l.scatterRange(0x268, 0x269, 0)
		l.scatterRange(0x267, 0x267, 0)
	case 80:
		l.placePreset(3, 3, 0x275, 0, 0)
		l.placePreset(ebx, 3, 0x275, 1, 0)
		l.placeRandom(0x276, 0, 0, 0xf)
		l.placeRandom(0x276, 1, 0, 0xf)
		l.placeRandom(0x277, 0, 0, 0xf)
		l.scatterRange(0x27b, 0x27b, 4)
		l.scatterRange(0x279, 0x27a, 0)
		l.scatterRange(0x278, 0x278, 0)
	case 81:
		l.placePreset(3, edi, 0x286, 0, 0)
		l.placePreset(ebx, edi, 0x286, 1, 0)
		l.placeRandom(0x287, 0, 0, 0xf)
		l.placeRandom(0x287, 1, 0, 0xf)
		l.placeRandom(0x277, 0, 0, 0xf)
		l.scatterRange(0x28b, 0x28b, 4)
		l.scatterRange(0x289, 0x28a, 0)
		l.scatterRange(0x288, 0x288, 0)
	case 82:
		l.placePreset(0, 0, 0x28c, 0, 0)
	}
}

// travincal is DRLG_PlaceTravincalPresets (0x682400).
func (l *Level) travincal() {
	l.trace("travincal")

	if l.Params.ID != 83 {
		return
	}

	for _, p := range [6][3]int{{0, 0, 0x28d}, {2, 0, 0x28e}, {6, 0, 0x28f}, {0, 4, 0x290}, {2, 4, 0x291}, {6, 4, 0x292}} {
		l.put(p[0], p[1], p[2])
	}
}

// generateAct3 is DRLG_GenerateAct3Outdoors (0x6824a0) minus the room pass.
// Act 3 never draws boundary edges: the whole level is tiled by presets.
func (l *Level) generateAct3() error {
	l.trace("markexit")
	l.markExits()
	l.trace("jungle_place")

	if l.Params.ID >= 76 && l.Params.ID <= 78 {
		if err := l.jungleCells(); err != nil {
			return err
		}
	}

	l.kurastSpecials()
	l.travincal()

	return nil
}
