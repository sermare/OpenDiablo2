package d2herostats

// Stamina drain and recovery, from the exe (observations only):
// 0x0057d220 (drain per running step) and 0x0057e4f0 (recovery per tick).
// Stamina (stat 10/11) is held in 1/256 units, so the functions below work in
// those raw units. The per-tick rate of both routines is UNVERIFIED: the engine
// assumes the original's 25 Hz frame (a level 1 Barbarian then runs dry in
// ~23 s, which matches the old empirical note in d2mapentity).

// FramesPerSecond is the assumed tick rate of the two routines (UNVERIFIED).
const FramesPerSecond = 25

// Unit modes the recovery distinguishes (player modes of the exe).
const (
	ModeNeutral     = 1
	ModeWalk        = 2
	ModeRun         = 3
	ModeGetHit      = 4
	ModeTownNeutral = 5
	ModeTownWalk    = 6
)

// StaminaDrainPerStep is the raw (1/256) stamina one running step costs
// (VERIFIED 0x0057d220, read from the disassembly):
//
//	d = charstats RunDrain * 2
//	if body armor is worn: d *= weight/10 + 1   (the armor's weight column)
//	d -= d * stat0x9a / 100                     (stamina drain percent)
//	d = max(d, 1)
//
// In town the routine returns before any of this (no drain). bodyWeight < 0
// means no armor is worn. Truncation is toward zero.
func StaminaDrainPerStep(runDrain, bodyWeight, drainPct int, inTown bool) int {
	if inTown {
		return 0
	}

	d := runDrain * 2
	if bodyWeight >= 0 {
		d *= bodyWeight/10 + 1
	}

	if drainPct != 0 {
		d -= drainPct * d / 100
	}

	if d < 1 {
		d = 1
	}

	return d
}

// StaminaRegenPerTick is the raw (1/256) stamina recovered in one tick
// (VERIFIED 0x0057e4f0): base = maxStamina(raw) >> shift with shift 8 for a
// standing hero (modes 1 and 5) and 9 for walking in town (mode 6) or in the
// field (mode 2, which only recovers while the stamina is at least 1 whole
// point); the other modes (running, hit, attacks...) only recover when stat
// 0x1c reaches 1000. Stat 0x1c (stamina recovery percent) adds base*pct/100.
// The caller caps the result at the maximum (ClampStamina).
func StaminaRegenPerTick(maxRaw, curRaw, mode, recoveryPct int) int {
	if curRaw >= maxRaw {
		return 0
	}

	shift := 8

	switch mode {
	case ModeNeutral, ModeTownNeutral:
	case ModeTownWalk:
		shift = 9
	case ModeWalk:
		if curRaw&^0xff == 0 {
			return 0
		}

		shift = 9
	default:
		if recoveryPct < 1000 {
			return 0
		}
	}

	base := maxRaw >> uint(shift)
	if recoveryPct != 0 {
		base += base * recoveryPct / 100
	}

	return base
}

// ClampStamina applies a recovery with the cap at the maximum.
func ClampStamina(curRaw, gain, maxRaw int) int {
	if curRaw+gain > maxRaw {
		return maxRaw
	}

	return curRaw + gain
}
