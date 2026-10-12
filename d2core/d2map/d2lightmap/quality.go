package d2lightmap

// Light quality level, LIGHT_GetQualityLevel 0x471460 (w37, read from the exe).
//
//	level 2 = occluded painter for kind 0 sources, level 1 = radial, level 0 = radial with caps
//
// Inputs: the fixed-mode toggle (0x4f27b0 returns DAT_0072c0a0; w44 VERIFIED: the options saver 0x478d70 sets it to 1
// for Light Quality option 0 and to 0 for options 1 and 2, and the main loop flips it), the raw quality setting
// (0x4c9980 returns DAT_0072902c, written by 0x4c9970 from the same options saver; the argument is not visible in
// the decompile) and the frame statistic (0x449050, w44 VERIFIED: frames per second averaged over a 3000 ms window by
// 0x448150). With the toggle 0 the level follows the frame stat: <= 9 gives 0,
// 10..12 gives 1, 13..15 keeps the previous level when it is above 0 (else 1), > 15 gives 2. With the
// toggle set the level is 0 when the raw setting is 0, otherwise it is capped at 1 by the same ladder.
// A change of level only takes effect once 2001 ms passed since the last accepted change.

// QualityState is the hysteresis memory (DAT_007ad704 level, DAT_007ad700 tick).
type QualityState struct {
	Level  int
	TickMS uint32
}

// Quality returns the light quality level for the inputs and updates the hysteresis state.
func (q *QualityState) Quality(toggle, raw, fps int, nowMS uint32) int {
	level := 2

	ladder := func() {
		switch {
		case fps <= 9:
			level = 0
		case fps > 15:
			// keeps level
		case fps > 12 && q.Level > 0:
			level = -q.Level // sentinel: returned as is below
		default:
			level = 1
		}
	}

	switch {
	case toggle == 0:
		ladder()
	case raw != 0:
		level = 1
		ladder()
	default:
		level = 0
	}

	if level < 0 {
		return -level
	}

	if level != q.Level && nowMS-q.TickMS < 2001 {
		return q.Level
	}

	q.Level, q.TickMS = level, nowMS

	return level
}

// RadialCap is the radius cap LIGHT_ApplyRadialLight 0x470520 applies at quality 0: a remote player's
// light (kind 0 not owned by the client) is limited to 16, a monster light (kind 1) to 8, and a kind 3
// light is not painted (ok false).
func RadialCap(kind int, ownPlayer bool, radius int) (r int, ok bool) {
	switch kind {
	case 0:
		if !ownPlayer && radius > 15 {
			return 16, true
		}
	case 1:
		return 8, true
	case 3:
		return 0, false
	}

	return radius, true
}
