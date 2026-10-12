package d2audio

// Wave 48 sound audit (Game.exe 1.14b, Ghidra D2_1_14b). Everything here was read in the exe unless marked UNVERIFIED.

// Footsteps are played on animation frames, not by a timer (UNITSND_PlayModeEndSound 0x4c7a20, VERIFIED). The
// function runs for the walk and run modes of players (and some monster modes). With the animation length T
// (frames, 8.8 fixed), the speed s (frames per tick, 8.8) and N steps per cycle (2 for a player; FsCnt of the
// MonSounds row for a monster, which also gives a phase offset FsOff), a step plays when
//   - N < 2, or the ticks since the last step are at least 2/3 of T/(N*s), AND
//   - the circular distance (modulo T/N) of the current frame to the phase offset is strictly smaller than that of
//     the frame one tick ahead and one tick back (the frame nearest to a step point).
// For a player with an 8 frame walk this is frames 0 and 4. A step after a pause longer than 1.5 intervals is
// played at 160/255 of the volume (the volume column of the exe: UNVERIFIED how it maps onto the engine volume,
// the port does not scale it).

const footstepFixed = 8

// ModularAbsDistance is MATH_ModularAbsDistance 0x4e1110: the shortest distance of a and b on a circle of length m.
func ModularAbsDistance(a, b, m int) int {
	if m <= 0 {
		return 0
	}

	a, b = ((a%m)+m)%m, ((b%m)+m)%m

	d1 := b - a
	if d1 < 0 {
		d1 = -d1
	}

	d2 := b - a - m
	if d2 < 0 {
		d2 = -d2
	}

	d3 := a - b - m
	if d3 < 0 {
		d3 = -d3
	}

	best := d1
	if d2 < best {
		best = d2
	}

	if d3 < best {
		best = d3
	}

	return best
}

// FootstepFrameDue says whether the frame (0-based, of count frames) is a step frame for steps steps per cycle and
// the phase offset in frames (FsOff, whole frames; 0 for heroes). steps below 1 never steps.
func FootstepFrameDue(frame, count, steps, offset int) bool {
	if steps < 1 || count < 1 {
		return false
	}

	m := (count << footstepFixed) / steps
	if m < 1 {
		return false
	}

	f := frame << footstepFixed
	o := offset << footstepFixed
	one := 1 << footstepFixed

	cur := ModularAbsDistance(f, o, m)

	return cur < ModularAbsDistance(f+one, o, m) && cur < ModularAbsDistance(f-one, o, m)
}

// FootstepSpacingOK is the minimum spacing between two steps (only with more than one step per cycle): at least
// 2/3 of the step interval, count/steps frames, must have passed.
func FootstepSpacingOK(framesSince, count, steps int) bool {
	if steps < 2 {
		return true
	}

	interval := count / steps

	return framesSince*3 >= interval*2
}

// FootstepTracker decides, frame after frame, when a walking unit steps.
type FootstepTracker struct {
	last   int
	since  int
	primed bool
}

// Reset forgets the walk (the unit left the walk or run mode): the next step is not held back by the spacing.
func (t *FootstepTracker) Reset() { t.primed = false; t.since = 1 << 20 }

// Observe feeds the current animation frame (call it every game frame) and reports whether a step plays now.
// A frame is only judged once, when it differs from the one seen last.
func (t *FootstepTracker) Observe(frame, count, steps, offset int) bool {
	if !t.primed {
		t.primed = true
		t.last = -1
		t.since = 1 << 20
	}

	if frame == t.last {
		return false
	}

	if t.last >= 0 && t.since < 1<<20 {
		d := frame - t.last
		if d <= 0 {
			d += count // the animation wrapped
		}

		t.since += d
	}

	t.last = frame

	if !FootstepFrameDue(frame, count, steps, offset) || !FootstepSpacingOK(t.since, count, steps) {
		return false
	}

	t.since = 0

	return true
}

// Block, kick and the extra swing modes (UNITSND_HandleUnitModeSound 0x4c9030, player branch, VERIFIED).
const (
	// BlockWeaponHandle is Sounds.txt row 0x18e, played when the hero enters the block mode (BL, mode 9) with a
	// weapon of hit class 1..9 (UNITSND_GetHitClassSoundId 0x4c7310); BlockArrowHandle is row 0x191 for the bow and
	// crossbow classes 10 and 11. Hit class 0 (no class) plays nothing.
	BlockWeaponHandle = "block_weapon_1"
	BlockArrowHandle  = "block_arrow_1"
	// KickHandle is row 0xff, played on the kick mode (KK, mode 12). The Assassin passes a delay of 4 ticks, every
	// other class none.
	KickHandle            = "weapon_kick_1"
	KickAssassinDelay     = 4
	classNumberBarbarian  = 4
	classNumberAssassin   = 6
	hitClassNamesBowFirst = 10
)

// BlockHandle returns the block sound for the weapons.txt "hit class" of the hero's weapon ("" plays nothing). An
// unarmed hero has class hth (1 in the exe's numbering, which plays the weapon block).
func BlockHandle(hitClass string) string {
	switch hitClass {
	case "", "hth", "1hss", "1hsl", "2hss", "2hsl", "1ht", "2ht", "club", "staf":
		return BlockWeaponHandle
	case "bow", "xbow":
		return BlockArrowHandle
	}

	return ""
}

// KickDelay is the delay of the kick sound for a hero class number (0 amazon .. 6 assassin).
func KickDelay(classNumber int) int {
	if classNumber == classNumberAssassin {
		return KickAssassinDelay
	}

	return 0
}

// SwingInSkillMode says whether a hero skill mode S3 (3) or S4 (4) swings the weapon like an attack: the Barbarian
// in S3 and the Assassin in S4 (the exe's mode 0xf / 0x10 branches). Other classes play nothing there.
func SwingInSkillMode(skillMode, classNumber int) bool {
	return skillMode == 3 && classNumber == classNumberBarbarian || skillMode == 4 && classNumber == classNumberAssassin
}
